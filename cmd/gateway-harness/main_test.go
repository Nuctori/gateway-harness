package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/rule"
	"github.com/Nuctori/gateway-harness/steward"
)

func TestMustWriteJSONFileCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "ledger.json")

	mustWriteJSONFile(path, map[string]string{"status": "ok"})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode written json: %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("unexpected json: %+v", got)
	}
}

func TestCompileRuleFixture(t *testing.T) {
	r := mustLoadRuleDocument("../../fixtures/newapi/context-rule.continuity-drop.json")

	compiled, err := rule.Compile(r)
	if err != nil {
		t.Fatalf("compile rule: %v", err)
	}
	if len(compiled.Programs) != 1 {
		t.Fatalf("unexpected compiled policy: %+v", compiled)
	}
	action := compiled.Programs[0].Steps[0].Do[0]
	if action.Action != "context.inject_ledger_summary" {
		t.Fatalf("unexpected compiled action: %+v", action)
	}

	encoded, err := json.Marshal(compiled)
	if err != nil {
		t.Fatalf("marshal compiled policy: %v", err)
	}
	if strings.Contains(string(encoded), "context.truncate") || strings.Contains(string(encoded), "budget") || strings.Contains(string(encoded), "ask_steward") {
		t.Fatalf("compiled rule introduced non-normalized behavior: %s", encoded)
	}
}

func TestCompileRuleStewardsFixture(t *testing.T) {
	r := mustLoadRuleDocument("../../fixtures/newapi/context-rule.ask-steward.json")

	specs, err := rule.CompileStewards(r)
	if err != nil {
		t.Fatalf("compile stewards: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("unexpected steward specs: %+v", specs)
	}
	if specs[0].Name != "newapi-compact-context-steward" {
		t.Fatalf("unexpected steward name: %+v", specs[0])
	}
	if specs[0].StewardModel != "kimi-for-coding" {
		t.Fatalf("unexpected steward model: %+v", specs[0])
	}
}

func TestExecuteGoalGateSidecarCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	proposal := strings.TrimSpace(`{
  "version": "0.1",
  "id": "proposal_goal_review_approve_001",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {
      "action": "goal.approve_complete",
      "reason": "Focused tests, CLI validation, and sidecar smoke all passed."
    }
  ]
}`)
	cmd := exec.Command(
		"go", "run", ".",
		"execute-goal-gate-sidecar",
		"../../fixtures/goal-gate/goal.before_complete.steward.json",
		"../../fixtures/goal-gate/goal.before_complete.steward-event.json",
		"../../fixtures/goal-gate/goal.before_complete.audit.json",
		"--",
		"go", "run", "./testdata/goalreviewhelper",
	)
	cmd.Dir = wd
	cmd.Env = append(os.Environ(), "GATEWAY_HARNESS_TEST_GOAL_PROPOSAL="+proposal)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execute-goal-gate-sidecar cli failed: %v\n%s", err, output)
	}
	var result steward.GoalGateSidecarResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode cli result: %v\n%s", err, output)
	}
	if result.Review.Decision != steward.GoalDecisionApprove {
		t.Fatalf("unexpected review decision: %+v", result.Review)
	}
	if !result.Outcome.AllowComplete || result.Outcome.ContinueWork {
		t.Fatalf("unexpected outcome: %+v", result.Outcome)
	}
	if result.AppendRecord.Event.Action != "goal.approve_complete" {
		t.Fatalf("unexpected append record action: %+v", result.AppendRecord.Event)
	}
	if result.AppendRecord.Event.Hook != steward.GoalBeforeCompleteHook {
		t.Fatalf("unexpected append record hook: %+v", result.AppendRecord.Event)
	}
	if result.AppendRecord.Event.Metadata["proposal_id"] == "" {
		t.Fatalf("expected proposal_id metadata: %+v", result.AppendRecord.Event)
	}
}

func TestValidateGoalGateConfigCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "goal-gate.config.json")
	mustWriteJSONFile(configPath, adapter.GoalGateConfig{
		Enabled:             true,
		Hook:                steward.GoalBeforeCompleteHook,
		Runner:              adapter.GoalGateRunner{Command: "python", Args: []string{"examples/smolagents/goal_reviewer.py"}},
		AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
		AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
		MaxContinueAttempts: 3,
		CooldownSeconds:     60,
	})
	cmd := exec.Command("go", "run", ".", "validate-goal-gate-config", configPath)
	cmd.Dir = wd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate-goal-gate-config failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "goal gate config ok") {
		t.Fatalf("unexpected validate output: %s", output)
	}
}

