package policy

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path"
	"strings"
)

func DryRun(p Policy, request []byte, options DryRunOptions) (DryRunResult, error) {
	if err := Validate(p); err != nil {
		return DryRunResult{}, err
	}
	if strings.TrimSpace(options.Hook) == "" {
		return DryRunResult{}, fmt.Errorf("hook is required")
	}
	if !SupportedHooks[options.Hook] || options.Hook == "*" {
		return DryRunResult{}, fmt.Errorf("unsupported explicit hook %q", options.Hook)
	}

	var obj map[string]any
	if err := json.Unmarshal(request, &obj); err != nil {
		return DryRunResult{}, fmt.Errorf("request must be a JSON object: %w", err)
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model, _ = obj["model"].(string)
	}
	if strings.TrimSpace(model) == "" {
		return DryRunResult{}, fmt.Errorf("model is required in request or options")
	}

	result := DryRunResult{
		Hook:            options.Hook,
		Model:           model,
		EstimatedTokens: options.EstimatedTokens,
	}
	for _, program := range p.Programs {
		if !programMatchesModel(program, model) {
			continue
		}
		programMatched := false
		for _, step := range program.Steps {
			if !stepMatchesHook(step, options.Hook) || !conditionMatches(step.When, model, options.EstimatedTokens) {
				continue
			}
			if !programMatched {
				result.MatchedPrograms = append(result.MatchedPrograms, program.Name)
				programMatched = true
			}
			for _, action := range step.Do {
				switch action.Action {
				case "context.inject", "context.inject_ledger_summary":
					patch, err := planContextInject(program.Name, obj, action)
					if err != nil {
						return DryRunResult{}, err
					}
					result.RequestPatches = append(result.RequestPatches, patch)
					result.AppliedActions = append(result.AppliedActions, action.Action)
				case "context.truncate":
					result.SkippedActions = append(result.SkippedActions, Skipped{
						Program: program.Name,
						Action:  action.Action,
						Reason:  "context.truncate is destructive and is not applied by dry-run",
					})
				default:
					result.SkippedActions = append(result.SkippedActions, Skipped{
						Program: program.Name,
						Action:  action.Action,
						Reason:  "unsupported dry-run action",
					})
				}
			}
		}
	}
	return result, nil
}

func programMatchesModel(program Program, model string) bool {
	for _, selector := range program.Models {
		if globMatches(selector, model) {
			return true
		}
	}
	return false
}

func stepMatchesHook(step Step, hook string) bool {
	for _, candidate := range EffectiveHooks(step) {
		if candidate == "*" || candidate == hook {
			return true
		}
	}
	return false
}

func conditionMatches(condition Condition, model string, estimatedTokens int) bool {
	if strings.TrimSpace(condition.ModelMatches) != "" && !globMatches(condition.ModelMatches, model) {
		return false
	}
	if condition.EstimatedTokensGT > 0 {
		return estimatedTokens > condition.EstimatedTokensGT
	}
	return true
}

func globMatches(pattern string, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}

func planContextInject(program string, obj map[string]any, action Action) (DryRunPatch, error) {
	if messages, ok := obj["messages"].([]any); ok {
		return newInjectPatch(program, "messages", insertMessageIndex(messages, action.Position), action), nil
	}
	if input, ok := obj["input"].([]any); ok {
		return newInjectPatch(program, "input", insertResponsesInputIndex(input, action.Position), action), nil
	}
	return DryRunPatch{}, fmt.Errorf("context.inject requires request.messages or request.input array")
}

func newInjectPatch(program string, target string, index int, action Action) DryRunPatch {
	hash := sha256.Sum256([]byte(action.Text))
	return DryRunPatch{
		Program:      program,
		Action:       action.Action,
		Target:       target,
		InsertIndex:  index,
		Role:         action.Role,
		Position:     action.Position,
		Source:       injectSource(action),
		LedgerRef:    action.LedgerRef,
		ArtifactRefs: append([]string(nil), action.ArtifactRefs...),
		ContentHash:  fmt.Sprintf("sha256:%x", hash),
		ContentChars: len([]rune(action.Text)),
		Reason:       action.Reason,
	}
}

func injectSource(action Action) string {
	if strings.TrimSpace(action.Source) != "" {
		return action.Source
	}
	if action.Action == "context.inject_ledger_summary" {
		return "ledger.summary"
	}
	return ""
}

func insertResponsesInputIndex(items []any, position string) int {
	prefixEnd := 0
	for prefixEnd < len(items) && isResponsesProtocolItem(items[prefixEnd]) {
		prefixEnd++
	}
	if prefixEnd == 0 {
		return insertMessageIndex(items, position)
	}
	return prefixEnd + insertMessageIndex(items[prefixEnd:], position)
}

func insertMessageIndex(items []any, position string) int {
	if position == "after_existing_system" {
		insertAt := 0
		for insertAt < len(items) && isSystemLike(items[insertAt]) {
			insertAt++
		}
		return insertAt
	}
	return 0
}

func isSystemLike(value any) bool {
	obj, ok := value.(map[string]any)
	if !ok {
		return false
	}
	role, _ := obj["role"].(string)
	return role == "system" || role == "developer"
}

func isResponsesProtocolItem(value any) bool {
	obj, ok := value.(map[string]any)
	if !ok {
		return false
	}
	kind, _ := obj["type"].(string)
	switch kind {
	case "item_reference", "function_call", "function_call_output", "reasoning":
		return true
	default:
		return false
	}
}
