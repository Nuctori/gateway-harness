package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
  gateway-harness schema`)
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
