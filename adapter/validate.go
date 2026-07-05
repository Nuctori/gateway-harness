package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Nuctori/gateway-harness/policy"
)

var SupportedRequestShapes = map[string]bool{
	"chat":              true,
	"responses":         true,
	"responses_compact": true,
}

var SupportedGuards = map[string]bool{
	"explicit_mutation_only":        true,
	"preserve_responses_tool_chain": true,
	"redacted_trace":                true,
}

func Decode(r io.Reader) (Manifest, error) {
	var m Manifest
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func Validate(m Manifest) error {
	if strings.TrimSpace(m.Adapter) == "" {
		return fmt.Errorf("adapter is required")
	}
	if len(m.Hooks) == 0 {
		return fmt.Errorf("adapter %q needs at least one hook", m.Adapter)
	}
	if len(m.Actions) == 0 {
		return fmt.Errorf("adapter %q needs at least one action", m.Adapter)
	}
	if err := validateSet("hook", m.Hooks, policy.SupportedHooks); err != nil {
		return fmt.Errorf("adapter %q %w", m.Adapter, err)
	}
	if err := validateSet("action", m.Actions, policy.SupportedActions); err != nil {
		return fmt.Errorf("adapter %q %w", m.Adapter, err)
	}
	if err := validateSet("request_shape", m.RequestShapes, SupportedRequestShapes); err != nil {
		return fmt.Errorf("adapter %q %w", m.Adapter, err)
	}
	if err := validateSet("guard", m.Guards, SupportedGuards); err != nil {
		return fmt.Errorf("adapter %q %w", m.Adapter, err)
	}
	return nil
}

func validateSet(name string, values []string, supported map[string]bool) error {
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("has empty %s", name)
		}
		if seen[value] {
			return fmt.Errorf("has duplicate %s %q", name, value)
		}
		if !supported[value] {
			return fmt.Errorf("has unsupported %s %q", name, value)
		}
		seen[value] = true
	}
	return nil
}

func Summarize(m Manifest) Summary {
	return Summary{
		Adapter:       m.Adapter,
		Hooks:         len(m.Hooks),
		Actions:       len(m.Actions),
		RequestShapes: len(m.RequestShapes),
		Guards:        len(m.Guards),
	}
}
