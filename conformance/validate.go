package conformance

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Nuctori/gateway-harness/adapter"
	"github.com/Nuctori/gateway-harness/policy"
)

const PreserveResponsesToolChainGuard = "preserve_responses_tool_chain"

func Decode(r io.Reader) (Fixture, error) {
	var f Fixture
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&f); err != nil {
		return Fixture{}, err
	}
	return f, nil
}

func Validate(f Fixture) error {
	if strings.TrimSpace(f.Name) == "" {
		return fmt.Errorf("fixture name is required")
	}
	if strings.TrimSpace(f.RequestShape) == "" {
		return fmt.Errorf("fixture %q request_shape is required", f.Name)
	}
	if err := adapter.Validate(f.Adapter); err != nil {
		return fmt.Errorf("fixture %q adapter: %w", f.Name, err)
	}
	if err := policy.Validate(f.Policy); err != nil {
		return fmt.Errorf("fixture %q policy: %w", f.Name, err)
	}
	if !contains(f.Adapter.RequestShapes, f.RequestShape) {
		return fmt.Errorf("fixture %q adapter %q does not support request_shape %q", f.Name, f.Adapter.Adapter, f.RequestShape)
	}
	if err := validatePolicyAgainstAdapter(f.Policy, f.Adapter); err != nil {
		return fmt.Errorf("fixture %q %w", f.Name, err)
	}
	for _, guard := range f.RequiredGuards {
		if !contains(f.Adapter.Guards, guard) {
			return fmt.Errorf("fixture %q adapter %q is missing required guard %q", f.Name, f.Adapter.Adapter, guard)
		}
	}
	if len(f.Request) > 0 {
		if err := validateRequestShape(f.RequestShape, f.Request); err != nil {
			return fmt.Errorf("fixture %q request: %w", f.Name, err)
		}
		if f.RequestShape == "responses" && responsesRequestHasStatefulToolChain(f.Request) && !contains(f.Adapter.Guards, PreserveResponsesToolChainGuard) {
			return fmt.Errorf("fixture %q responses tool-chain request requires guard %q", f.Name, PreserveResponsesToolChainGuard)
		}
	}
	return nil
}

func validatePolicyAgainstAdapter(p policy.Policy, m adapter.Manifest) error {
	for _, program := range p.Programs {
		for _, step := range program.Steps {
			for _, hook := range policy.EffectiveHooks(step) {
				if !contains(m.Hooks, hook) && !contains(m.Hooks, "*") {
					return fmt.Errorf("adapter %q does not support policy hook %q", m.Adapter, hook)
				}
			}
			for _, action := range step.Do {
				if !contains(m.Actions, action.Action) {
					return fmt.Errorf("adapter %q does not support policy action %q", m.Adapter, action.Action)
				}
			}
		}
	}
	return nil
}

func validateRequestShape(shape string, raw json.RawMessage) error {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("invalid JSON object: %w", err)
	}
	switch shape {
	case "chat":
		if _, ok := obj["messages"]; !ok {
			return fmt.Errorf("chat request must include messages")
		}
	case "responses":
		if _, ok := obj["input"]; !ok {
			return fmt.Errorf("responses request must include input")
		}
	case "responses_compact":
		if _, ok := obj["input"]; !ok {
			return fmt.Errorf("responses_compact request must include input")
		}
	default:
		return fmt.Errorf("unsupported request_shape %q", shape)
	}
	return nil
}

func responsesRequestHasStatefulToolChain(raw json.RawMessage) bool {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}
	if strings.TrimSpace(asString(obj["previous_response_id"])) != "" {
		return true
	}
	input, ok := obj["input"].([]any)
	if !ok {
		return false
	}
	for _, item := range input {
		itemObj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		switch asString(itemObj["type"]) {
		case "function_call", "function_call_output", "item_reference", "reasoning":
			return true
		}
	}
	return false
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func Summarize(f Fixture) Summary {
	policySummary := policy.Summarize(f.Policy)
	return Summary{
		Name:           f.Name,
		Adapter:        f.Adapter.Adapter,
		RequestShape:   f.RequestShape,
		Programs:       policySummary.Programs,
		Steps:          policySummary.Steps,
		Actions:        policySummary.Actions,
		RequiredGuards: len(f.RequiredGuards),
	}
}
