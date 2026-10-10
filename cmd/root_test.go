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

func TestDefaultIsGlobalWithAveragePrice(t *testing.T) {
	out, _, err := run(t, options{data: "Sample text for token counting.", format: "json"})
	if err != nil {
		t.Fatal(err)
	}
	r := decode(t, out)
	if len(r.Results) != 1 || r.Results[0].Model != "global" || r.Summary != nil {
		t.Fatalf("unexpected report: %+v", r)
	}
	// fakeRegistry prices: 3.0, 2.5 and 0.15 per 1M -> mean 1.883333
	got := r.Results[0].InputPricePer1M
	if got == nil || *got < 1.88 || *got > 1.89 {
		t.Fatalf("expected average price ~1.883, got %v", got)
	}
	if r.Results[0].CostUSD == nil || r.Results[0].CostUSD.Net <= 0 {
		t.Fatalf("expected cost on global result: %+v", r.Results[0])
	}
	if r.Catalog == nil || r.Catalog.Models != 3 {
		t.Fatalf("expected catalog info with 3 models: %+v", r.Catalog)
	}
}

func TestGlobalWithoutPricesOmitsCost(t *testing.T) {
	opts := options{data: "hello", format: "json", registry: func(models.Options) *models.Registry {
		return models.NewRegistry(nil, models.SourceBuiltin, time.Time{}, "")
	}}
	var out bytes.Buffer
	if err := runEstimate(strings.NewReader(""), &out, &bytes.Buffer{}, opts); err != nil {
		t.Fatal(err)
	}
	if r := decode(t, out.String()); r.Results[0].CostUSD != nil || r.Results[0].InputPricePer1M != nil {
		t.Fatalf("expected no pricing: %+v", r.Results[0])
	}
}

func TestGlobalTextShowsAveragePriceNote(t *testing.T) {
	out, _, err := run(t, options{data: "hello world", format: "text"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"global", "COST (NET)", "average input price across 3 models"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
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

// execute runs the real cobra command tree with the given arguments.
func execute(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	opts := &options{registry: fakeRegistry, cacheDir: t.TempDir()}
	root := newRootCmd("test", opts)
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestCLICommaSeparatedAndRepeatedModelFlags(t *testing.T) {
	for _, args := range [][]string{
		{"-d", "hello world", "-m", "gpt-4o,claude-sonnet-4-5", "--format", "json"},
		{"-d", "hello world", "-m", "gpt-4o", "-m", "claude-sonnet-4-5", "--format", "json"},
		{"-d", "hello world", "--model", "gpt-4o, claude-sonnet-4-5", "--format", "json"},
	} {
		out, _, err := execute(t, "", args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if got := len(decode(t, out).Results); got != 2 {
			t.Fatalf("%v: expected 2 results, got %d", args, got)
		}
	}
}

func TestCLILegacyAllKeyword(t *testing.T) {
	out, _, err := execute(t, "", "-d", "hello", "-m", "all", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(decode(t, out).Results); got != 3 {
		t.Fatalf("expected 3 results, got %d", got)
	}
}

func TestCLIStdin(t *testing.T) {
	out, _, err := execute(t, "piped text for estimation", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	if r := decode(t, out); r.Source != "stdin" || r.Results[0].Model != "global" {
		t.Fatalf("unexpected report: %+v", r)
	}
}

func TestCLINoInputFails(t *testing.T) {
	if _, _, err := execute(t, "   "); err == nil {
		t.Fatal("expected error for blank stdin")
	}
}

func TestCLIModelsCommand(t *testing.T) {
	out, _, err := execute(t, "", "models", "gpt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "gpt-4o-mini") || strings.Contains(out, "claude-sonnet") {
		t.Fatalf("filter not applied:\n%s", out)
	}

	out, _, err = execute(t, "", "models", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var list []models.Model
	if err := json.Unmarshal([]byte(out), &list); err != nil || len(list) != 3 {
		t.Fatalf("unexpected json (%v): %s", err, out)
	}
}

func TestCLICacheCommands(t *testing.T) {
	dir := t.TempDir()
	opts := &options{registry: fakeRegistry, cacheDir: dir}
	run := func(args ...string) string {
		root := newRootCmd("test", opts)
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}

	if got := strings.TrimSpace(run("cache", "path")); got != models.CachePath(dir) {
		t.Fatalf("unexpected path %q", got)
	}
	if err := os.WriteFile(models.CachePath(dir), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("cache", "clear")
	if _, err := os.Stat(models.CachePath(dir)); !os.IsNotExist(err) {
		t.Fatalf("cache file should be removed: %v", err)
	}
}

func TestCLIRefreshFlagReachesRegistry(t *testing.T) {
	var gotRefresh bool
	opts := &options{cacheDir: t.TempDir(), registry: func(o models.Options) *models.Registry {
		gotRefresh = o.Refresh
		return fakeRegistry(o)
	}}
	root := newRootCmd("test", opts)
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"-d", "hello", "-m", "gpt-4o", "--refresh"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !gotRefresh {
		t.Fatal("--refresh did not reach the registry loader")
	}
}

func TestCLIWarningGoesToStderr(t *testing.T) {
	opts := &options{cacheDir: t.TempDir(), registry: func(models.Options) *models.Registry {
		return models.NewRegistry([]models.Model{{ID: "gpt-4o", Provider: "openai", InputPerToken: 1e-6}},
			models.SourceBuiltin, time.Time{}, "using built-in list")
	}}
	root := newRootCmd("test", opts)
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"-d", "hello", "-m", "gpt-4o", "--format", "json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "using built-in list") {
		t.Fatalf("warning missing from stderr: %q", errOut.String())
	}
	decode(t, out.String()) // stdout must stay valid JSON
}
