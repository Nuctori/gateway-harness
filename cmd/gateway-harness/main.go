package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/conformance"
	"github.com/Nuctori/gateway-harness/ledger"
	"github.com/Nuctori/gateway-harness/policy"
	"github.com/Nuctori/gateway-harness/schema"
	"github.com/Nuctori/gateway-harness/steward"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "validate":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		p := mustLoadPolicy(os.Args[2])
		if err := policy.Validate(p); err != nil {
			fmt.Fprintf(os.Stderr, "invalid policy: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("policy ok")
	case "explain":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		p := mustLoadPolicy(os.Args[2])
		if err := policy.Validate(p); err != nil {
			fmt.Fprintf(os.Stderr, "invalid policy: %v\n", err)
			os.Exit(1)
		}
		printSummary(p)
	case "schema":
		fmt.Print(schema.PolicyJSON)
	case "dry-run-policy":
		if len(os.Args) != 5 && len(os.Args) != 6 {
			usage()
			os.Exit(2)
		}
		p := mustLoadPolicy(os.Args[2])
		options := policy.DryRunOptions{Hook: os.Args[3]}
		if len(os.Args) == 6 {
			estimatedTokens, err := strconv.Atoi(os.Args[5])
			if err != nil || estimatedTokens < 0 {
				fmt.Fprintf(os.Stderr, "estimated_tokens must be a non-negative integer\n")
				os.Exit(2)
			}
			options.EstimatedTokens = estimatedTokens
		}
		result, err := policy.DryRun(p, mustReadFile("open policy dry-run request", os.Args[4]), options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "policy dry-run failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(result)
	case "validate-adapter":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		m := mustLoadAdapterManifest(os.Args[2])
		if err := adapter.Validate(m); err != nil {
			fmt.Fprintf(os.Stderr, "invalid adapter manifest: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("adapter manifest ok")
	case "explain-adapter":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		m := mustLoadAdapterManifest(os.Args[2])
		if err := adapter.Validate(m); err != nil {
			fmt.Fprintf(os.Stderr, "invalid adapter manifest: %v\n", err)
			os.Exit(1)
		}
		printAdapterSummary(m)
	case "adapter-schema":
		fmt.Print(schema.AdapterJSON)
	case "conformance-schema":
		fmt.Print(schema.ConformanceJSON)
	case "validate-ledger":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		l := mustLoadLedger(os.Args[2])
		if err := ledger.Validate(l); err != nil {
			fmt.Fprintf(os.Stderr, "invalid ledger: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ledger ok")
	case "explain-ledger":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		l := mustLoadLedger(os.Args[2])
		if err := ledger.Validate(l); err != nil {
			fmt.Fprintf(os.Stderr, "invalid ledger: %v\n", err)
			os.Exit(1)
		}
		printLedgerSummary(l)
	case "ledger-schema":
		fmt.Print(schema.LedgerJSON)
	case "validate-steward":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		s := mustLoadStewardSpec(os.Args[2])
		if err := steward.Validate(s); err != nil {
			fmt.Fprintf(os.Stderr, "invalid steward spec: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("steward spec ok")
	case "explain-steward":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		s := mustLoadStewardSpec(os.Args[2])
		if err := steward.Validate(s); err != nil {
			fmt.Fprintf(os.Stderr, "invalid steward spec: %v\n", err)
			os.Exit(1)
		}
		printStewardSummary(s)
	case "steward-schema":
		fmt.Print(schema.StewardJSON)
	case "validate-steward-proposal":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		s := mustLoadStewardSpec(os.Args[2])
		p := mustLoadStewardProposal(os.Args[3])
		if err := steward.ValidateProposal(s, p); err != nil {
			fmt.Fprintf(os.Stderr, "invalid steward proposal: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("steward proposal ok")
	case "explain-steward-proposal":
		if len(os.Args) != 4 {
			usage()
			os.Exit(2)
		}
		s := mustLoadStewardSpec(os.Args[2])
		p := mustLoadStewardProposal(os.Args[3])
		if err := steward.ValidateProposal(s, p); err != nil {
			fmt.Fprintf(os.Stderr, "invalid steward proposal: %v\n", err)
			os.Exit(1)
		}
		printStewardProposalSummary(p)
	case "steward-proposal-schema":
		fmt.Print(schema.StewardProposalJSON)
	case "dry-run-steward-proposal":
		if len(os.Args) != 5 {
			usage()
			os.Exit(2)
		}
		s := mustLoadStewardSpec(os.Args[2])
		p := mustLoadStewardProposal(os.Args[3])
		request := mustReadFile("open steward dry-run request", os.Args[4])
		result, err := steward.DryRunProposal(s, p, request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "steward proposal dry-run failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(result)
	case "validate-conformance":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		f := mustLoadConformanceFixture(os.Args[2])
		if err := conformance.Validate(f); err != nil {
			fmt.Fprintf(os.Stderr, "invalid conformance fixture: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("conformance fixture ok")
	case "explain-conformance":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		f := mustLoadConformanceFixture(os.Args[2])
		if err := conformance.Validate(f); err != nil {
			fmt.Fprintf(os.Stderr, "invalid conformance fixture: %v\n", err)
			os.Exit(1)
		}
		printConformanceSummary(f)
	case "replay-conformance":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		f := mustLoadConformanceFixture(os.Args[2])
		result, err := conformance.ReplayFakeUpstream(context.Background(), f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "conformance replay failed: %v\n", err)
			os.Exit(1)
		}
		printReplayResult(result)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `gateway-harness

Usage:
  gateway-harness validate <policy.json>
  gateway-harness explain <policy.json>
  gateway-harness schema
  gateway-harness dry-run-policy <policy.json> <hook> <request.json> [estimated_tokens]
  gateway-harness validate-adapter <adapter.capability.json>
  gateway-harness explain-adapter <adapter.capability.json>
  gateway-harness adapter-schema
  gateway-harness validate-conformance <fixture.json>
  gateway-harness explain-conformance <fixture.json>
  gateway-harness replay-conformance <fixture.json>
  gateway-harness conformance-schema
  gateway-harness validate-ledger <ledger.json>
  gateway-harness explain-ledger <ledger.json>
  gateway-harness ledger-schema
  gateway-harness validate-steward <steward.json>
  gateway-harness explain-steward <steward.json>
  gateway-harness steward-schema
  gateway-harness validate-steward-proposal <steward.json> <proposal.json>
  gateway-harness explain-steward-proposal <steward.json> <proposal.json>
  gateway-harness steward-proposal-schema
  gateway-harness dry-run-steward-proposal <steward.json> <proposal.json> <request.json>`)
}

func mustLoadPolicy(path string) policy.Policy {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open policy: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	p, err := policy.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode policy: %v\n", err)
		os.Exit(1)
	}
	return p
}

func mustLoadAdapterManifest(path string) adapter.Manifest {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open adapter manifest: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	m, err := adapter.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode adapter manifest: %v\n", err)
		os.Exit(1)
	}
	return m
}

func mustLoadConformanceFixture(path string) conformance.Fixture {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open conformance fixture: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	f, err := conformance.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode conformance fixture: %v\n", err)
		os.Exit(1)
	}
	return f
}

func mustLoadLedger(path string) ledger.Ledger {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open ledger: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	l, err := ledger.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode ledger: %v\n", err)
		os.Exit(1)
	}
	return l
}

func mustLoadStewardSpec(path string) steward.Spec {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open steward spec: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	s, err := steward.Decode(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode steward spec: %v\n", err)
		os.Exit(1)
	}
	return s
}

func mustLoadStewardProposal(path string) steward.Proposal {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open steward proposal: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	p, err := steward.DecodeProposal(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode steward proposal: %v\n", err)
		os.Exit(1)
	}
	return p
}

func mustReadFile(label string, path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
		os.Exit(1)
	}
	return data
}

func printSummary(p policy.Policy) {
	summary := policy.Summarize(p)
	data := map[string]any{
		"programs": summary.Programs,
		"steps":    summary.Steps,
		"actions":  summary.Actions,
		"hooks":    summary.Hooks,
	}
	encoded, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(encoded))
	for _, program := range p.Programs {
		fmt.Printf("- %s models=%s tags=%s steps=%d\n", program.Name, strings.Join(program.Models, ","), strings.Join(program.Tags, ","), len(program.Steps))
	}
}

func printAdapterSummary(m adapter.Manifest) {
	summary := adapter.Summarize(m)
	data := map[string]any{
		"adapter":        summary.Adapter,
		"hooks":          summary.Hooks,
		"actions":        summary.Actions,
		"request_shapes": summary.RequestShapes,
		"guards":         summary.Guards,
	}
	encoded, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(encoded))
	fmt.Printf("- %s hooks=%s actions=%s guards=%s\n", m.Adapter, strings.Join(m.Hooks, ","), strings.Join(m.Actions, ","), strings.Join(m.Guards, ","))
}

func printConformanceSummary(f conformance.Fixture) {
	summary := conformance.Summarize(f)
	data := map[string]any{
		"name":            summary.Name,
		"adapter":         summary.Adapter,
		"request_shape":   summary.RequestShape,
		"programs":        summary.Programs,
		"steps":           summary.Steps,
		"actions":         summary.Actions,
		"required_guards": summary.RequiredGuards,
	}
	encoded, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(encoded))
}

func printReplayResult(result conformance.ReplayResult) {
	data := map[string]any{
		"name":         result.Name,
		"path":         result.Path,
		"status_code":  result.StatusCode,
		"request_body": result.RequestBody,
	}
	encoded, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(encoded))
}

func printJSON(value any) {
	encoded, _ := json.MarshalIndent(value, "", "  ")
	fmt.Println(string(encoded))
}

func printLedgerSummary(l ledger.Ledger) {
	summary := ledger.Summarize(l)
	data := map[string]any{
		"projects":  summary.Projects,
		"sessions":  summary.Sessions,
		"events":    summary.Events,
		"artifacts": summary.Artifacts,
	}
	encoded, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(encoded))
	for _, project := range l.Projects {
		for _, session := range project.Sessions {
			fmt.Printf("- %s/%s events=%d artifacts=%d tags=%s\n", project.ID, session.ID, len(session.Events), len(session.Artifacts), strings.Join(session.Tags, ","))
		}
	}
}

func printStewardSummary(s steward.Spec) {
	summary := steward.Summarize(s)
	data := map[string]any{
		"name":            summary.Name,
		"hooks":           summary.Hooks,
		"inputs":          summary.Inputs,
		"allowed_actions": summary.AllowedActions,
		"artifact_types":  summary.ArtifactTypes,
		"required_guards": summary.RequiredGuards,
	}
	encoded, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(encoded))
	fmt.Printf("- %s model=%s hooks=%s actions=%s\n", s.Name, s.StewardModel, strings.Join(s.Hooks, ","), strings.Join(s.AllowedActions, ","))
}

func printStewardProposalSummary(p steward.Proposal) {
	summary := steward.SummarizeProposal(p)
	data := map[string]any{
		"id":      summary.ID,
		"steward": summary.Steward,
		"hook":    summary.Hook,
		"outputs": summary.Outputs,
	}
	encoded, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(encoded))
	for _, output := range p.Outputs {
		fmt.Printf("- %s reason=%s\n", output.Action, output.Reason)
	}
}
