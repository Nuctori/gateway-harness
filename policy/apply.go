package policy

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Apply(p Policy, request []byte, options ApplyOptions) (ApplyResult, error) {
	if err := Validate(p); err != nil {
		return ApplyResult{}, err
	}
	if strings.TrimSpace(options.Hook) == "" {
		return ApplyResult{}, fmt.Errorf("hook is required")
	}
	if !SupportedHooks[options.Hook] || options.Hook == "*" {
		return ApplyResult{}, fmt.Errorf("unsupported explicit hook %q", options.Hook)
	}

	var obj map[string]any
	if err := json.Unmarshal(request, &obj); err != nil {
		return ApplyResult{}, fmt.Errorf("request must be a JSON object: %w", err)
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model, _ = obj["model"].(string)
	}
	if strings.TrimSpace(model) == "" {
		return ApplyResult{}, fmt.Errorf("model is required in request or options")
	}

	result := ApplyResult{
		Hook:            options.Hook,
		Model:           model,
		EstimatedTokens: options.EstimatedTokens,
		Trace: ApplyTrace{
			Summary: TraceSummary{ContentMode: "redacted"},
		},
	}
	continuityDrop := options.ContextContinuityDrop || options.Hook == "context.continuity_drop.detected"
	for _, program := range p.Programs {
		if !programMatchesModel(program, model) {
			continue
		}
		programMatched := false
		for _, step := range program.Steps {
			if !stepMatchesHook(step, options.Hook, continuityDrop) || !conditionMatches(step.When, model, options.EstimatedTokens, continuityDrop) {
				continue
			}
			if !programMatched {
				result.MatchedPrograms = append(result.MatchedPrograms, program.Name)
				programMatched = true
			}
			for _, action := range step.Do {
				switch action.Action {
				case "context.inject", "context.inject_ledger_summary":
					operation, err := applyContextInject(program.Name, obj, action)
					if err != nil {
						return ApplyResult{}, err
					}
					result.Trace.Operations = append(result.Trace.Operations, operation)
					result.AppliedActions = append(result.AppliedActions, action.Action)
				case "context.truncate":
					result.SkippedActions = append(result.SkippedActions, Skipped{
						Program: program.Name,
						Action:  action.Action,
						Reason:  "context.truncate is destructive and is not applied by policy.Apply",
					})
				default:
					result.SkippedActions = append(result.SkippedActions, Skipped{
						Program: program.Name,
						Action:  action.Action,
						Reason:  "unsupported apply action",
					})
				}
			}
		}
	}

	encoded, err := json.Marshal(obj)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("encode applied request: %w", err)
	}
	result.Trace.Summary.Ops = len(result.Trace.Operations)
	result.Request = encoded
	return result, nil
}

func applyContextInject(program string, obj map[string]any, action Action) (TraceOperation, error) {
	if messages, ok := obj["messages"].([]any); ok {
		index := insertMessageIndex(messages, action.Position)
		obj["messages"] = insertAt(messages, index, contextItem(action))
		return newTraceOperation(program, "messages", index, action), nil
	}
	if input, ok := obj["input"].([]any); ok {
		index := insertResponsesInputIndex(input, action.Position)
		obj["input"] = insertAt(input, index, contextItem(action))
		return newTraceOperation(program, "input", index, action), nil
	}
	return TraceOperation{}, fmt.Errorf("context.inject requires request.messages or request.input array")
}

func contextItem(action Action) map[string]any {
	return map[string]any{
		"role":    action.Role,
		"content": action.Text,
	}
}

func insertAt(items []any, index int, item any) []any {
	if index < 0 {
		index = 0
	}
	if index > len(items) {
		index = len(items)
	}
	out := make([]any, 0, len(items)+1)
	out = append(out, items[:index]...)
	out = append(out, item)
	out = append(out, items[index:]...)
	return out
}

func newTraceOperation(program string, target string, index int, action Action) TraceOperation {
	patch := newInjectPatch(program, target, index, action)
	return TraceOperation{
		Program:      program,
		Op:           "insert",
		Action:       action.Action,
		Target:       patch.Target,
		InsertIndex:  patch.InsertIndex,
		Role:         patch.Role,
		Source:       patch.Source,
		LedgerRef:    patch.LedgerRef,
		ArtifactRefs: append([]string(nil), patch.ArtifactRefs...),
		ContentHash:  patch.ContentHash,
		ContentChars: patch.ContentChars,
		Reason:       patch.Reason,
	}
}
