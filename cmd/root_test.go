package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunEstimateTextOutput(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}

	opts := options{
		model:  "gpt-4o",
		data:   "Token sayımı için test metni.",
		format: "text",
	}

	if err := runEstimate(stdin, stdout, opts); err != nil {
		t.Fatalf("runEstimate returned error: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "model: gpt-4o") {
		t.Fatalf("expected model in output, got: %s", out)
	}
	if !strings.Contains(out, "overall:") {
		t.Fatalf("expected overall summary in output, got: %s", out)
	}
}

func TestRunEstimateJSONOutput(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}

	opts := options{
		model:  "all",
		data:   "JSON çıktısı için test metni.",
		format: "json",
	}

	if err := runEstimate(stdin, stdout, opts); err != nil {
		t.Fatalf("runEstimate returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if payload["source"] != "flag:data" {
		t.Fatalf("expected source=flag:data got %v", payload["source"])
	}
}

func TestRunEstimateDirWithFilters(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.go"), []byte("package a // hello world"), 0o644)
	os.MkdirAll(filepath.Join(root, "sub"), 0o755)
	os.WriteFile(filepath.Join(root, "sub", "b.go"), []byte("package b // nested file"), 0o644)
	os.WriteFile(filepath.Join(root, "notes.md"), []byte("ignore me"), 0o644)

	stdout := &bytes.Buffer{}
	opts := options{model: "gpt-4o", dir: root, include: []string{"*.go"}, format: "json"}
	if err := runEstimate(strings.NewReader(""), stdout, opts); err != nil {
		t.Fatal(err)
	}

	var payload struct {
		Source string   `json:"source"`
		Files  []string `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Files) != 2 || payload.Files[0] != "a.go" || payload.Files[1] != "sub/b.go" {
		t.Fatalf("unexpected files: %v", payload.Files)
	}
}
