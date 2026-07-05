package policy

import (
	"strings"
	"testing"
)

func TestValidateAcceptsNewAPIPolicy(t *testing.T) {
	p, err := Decode(strings.NewReader(`{
		"version": "0.1",
		"programs": [{
			"name": "newapi-coding",
			"models": ["*"],
			"tags": ["domain:coding"],
			"budget": {"max_patch_ops": 16, "max_added_tokens": 1200},
			"steps": [{
				"hook": "responses.compact.before_upstream",
				"when": {"model_matches": "*"},
				"do": [{
					"action": "context.inject",
					"role": "system",
					"position": "after_existing_system",
					"text": "Preserve user intent."
				}]
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err != nil {
		t.Fatalf("validate: %v", err)
	}
	summary := Summarize(p)
	if summary.Programs != 1 || summary.Steps != 1 || summary.Actions != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestValidateRejectsUnsupportedHook(t *testing.T) {
	p, err := Decode(strings.NewReader(`{
		"programs": [{
			"name": "bad",
			"models": ["*"],
			"steps": [{
				"hook": "memory.after_magic",
				"do": [{"action": "context.inject", "text": "x"}]
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil {
		t.Fatal("expected unsupported hook error")
	}
}

func TestValidateRejectsUnsupportedAction(t *testing.T) {
	p, err := Decode(strings.NewReader(`{
		"programs": [{
			"name": "bad",
			"models": ["*"],
			"steps": [{
				"do": [{"action": "context.exec", "text": "x"}]
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil {
		t.Fatal("expected unsupported action error")
	}
}

func TestEffectiveHooksDefaultsToBeforeUpstream(t *testing.T) {
	hooks := EffectiveHooks(Step{})
	if len(hooks) != 1 || hooks[0] != DefaultHook {
		t.Fatalf("unexpected hooks: %#v", hooks)
	}
}

