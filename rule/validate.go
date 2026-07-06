package rule

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Nuctori/gateway-harness/policy"
)

const (
	OperationInjectCapsule = "inject_capsule"
)

func Decode(r io.Reader) (Document, error) {
	var d Document
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&d); err != nil {
		return Document{}, err
	}
	return d, nil
}

func Validate(d Document) error {
	if len(d.Rules) == 0 {
		return fmt.Errorf("rules needs at least one rule")
	}
	for i, r := range d.Rules {
		if strings.TrimSpace(r.Name) == "" {
			return fmt.Errorf("rule %d name is required", i)
		}
		if err := validateTrigger(r.Trigger); err != nil {
			return fmt.Errorf("rule %q trigger: %w", r.Name, err)
		}
		if err := validateScope(r.Scope); err != nil {
			return fmt.Errorf("rule %q scope: %w", r.Name, err)
		}
		if err := validateOperation(r.Operation); err != nil {
			return fmt.Errorf("rule %q operation: %w", r.Name, err)
		}
		if err := validateAudit(r.Audit); err != nil {
			return fmt.Errorf("rule %q audit: %w", r.Name, err)
		}
	}
	return nil
}

func validateTrigger(trigger Trigger) error {
	if len(trigger.Hooks) == 0 {
		return fmt.Errorf("hooks needs at least one hook")
	}
	for _, hook := range trigger.Hooks {
		hook = strings.TrimSpace(hook)
		if hook == "" {
			return fmt.Errorf("hook is required")
		}
		if !policy.SupportedHooks[hook] {
			return fmt.Errorf("unsupported hook %q", hook)
		}
	}
	return nil
}

func validateScope(scope Scope) error {
	if len(scope.Models) == 0 {
		return fmt.Errorf("models needs at least one model selector")
	}
	for _, model := range scope.Models {
		if strings.TrimSpace(model) == "" {
			return fmt.Errorf("model selector is required")
		}
	}
	return nil
}

func validateOperation(operation Operation) error {
	if operation.Type != OperationInjectCapsule {
		return fmt.Errorf("unsupported operation %q", operation.Type)
	}
	if strings.TrimSpace(operation.Role) == "" {
		return fmt.Errorf("role is required")
	}
	switch operation.Position {
	case "after_existing_system", "before_messages":
	default:
		return fmt.Errorf("unsupported position %q", operation.Position)
	}
	if strings.TrimSpace(operation.Text) == "" {
		return fmt.Errorf("text is required")
	}
	return nil
}

func validateAudit(audit Audit) error {
	if len(audit.ArtifactRefs) > 0 && strings.TrimSpace(audit.LedgerRef) == "" {
		return fmt.Errorf("artifact_refs require ledger_ref")
	}
	for _, ref := range audit.ArtifactRefs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("artifact_ref is required")
		}
	}
	return nil
}

func Summarize(d Document) Summary {
	summary := Summary{Rules: len(d.Rules)}
	hooks := map[string]bool{}
	operations := map[string]bool{}
	for _, r := range d.Rules {
		for _, hook := range r.Trigger.Hooks {
			hooks[hook] = true
		}
		operations[r.Operation.Type] = true
	}
	for hook := range hooks {
		summary.Hooks = append(summary.Hooks, hook)
	}
	for operation := range operations {
		summary.Operations = append(summary.Operations, operation)
	}
	sort.Strings(summary.Hooks)
	sort.Strings(summary.Operations)
	return summary
}
