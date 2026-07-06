package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nuctori/gateway-harness/ledger"
)

func TestGoalGateHostExampleWritesLedger(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	ledgerPath := filepath.Join(t.TempDir(), "goal-gate-host.ledger.json")
	proposal := `{"version":"0.1","id":"proposal_goal_review_approve_host_test","steward":"goal-completion-reviewer","hook":"goal.before_complete","outputs":[{"action":"goal.approve_complete","reason":"host example test approved"}]}`
	cmd := exec.Command("go", "run", "./examples/goal-gate-host", "-ledger", ledgerPath)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GATEWAY_HARNESS_TEST_GOAL_PROPOSAL="+proposal)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run host example: %v\n%s", err, output)
	}
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	var got ledger.Ledger
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode ledger: %v\n%s", err, data)
	}
	if len(got.Projects) != 1 || len(got.Projects[0].Sessions) != 1 || len(got.Projects[0].Sessions[0].Events) != 1 {
		t.Fatalf("unexpected ledger shape: %+v", got)
	}
	event := got.Projects[0].Sessions[0].Events[0]
	if event.Action != "goal.approve_complete" || event.Hook != "goal.before_complete" {
		t.Fatalf("unexpected ledger event: %+v", event)
	}
}

func TestGoalGateHostExampleDisabledConfigSkipsLedgerAppend(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	ledgerPath := filepath.Join(t.TempDir(), "goal-gate-host-disabled.ledger.json")
	cmd := exec.Command(
		"go", "run", "./examples/goal-gate-host",
		"-config", "examples/newapi/goal-gate.config.json",
		"-ledger", ledgerPath,
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run host example with disabled config: %v\n%s", err, output)
	}
	if _, err := os.Stat(ledgerPath); !os.IsNotExist(err) {
		t.Fatalf("expected no ledger file for disabled config, got err=%v", err)
	}
}

func TestGoalGateHostExampleRejectContinuePath(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	ledgerPath := filepath.Join(t.TempDir(), "goal-gate-host-reject.ledger.json")
	proposal := `{"version":"0.1","id":"proposal_goal_review_reject_host_test","steward":"goal-completion-reviewer","hook":"goal.before_complete","outputs":[{"action":"goal.reject_complete","reason":"deployment verification missing"},{"action":"goal.request_continue","reason":"continue with verification","instruction":"Deploy the image and run the smoke test."}]}`
	cmd := exec.Command("go", "run", "./examples/goal-gate-host", "-ledger", ledgerPath)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GATEWAY_HARNESS_TEST_GOAL_PROPOSAL="+proposal)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run host example reject path: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "continue_work") || !strings.Contains(string(output), "Deploy the image and run the smoke test.") {
		t.Fatalf("expected continue output in host example: %s", output)
	}
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	var got ledger.Ledger
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode ledger: %v\n%s", err, data)
	}
	event := got.Projects[0].Sessions[0].Events[0]
	if event.Action != "goal.request_continue" || event.Type != "harness_action" {
		t.Fatalf("unexpected reject ledger event: %+v", event)
	}
}

func TestGoalGateHostExampleRunnerFailureStillWritesErrorLedger(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	ledgerPath := filepath.Join(t.TempDir(), "goal-gate-host-runner-fail.ledger.json")
	configPath := filepath.Join(t.TempDir(), "goal-gate.config.json")
	config := map[string]any{
		"enabled": true,
		"hook":    "goal.before_complete",
		"runner": map[string]any{
			"command": "go",
			"workdir": repoRoot,
			"args":    []string{"run", "./cmd/gateway-harness/testdata/does-not-exist"},
		},
		"allowed_inputs": []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
		"allowed_actions": []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
		"max_continue_attempts": 3,
		"cooldown_seconds":      60,
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cmd := exec.Command(
		"go", "run", "./examples/goal-gate-host",
		"-config", configPath,
		"-ledger", ledgerPath,
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected host example to fail, output=%s", output)
	}
	data, err = os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	var got ledger.Ledger
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode ledger: %v\n%s", err, data)
	}
	event := got.Projects[0].Sessions[0].Events[0]
	if event.Type != "error" || event.ErrorCode != "goal_gate_runner_failed" {
		t.Fatalf("unexpected failure ledger event: %+v", event)
	}
	if event.Action != "goal.review.failed" {
		t.Fatalf("unexpected failure ledger action: %+v", event)
	}
}
