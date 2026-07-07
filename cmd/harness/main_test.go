package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var harnessBin string

func TestMain(m *testing.M) {
	// Walk up from the package source directory to locate go.mod.
	cwd, _ := os.Getwd()
	root := cwd
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		if parent := filepath.Dir(root); parent != root {
			root = parent
		} else {
			fmt.Fprintln(os.Stderr, "cannot find module root from", cwd)
			os.Exit(1)
		}
	}

	// Build the harness binary once for all E2E tests.
	bin := filepath.Join(os.TempDir(), "harness-e2e")
	if os.PathSeparator == '\\' { // Windows
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/harness")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintln(os.Stderr, "build harness:", err)
		fmt.Fprintln(os.Stderr, string(out))
		os.Exit(1)
	}
	harnessBin = bin

	code := m.Run()
	os.Remove(bin)
	os.Exit(code)
}

// moduleRoot walks up from cwd to find the go.mod root.
func moduleRoot(t *testing.T) string {
	t.Helper()
	cwd, _ := os.Getwd()
	root := cwd
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root
		}
		if parent := filepath.Dir(root); parent != root {
			root = parent
		} else {
			t.Fatal("cannot find module root from", cwd)
			return ""
		}
	}
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(moduleRoot(t), "examples", "fixtures", name)
}

func policyFixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(moduleRoot(t), "examples", "policies", name)
}

// runHarness executes the harness binary with the given args and
// returns the parsed JSON object, raw output, and any exec error.
func runHarness(t *testing.T, args ...string) (map[string]any, string, error) {
	t.Helper()
	out, err := exec.Command(harnessBin, args...).CombinedOutput()
	output := string(out)
	if err != nil {
		return nil, output, fmt.Errorf("harness %v: %w\n%s", args, err, output)
	}
	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, output, fmt.Errorf("json decode: %w\n%s", err, output)
	}
	return result, output, nil
}

// ── policy validate ────────────────────────────────────────────

func TestCLI_PolicyValidate(t *testing.T) {
	result, _, err := runHarness(t, "policy", "validate", policyFixture(t, "coding.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if result["ok"] != true {
		t.Errorf("ok = %v, want true", result["ok"])
	}
	if result["policy"] != "coding-model-policy" {
		t.Errorf("policy = %q", result["policy"])
	}
}

func TestCLI_PolicyValidateInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	bad := filepath.Join(tmpDir, "bad.yaml")
	os.WriteFile(bad, []byte("name: ''\nhooks: {}"), 0644)

	_, _, err := runHarness(t, "policy", "validate", bad)
	if err == nil {
		t.Fatal("expected error for invalid policy")
	}
}

// ── run ────────────────────────────────────────────────────────

func TestCLI_RunPrePrompt(t *testing.T) {
	result, _, err := runHarness(t, "run",
		"--event", fixture(t, "pre-prompt.json"),
		"--policy", policyFixture(t, "coding.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	dec := result["decision"].(map[string]any)
	if dec["event_type"] != "request.pre_prompt" {
		t.Errorf("event_type = %q", dec["event_type"])
	}
	if dec["matched_policy"] != "coding-model-policy" {
		t.Errorf("matched_policy = %q", dec["matched_policy"])
	}
	tr := result["trace"].(map[string]any)
	outs := tr["outcomes"].([]any)
	if len(outs) == 0 {
		t.Fatal("trace outcomes empty")
	}
	if outs[0].(map[string]any)["type"] != "decision.applied" {
		t.Errorf("outcome type = %v", outs[0].(map[string]any)["type"])
	}
}

func TestCLI_RunUpstreamError(t *testing.T) {
	result, _, err := runHarness(t, "run",
		"--event", fixture(t, "upstream-error-429.json"),
		"--policy", policyFixture(t, "coding.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	dec := result["decision"].(map[string]any)
	actions := dec["actions"].([]any)
	if len(actions) == 0 {
		t.Fatal("expected actions for error event")
	}
	act := actions[0].(map[string]any)
	if act["type"] != "retry.with_model" {
		t.Errorf("action type = %q", act["type"])
	}
	if act["to_model"] != "gpt-5.5" {
		t.Errorf("to_model = %q", act["to_model"])
	}
}

// ── context dry-run ────────────────────────────────────────────

func TestCLI_ContextDryRun(t *testing.T) {
	result, _, err := runHarness(t, "context", "dry-run",
		"--event", fixture(t, "pre-context-any-model.json"),
		"--policy", policyFixture(t, "context-harness.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	dec := result["decision"].(map[string]any)
	if dec["event_type"] != "request.pre_context" {
		t.Errorf("event_type = %q", dec["event_type"])
	}
	cp := dec["context_patch"].(map[string]any)
	ops := cp["operations"].([]any)
	if len(ops) == 0 {
		t.Fatal("expected patch operations")
	}
	op := ops[0].(map[string]any)
	if op["op"] != "append" {
		t.Errorf("op = %q", op["op"])
	}
	// Redacted output must not contain raw content.
	if _, ok := op["content"]; ok {
		t.Error("dry-run must redact operation content")
	}
	if _, ok := op["content_hash"]; !ok {
		t.Error("dry-run must include content_hash")
	}
}

func TestCLI_ContextDryRunKimi(t *testing.T) {
	result, _, err := runHarness(t, "context", "dry-run",
		"--event", fixture(t, "pre-context-kimi.json"),
		"--policy", policyFixture(t, "context-harness.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cp := result["decision"].(map[string]any)["context_patch"].(map[string]any)
	summary := cp["summary"].(map[string]any)
	ops := summary["ops"].(float64)
	if ops < 1 {
		t.Errorf("expected >=1 ops, got %v", ops)
	}
}

// ── trace replay ───────────────────────────────────────────────

func TestCLI_TraceReplay(t *testing.T) {
	result, _, err := runHarness(t, "trace", "replay", fixture(t, "context-patch-trace.json"))
	if err != nil {
		t.Fatal(err)
	}
	if result["trace_id"] != "trace_context_gpt" {
		t.Errorf("trace_id = %q", result["trace_id"])
	}
	for _, key := range []string{"events", "decisions", "outcomes"} {
		if items, ok := result[key].([]any); !ok || len(items) == 0 {
			t.Errorf("trace.%s is empty", key)
		}
	}
}

// ── error cases ────────────────────────────────────────────────

func TestCLI_UnknownCommand(t *testing.T) {
	_, raw, _ := runHarness(t, "nonexistent")
	if !strings.Contains(raw, "unknown command") {
		t.Errorf("expected 'unknown command', got: %s", raw)
	}
}

func TestCLI_MissingFlags(t *testing.T) {
	_, raw, _ := runHarness(t, "run")
	if !strings.Contains(raw, "--event") {
		t.Errorf("expected '--event' in error, got: %s", raw)
	}
}
