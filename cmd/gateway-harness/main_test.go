package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nuctori/gateway-harness/rule"
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
