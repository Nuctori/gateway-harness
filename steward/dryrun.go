package steward

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func DryRunProposal(s Spec, p Proposal, request []byte) (DryRunResult, error) {
	if err := ValidateProposal(s, p); err != nil {
		return DryRunResult{}, err
	}

	var obj map[string]any
	if err := json.Unmarshal(request, &obj); err != nil {
		return DryRunResult{}, fmt.Errorf("request must be a JSON object: %w", err)
	}

	result := DryRunResult{
		ProposalID: p.ID,
		Steward:    p.Steward,
		Hook:       p.Hook,
	}
	for _, output := range p.Outputs {
		switch output.Action {
		case "context.inject":
			patch, err := planContextInject(obj, output)
			if err != nil {
				return DryRunResult{}, err
			}
			result.RequestPatches = append(result.RequestPatches, patch)
			result.AppliedActions = append(result.AppliedActions, output.Action)
		case "ledger.artifact.create":
			result.Artifacts = append(result.Artifacts, DryRunRef{
				Type:        output.ArtifactType,
				ContentHash: output.ContentHash,
				Ref:         output.Ref,
			})
			result.AppliedActions = append(result.AppliedActions, output.Action)
		case "diagnosis.note.create":
			result.Diagnostics = append(result.Diagnostics, DryRunRef{
				NoteHash: output.NoteHash,
				Ref:      output.Ref,
				Severity: output.Severity,
			})
			result.AppliedActions = append(result.AppliedActions, output.Action)
		case "session.tags.update":
			result.SessionTags = append(result.SessionTags, output.Tags...)
			result.AppliedActions = append(result.AppliedActions, output.Action)
		case "goal.approve_complete", "goal.reject_complete", "goal.request_continue":
			result.GoalActions = append(result.GoalActions, DryRunGoal{
				Action:      output.Action,
				Reason:      output.Reason,
				Instruction: output.Instruction,
			})
			result.AppliedActions = append(result.AppliedActions, output.Action)
		default:
			return DryRunResult{}, fmt.Errorf("unsupported dry-run action %q", output.Action)
		}
	}
	return result, nil
}

func planContextInject(obj map[string]any, output Output) (DryRunPatch, error) {
	if messages, ok := obj["messages"].([]any); ok {
		return newInjectPatch("messages", insertMessageIndex(messages, output.Position), output), nil
	}
	if input, ok := obj["input"].([]any); ok {
		return newInjectPatch("input", insertResponsesInputIndex(input, output.Position), output), nil
	}
	return DryRunPatch{}, fmt.Errorf("context.inject requires request.messages or request.input array")
}

func newInjectPatch(target string, index int, output Output) DryRunPatch {
	hash := sha256.Sum256([]byte(output.Text))
	return DryRunPatch{
		Action:       output.Action,
		Target:       target,
		InsertIndex:  index,
		Role:         output.Role,
		Position:     output.Position,
		ContentHash:  fmt.Sprintf("sha256:%x", hash),
		ContentChars: len([]rune(output.Text)),
	}
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
	if position == "before_messages" {
		return 0
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
