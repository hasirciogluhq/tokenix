package models

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const catalog = `{
  "sample_spec": {"input_cost_per_token": "number", "litellm_provider": "x", "mode": "chat"},
  "gpt-4o": {"litellm_provider": "openai", "mode": "chat", "input_cost_per_token": 2.5e-6, "output_cost_per_token": 1e-5, "max_input_tokens": 128000},
  "ft:gpt-4o": {"litellm_provider": "openai", "mode": "chat", "input_cost_per_token": 3.75e-6},
  "text-embedding-3-small": {"litellm_provider": "openai", "mode": "embedding", "input_cost_per_token": 2e-8},
  "gemini/gemini-2.5-pro": {"litellm_provider": "gemini", "mode": "chat", "input_cost_per_token": 1.25e-6, "output_cost_per_token": 1e-5},
  "vertex_ai/gemini-2.5-pro": {"litellm_provider": "vertex_ai", "mode": "chat", "input_cost_per_token": 9e-6},
  "claude-sonnet-4-5": {"litellm_provider": "anthropic", "mode": "chat", "input_cost_per_token": 3e-6, "output_cost_per_token": 1.5e-5},
  "claude-sonnet-4-5-20250929": {"litellm_provider": "anthropic", "mode": "chat", "input_cost_per_token": 3e-6, "output_cost_per_token": 1.5e-5}
}`

func server(t *testing.T, status int, body string) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestParseCatalogFilters(t *testing.T) {
	list, err := parseCatalog([]byte(catalog))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, m := range list {
		got[m.ID] = m.Provider
	}
	want := map[string]string{
		"gpt-4o": "openai", "gemini-2.5-pro": "gemini",
		"claude-sonnet-4-5": "anthropic", "claude-sonnet-4-5-20250929": "anthropic",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for id, p := range want {
		if got[id] != p {
			t.Fatalf("missing %s: %v", id, got)
		}
	}
}

func TestLoadFetchesThenUsesCache(t *testing.T) {
	srv, hits := server(t, 200, catalog)
	dir := t.TempDir()
	opts := Options{URL: srv.URL, CacheDir: dir}

	if r := Load(opts); r.Source != SourceNetwork || *hits != 1 {
		t.Fatalf("first load: source=%s hits=%d", r.Source, *hits)
	}
	if r := Load(opts); r.Source != SourceCache || *hits != 1 {
		t.Fatalf("second load should hit cache: source=%s hits=%d", r.Source, *hits)
	}
}

func TestRefreshForcesFetch(t *testing.T) {
	srv, hits := server(t, 200, catalog)
	opts := Options{URL: srv.URL, CacheDir: t.TempDir()}
	Load(opts)
	opts.Refresh = true
	if r := Load(opts); r.Source != SourceNetwork || *hits != 2 {
		t.Fatalf("refresh: source=%s hits=%d", r.Source, *hits)
	}
}

func TestCacheExpiresAfterTTL(t *testing.T) {
	srv, hits := server(t, 200, catalog)
	now := time.Now()
	opts := Options{URL: srv.URL, CacheDir: t.TempDir(), Now: func() time.Time { return now }}
	Load(opts)
	now = now.Add(23 * time.Hour)
	if r := Load(opts); r.Source != SourceCache {
		t.Fatalf("23h old cache should be fresh, got %s", r.Source)
	}
	now = now.Add(2 * time.Hour)
	if r := Load(opts); r.Source != SourceNetwork || *hits != 2 {
		t.Fatalf("25h old cache should be refetched, got %s hits=%d", r.Source, *hits)
	}
}

func TestStaleCacheFallbackOnNetworkFailure(t *testing.T) {
	good, _ := server(t, 200, catalog)
	dir := t.TempDir()
	now := time.Now()
	Load(Options{URL: good.URL, CacheDir: dir, Now: func() time.Time { return now }})

	bad, _ := server(t, 500, "boom")
	r := Load(Options{URL: bad.URL, CacheDir: dir, Now: func() time.Time { return now.Add(48 * time.Hour) }})
	if r.Source != SourceStaleCache || r.Warning == "" || len(r.Models) == 0 {
		t.Fatalf("expected stale cache fallback, got source=%s warning=%q", r.Source, r.Warning)
	}
}

func TestBuiltinFallbackWithoutCache(t *testing.T) {
	bad, _ := server(t, 500, "boom")
	r := Load(Options{URL: bad.URL, CacheDir: t.TempDir()})
	if r.Source != SourceBuiltin || r.Warning == "" {
		t.Fatalf("expected builtin fallback, got %s", r.Source)
	}
	if _, err := r.Lookup("gpt-4o"); err != nil {
		t.Fatal(err)
	}
}

func TestClearCache(t *testing.T) {
	srv, _ := server(t, 200, catalog)
	dir := t.TempDir()
	Load(Options{URL: srv.URL, CacheDir: dir})
	if err := ClearCache(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(CachePath(dir)); !os.IsNotExist(err) {
		t.Fatalf("cache should be gone: %v", err)
	}
	if err := ClearCache(dir); err != nil {
		t.Fatalf("clearing a missing cache must not fail: %v", err)
	}
}

func TestLookup(t *testing.T) {
	list, _ := parseCatalog([]byte(catalog))
	r := NewRegistry(list, SourceCache, time.Now(), "")

	if m, err := r.Lookup("GPT-4O"); err != nil || m.ID != "gpt-4o" {
		t.Fatalf("case-insensitive exact: %v %v", m, err)
	}
	if m, err := r.Lookup("gemini-2.5"); err != nil || m.ID != "gemini-2.5-pro" {
		t.Fatalf("unique prefix: %v %v", m, err)
	}
	if _, err := r.Lookup("claude-sonnet"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
	if _, err := r.Lookup("llama"); err == nil || !strings.Contains(err.Error(), "unknown model") {
		t.Fatalf("expected unknown error, got %v", err)
	}
}

func TestAverageInputPerToken(t *testing.T) {
	r := NewRegistry([]Model{
		{ID: "a", Provider: "x", InputPerToken: 1e-6},
		{ID: "b", Provider: "x", InputPerToken: 3e-6},
	}, SourceCache, time.Now(), "")
	if got := r.AverageInputPerToken(); got < 1.999e-6 || got > 2.001e-6 {
		t.Fatalf("unexpected average %v", got)
	}
	if NewRegistry(nil, SourceCache, time.Now(), "").AverageInputPerToken() != 0 {
		t.Fatal("empty registry should average to 0")
	}
}
