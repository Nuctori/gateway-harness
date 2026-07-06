package rule

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Nuctori/gateway-harness/policy"
	"github.com/Nuctori/gateway-harness/steward"
)

const (
	OperationInjectCapsule = "inject_capsule"
	OperationAskSteward    = "ask_steward"
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
		if err := validateOperation(r.Operation, r.Trigger); err != nil {
			return fmt.Errorf("rule %q operation: %w", r.Name, err)
		}
		if err := validateAudit(r.Audit); err != nil {
			return fmt.Errorf("rule %q audit: %w", r.Name, err)
		}
		if r.Operation.Type == OperationAskSteward && hasAuditProvenance(r.Audit) {
			return fmt.Errorf("rule %q audit: ask_steward must not include rule audit provenance; use steward outputs and guards", r.Name)
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

func validateOperation(operation Operation, trigger Trigger) error {
	switch operation.Type {
	case OperationInjectCapsule:
		return validateInjectCapsuleOperation(operation)
	case OperationAskSteward:
		return validateAskStewardOperation(operation, trigger)
	default:
		return fmt.Errorf("unsupported operation %q", operation.Type)
	}
}

func validateInjectCapsuleOperation(operation Operation) error {
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
	if strings.TrimSpace(operation.StewardName) != "" ||
		strings.TrimSpace(operation.StewardModel) != "" ||
		len(operation.Inputs) > 0 ||
		len(operation.AllowedActions) > 0 ||
		len(operation.ArtifactTypes) > 0 ||
		len(operation.RequiredGuards) > 0 {
		return fmt.Errorf("inject_capsule must not include steward fields")
	}
	return nil
}

func validateAskStewardOperation(operation Operation, trigger Trigger) error {
	if strings.TrimSpace(operation.Role) != "" ||
		strings.TrimSpace(operation.Position) != "" ||
		strings.TrimSpace(operation.Text) != "" {
		return fmt.Errorf("ask_steward must not include inject fields")
	}
	if containsString(operation.AllowedActions, "policy.patch.propose") {
		return fmt.Errorf("ask_steward does not support policy.patch.propose in normalized rules")
	}
	if containsString(operation.AllowedActions, "context.truncate") {
		return fmt.Errorf("ask_steward does not support context.truncate in normalized rules")
	}
	spec := steward.Spec{
		Name:           firstNonEmpty(operation.StewardName, "rule-steward"),
		StewardModel:   operation.StewardModel,
		Hooks:          append([]string(nil), trigger.Hooks...),
		Inputs:         append([]string(nil), operation.Inputs...),
		AllowedActions: append([]string(nil), operation.AllowedActions...),
		ArtifactTypes:  append([]string(nil), operation.ArtifactTypes...),
		RequiredGuards: append([]string(nil), operation.RequiredGuards...),
	}
	if err := steward.Validate(spec); err != nil {
		return err
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

func hasAuditProvenance(audit Audit) bool {
	return strings.TrimSpace(audit.LedgerRef) != "" || len(audit.ArtifactRefs) > 0
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
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
