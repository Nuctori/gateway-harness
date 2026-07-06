package schema

import "testing"

func TestHookCatalogIncludesLocalizedGoalLifecycleEntries(t *testing.T) {
	catalog := HookCatalog()
	foundBeforeResume := false
	foundAfterComplete := false
	for _, entry := range catalog {
		id, _ := entry["id"].(string)
		locales, _ := entry["locales"].(map[string]any)
		if len(locales) == 0 {
			t.Fatalf("expected locales for hook %q", id)
		}
		switch id {
		case "goal.before_resume":
			foundBeforeResume = true
		case "goal.after_complete":
			foundAfterComplete = true
		}
	}
	if !foundBeforeResume || !foundAfterComplete {
		t.Fatalf("expected new goal lifecycle hooks in catalog: %#v", catalog)
	}
}

func TestGoalGateHookOptionsOnlyExposeHostSelectableHooks(t *testing.T) {
	options := goalGateHookOptions()
	if len(options) != 1 {
		t.Fatalf("expected exactly one goal gate selectable hook, got %d", len(options))
	}
	if got := options[0]["value"]; got != "goal.before_complete" {
		t.Fatalf("unexpected selectable hook: %#v", options[0])
	}
	if descriptions, ok := options[0]["descriptions"].(map[string]string); !ok || descriptions["zh-CN"] == "" || descriptions["en-US"] == "" {
		t.Fatalf("expected bilingual descriptions on hook option: %#v", options[0])
	}
}
