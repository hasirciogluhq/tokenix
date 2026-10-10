package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hasirciogluhq/tokenix/internal/models"
)

func fakeRegistry(models.Options) *models.Registry {
	return models.NewRegistry([]models.Model{
		{ID: "claude-sonnet-4-5", Provider: "anthropic", InputPerToken: 3e-6, OutputPerToken: 15e-6},
		{ID: "gpt-4o", Provider: "openai", InputPerToken: 2.5e-6, OutputPerToken: 10e-6},
		{ID: "gpt-4o-mini", Provider: "openai", InputPerToken: 0.15e-6, OutputPerToken: 0.6e-6},
	}, models.SourceCache, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "")
}

func run(t *testing.T, opts options) (string, string, error) {
	t.Helper()
	opts.registry = fakeRegistry
	var out, errOut bytes.Buffer
	err := runEstimate(strings.NewReader(""), &out, &errOut, opts)
	return out.String(), errOut.String(), err
}

func decode(t *testing.T, s string) report {
	t.Helper()
	var r report
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, s)
	}
	return r
}

func TestDefaultIsGlobalWithoutCatalog(t *testing.T) {
	opts := options{data: "Sample text for token counting.", format: "json"}
	opts.registry = func(models.Options) *models.Registry {
		t.Fatal("registry must not be loaded in global mode")
		return nil
	}
	var out bytes.Buffer
	if err := runEstimate(strings.NewReader(""), &out, &bytes.Buffer{}, opts); err != nil {
		t.Fatal(err)
	}
	r := decode(t, out.String())
	if len(r.Results) != 1 || r.Results[0].Model != "global" || r.Catalog != nil || r.Summary != nil {
		t.Fatalf("unexpected report: %+v", r)
	}
}

func TestSingleAndCommaSeparatedModels(t *testing.T) {
	out, _, err := run(t, options{data: "hello world example", models: []string{"gpt-4o", "claude-sonnet-4-5"}, format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	r := decode(t, out)
	if len(r.Results) != 2 || r.Summary == nil {
		t.Fatalf("expected 2 results and a summary: %+v", r)
	}
	if r.Results[0].CostUSD == nil || r.Results[0].InputPricePer1M == nil || *r.Results[0].InputPricePer1M != 2.5 {
		t.Fatalf("expected pricing on result: %+v", r.Results[0])
	}
}

func TestAllModels(t *testing.T) {
	out, _, err := run(t, options{data: "hello world example", all: true, format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(decode(t, out).Results); got != 3 {
		t.Fatalf("expected 3 results, got %d", got)
	}
}

func TestPrefixAndDuplicateModels(t *testing.T) {
	out, _, err := run(t, options{data: "hello", models: []string{"claude", "claude-sonnet-4-5"}, format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(decode(t, out).Results); got != 1 {
		t.Fatalf("expected deduplicated result, got %d", got)
	}
}

func TestUnknownModelErrors(t *testing.T) {
	_, _, err := run(t, options{data: "hello", models: []string{"nope-9000"}, format: "text"})
	if err == nil || !strings.Contains(err.Error(), "unknown model") {
		t.Fatalf("expected unknown model error, got %v", err)
	}
}

func TestTextOutputHasTableAndCost(t *testing.T) {
	out, _, err := run(t, options{data: "hello world", models: []string{"gpt-4o"}, format: "text"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"MODEL", "gpt-4o", "COST (NET)", "$"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
	}
}

func TestInvalidFormat(t *testing.T) {
	if _, _, err := run(t, options{data: "x", format: "yaml"}); err == nil {
		t.Fatal("expected format error")
	}
}

func TestDirWithFilters(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.go"), []byte("package a // hello world"), 0o644)
	os.MkdirAll(filepath.Join(root, "sub"), 0o755)
	os.WriteFile(filepath.Join(root, "sub", "b.go"), []byte("package b // nested file"), 0o644)
	os.WriteFile(filepath.Join(root, "notes.md"), []byte("ignore me"), 0o644)

	out, _, err := run(t, options{dir: root, include: []string{"*.go"}, format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	files := decode(t, out).Files
	if len(files) != 2 || files[0] != "a.go" || files[1] != "sub/b.go" {
		t.Fatalf("unexpected files: %v", files)
	}
}
