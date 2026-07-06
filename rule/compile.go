package rule

import (
	"strings"

	"github.com/Nuctori/gateway-harness/policy"
)

func Compile(d Document) (policy.Policy, error) {
	if err := Validate(d); err != nil {
		return policy.Policy{}, err
	}

	programs := make([]policy.Program, 0, len(d.Rules))
	for _, r := range d.Rules {
		action := compileAction(r.Operation, r.Audit)
		step := policy.Step{
			Hooks: append([]string(nil), r.Trigger.Hooks...),
			When: policy.Condition{
				ModelMatches:          strings.TrimSpace(r.Scope.ModelMatches),
				ContextContinuityDrop: r.Trigger.ContinuityDrop,
			},
			Do: []policy.Action{action},
		}
		programs = append(programs, policy.Program{
			Name:   r.Name,
			Models: append([]string(nil), r.Scope.Models...),
			Tags:   append([]string(nil), r.Tags...),
			Steps:  []policy.Step{step},
		})
	}

	compiled := policy.Policy{
		Version:  d.Version,
		Programs: programs,
	}
	if err := policy.Validate(compiled); err != nil {
		return policy.Policy{}, err
	}
	return compiled, nil
}

func compileAction(operation Operation, audit Audit) policy.Action {
	action := policy.Action{
		Action:   "context.inject",
		Role:     operation.Role,
		Position: operation.Position,
		Text:     operation.Text,
		Reason:   operation.Reason,
	}
	if strings.TrimSpace(audit.LedgerRef) != "" {
		action.Action = "context.inject_ledger_summary"
		action.Source = "ledger.summary"
		action.LedgerRef = audit.LedgerRef
		action.ArtifactRefs = append([]string(nil), audit.ArtifactRefs...)
	}
	return action
}
