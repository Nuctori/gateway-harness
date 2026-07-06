package ledger

import (
	"strings"
	"testing"
)

func TestAppendCreatesProjectSessionAndEvent(t *testing.T) {
	record, err := DecodeAppendRecord(strings.NewReader(`{
	  "project": {"id": "project_gateway_harness", "name": "Gateway Harness", "tags": ["repo:gateway-harness"]},
	  "session": {"id": "session_codex_context_harness", "title": "Codex work", "started_at": "2026-07-06T04:00:00Z", "tags": ["domain:coding"]},
	  "artifacts": [
	    {"id": "artifact_continuity_summary_1", "type": "compact_summary", "content_hash": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}
	  ],
	  "event": {
	    "id": "evt_continuity_drop_1",
	    "type": "compact",
	    "at": "2026-07-06T04:01:00Z",
	    "hook": "context.continuity_drop.detected",
	    "action": "context.inject_ledger_summary",
	    "artifact_refs": ["artifact_continuity_summary_1"],
	    "metadata": {"retained_percent": "9"}
	  }
	}`))
	if err != nil {
		t.Fatalf("decode append record: %v", err)
	}

	got, result, err := Append(Ledger{Version: "0.3"}, record)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if result.ProjectID != "project_gateway_harness" || result.SessionID != "session_codex_context_harness" || result.EventID != "evt_continuity_drop_1" {
		t.Fatalf("unexpected append result: %+v", result)
	}
	if result.Events != 1 || result.Artifacts != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if err := Validate(got); err != nil {
		t.Fatalf("validate appended ledger: %v", err)
	}
}

func TestAppendMergesIntoExistingSession(t *testing.T) {
	l, err := Decode(strings.NewReader(projectSessionLedgerJSON))
	if err != nil {
		t.Fatalf("decode ledger: %v", err)
	}
	record := AppendRecord{
		Project: AppendProject{ID: "project_gateway_harness", Tags: []string{"repo:gateway-harness", "adapter:newapi"}},
		Session: AppendSession{ID: "session_codex_context_harness", Tags: []string{"domain:coding", "continuity:drop"}},
		Event: Event{
			ID:           "evt_continuity_drop_2",
			Type:         "harness_action",
			At:           "2026-07-06T04:02:00Z",
			Hook:         "context.continuity_drop.detected",
			Action:       "context.inject_ledger_summary",
			ArtifactRefs: []string{"artifact_compact_summary_1"},
		},
	}

	got, result, err := Append(l, record)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if result.Events != 5 || result.Artifacts != 2 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	results := Query(got, QueryOptions{Tags: []string{"continuity:drop"}, EventTypes: []string{"harness_action"}})
	if len(results) != 1 {
		t.Fatalf("expected continuity-tagged session, got %+v", results)
	}
}

func TestAppendRejectsRawMetadataKey(t *testing.T) {
	record := AppendRecord{
		Project: AppendProject{ID: "project_gateway_harness"},
		Session: AppendSession{ID: "session_codex_context_harness", StartedAt: "2026-07-06T04:00:00Z"},
		Event: Event{
			ID:       "evt_bad",
			Type:     "request",
			At:       "2026-07-06T04:01:00Z",
			Metadata: map[string]string{"prompt": "raw prompt"},
		},
	}

	if _, _, err := Append(Ledger{Version: "0.3"}, record); err == nil {
		t.Fatal("expected raw metadata key rejection")
	}
}

func TestAppendDoesNotMutateInputLedgerOnFailure(t *testing.T) {
	l, err := Decode(strings.NewReader(projectSessionLedgerJSON))
	if err != nil {
		t.Fatalf("decode ledger: %v", err)
	}
	originalEvents := len(l.Projects[0].Sessions[0].Events)
	record := AppendRecord{
		Project: AppendProject{ID: "project_gateway_harness"},
		Session: AppendSession{ID: "session_codex_context_harness"},
		Event: Event{
			ID:           "evt_bad_ref",
			Type:         "compact",
			At:           "2026-07-06T04:01:00Z",
			ArtifactRefs: []string{"artifact_missing"},
		},
	}

	if _, _, err := Append(l, record); err == nil {
		t.Fatal("expected missing artifact ref rejection")
	}
	if len(l.Projects[0].Sessions[0].Events) != originalEvents {
		t.Fatal("append mutated input ledger on failure")
	}
}

func TestAppendRequiresStartedAtForNewSession(t *testing.T) {
	record := AppendRecord{
		Project: AppendProject{ID: "project_gateway_harness"},
		Session: AppendSession{ID: "session_codex_context_harness"},
		Event:   Event{ID: "evt_request_1", Type: "request", At: "2026-07-06T04:01:00Z"},
	}

	if _, _, err := Append(Ledger{Version: "0.3"}, record); err == nil {
		t.Fatal("expected started_at requirement for new session")
	}
}
