package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
)

type ReplayResult struct {
	Name        string
	Path        string
	StatusCode  int
	RequestBody int
}

func ReplayFakeUpstream(ctx context.Context, f Fixture) (ReplayResult, error) {
	if err := Validate(f); err != nil {
		return ReplayResult{}, err
	}
	if len(f.Request) == 0 {
		return ReplayResult{}, fmt.Errorf("fixture %q request is required for replay", f.Name)
	}

	path, err := requestShapePath(f.RequestShape)
	if err != nil {
		return ReplayResult{}, err
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method must be POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != path {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusBadRequest)
			return
		}
		if err := validateRequestShape(f.RequestShape, body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := validateFakeUpstreamProtocol(f.RequestShape, body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+path, bytes.NewReader(f.Request))
	if err != nil {
		return ReplayResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ReplayResult{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ReplayResult{}, fmt.Errorf("fake upstream returned %d: %s", resp.StatusCode, string(body))
	}

	return ReplayResult{
		Name:        f.Name,
		Path:        path,
		StatusCode:  resp.StatusCode,
		RequestBody: len(f.Request),
	}, nil
}

func requestShapePath(shape string) (string, error) {
	switch shape {
	case "chat":
		return "/v1/chat/completions", nil
	case "responses":
		return "/v1/responses", nil
	case "responses_compact":
		return "/v1/responses/compact", nil
	default:
		return "", fmt.Errorf("unsupported request_shape %q", shape)
	}
}

func validateFakeUpstreamProtocol(shape string, raw json.RawMessage) error {
	if shape != "responses" {
		return nil
	}
	if !responsesRequestHasStatefulToolChain(raw) {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}
	input, ok := obj["input"].([]any)
	if !ok {
		return fmt.Errorf("responses tool-chain request input must be an array")
	}
	hasItemReference := false
	hasFunctionCallOutput := false
	for _, item := range input {
		itemObj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		switch asString(itemObj["type"]) {
		case "item_reference":
			if asString(itemObj["id"]) == "" {
				return fmt.Errorf("item_reference id is required")
			}
			hasItemReference = true
		case "function_call_output":
			if asString(itemObj["call_id"]) == "" {
				return fmt.Errorf("function_call_output call_id is required")
			}
			hasFunctionCallOutput = true
		}
	}
	if hasFunctionCallOutput && !hasItemReference {
		return fmt.Errorf("stateful function_call_output requires an item_reference in HTTP replay")
	}
	return nil
}
