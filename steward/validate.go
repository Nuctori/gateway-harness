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
	GuardExplicitInvocationOnly = "explicit_invocation_only"
	GuardStructuredOutputOnly   = "structured_output_only"
	GuardValidateOutputActions  = "validate_output_actions"
	GuardRedactedInputOnly      = "redacted_input_only"
	GuardArtifactHashRequired   = "artifact_hash_required"
)

var SupportedInputs = map[string]bool{
	"adapter_capability":    true,
	"artifact_refs":         true,
	"blockers":              true,
	"changed_files":         true,
	"error_summary":         true,
	"goal_state":            true,
	"ledger_event_metadata": true,
	"policy_summary":        true,
	"redacted_trace":        true,
	"recent_trace":          true,
	"session_tags":          true,
	"test_results":          true,
	"user_goal":             true,
	"verification_summary":  true,
	"work_summary":          true,
}

var SupportedOutputActions = map[string]bool{
	"context.inject":         true,
	"diagnosis.note.create":  true,
	"goal.approve_complete":  true,
	"goal.reject_complete":   true,
	"goal.request_continue":  true,
	"ledger.artifact.create": true,
	"session.tags.update":    true,
}

var SupportedGuards = map[string]bool{
	GuardExplicitInvocationOnly: true,
	GuardStructuredOutputOnly:   true,
	GuardValidateOutputActions:  true,
	GuardRedactedInputOnly:      true,
	GuardArtifactHashRequired:   true,
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

func DecodeProposal(r io.Reader) (Proposal, error) {
	var p Proposal
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		return Proposal{}, err
	}
	return p, nil
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

func ValidateProposal(s Spec, p Proposal) error {
	if err := Validate(s); err != nil {
		return fmt.Errorf("spec: %w", err)
	}
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("proposal id is required")
	}
	if p.Steward != s.Name {
		return fmt.Errorf("proposal %q steward %q does not match spec %q", p.ID, p.Steward, s.Name)
	}
	if !contains(s.Hooks, p.Hook) {
		return fmt.Errorf("proposal %q hook %q is not enabled by steward %q", p.ID, p.Hook, s.Name)
	}
	if len(p.Outputs) == 0 {
		return fmt.Errorf("proposal %q needs at least one output", p.ID)
	}
	for i, output := range p.Outputs {
		if err := validateProposalOutput(s, p.Hook, output); err != nil {
			return fmt.Errorf("proposal %q output %d %w", p.ID, i, err)
		}
	}
	return nil
}

func validateProposalOutput(s Spec, hook string, output Output) error {
	if !contains(s.AllowedActions, output.Action) {
		return fmt.Errorf("action %q is not allowed by steward %q", output.Action, s.Name)
	}
	switch output.Action {
	case "context.inject":
		if err := rejectFields(output, "instruction", "artifact_type", "content_hash", "ref", "tags", "severity", "note_hash"); err != nil {
			return err
		}
		return validateProposalPolicyAction(hook, output)
	case "ledger.artifact.create":
		if err := rejectFields(output, "instruction", "role", "position", "text", "tags", "severity", "note_hash"); err != nil {
			return err
		}
		if !contains(s.ArtifactTypes, output.ArtifactType) {
			return fmt.Errorf("artifact_type %q is not allowed by steward %q", output.ArtifactType, s.Name)
		}
		if strings.TrimSpace(output.ContentHash) == "" {
			return fmt.Errorf("ledger.artifact.create content_hash is required")
		}
		if strings.TrimSpace(output.Ref) == "" {
			return fmt.Errorf("ledger.artifact.create ref is required")
		}
	case "diagnosis.note.create":
		if err := rejectFields(output, "instruction", "role", "position", "text", "artifact_type", "content_hash", "tags"); err != nil {
			return err
		}
		if strings.TrimSpace(output.NoteHash) == "" {
			return fmt.Errorf("diagnosis.note.create note_hash is required")
		}
		if strings.TrimSpace(output.Ref) == "" {
			return fmt.Errorf("diagnosis.note.create ref is required")
		}
		if strings.TrimSpace(output.Severity) == "" {
			return fmt.Errorf("diagnosis.note.create severity is required")
		}
	case "session.tags.update":
		if err := rejectFields(output, "instruction", "role", "position", "text", "artifact_type", "content_hash", "ref", "severity", "note_hash"); err != nil {
			return err
		}
		return validateNonEmptyUnique("tag", output.Tags)
	case "goal.approve_complete":
		return rejectFields(output, "instruction", "role", "position", "text", "artifact_type", "content_hash", "ref", "tags", "severity", "note_hash")
	case "goal.reject_complete":
		if err := rejectFields(output, "instruction", "role", "position", "text", "artifact_type", "content_hash", "ref", "tags", "severity", "note_hash"); err != nil {
			return err
		}
		if strings.TrimSpace(output.Reason) == "" {
			return fmt.Errorf("goal.reject_complete reason is required")
		}
	case "goal.request_continue":
		if err := rejectFields(output, "role", "position", "text", "artifact_type", "content_hash", "ref", "tags", "severity", "note_hash"); err != nil {
			return err
		}
		if strings.TrimSpace(output.Instruction) == "" {
			return fmt.Errorf("goal.request_continue instruction is required")
		}
	default:
		return fmt.Errorf("unsupported proposal action %q", output.Action)
	}
	return nil
}

func validateProposalPolicyAction(hook string, output Output) error {
	action := policy.Action{
		Action:   output.Action,
		Role:     output.Role,
		Position: output.Position,
		Text:     output.Text,
		Reason:   output.Reason,
	}
	return policy.Validate(policy.Policy{Programs: []policy.Program{{
		Name:   "steward-proposal",
		Models: []string{"*"},
		Steps:  []policy.Step{{Hook: hook, Do: []policy.Action{action}}},
	}}})
}

func rejectFields(output Output, names ...string) error {
	for _, name := range names {
		if outputFieldSet(output, name) {
			return fmt.Errorf("field %q is not allowed for action %q", name, output.Action)
		}
	}
	return nil
}

func outputFieldSet(output Output, name string) bool {
	switch name {
	case "instruction":
		return strings.TrimSpace(output.Instruction) != ""
	case "role":
		return strings.TrimSpace(output.Role) != ""
	case "position":
		return strings.TrimSpace(output.Position) != ""
	case "text":
		return strings.TrimSpace(output.Text) != ""
	case "artifact_type":
		return strings.TrimSpace(output.ArtifactType) != ""
	case "content_hash":
		return strings.TrimSpace(output.ContentHash) != ""
	case "ref":
		return strings.TrimSpace(output.Ref) != ""
	case "tags":
		return len(output.Tags) > 0
	case "severity":
		return strings.TrimSpace(output.Severity) != ""
	case "note_hash":
		return strings.TrimSpace(output.NoteHash) != ""
	default:
		return false
	}
}

func validateNonEmptyUnique(name string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("requires at least one %s", name)
	}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("has empty %s", name)
		}
		if seen[value] {
			return fmt.Errorf("has duplicate %s %q", name, value)
		}
		seen[value] = true
	}
	return nil
}

func SummarizeProposal(p Proposal) ProposalSummary {
	return ProposalSummary{
		ID:      p.ID,
		Steward: p.Steward,
		Hook:    p.Hook,
		Outputs: len(p.Outputs),
	}
}
