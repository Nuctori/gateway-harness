package main

import (
	"testing"

	"github.com/Nuctori/gateway-harness/adapter"
)

func TestComposeGoalGateConfigDraftEnablesExplicitReviewDefaults(t *testing.T) {
	resp := ComposeGoalGateConfigDraft(adapter.GoalGateConfig{}, "帮我启用 goal gate，用 smolagents 审查完成状态，最多继续 4 次，冷却 90 秒")
	if !resp.Proposal.Enabled {
		t.Fatalf("expected proposal enabled: %+v", resp.Proposal)
	}
	if resp.Proposal.Runner.Command != "python" {
		t.Fatalf("expected python runner: %+v", resp.Proposal.Runner)
	}
	if resp.Proposal.MaxContinueAttempts != 4 {
		t.Fatalf("expected max_continue_attempts=4: %+v", resp.Proposal)
	}
	if resp.Proposal.CooldownSeconds != 90 {
		t.Fatalf("expected cooldown_seconds=90: %+v", resp.Proposal)
	}
	if len(resp.Changes) == 0 {
		t.Fatalf("expected visible config changes")
	}
	if !resp.RequiresConfirmation {
		t.Fatalf("expected explicit confirmation requirement")
	}
}

func TestComposeGoalGateConfigDraftKeepsDisableExplicit(t *testing.T) {
	resp := ComposeGoalGateConfigDraft(adapter.GoalGateConfig{Enabled: true}, "关闭 goal gate")
	if resp.Proposal.Enabled {
		t.Fatalf("expected proposal disabled: %+v", resp.Proposal)
	}
	if len(resp.Warnings) == 0 {
		t.Fatalf("expected warning about disabled state")
	}
	if resp.AssistantMessage == "" {
		t.Fatalf("expected assistant explanation")
	}
}

func TestComposeGoalGateConfigDraftDefaultsToMinimalReviewInputs(t *testing.T) {
	resp := ComposeGoalGateConfigDraft(adapter.GoalGateConfig{}, "帮我启用 goal gate，用 smolagents 审查完成状态")
	for _, want := range []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace"} {
		if !containsInputValue(resp.Proposal.AllowedInputs, want) {
			t.Fatalf("expected default minimal input %q: %+v", want, resp.Proposal.AllowedInputs)
		}
	}
	for _, forbidden := range []string{"changed_files", "verification_summary", "user_goal"} {
		if containsInputValue(resp.Proposal.AllowedInputs, forbidden) {
			t.Fatalf("did not expect extended input %q by default: %+v", forbidden, resp.Proposal.AllowedInputs)
		}
	}
}

func TestComposeGoalGateConfigDraftAllowsExplicitExtendedInputs(t *testing.T) {
	resp := ComposeGoalGateConfigDraft(adapter.GoalGateConfig{}, "帮我启用 goal gate，并授权更完整上下文，包括 changed files、verification summary 和 user goal")
	for _, want := range []string{"changed_files", "verification_summary", "user_goal"} {
		if !containsInputValue(resp.Proposal.AllowedInputs, want) {
			t.Fatalf("expected explicitly authorized extended input %q: %+v", want, resp.Proposal.AllowedInputs)
		}
	}
}

func TestComposeGoalGateConfigDraftWarnsForCatalogOnlyGoalHooks(t *testing.T) {
	resp := ComposeGoalGateConfigDraft(adapter.GoalGateConfig{}, "帮我看看 goal before resume 和完成后 hook")
	if len(resp.Warnings) < 2 {
		t.Fatalf("expected warnings for catalog-only hooks: %+v", resp.Warnings)
	}
	if resp.Proposal.Hook != "goal.before_complete" {
		t.Fatalf("expected runtime hook to remain goal.before_complete: %+v", resp.Proposal)
	}
}

func containsInputValue(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
