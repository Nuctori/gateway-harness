package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/steward"
)

func TestDemoHostBundlePublishesChatFirstInteractionModel(t *testing.T) {
	mux := newDemoHostMux(testDemoHostPaths(t))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/goal-gate/bundle", nil)

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bundle status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	interaction, ok := payload["interaction_model"].(map[string]any)
	if !ok {
		t.Fatalf("expected interaction_model in bundle: %+v", payload)
	}
	if interaction["primary_surface"] != "chat_assistant" {
		t.Fatalf("unexpected primary surface: %+v", interaction)
	}
	if interaction["assistant_applies_directly"] != false {
		t.Fatalf("assistant must not auto-apply drafts: %+v", interaction)
	}
}

func TestDemoHostConfigAssistantReturnsProposalOnly(t *testing.T) {
	mux := newDemoHostMux(testDemoHostPaths(t))
	body, err := json.Marshal(ConfigAssistantRequest{
		Message: "帮我启用 goal gate，用 smolagents 审查完成状态，允许 context.inject，最多继续 5 次",
		Config:  adapter.GoalGateConfig{},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/goal-gate/config-assistant", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("config assistant status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload ConfigAssistantResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode config assistant response: %v", err)
	}
	if !payload.RequiresConfirmation {
		t.Fatalf("expected explicit confirmation requirement: %+v", payload)
	}
	if !payload.Proposal.Enabled {
		t.Fatalf("expected enabled proposal: %+v", payload.Proposal)
	}
	if payload.Proposal.MaxContinueAttempts != 5 {
		t.Fatalf("expected requested retry limit: %+v", payload.Proposal)
	}
	if !containsString(payload.Proposal.AllowedActions, "context.inject") {
		t.Fatalf("expected explicit context.inject opt-in: %+v", payload.Proposal.AllowedActions)
	}
	if len(payload.Changes) == 0 {
		t.Fatalf("expected visible field-level changes")
	}
}

func TestDemoHostExecuteSkipsWhenConfigDisabled(t *testing.T) {
	mux := newDemoHostMux(testDemoHostPaths(t))
	payload := map[string]any{
		"config": map[string]any{
			"enabled": false,
			"hook":    steward.GoalBeforeCompleteHook,
		},
		"event": mustLoadStewardEvent(testDemoHostPaths(t).EventPath),
		"audit": map[string]any{
			"project":        map[string]any{"id": "project_gateway_harness"},
			"session":        map[string]any{"id": "session_goal_gate", "started_at": "2026-07-06T04:00:00Z"},
			"event_id":       "evt_goal_gate_http_disabled",
			"at":             "2026-07-06T04:01:00Z",
			"policy_version": "0.2",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal execute payload: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/goal-gate/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("execute disabled status=%d body=%s", rec.Code, rec.Body.String())
	}

	var result adapter.GoalGateResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode execute result: %v", err)
	}
	if result.Enabled || result.Triggered {
		t.Fatalf("disabled config must skip goal gate: %+v", result)
	}
}

func TestDemoHostExecuteRunsConfiguredRunner(t *testing.T) {
	paths := testDemoHostPaths(t)
	mux := newDemoHostMux(paths)
	payload := map[string]any{
		"config": mustLoadGoalGateConfig(paths.ConfigPath),
		"event":  buildConfiguredDemoEvent(mustLoadStewardEvent(paths.EventPath), mustLoadGoalGateConfig(paths.ConfigPath)),
		"audit": map[string]any{
			"project":        map[string]any{"id": "project_gateway_harness", "name": "Gateway Harness"},
			"session":        map[string]any{"id": "session_goal_gate", "title": "Goal Gate review", "started_at": "2026-07-06T04:00:00Z"},
			"event_id":       "evt_goal_gate_http_enabled",
			"at":             "2026-07-06T04:01:00Z",
			"policy_version": "0.2",
			"trace_hash":     "sha256:5555555555555555555555555555555555555555555555555555555555555555",
			"model":          "external-agent",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal execute payload: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/goal-gate/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("execute enabled status=%d body=%s", rec.Code, rec.Body.String())
	}

	var result adapter.GoalGateResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode execute result: %v", err)
	}
	if !result.Enabled || !result.Triggered {
		t.Fatalf("expected enabled goal gate path: %+v", result)
	}
	if result.Sidecar == nil || !result.Sidecar.Outcome.AllowComplete {
		t.Fatalf("expected configured runner to approve completion: %+v", result)
	}
	if result.AppendRecord == nil || result.AppendRecord.Event.Action != "goal.approve_complete" {
		t.Fatalf("expected auditable append record: %+v", result)
	}
	if result.Sidecar.Review.ProposalID == "" {
		t.Fatalf("expected proposal id in review result: %+v", result.Sidecar.Review)
	}
}

func TestDemoHostExecuteRejectContinuePath(t *testing.T) {
	t.Setenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL", `{
  "version": "0.1",
  "id": "proposal_goal_review_reject_http_001",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {
      "action": "goal.reject_complete",
      "reason": "deployment verification is still missing"
    },
    {
      "action": "goal.request_continue",
      "reason": "continue with verification",
      "instruction": "Deploy the image and run the smoke test."
    }
  ]
}`)
	paths := testDemoHostPaths(t)
	mux := newDemoHostMux(paths)
	payload := map[string]any{
		"config": mustLoadGoalGateConfig(paths.ConfigPath),
		"event":  buildConfiguredDemoEvent(mustLoadStewardEvent(paths.EventPath), mustLoadGoalGateConfig(paths.ConfigPath)),
		"audit": map[string]any{
			"project":        map[string]any{"id": "project_gateway_harness", "name": "Gateway Harness"},
			"session":        map[string]any{"id": "session_goal_gate", "title": "Goal Gate review", "started_at": "2026-07-06T04:00:00Z"},
			"event_id":       "evt_goal_gate_http_reject",
			"at":             "2026-07-06T04:01:00Z",
			"policy_version": "0.2",
			"trace_hash":     "sha256:5959595959595959595959595959595959595959595959595959595959595959",
			"model":          "external-agent",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal execute payload: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/goal-gate/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("execute reject status=%d body=%s", rec.Code, rec.Body.String())
	}

	var result adapter.GoalGateResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode execute result: %v", err)
	}
	if result.Sidecar == nil || result.Sidecar.Outcome.AllowComplete || !result.Sidecar.Outcome.ContinueWork {
		t.Fatalf("expected reject+continue outcome: %+v", result)
	}
	if result.Sidecar.Outcome.ContinueInstruction == "" {
		t.Fatalf("expected continue instruction: %+v", result.Sidecar.Outcome)
	}
	if result.AppendRecord == nil || result.AppendRecord.Event.Action != "goal.request_continue" {
		t.Fatalf("expected goal.request_continue append record: %+v", result)
	}
}

func TestDemoHostExecuteRejectsInvalidProposal(t *testing.T) {
	t.Setenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL", `{
  "version": "0.1",
  "id": "proposal_goal_review_invalid_http_001",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {
      "action": "context.truncate",
      "reason": "not allowed"
    }
  ]
}`)
	paths := testDemoHostPaths(t)
	mux := newDemoHostMux(paths)
	payload := map[string]any{
		"config": mustLoadGoalGateConfig(paths.ConfigPath),
		"event":  buildConfiguredDemoEvent(mustLoadStewardEvent(paths.EventPath), mustLoadGoalGateConfig(paths.ConfigPath)),
		"audit": map[string]any{
			"project":        map[string]any{"id": "project_gateway_harness"},
			"session":        map[string]any{"id": "session_goal_gate", "started_at": "2026-07-06T04:00:00Z"},
			"event_id":       "evt_goal_gate_http_invalid_proposal",
			"at":             "2026-07-06T04:01:00Z",
			"policy_version": "0.2",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal execute payload: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/goal-gate/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid proposal status=%d body=%s", rec.Code, rec.Body.String())
	}

	var result adapter.GoalGateResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode invalid proposal result: %v", err)
	}
	if result.Failure == nil || result.Failure.Code != "goal_gate_proposal_invalid" || result.Failure.Stage != "proposal" {
		t.Fatalf("unexpected invalid proposal failure: %+v", result)
	}
	if result.AppendRecord == nil || result.AppendRecord.Event.Action != "goal.review.failed" {
		t.Fatalf("expected auditable failure append record: %+v", result)
	}
}

func testDemoHostPaths(t *testing.T) demoHostPaths {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	return demoHostPaths{
		ConfigPath: filepath.Join(repoRoot, "examples", "goal-gate-host", "goal-gate.demo.config.json"),
		SpecPath:   filepath.Join(repoRoot, "fixtures", "goal-gate", "goal.before_complete.steward.json"),
		EventPath:  filepath.Join(repoRoot, "fixtures", "goal-gate", "goal.before_complete.steward-event.json"),
		AuditPath:  filepath.Join(repoRoot, "fixtures", "goal-gate", "goal.before_complete.audit.json"),
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
