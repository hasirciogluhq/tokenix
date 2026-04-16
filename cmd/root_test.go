package cmd

import (
	"bytes"
	"encoding/json"
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
