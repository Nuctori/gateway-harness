package schema

func GoalGateFormModel() map[string]any {
	return map[string]any{
		"title":             "Gateway Harness Goal Gate",
		"description":       "Host-facing form model for explicit Goal Gate completion review configuration.",
		"transparency_note": "Default-off. Unless enabled here, Goal Gate must not invoke an external runner, intercept completion, inject context, or write ledger records.",
		"schema_ref":        "https://gateway-harness.dev/schema/gateway-harness.goal-gate-config.schema.json",
		"sections": []map[string]any{
			{
				"id":          "activation",
				"title":       "Activation",
				"description": "Turn Goal Gate on explicitly and select the lifecycle hook.",
				"fields": []map[string]any{
					{
						"path":                  "enabled",
						"title":                 "Enable Goal Gate",
						"control":               "toggle",
						"recommended":           true,
						"description":           "Default-off master switch for host-side completion interception.",
						"required_when_enabled": false,
					},
					{
						"path":                  "hook",
						"title":                 "Lifecycle Hook",
						"control":               "select",
						"required_when_enabled": true,
						"recommended":           true,
						"description":           "Explicit lifecycle point where Goal Gate may run. See hook_catalog for the broader Harness hook map and localized explanations.",
						"catalog_ref":           "hook_catalog",
						"options":               goalGateHookOptions(),
					},
				},
			},
			{
				"id":          "runner",
				"title":       "Runner",
				"description": "Configure the external AI reviewer process. Gateway Harness does not embed a model or agent runtime.",
				"fields": []map[string]any{
					{
						"path":                  "runner.command",
						"title":                 "Runner Command",
						"control":               "command",
						"required_when_enabled": true,
						"recommended":           true,
						"description":           "Executable to launch, for example python or a dedicated sidecar binary.",
					},
					{
						"path":                  "runner.workdir",
						"title":                 "Runner Working Directory",
						"control":               "text",
						"required_when_enabled": false,
						"recommended":           true,
						"description":           "Optional working directory. Relative paths should be resolved against the config file location.",
					},
					{
						"path":                  "runner.args",
						"title":                 "Runner Arguments",
						"control":               "text",
						"required_when_enabled": false,
						"recommended":           true,
						"description":           "Explicit argument list for the external runner.",
					},
				},
			},
			{
				"id":          "review_scope",
				"title":       "Review Scope",
				"description": "Choose what redacted context the reviewer may see and what actions it may propose.",
				"fields": []map[string]any{
					{
						"path":                  "allowed_inputs",
						"title":                 "Allowed Event Inputs",
						"control":               "multi_select",
						"required_when_enabled": true,
						"recommended":           true,
						"description":           "Only these redacted input fields may be passed to the external reviewer. Default to the minimal summary set; broader context must be explicitly opted in.",
						"options": []map[string]any{
							{"value": "goal_state", "title": "Goal State", "description": "Required machine-readable goal status and retry state."},
							{"value": "work_summary", "title": "Work Summary", "description": "Concise summary of completed work."},
							{"value": "test_results", "title": "Test Results", "description": "Focused test outcomes or command summaries."},
							{"value": "blockers", "title": "Blockers", "description": "Known missing pieces or unresolved blockers."},
							{"value": "recent_trace", "title": "Recent Trace", "description": "Recent redacted execution trace snippets."},
							{"value": "changed_files", "title": "Changed Files", "description": "Relevant changed file paths or summaries."},
							{"value": "verification_summary", "title": "Verification Summary", "description": "Broader verification or deployment status."},
							{"value": "user_goal", "title": "User Goal", "description": "Original user objective in concise redacted form."},
						},
					},
					{
						"path":                  "allowed_actions",
						"title":                 "Allowed Proposal Actions",
						"control":               "multi_select",
						"required_when_enabled": true,
						"recommended":           true,
						"description":           "Only these actions may be accepted from the external reviewer.",
						"options": []map[string]any{
							{"value": "goal.approve_complete", "title": "Approve Complete", "description": "Allow completion."},
							{"value": "goal.reject_complete", "title": "Reject Complete", "description": "Block completion."},
							{"value": "goal.request_continue", "title": "Request Continue", "description": "Return an explicit continuation instruction."},
							{"value": "diagnosis.note.create", "title": "Create Diagnosis Note", "description": "Attach a redacted diagnosis note."},
							{"value": "ledger.artifact.create", "title": "Create Ledger Artifact", "description": "Reference a hashed artifact in the ledger."},
							{"value": "context.inject", "title": "Inject Context", "description": "Optional advanced action for explicit follow-up guidance only."},
						},
					},
				},
			},
			{
				"id":          "safety",
				"title":       "Safety Limits",
				"description": "Prevent infinite continue loops and keep retries explicit.",
				"fields": []map[string]any{
					{
						"path":                  "max_continue_attempts",
						"title":                 "Max Continue Attempts",
						"control":               "number",
						"required_when_enabled": false,
						"recommended":           true,
						"description":           "Host-side default retry limit when the event omitted one.",
					},
					{
						"path":                  "cooldown_seconds",
						"title":                 "Cooldown Seconds",
						"control":               "number",
						"required_when_enabled": false,
						"recommended":           true,
						"description":           "Host-side default cooldown between repeated continuation attempts.",
					},
				},
			},
		},
	}
}
