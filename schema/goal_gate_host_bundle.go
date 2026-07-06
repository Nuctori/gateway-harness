package schema

import "encoding/json"

func GoalGateHostBundle() map[string]any {
	return map[string]any{
		"title":             "Gateway Harness Goal Gate Host Bundle",
		"description":       "Single host-facing bundle for wiring Goal Gate configuration, UI rendering, and result handling.",
		"transparency_note": "Default-off. Unless enabled by explicit host config, Goal Gate must not call an external runner, intercept completion, inject continuation context, or append ledger records.",
		"interaction_model": map[string]any{
			"primary_surface":            "chat_assistant",
			"secondary_surface":          "settings_form",
			"assistant_generates":        "explicit_config_draft",
			"assistant_applies_directly": false,
			"manual_edit_supported":      true,
			"ai_config_supported":        true,
		},
		"i18n": map[string]any{
			"default_locale":    "zh-CN",
			"supported_locales": []string{"zh-CN", "en-US"},
		},
		"hook_catalog":  HookCatalog(),
		"config_schema": mustParseJSON(GoalGateConfigJSON),
		"form_schema":   mustParseJSON(GoalGateFormJSON),
		"form_model":    GoalGateFormModel(),
		"result_schema": mustParseJSON(GoalGateResultJSON),
		"assistant_examples": []map[string]any{
			{
				"user":    "Enable Goal Gate with the smolagents reviewer and keep three continue attempts.",
				"outcome": "Generate an explicit enabled draft with python runner defaults and the minimal default review scope.",
			},
			{
				"user":    "Add context.inject so the reviewer may propose a continuation patch, but keep everything explicit.",
				"outcome": "Keep the host transparent and add only the explicit context.inject allowance to allowed_actions.",
			},
			{
				"user":    "Authorize fuller redacted context including changed files and verification summary.",
				"outcome": "Expand allowed_inputs only because the user explicitly requested a broader review scope.",
			},
		},
		"examples": map[string]any{
			"config": map[string]any{
				"enabled": false,
				"hook":    "goal.before_complete",
				"runner": map[string]any{
					"command": "python",
					"workdir": "../..",
					"args":    []string{"examples/smolagents/goal_reviewer.py"},
				},
				"allowed_inputs":        []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace"},
				"allowed_actions":       []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
				"max_continue_attempts": 3,
				"cooldown_seconds":      60,
			},
			"approve_result": map[string]any{
				"allow_complete":       true,
				"continue_work":        false,
				"continue_instruction": "",
				"continuation_patches": []any{},
				"next_goal_state": map[string]any{
					"status":                "complete",
					"attempt":               1,
					"max_continue_attempts": 3,
					"cooldown_seconds":      60,
				},
			},
			"reject_result": map[string]any{
				"allow_complete":       false,
				"continue_work":        true,
				"continue_instruction": "Deploy the image and run the smoke test.",
				"continuation_patches": []any{},
				"next_goal_state": map[string]any{
					"status":                     "pending_complete",
					"attempt":                    2,
					"max_continue_attempts":      3,
					"cooldown_seconds":           60,
					"last_rejection_reason_hash": "sha256:example",
				},
			},
			"failure_result": map[string]any{
				"enabled":   true,
				"triggered": true,
				"failure": map[string]any{
					"stage":          "runner",
					"code":           "goal_gate_runner_failed",
					"message":        "agent command failed",
					"runner_command": "python",
				},
			},
		},
	}
}

func mustParseJSON(text string) any {
	var value any
	if err := json.Unmarshal([]byte(text), &value); err != nil {
		panic(err)
	}
	return value
}
