package steward

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const maxAgentStdoutBytes = 1 << 20

type Event struct {
	Hook     string          `json:"hook"`
	Redacted bool            `json:"redacted"`
	Inputs   json.RawMessage `json:"inputs,omitempty"`
}

var reservedRawInputKeys = map[string]bool{
	"content":        true,
	"conversation":   true,
	"full_text":      true,
	"input":          true,
	"message":        true,
	"messages":       true,
	"output":         true,
	"prompt":         true,
	"raw_content":    true,
	"raw_input":      true,
	"raw_output":     true,
	"raw_prompt":     true,
	"raw_response":   true,
	"raw_transcript": true,
	"response":       true,
	"transcript":     true,
}

func DecodeEvent(r io.Reader) (Event, []byte, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return Event{}, nil, err
	}
	var e Event
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&e); err != nil {
		return Event{}, nil, err
	}
	return e, raw, nil
}

func ValidateEvent(s Spec, e Event) error {
	_, err := canonicalEventJSON(s, e)
	return err
}

func canonicalEventJSON(s Spec, e Event) ([]byte, error) {
	if err := Validate(s); err != nil {
		return nil, fmt.Errorf("spec: %w", err)
	}
	if strings.TrimSpace(e.Hook) == "" {
		return nil, fmt.Errorf("event hook is required")
	}
	if !contains(s.Hooks, e.Hook) {
		return nil, fmt.Errorf("event hook %q is not enabled by steward %q", e.Hook, s.Name)
	}
	if !e.Redacted {
		return nil, fmt.Errorf("event must set redacted=true before invoking an external AI steward")
	}
	if len(e.Inputs) == 0 {
		return nil, fmt.Errorf("event inputs are required")
	}
	var inputs map[string]any
	decoder := json.NewDecoder(bytes.NewReader(e.Inputs))
	decoder.UseNumber()
	if err := decoder.Decode(&inputs); err != nil {
		return nil, fmt.Errorf("event inputs must be a JSON object: %w", err)
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("event inputs are required")
	}
	for key, value := range inputs {
		if !contains(s.Inputs, key) {
			return nil, fmt.Errorf("event input %q is not declared by steward %q", key, s.Name)
		}
		if err := rejectReservedRawInputKeys("inputs."+key, value); err != nil {
			return nil, err
		}
	}
	canonical := struct {
		Hook     string         `json:"hook"`
		Redacted bool           `json:"redacted"`
		Inputs   map[string]any `json:"inputs"`
	}{
		Hook:     e.Hook,
		Redacted: e.Redacted,
		Inputs:   inputs,
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("canonicalize event: %w", err)
	}
	return encoded, nil
}

func RunExternalAgent(ctx context.Context, s Spec, e Event, _ []byte, command string, args ...string) (Proposal, error) {
	canonicalEvent, err := canonicalEventJSON(s, e)
	if err != nil {
		return Proposal{}, err
	}
	if strings.TrimSpace(command) == "" {
		return Proposal{}, fmt.Errorf("agent command is required")
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = bytes.NewReader(canonicalEvent)
	var stdout bytes.Buffer
	cmd.Stdout = &limitedWriter{w: &stdout, remaining: maxAgentStdoutBytes}
	var stderr bytes.Buffer
	cmd.Stderr = &limitedWriter{w: &stderr, remaining: 4096}
	if err := cmd.Run(); err != nil {
		return Proposal{}, fmt.Errorf("agent command failed; stderr suppressed to avoid leaking prompt data: %w", err)
	}

	p, err := DecodeProposal(bytes.NewReader(stdout.Bytes()))
	if err != nil {
		return Proposal{}, fmt.Errorf("decode agent proposal: %w", err)
	}
	if p.Hook != e.Hook {
		return Proposal{}, fmt.Errorf("proposal hook %q does not match event hook %q", p.Hook, e.Hook)
	}
	if err := ValidateProposal(s, p); err != nil {
		return Proposal{}, fmt.Errorf("proposal: %w", err)
	}
	return p, nil
}

func rejectReservedRawInputKeys(path string, value any) error {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			normalized := strings.ToLower(strings.TrimSpace(key))
			if reservedRawInputKeys[normalized] {
				return fmt.Errorf("event input %s.%s is reserved for raw content and must be passed as a hash or artifact reference", path, key)
			}
			if err := rejectReservedRawInputKeys(path+"."+key, child); err != nil {
				return err
			}
		}
	case []any:
		for i, child := range v {
			if err := rejectReservedRawInputKeys(fmt.Sprintf("%s[%d]", path, i), child); err != nil {
				return err
			}
		}
	}
	return nil
}

type limitedWriter struct {
	w         io.Writer
	remaining int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, fmt.Errorf("output limit exceeded")
	}
	if len(p) > w.remaining {
		_, _ = w.w.Write(p[:w.remaining])
		w.remaining = 0
		return 0, fmt.Errorf("output limit exceeded")
	}
	n, err := w.w.Write(p)
	w.remaining -= n
	return n, err
}
