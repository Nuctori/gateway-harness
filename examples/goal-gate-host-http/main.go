package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/ledger"
	"github.com/Nuctori/gateway-harness/schema"
	"github.com/Nuctori/gateway-harness/steward"
)

type demoHostPaths struct {
	ConfigPath string
	SpecPath   string
	EventPath  string
	AuditPath  string
}

func main() {
	addr := flag.String("addr", "127.0.0.1:4070", "HTTP listen address")
	configPath := flag.String("config", filepath.Clean("examples/goal-gate-host/goal-gate.demo.config.json"), "Goal Gate host config path")
	specPath := flag.String("spec", filepath.Clean("fixtures/goal-gate/goal.before_complete.steward.json"), "Goal Gate steward spec path")
	eventPath := flag.String("event", filepath.Clean("fixtures/goal-gate/goal.before_complete.steward-event.json"), "Goal Gate event path")
	auditPath := flag.String("audit", filepath.Clean("fixtures/goal-gate/goal.before_complete.audit.json"), "Goal Gate audit input path")
	flag.Parse()

	paths := demoHostPaths{
		ConfigPath: *configPath,
		SpecPath:   *specPath,
		EventPath:  *eventPath,
		AuditPath:  *auditPath,
	}
	mux := newDemoHostMux(paths)

	fmt.Fprintf(os.Stdout, "goal gate host demo listening on http://%s\n", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
}

func newDemoHostMux(paths demoHostPaths) *http.ServeMux {
	bundle := schema.GoalGateHostBundle()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/goal-gate/bundle", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, bundle)
	})

	mux.HandleFunc("/api/goal-gate/example-request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cfg := mustLoadGoalGateConfig(paths.ConfigPath)
		event := buildConfiguredDemoEvent(mustLoadStewardEvent(paths.EventPath), cfg)
		audit := mustLoadGoalGateAuditInput(paths.AuditPath)
		writeJSON(w, map[string]any{
			"config": cfg,
			"event":  event,
			"audit": map[string]any{
				"project":        audit.Project,
				"session":        audit.Session,
				"event_id":       audit.EventID,
				"at":             audit.At.UTC().Format(time.RFC3339),
				"policy_version": audit.PolicyVersion,
				"trace_hash":     audit.TraceHash,
				"model":          audit.Model,
			},
		})
	})

	mux.HandleFunc("/api/goal-gate/config-assistant", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload ConfigAssistantRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("decode request: %v", err), http.StatusBadRequest)
			return
		}
		writeJSON(w, ComposeGoalGateConfigDraft(payload.Config, payload.Message))
	})

	mux.HandleFunc("/api/goal-gate/execute", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Config adapter.GoalGateConfig `json:"config"`
			Event  steward.Event          `json:"event"`
			Audit  struct {
				Project       ledger.AppendProject `json:"project"`
				Session       ledger.AppendSession `json:"session"`
				EventID       string               `json:"event_id"`
				At            string               `json:"at"`
				PolicyVersion string               `json:"policy_version"`
				TraceHash     string               `json:"trace_hash"`
				Model         string               `json:"model"`
			} `json:"audit"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("decode request: %v", err), http.StatusBadRequest)
			return
		}

		spec := mustLoadStewardSpec(paths.SpecPath)
		audit := mustAuditFromPayload(payload.Audit)
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		result, err := adapter.ExecuteGoalGate(ctx, adapter.GoalGateRequest{
			Config:  payload.Config,
			Spec:    spec,
			Event:   payload.Event,
			Audit:   audit,
			NowUnix: time.Now().UTC().Unix(),
		})
		if err != nil {
			var execErr *adapter.GoalGateExecutionError
			if errors.As(err, &execErr) {
				writeJSONWithStatus(w, http.StatusBadRequest, execErr.Result)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, result)
	})

	uiDir := filepath.Dir(paths.ConfigPath)
	mux.Handle("/", http.FileServer(http.Dir(uiDir)))
	return mux
}

func mustLoadGoalGateConfig(path string) adapter.GoalGateConfig {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	cfg, err := adapter.DecodeGoalGateConfig(file)
	if err != nil {
		panic(err)
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
		panic(err)
	}
	defer file.Close()
	s, err := steward.Decode(file)
	if err != nil {
		panic(err)
	}
	return s
}

func mustLoadStewardEvent(path string) steward.Event {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	event, _, err := steward.DecodeEvent(file)
	if err != nil {
		panic(err)
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
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var input struct {
		Project       ledger.AppendProject `json:"project"`
		Session       ledger.AppendSession `json:"session"`
		EventID       string               `json:"event_id"`
		At            string               `json:"at"`
		PolicyVersion string               `json:"policy_version"`
		TraceHash     string               `json:"trace_hash"`
		Model         string               `json:"model"`
	}
	if err := json.Unmarshal(data, &input); err != nil {
		panic(err)
	}
	return mustAuditFromPayload(input)
}

func mustAuditFromPayload(input struct {
	Project       ledger.AppendProject `json:"project"`
	Session       ledger.AppendSession `json:"session"`
	EventID       string               `json:"event_id"`
	At            string               `json:"at"`
	PolicyVersion string               `json:"policy_version"`
	TraceHash     string               `json:"trace_hash"`
	Model         string               `json:"model"`
}) steward.GoalGateAuditInput {
	at, err := time.Parse(time.RFC3339, input.At)
	if err != nil {
		panic(err)
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

func writeJSON(w http.ResponseWriter, v any) {
	writeJSONWithStatus(w, http.StatusOK, v)
}

func writeJSONWithStatus(w http.ResponseWriter, status int, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(bytes.TrimSpace(data), '\n'))
}
