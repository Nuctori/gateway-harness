package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMustWriteJSONFileCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "ledger.json")

	mustWriteJSONFile(path, map[string]string{"status": "ok"})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode written json: %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("unexpected json: %+v", got)
	}
}