func TestGoalGateConfigSchemaCLIPublishesGUIHints(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	cmd := exec.Command("go", "run", ".", "goal-gate-config-schema")
	cmd.Dir = wd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goal-gate-config-schema failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, needle := range []string{
		"Enable Goal Gate",
		"Allowed Event Inputs",
		"Allowed Proposal Actions",
		"Goal State",
		"Approve Complete",
		"Default-off",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected schema hint %q in output: %s", needle, output)
		}
	}
}

func TestGoalGateFormSchemaCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	cmd := exec.Command("go", "run", ".", "goal-gate-form-schema")
	cmd.Dir = wd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goal-gate-form-schema failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, needle := range []string{
		"Goal Gate Form Model",
		"transparency_note",
		"sections",
		"multi_select",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected form schema hint %q in output: %s", needle, output)
		}
	}
}

func TestGoalGateResultSchemaCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	cmd := exec.Command("go", "run", ".", "goal-gate-result-schema")
	cmd.Dir = wd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goal-gate-result-schema failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, needle := range []string{
		"Goal Gate Result",
		"continuation_patches",
		"append_record",
		"allow_complete",
		"failure",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected result schema hint %q in output: %s", needle, output)
		}
	}
}

func TestGoalGateFormModelCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	cmd := exec.Command("go", "run", ".", "goal-gate-form-model")
	cmd.Dir = wd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goal-gate-form-model failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, needle := range []string{
		"Default-off. Unless enabled here",
		"activation",
		"review_scope",
		"Allowed Event Inputs",
		"Allowed Proposal Actions",
		"multi_select",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected form model hint %q in output: %s", needle, output)
		}
	}
}

func TestGoalGateHostBundleCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	cmd := exec.Command("go", "run", ".", "goal-gate-host-bundle")
	cmd.Dir = wd
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goal-gate-host-bundle failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, needle := range []string{
		"Goal Gate Host Bundle",
		"interaction_model",
		"hook_catalog",
		"goal.before_resume",
		"goal.after_complete",
		"zh-CN",
		"assistant_examples",
		"config_schema",
		"form_model",
		"result_schema",
		"approve_result",
		"reject_result",
		"failure_result",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("expected host bundle field %q in output: %s", needle, output)
		}
	}
}

func TestMustLoadGoalGateConfigResolvesRelativeWorkdir(t *testing.T) {
	base := t.TempDir()
	configDir := filepath.Join(base, "nested")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "goal-gate.config.json")
	mustWriteJSONFile(configPath, adapter.GoalGateConfig{
		Enabled:             true,
		Hook:                steward.GoalBeforeCompleteHook,
		Runner:              adapter.GoalGateRunner{Command: "python", Workdir: "..", Args: []string{"examples/smolagents/goal_reviewer.py"}},
		AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
		AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
		MaxContinueAttempts: 3,
		CooldownSeconds:     60,
	})
	cfg := mustLoadGoalGateConfig(configPath)
	want := filepath.Clean(base)
	if cfg.Runner.Workdir != want {
		t.Fatalf("unexpected resolved workdir: got %q want %q", cfg.Runner.Workdir, want)
	}
}

func TestExecuteGoalGateCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	proposal := strings.TrimSpace(`{
  "version": "0.1",
  "id": "proposal_goal_review_approve_002",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {
      "action": "goal.approve_complete",
      "reason": "Goal Gate host CLI executed successfully."
    }
  ]
}`)
	configPath := filepath.Join(t.TempDir(), "goal-gate.config.json")
	mustWriteJSONFile(configPath, adapter.GoalGateConfig{
		Enabled:             true,
		Hook:                steward.GoalBeforeCompleteHook,
		Runner:              adapter.GoalGateRunner{Command: "go", Workdir: repoRoot, Args: []string{"run", "./cmd/gateway-harness/testdata/goalreviewhelper"}},
		AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
		AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
		MaxContinueAttempts: 3,
		CooldownSeconds:     60,
	})
	cmd := exec.Command(
		"go", "run", "./cmd/gateway-harness",
		"execute-goal-gate",
		configPath,
		"fixtures/goal-gate/goal.before_complete.steward.json",
		"fixtures/goal-gate/goal.before_complete.steward-event.json",
		"fixtures/goal-gate/goal.before_complete.audit.json",
	)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GATEWAY_HARNESS_TEST_GOAL_PROPOSAL="+proposal)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execute-goal-gate failed: %v\n%s", err, output)
	}
	var result adapter.GoalGateResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode execute-goal-gate result: %v\n%s", err, output)
	}
	if !result.Enabled || !result.Triggered || result.Sidecar == nil || result.AppendRecord == nil {
		t.Fatalf("unexpected goal gate result: %+v", result)
	}
	if !result.Sidecar.Outcome.AllowComplete {
		t.Fatalf("unexpected sidecar outcome: %+v", result.Sidecar.Outcome)
	}
	if result.AppendRecord.Event.Action != "goal.approve_complete" {
		t.Fatalf("unexpected append record action: %+v", result.AppendRecord.Event)
	}
}

