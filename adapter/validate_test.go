package adapter

import (
	"strings"
	"testing"
)

func TestValidateAcceptsNewAPIManifest(t *testing.T) {
	m, err := Decode(strings.NewReader(`{
		"version": "0.2",
		"adapter": "newapi",
		"hooks": ["request.before_upstream", "responses.compact.before_upstream"],
		"actions": ["context.inject", "context.truncate"],
		"request_shapes": ["chat", "responses", "responses_compact"],
		"guards": ["explicit_mutation_only", "preserve_responses_tool_chain", "redacted_trace"]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(m); err != nil {
		t.Fatalf("validate: %v", err)
	}
	summary := Summarize(m)
	if summary.Adapter != "newapi" || summary.Hooks != 2 || summary.Actions != 2 || summary.Guards != 3 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestValidateRejectsUnsupportedHook(t *testing.T) {
	m, err := Decode(strings.NewReader(`{
		"adapter": "bad",
		"hooks": ["magic.before_anything"],
		"actions": ["context.inject"]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(m); err == nil {
		t.Fatal("expected unsupported hook error")
	}
}

func TestValidateRejectsUnsupportedGuard(t *testing.T) {
	m, err := Decode(strings.NewReader(`{
		"adapter": "bad",
		"hooks": ["request.before_upstream"],
		"actions": ["context.inject"],
		"guards": ["hidden_context_budget"]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(m); err == nil {
		t.Fatal("expected unsupported guard error")
	}
}
