package ledger

import (
	"strings"
	"testing"
)

func TestValidateAcceptsProjectSessionLedger(t *testing.T) {
	l, err := Decode(strings.NewReader(projectSessionLedgerJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(l); err != nil {
		t.Fatalf("validate: %v", err)
	}
	summary := Summarize(l)
	if summary.Projects != 1 || summary.Sessions != 1 || summary.Events != 4 || summary.Artifacts != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestValidateRejectsRawContentField(t *testing.T) {
	raw := strings.Replace(projectSessionLedgerJSON, `"metadata": {"policy": "context-harness"}`, `"content": "raw prompt", "metadata": {"policy": "context-harness"}`, 1)
	if _, err := Decode(strings.NewReader(raw)); err == nil {
		t.Fatal("expected unknown raw content field error")
	}
}

func TestValidateRejectsDuplicateEventID(t *testing.T) {
	raw := strings.Replace(projectSessionLedgerJSON, `"id": "evt_harness_1"`, `"id": "evt_request_1"`, 1)
	l, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(l); err == nil {
		t.Fatal("expected duplicate event id error")
	}
}

func TestValidateRejectsRawMetadataKey(t *testing.T) {
	raw := strings.Replace(projectSessionLedgerJSON, `"metadata": {"policy": "context-harness"}`, `"metadata": {"prompt": "raw prompt"}`, 1)
	l, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(l); err == nil {
		t.Fatal("expected raw metadata key error")
	}
}

const projectSessionLedgerJSON = `{
  "version": "0.3",
  "projects": [
    {
      "id": "project_gateway_harness",
      "name": "Gateway Harness",
      "tags": ["repo:gateway-harness", "adapter:newapi"],
      "sessions": [
        {
          "id": "session_codex_context_harness",
          "title": "Codex context harness work",
          "started_at": "2026-07-05T18:00:00Z",
          "tags": ["domain:coding", "risk:medium"],
          "events": [
            {
              "id": "evt_request_1",
              "type": "request",
              "at": "2026-07-05T18:01:00Z",
              "request_shape": "responses",
              "model": "gpt-5.4-mini",
              "metadata": {"policy": "context-harness"}
            },
            {
              "id": "evt_harness_1",
              "type": "harness_action",
              "at": "2026-07-05T18:01:01Z",
              "hook": "responses.before_upstream",
              "action": "context.inject",
              "policy_version": "0.2",
              "trace_hash": "sha256:1111111111111111111111111111111111111111111111111111111111111111"
            },
            {
              "id": "evt_compact_1",
              "type": "compact",
              "at": "2026-07-05T18:10:00Z",
              "hook": "responses.compact.before_upstream",
              "trace_hash": "sha256:2222222222222222222222222222222222222222222222222222222222222222"
            },
            {
              "id": "evt_error_1",
              "type": "error",
              "at": "2026-07-05T18:11:00Z",
              "error_code": "upstream_400_function_call_output_reference"
            }
          ],
          "artifacts": [
            {
              "id": "artifact_trace_1",
              "type": "trace",
              "content_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
            },
            {
              "id": "artifact_compact_summary_1",
              "type": "compact_summary",
              "content_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
              "ref": "memory://summaries/session_codex_context_harness/compact_1"
            }
          ]
        }
      ]
    }
  ]
}`
