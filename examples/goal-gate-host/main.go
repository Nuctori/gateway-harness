package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/ledger"
	"github.com/Nuctori/gateway-harness/steward"
)

func main() {
	configPath := flag.String("config", filepath.Clean("examples/goal-gate-host/goal-gate.demo.config.json"), "Goal Gate host config path")
	specPath := flag.String("spec", filepath.Clean("fixtures/goal-gate/goal.before_complete.steward.json"), "Goal Gate steward spec path")
	eventPath := flag.String("event", filepath.Clean("fixtures/goal-gate/goal.before_complete.steward-event.json"), "Goal Gate event path")
	auditPath := flag.String("audit", filepath.Clean("fixtures/goal-gate/goal.before_complete.audit.json"), "Goal Gate audit input path")
	ledgerPath := flag.String("ledger", filepath.Clean("outputs/goal-gate-host.ledger.json"), "Target ledger path")
	flag.Parse()

	cfg := mustLoadGoalGateConfig(*configPath)
	spec := mustLoadStewardSpec(*specPath)
	event := mustLoadStewardEvent(*eventPath)
	event = buildConfiguredDemoEvent(event, cfg)
	audit := mustLoadGoalGateAuditInput(*auditPath)

	result, err := adapter.ExecuteGoalGate(context.Background(), adapter.GoalGateRequest{
		Config:  cfg,
		Spec:    spec,
		Event:   event,
		Audit:   audit,
		NowUnix: time.Now().UTC().Unix(),
	})
	if err != nil {
		var execErr *adapter.GoalGateExecutionError
		if errors.As(err, &execErr) {
			appendResult, appendErr := appendLedger(*ledgerPath, execErr.Result)
			if appendErr != nil {
				fmt.Fprintf(os.Stderr, "append ledger failed: %v\n", appendErr)
				os.Exit(1)
			}
			printJSON(map[string]any{
				"allow_complete":       false,
				"continue_work":        false,
				"continue_instruction": "",
				"continuation_patches": nil,
				"next_goal_state":      nil,
				"append_result":        appendResult,
				"ledger_path":          ledgerPathOrEmpty(*ledgerPath, execErr.Result),
				"failure":              execErr.Result.Failure,
			})
		}
		fmt.Fprintf(os.Stderr, "goal gate host execution failed: %v\n", err)
		os.Exit(1)
	}

	appendResult, err := appendLedger(*ledgerPath, result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "append ledger failed: %v\n", err)
		os.Exit(1)
	}

	printJSON(map[string]any{
		"allow_complete":       result.Sidecar != nil && result.Sidecar.Outcome.AllowComplete,
		"continue_work":        result.Sidecar != nil && result.Sidecar.Outcome.ContinueWork,
		"continue_instruction": continueInstruction(result),
		"continuation_patches": continuationPatches(result),
		"next_goal_state":      nextGoalState(result),
		"append_result":        appendResult,
		"ledger_path":          ledgerPathOrEmpty(*ledgerPath, result),
	})
}

func mustLoadGoalGateConfig(path string) adapter.GoalGateConfig {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open goal gate config: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	cfg, err := adapter.DecodeGoalGateConfig(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode goal gate config: %v\n", err)
		os.Exit(1)
	}
	configDir := filepath.Dir(path)
	if strings.TrimSpace(cfg.Runner.Workdir) != "" && !filepath.IsAbs(cfg.Runner.Workdir) {
		cfg.Runner.Workdir = filepath.Clean(filepath.Join(configDir, cfg.Runner.Workdir))
	}
	return cfg
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

func mustLoadStewardEvent(path string) steward.Event {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open steward event: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	event, _, err := steward.DecodeEvent(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode steward event: %v\n", err)
		os.Exit(1)
	}
	return event
}

func buildConfiguredDemoEvent(event steward.Event, cfg adapter.GoalGateConfig) steward.Event {
	if len(cfg.AllowedInputs) == 0 || len(event.Inputs) == 0 {
		return event
	}
	var inputs map[string]json.RawMessage
	if err := json.Unmarshal(event.Inputs, &inputs); err != nil {
		return event
	}
	allowed := make(map[string]bool, len(cfg.AllowedInputs))
	for _, name := range cfg.AllowedInputs {
		allowed[strings.TrimSpace(name)] = true
	}
	filtered := make(map[string]json.RawMessage, len(allowed))
	for key, raw := range inputs {
		if allowed[key] {
			filtered[key] = raw
		}
	}
	encoded, err := json.Marshal(filtered)
	if err != nil {
		return event
	}
	event.Inputs = encoded
	return event
}

func mustLoadGoalGateAuditInput(path string) steward.GoalGateAuditInput {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open goal gate audit input: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()
	var input struct {
		Project       ledger.AppendProject `json:"project"`
		Session       ledger.AppendSession `json:"session"`
		EventID       string               `json:"event_id"`
		At            string               `json:"at"`
		PolicyVersion string               `json:"policy_version"`
		TraceHash     string               `json:"trace_hash"`
		Model         string               `json:"model"`
	}
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		fmt.Fprintf(os.Stderr, "decode goal gate audit input: %v\n", err)
		os.Exit(1)
	}
	at, err := time.Parse(time.RFC3339, input.At)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode goal gate audit input: at must be RFC3339\n")
		os.Exit(1)
	}
	return steward.GoalGateAuditInput{
		Project:       input.Project,
		Session:       input.Session,
		EventID:       input.EventID,
		At:            at,
		PolicyVersion: input.PolicyVersion,
		TraceHash:     input.TraceHash,
		Model:         input.Model,
	}
}

func appendLedger(path string, result adapter.GoalGateResult) (*ledger.AppendResult, error) {
	if result.AppendRecord == nil {
		return nil, nil
	}
	current := ledger.Ledger{}
	if file, err := os.Open(path); err == nil {
		defer file.Close()
		decoded, err := ledger.Decode(file)
		if err != nil {
			return nil, err
		}
		current = decoded
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	next, appendResult, err := ledger.Append(current, *result.AppendRecord)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return &appendResult, nil
}

func continueInstruction(result adapter.GoalGateResult) string {
	if result.Sidecar == nil {
		return ""
	}
	return result.Sidecar.Outcome.ContinueInstruction
}

func nextGoalState(result adapter.GoalGateResult) any {
	if result.Sidecar == nil {
		return nil
	}
	return result.Sidecar.Outcome.NextGoalState
}

func continuationPatches(result adapter.GoalGateResult) any {
	if result.Sidecar == nil {
		return nil
	}
	return result.Sidecar.Review.ContinuationPatches
}

func ledgerPathOrEmpty(path string, result adapter.GoalGateResult) string {
	if result.AppendRecord == nil {
		return ""
	}
	return path
}

func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(append(data, '\n'))
}
