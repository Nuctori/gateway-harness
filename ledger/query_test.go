package ledger

import (
	"strings"
	"testing"
)

func TestQueryFiltersSessionsByProjectSessionTagAndEventType(t *testing.T) {
	l, err := Decode(strings.NewReader(projectSessionLedgerJSON))
	if err != nil {
		t.Fatalf("decode ledger: %v", err)
	}

	results := Query(l, QueryOptions{
		ProjectID:  "project_gateway_harness",
		SessionID:  "session_codex_context_harness",
		Tags:       []string{"adapter:newapi", "domain:coding"},
		EventTypes: []string{"compact"},
	})

	if len(results) != 1 {
		t.Fatalf("expected one result, got %+v", results)
	}
	result := results[0]
	if result.ProjectID != "project_gateway_harness" || result.SessionID != "session_codex_context_harness" {
		t.Fatalf("unexpected result identity: %+v", result)
	}
	if len(result.MatchedEvents) != 1 || result.MatchedEvents[0] != "evt_compact_1" {
		t.Fatalf("unexpected matched events: %+v", result.MatchedEvents)
	}
	if result.EventCounts["compact"] != 1 || result.Artifacts != 2 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.ArtifactRefs) != 2 || result.ArtifactRefs[0] != "artifact_compact_summary_1" {
		t.Fatalf("unexpected artifact refs: %+v", result.ArtifactRefs)
	}
}

func TestQueryRequiresAllTags(t *testing.T) {
	l, err := Decode(strings.NewReader(projectSessionLedgerJSON))
	if err != nil {
		t.Fatalf("decode ledger: %v", err)
	}

	results := Query(l, QueryOptions{
		Tags: []string{"adapter:newapi", "domain:writing"},
	})

	if len(results) != 0 {
		t.Fatalf("expected no results, got %+v", results)
	}
}
