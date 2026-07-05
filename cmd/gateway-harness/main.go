package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/conformance"
	"github.com/Nuctori/gateway-harness/policy"
	"github.com/Nuctori/gateway-harness/schema"
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
  gateway-harness validate-adapter <adapter.capability.json>
  gateway-harness explain-adapter <adapter.capability.json>
  gateway-harness adapter-schema
  gateway-harness validate-conformance <fixture.json>
  gateway-harness explain-conformance <fixture.json>
  gateway-harness conformance-schema`)
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
