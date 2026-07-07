package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/plan"
	"github.com/nuctori/gateway-harness/core/policy"
	hruntime "github.com/nuctori/gateway-harness/core/runtime"
	"github.com/nuctori/gateway-harness/core/trace"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "policy":
		return runPolicy(args[1:])
	case "run":
		return runEvaluate(args[1:], false)
	case "trace":
		return runTrace(args[1:])
	case "context":
		return runContext(args[1:])
	case "help", "-h", "--help":
		return usage()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() error {
	fmt.Println(`harness commands:
  harness policy validate <policy.yaml>
  harness run --event <event.json|yaml> --policy <policy.yaml>
  harness trace replay <trace.json|yaml>
  harness context validate <policy.yaml>
  harness context dry-run --event <event.json|yaml> --policy <policy.yaml>`)
	return nil
}

func runPolicy(args []string) error {
	if len(args) != 2 || args[0] != "validate" {
		return fmt.Errorf("usage: harness policy validate <policy.yaml>")
	}
	p, err := policy.LoadFile(args[1])
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"ok": true, "policy": p.Name})
}

func runContext(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: harness context <validate|dry-run>")
	}
	switch args[0] {
	case "validate":
		if len(args) != 2 {
			return fmt.Errorf("usage: harness context validate <policy.yaml>")
		}
		return runPolicy([]string{"validate", args[1]})
	case "dry-run":
		return runEvaluate(args[1:], true)
	default:
		return fmt.Errorf("unknown context command %q", args[0])
	}
}

func runEvaluate(args []string, redactContext bool) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	eventPath := fs.String("event", "", "event fixture path")
	policyPath := fs.String("policy", "", "policy path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *eventPath == "" || *policyPath == "" {
		return fmt.Errorf("--event and --policy are required")
	}
	p, err := policy.LoadFile(*policyPath)
	if err != nil {
		return err
	}
	var ev event.Event
	if err := loadStructured(*eventPath, &ev); err != nil {
		return err
	}
	eval, err := hruntime.NewEngine(p).Evaluate(ev)
	if err != nil {
		return err
	}
	if redactContext && eval.Decision.ContextPatch != nil {
		// Replace raw content with content_hash in context operations.
		redacted := *eval.Decision.ContextPatch
		redacted.Operations = make([]plan.PatchOperation, len(eval.Decision.ContextPatch.Operations))
		for i, op := range eval.Decision.ContextPatch.Operations {
			redacted.Operations[i] = op.Redacted()
		}
		eval.Decision.ContextPatch = &redacted
	}
	return printJSON(eval)
}

func runTrace(args []string) error {
	if len(args) != 2 || args[0] != "replay" {
		return fmt.Errorf("usage: harness trace replay <trace.json|yaml>")
	}
	var tr trace.Trace
	if err := loadStructured(args[1], &tr); err != nil {
		return err
	}
	return printJSON(map[string]any{
		"trace_id":   tr.TraceID,
		"request_id": tr.RequestID,
		"events":     tr.Events,
		"decisions":  tr.Decisions,
		"outcomes":   tr.Outcomes,
	})
}

func loadStructured(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	switch filepath.Ext(path) {
	case ".json":
		return json.Unmarshal(raw, out)
	default:
		return yaml.Unmarshal(raw, out)
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