func TestExecuteGoalGateCLIRejectContinuePath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	proposal := strings.TrimSpace(`{
  "version": "0.1",
  "id": "proposal_goal_review_reject_002",
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
	configPath := filepath.Join(t.TempDir(), "goal-gate.config.json")
	mustWriteJSONFile(configPath, adapter.GoalGateConfig{
		Enabled:             true,
		Hook:                steward.GoalBeforeCompleteHook,
		Runner:              adapter.GoalGateRunner{Command: "go", Workdir: repoRoot, Args: []string{"run", "./cmd/gateway-harness/testdata/goalreviewhelper"}},
		AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
		AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
		MaxContinueAttempts: 3,
		CooldownSeconds:     60,
	})
	cmd := exec.Command(
		"go", "run", "./cmd/gateway-harness",
		"execute-goal-gate",
		configPath,
		"fixtures/goal-gate/goal.before_complete.steward.json",
		"fixtures/goal-gate/goal.before_complete.steward-event.json",
		"fixtures/goal-gate/goal.before_complete.audit.json",
	)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GATEWAY_HARNESS_TEST_GOAL_PROPOSAL="+proposal)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execute-goal-gate reject path failed: %v\n%s", err, output)
	}
	var result adapter.GoalGateResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode execute-goal-gate reject result: %v\n%s", err, output)
	}
	if result.Sidecar == nil || result.Sidecar.Outcome.AllowComplete || !result.Sidecar.Outcome.ContinueWork {
		t.Fatalf("unexpected reject outcome: %+v", result)
	}
	if result.Sidecar.Outcome.ContinueInstruction == "" {
		t.Fatalf("expected continue instruction: %+v", result.Sidecar.Outcome)
	}
	if result.AppendRecord == nil || result.AppendRecord.Event.Action != "goal.request_continue" {
		t.Fatalf("expected goal.request_continue append record: %+v", result)
	}
}

func TestExecuteGoalGateCLIRunnerFailurePrintsStructuredResult(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	configPath := filepath.Join(t.TempDir(), "goal-gate.config.json")
	mustWriteJSONFile(configPath, adapter.GoalGateConfig{
		Enabled:             true,
		Hook:                steward.GoalBeforeCompleteHook,
		Runner:              adapter.GoalGateRunner{Command: "go", Workdir: repoRoot, Args: []string{"run", "./cmd/gateway-harness/testdata/does-not-exist"}},
		AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
		AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
		MaxContinueAttempts: 3,
		CooldownSeconds:     60,
	})
	cmd := exec.Command(
		"go", "run", "./cmd/gateway-harness",
		"execute-goal-gate",
		configPath,
		"fixtures/goal-gate/goal.before_complete.steward.json",
		"fixtures/goal-gate/goal.before_complete.steward-event.json",
		"fixtures/goal-gate/goal.before_complete.audit.json",
	)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected execute-goal-gate to fail, output=%s", output)
	}
	var result adapter.GoalGateResult
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("decode structured failure result: %v\n%s", err, output)
	}
	if result.Failure == nil || result.Failure.Code != "goal_gate_runner_failed" {
		t.Fatalf("unexpected failure result: %+v", result)
	}
	if result.AppendRecord == nil || result.AppendRecord.Event.ErrorCode != "goal_gate_runner_failed" {
		t.Fatalf("expected failure append record: %+v", result)
	}
	if !strings.Contains(string(output), "execute goal gate failed") {
		t.Fatalf("expected stderr failure summary in combined output: %s", output)
	}
}
