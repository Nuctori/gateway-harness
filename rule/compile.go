package rule

import (
	"fmt"
	"strings"

	"github.com/Nuctori/gateway-harness/policy"
	"github.com/Nuctori/gateway-harness/steward"
)

func Compile(d Document) (policy.Policy, error) {
	if err := Validate(d); err != nil {
		return policy.Policy{}, err
	}

	programs := make([]policy.Program, 0, len(d.Rules))
	for _, r := range d.Rules {
		if r.Operation.Type != OperationInjectCapsule {
			return policy.Policy{}, fmt.Errorf("rule %q operation %q cannot compile to policy; use compile-rule-stewards", r.Name, r.Operation.Type)
		}
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

func CompileStewards(d Document) ([]steward.Spec, error) {
	if err := Validate(d); err != nil {
		return nil, err
	}

	specs := []steward.Spec{}
	for _, r := range d.Rules {
		if r.Operation.Type != OperationAskSteward {
			continue
		}
		spec := steward.Spec{
			Version:        d.Version,
			Name:           firstNonEmpty(r.Operation.StewardName, r.Name),
			StewardModel:   r.Operation.StewardModel,
			Hooks:          append([]string(nil), r.Trigger.Hooks...),
			Inputs:         append([]string(nil), r.Operation.Inputs...),
			AllowedActions: append([]string(nil), r.Operation.AllowedActions...),
			ArtifactTypes:  append([]string(nil), r.Operation.ArtifactTypes...),
			RequiredGuards: append([]string(nil), r.Operation.RequiredGuards...),
		}
		if err := steward.Validate(spec); err != nil {
			return nil, fmt.Errorf("rule %q steward: %w", r.Name, err)
		}
		specs = append(specs, spec)
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("no ask_steward rules to compile")
	}
	return specs, nil
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
