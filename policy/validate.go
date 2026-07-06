package policy

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Nuctori/gateway-harness/hooks"
)

var SupportedHooks = func() map[string]bool {
	supported := hooks.SupportedMap()
	supported["*"] = true
	return supported
}()

var SupportedActions = map[string]bool{
	"context.inject":                true,
	"context.inject_ledger_summary": true,
	"context.truncate":              true,
}

func Decode(r io.Reader) (Policy, error) {
	var p Policy
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return Policy{}, err
	}
	return p, nil
}

func Validate(p Policy) error {
	if p.Programs == nil {
		return nil
	}
	for i, program := range p.Programs {
		if strings.TrimSpace(program.Name) == "" {
			return fmt.Errorf("program %d name is required", i)
		}
		if len(program.Models) == 0 {
			return fmt.Errorf("program %q needs at least one model selector", program.Name)
		}
		if len(program.Steps) == 0 {
			return fmt.Errorf("program %q needs at least one step", program.Name)
		}
		for j, step := range program.Steps {
			if err := validateHooks(step); err != nil {
				return fmt.Errorf("program %q step %d %w", program.Name, j, err)
			}
			if step.When.EstimatedTokensGT < 0 {
				return fmt.Errorf("program %q step %d has invalid estimated_tokens_gt", program.Name, j)
			}
			if len(step.Do) == 0 {
				return fmt.Errorf("program %q step %d needs at least one action", program.Name, j)
			}
			for k, action := range step.Do {
				if err := validateAction(action); err != nil {
					return fmt.Errorf("program %q step %d action %d %w", program.Name, j, k, err)
				}
			}
		}
	}
	return nil
}

func validateHooks(step Step) error {
	hooks := EffectiveHooks(step)
	if len(hooks) == 0 {
		return fmt.Errorf("needs explicit hook or hooks")
	}
	for _, hook := range hooks {
		if !SupportedHooks[hook] {
			return fmt.Errorf("unsupported hook %q", hook)
		}
	}
	return nil
}

func validateAction(action Action) error {
	if !SupportedActions[action.Action] {
		return fmt.Errorf("unsupported action %q", action.Action)
	}
	switch action.Action {
	case "context.inject":
		if err := validateInjectShape(action); err != nil {
			return fmt.Errorf("context.inject %w", err)
		}
		if hasLedgerProvenance(action) {
			return fmt.Errorf("context.inject ledger provenance fields require context.inject_ledger_summary")
		}
		if hasTruncateFields(action) {
			return fmt.Errorf("context.inject truncate fields are not allowed")
		}
	case "context.inject_ledger_summary":
		if err := validateInjectShape(action); err != nil {
			return fmt.Errorf("context.inject_ledger_summary %w", err)
		}
		if strings.TrimSpace(action.LedgerRef) == "" {
			return fmt.Errorf("context.inject_ledger_summary ledger_ref is required")
		}
		if strings.TrimSpace(action.Source) != "" && action.Source != "ledger.summary" {
			return fmt.Errorf("context.inject_ledger_summary source must be ledger.summary")
		}
		if hasTruncateFields(action) {
			return fmt.Errorf("context.inject_ledger_summary truncate fields are not allowed")
		}
	case "context.truncate":
		if hasInjectFields(action) || hasLedgerProvenance(action) {
			return fmt.Errorf("context.truncate inject fields are not allowed")
		}
		if strings.TrimSpace(action.Strategy) != "" {
			return fmt.Errorf("context.truncate strategy is not supported")
		}
		if action.KeepLastMessages != nil && *action.KeepLastMessages < 0 {
			return fmt.Errorf("context.truncate limits must not be negative")
		}
	}
	return nil
}

func validateInjectShape(action Action) error {
	if strings.TrimSpace(action.Text) == "" {
		return fmt.Errorf("text is required")
	}
	if strings.TrimSpace(action.Role) == "" {
		return fmt.Errorf("role is required")
	}
	if strings.TrimSpace(action.Position) == "" {
		return fmt.Errorf("position is required")
	}
	return nil
}

func hasLedgerProvenance(action Action) bool {
	return strings.TrimSpace(action.Source) != "" ||
		strings.TrimSpace(action.LedgerRef) != "" ||
		len(action.ArtifactRefs) > 0
}

func hasTruncateFields(action Action) bool {
	return strings.TrimSpace(action.Strategy) != "" ||
		action.KeepLastMessages != nil ||
		len(action.PreserveRoles) > 0
}

func hasInjectFields(action Action) bool {
	return strings.TrimSpace(action.Role) != "" ||
		strings.TrimSpace(action.Position) != "" ||
		strings.TrimSpace(action.Text) != ""
}

func EffectiveHooks(step Step) []string {
	hooks := make([]string, 0, len(step.Hooks)+1)
	hooks = append(hooks, step.Hooks...)
	if strings.TrimSpace(step.Hook) != "" {
		hooks = append(hooks, step.Hook)
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(hooks))
	for _, hook := range hooks {
		hook = strings.TrimSpace(hook)
		if hook == "" || seen[hook] {
			continue
		}
		seen[hook] = true
		out = append(out, hook)
	}
	return out
}

func Summarize(p Policy) Summary {
	summary := Summary{Programs: len(p.Programs)}
	hooks := map[string]bool{}
	for _, program := range p.Programs {
		summary.Steps += len(program.Steps)
		for _, step := range program.Steps {
			summary.Actions += len(step.Do)
			for _, hook := range EffectiveHooks(step) {
				hooks[hook] = true
			}
		}
	}
	for hook := range hooks {
		summary.Hooks = append(summary.Hooks, hook)
	}
	sort.Strings(summary.Hooks)
	return summary
}
