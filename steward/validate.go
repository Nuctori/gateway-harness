package steward

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Nuctori/gateway-harness/ledger"
	"github.com/Nuctori/gateway-harness/policy"
)

const (
	GuardExplicitInvocationOnly      = "explicit_invocation_only"
	GuardStructuredOutputOnly        = "structured_output_only"
	GuardValidateOutputActions       = "validate_output_actions"
	GuardRedactedInputOnly           = "redacted_input_only"
	GuardArtifactHashRequired        = "artifact_hash_required"
	GuardHumanApprovalForPolicyPatch = "human_approval_for_policy_patch"
)

var SupportedInputs = map[string]bool{
	"adapter_capability":    true,
	"artifact_refs":         true,
	"error_summary":         true,
	"ledger_event_metadata": true,
	"policy_summary":        true,
	"redacted_trace":        true,
	"session_tags":          true,
	"user_goal":             true,
}

var SupportedOutputActions = map[string]bool{
	"context.inject":         true,
	"context.truncate":       true,
	"diagnosis.note.create":  true,
	"ledger.artifact.create": true,
	"policy.patch.propose":   true,
	"session.tags.update":    true,
}

var SupportedGuards = map[string]bool{
	GuardExplicitInvocationOnly:      true,
	GuardStructuredOutputOnly:        true,
	GuardValidateOutputActions:       true,
	GuardRedactedInputOnly:           true,
	GuardArtifactHashRequired:        true,
	GuardHumanApprovalForPolicyPatch: true,
}

func Decode(r io.Reader) (Spec, error) {
	var s Spec
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&s); err != nil {
		return Spec{}, err
	}
	return s, nil
}

func Validate(s Spec) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("steward name is required")
	}
	if strings.TrimSpace(s.StewardModel) == "" {
		return fmt.Errorf("steward %q steward_model is required", s.Name)
	}
	if len(s.Hooks) == 0 {
		return fmt.Errorf("steward %q needs at least one hook", s.Name)
	}
	if len(s.Inputs) == 0 {
		return fmt.Errorf("steward %q needs at least one input", s.Name)
	}
	if len(s.AllowedActions) == 0 {
		return fmt.Errorf("steward %q needs at least one allowed action", s.Name)
	}
	if err := validateHooks(s.Hooks); err != nil {
		return fmt.Errorf("steward %q %w", s.Name, err)
	}
	if err := validateSet("input", s.Inputs, SupportedInputs); err != nil {
		return fmt.Errorf("steward %q %w", s.Name, err)
	}
	if err := validateSet("allowed_action", s.AllowedActions, SupportedOutputActions); err != nil {
		return fmt.Errorf("steward %q %w", s.Name, err)
	}
	if err := validateSet("artifact_type", s.ArtifactTypes, ledger.SupportedArtifactTypes); err != nil {
		return fmt.Errorf("steward %q %w", s.Name, err)
	}
	if err := validateSet("required_guard", s.RequiredGuards, SupportedGuards); err != nil {
		return fmt.Errorf("steward %q %w", s.Name, err)
	}
	if err := requireGuards(s); err != nil {
		return fmt.Errorf("steward %q %w", s.Name, err)
	}
	return nil
}

func validateHooks(hooks []string) error {
	seen := map[string]bool{}
	for _, hook := range hooks {
		hook = strings.TrimSpace(hook)
		if hook == "" {
			return fmt.Errorf("has empty hook")
		}
		if hook == "*" {
			return fmt.Errorf("must not use wildcard hook for AI-in-the-loop")
		}
		if seen[hook] {
			return fmt.Errorf("has duplicate hook %q", hook)
		}
		if !policy.SupportedHooks[hook] {
			return fmt.Errorf("has unsupported hook %q", hook)
		}
		seen[hook] = true
	}
	return nil
}

func validateSet(name string, values []string, supported map[string]bool) error {
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("has empty %s", name)
		}
		if seen[value] {
			return fmt.Errorf("has duplicate %s %q", name, value)
		}
		if !supported[value] {
			return fmt.Errorf("has unsupported %s %q", name, value)
		}
		seen[value] = true
	}
	return nil
}

func requireGuards(s Spec) error {
	required := []string{
		GuardExplicitInvocationOnly,
		GuardStructuredOutputOnly,
		GuardValidateOutputActions,
		GuardRedactedInputOnly,
	}
	for _, guard := range required {
		if !contains(s.RequiredGuards, guard) {
			return fmt.Errorf("requires guard %q", guard)
		}
	}
	if contains(s.AllowedActions, "ledger.artifact.create") && !contains(s.RequiredGuards, GuardArtifactHashRequired) {
		return fmt.Errorf("ledger.artifact.create requires guard %q", GuardArtifactHashRequired)
	}
	if contains(s.AllowedActions, "ledger.artifact.create") && len(s.ArtifactTypes) == 0 {
		return fmt.Errorf("ledger.artifact.create requires at least one artifact_type")
	}
	if len(s.ArtifactTypes) > 0 && !contains(s.AllowedActions, "ledger.artifact.create") {
		return fmt.Errorf("artifact_types require allowed action %q", "ledger.artifact.create")
	}
	if contains(s.AllowedActions, "policy.patch.propose") && !contains(s.RequiredGuards, GuardHumanApprovalForPolicyPatch) {
		return fmt.Errorf("policy.patch.propose requires guard %q", GuardHumanApprovalForPolicyPatch)
	}
	return nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func Summarize(s Spec) Summary {
	return Summary{
		Name:           s.Name,
		Hooks:          len(s.Hooks),
		Inputs:         len(s.Inputs),
		AllowedActions: len(s.AllowedActions),
		ArtifactTypes:  len(s.ArtifactTypes),
		RequiredGuards: len(s.RequiredGuards),
	}
}
