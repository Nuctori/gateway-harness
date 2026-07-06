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

func TestValidateAcceptsContinuityDropHookAndCondition(t *testing.T) {
	p, err := Decode(strings.NewReader(`{
		"programs": [{
			"name": "continuity-drop",
			"models": ["*"],
			"steps": [{
				"hook": "context.continuity_drop.detected",
				"when": {"context_continuity_drop": true},
				"do": [{
					"action": "context.inject",
					"role": "system",
					"position": "after_existing_system",
					"text": "Preserve explicit project continuity."
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
}

func TestValidateAcceptsGoalResumeHook(t *testing.T) {
	p, err := Decode(strings.NewReader(`{
		"programs": [{
			"name": "goal-resume",
			"models": ["*"],
			"steps": [{
				"hook": "goal.before_resume",
				"do": [{
					"action": "context.inject",
					"role": "system",
					"position": "after_existing_system",
					"text": "Resume with the normalized project context."
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

func TestEffectiveHooksRequiresExplicitHook(t *testing.T) {
	hooks := EffectiveHooks(Step{})
	if len(hooks) != 0 {
		t.Fatalf("unexpected hooks: %#v", hooks)
	}
}

func TestValidateRejectsMissingHook(t *testing.T) {
	p, err := Decode(strings.NewReader(`{
		"programs": [{
			"name": "bad",
			"models": ["*"],
			"steps": [{
				"do": [{"action": "context.inject", "role": "system", "position": "after_existing_system", "text": "x"}]
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil {
		t.Fatal("expected missing hook error")
	}
}

func TestValidateRejectsImplicitInjectRoleAndPosition(t *testing.T) {
	p, err := Decode(strings.NewReader(`{
		"programs": [{
			"name": "bad",
			"models": ["*"],
			"steps": [{
				"hook": "request.before_upstream",
				"do": [{"action": "context.inject", "text": "x"}]
			}]
		}]
	}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil {
		t.Fatal("expected explicit inject role and position error")
	}
}
