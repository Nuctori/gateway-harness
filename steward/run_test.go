package steward

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunExternalAgentValidatesProposalFromCommand(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, raw, err := DecodeEvent(strings.NewReader(compactStewardEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	p, err := RunExternalAgent(context.Background(), s, e, raw, os.Args[0], "-test.run=TestStewardAgentHelperProcess", "--", "--steward-helper")
	if err != nil {
		t.Fatalf("run external agent: %v", err)
	}
	if p.Steward != s.Name || p.Hook != e.Hook || len(p.Outputs) != 4 {
		t.Fatalf("unexpected proposal: %+v", p)
	}
}

func TestRunExternalAgentRejectsUnredactedEvent(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	rawEvent := strings.Replace(compactStewardEventJSON, `"redacted": true`, `"redacted": false`, 1)
	e, raw, err := DecodeEvent(strings.NewReader(rawEvent))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if _, err := RunExternalAgent(context.Background(), s, e, raw, os.Args[0], "-test.run=TestStewardAgentHelperProcess", "--", "--steward-helper"); err == nil {
		t.Fatal("expected unredacted event rejection")
	}
}

func TestRunExternalAgentCanonicalizesEventBeforeCommand(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	rawEvent := strings.Replace(
		compactStewardEventJSON,
		`"redacted_trace": [
      {"role": "user", "content_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
    ]`,
		`"redacted_trace": [{"role": "user", "content": "raw prompt text"}],
    "redacted_trace": [
      {"role": "user", "content_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
    ]`,
		1,
	)
	e, raw, err := DecodeEvent(strings.NewReader(rawEvent))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if _, err := RunExternalAgent(context.Background(), s, e, raw, os.Args[0], "-test.run=TestStewardAgentHelperProcess", "--", "--steward-helper"); err != nil {
		t.Fatalf("expected canonicalized event to hide duplicate-key raw content, got %v", err)
	}
}

func TestValidateEventRejectsUndeclaredInput(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	rawEvent := strings.Replace(compactStewardEventJSON, `"user_goal": "continue current gateway harness implementation"`, `"undeclared": "nope"`, 1)
	e, _, err := DecodeEvent(strings.NewReader(rawEvent))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if err := ValidateEvent(s, e); err == nil {
		t.Fatal("expected undeclared input rejection")
	}
}

func TestValidateEventRejectsRawNestedContent(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	rawEvent := strings.Replace(compactStewardEventJSON, `"content_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`, `"content": "raw prompt text"`, 1)
	e, _, err := DecodeEvent(strings.NewReader(rawEvent))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if err := ValidateEvent(s, e); err == nil {
		t.Fatal("expected raw nested content rejection")
	}
}

func TestValidateGoalEventAcceptsDeclaredInputs(t *testing.T) {
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, _, err := DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if err := ValidateEvent(s, e); err != nil {
		t.Fatalf("validate event: %v", err)
	}
}

func TestStewardAgentHelperProcess(t *testing.T) {
	if !hasArg("--steward-helper") {
		t.Skip("helper process only")
	}
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if strings.Contains(string(stdin), "raw prompt text") {
		fmt.Fprintln(os.Stderr, "raw prompt leaked to helper")
		os.Exit(2)
	}
	_, _ = fmt.Fprint(os.Stdout, compactStewardProposalJSON)
	os.Exit(0)
}

func hasArg(want string) bool {
	for _, arg := range os.Args {
		if arg == want {
			return true
		}
	}
	return false
}

const compactStewardEventJSON = `{
  "hook": "responses.compact.before_upstream",
  "redacted": true,
  "inputs": {
    "user_goal": "continue current gateway harness implementation",
    "session_tags": ["project:gateway-harness"],
    "redacted_trace": [
      {"role": "user", "content_hash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
    ]
  }
}`
