package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	body := string(stdin)
	if !strings.Contains(body, "goal.before_complete") {
		fmt.Fprintln(os.Stderr, "goal hook missing from helper stdin")
		os.Exit(2)
	}
	if !strings.Contains(body, `"redacted":true`) {
		fmt.Fprintln(os.Stderr, "expected canonical redacted event")
		os.Exit(2)
	}
	proposal := strings.TrimSpace(os.Getenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL"))
	if proposal == "" {
		proposal = `{
  "version": "0.1",
  "id": "proposal_goal_review_approve_default",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {
      "action": "goal.approve_complete",
      "reason": "helper default approval"
    }
  ]
}`
	}
	_, _ = fmt.Fprint(os.Stdout, proposal)
}
