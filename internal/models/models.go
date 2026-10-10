// Package models provides the list of known LLMs and their pricing.
//
// Data comes from the community-maintained LiteLLM price catalog, is reduced
// to first-party chat models and cached on disk for one day.
package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DefaultURL is the upstream price catalog.
const DefaultURL = "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json"

// DefaultTTL is how long a cache file is considered fresh.
const DefaultTTL = 24 * time.Hour

const cacheFileName = "models.json"

// Registry sources, reported in Registry.Source.
const (
	SourceCache      = "cache"
	SourceNetwork    = "network"
	SourceStaleCache = "stale-cache"
	SourceBuiltin    = "builtin"
)

// Model describes one model and its input/output price in USD per token.
type Model struct {
	ID             string  `json:"id"`
	Provider       string  `json:"provider"`
	InputPerToken  float64 `json:"input_per_token"`
	OutputPerToken float64 `json:"output_per_token"`
	MaxInputTokens int     `json:"max_input_tokens,omitempty"`
}

// Registry is a loaded model list with lookup helpers.
type Registry struct {
	Models    []Model
	Source    string
	FetchedAt time.Time
	// Warning is set when the registry had to fall back to older data.
	Warning string

	byID map[string]Model
}

// Options configure Load. Zero values pick sensible defaults.
type Options struct {
	URL      string
	CacheDir string
	TTL      time.Duration
	// Refresh ignores the cache and forces a download.
	Refresh bool
	Client  *http.Client
	Now     func() time.Time
}

type cacheFile struct {
	FetchedAt time.Time `json:"fetched_at"`
	Source    string    `json:"source"`
	Models    []Model   `json:"models"`
}

// CacheDir returns the default cache directory for tokenix.
func CacheDir() string {
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "tokenix")
	}
	return filepath.Join(os.TempDir(), "tokenix")
}

// CachePath returns the cache file path inside dir (default dir when empty).
func CachePath(dir string) string {
	if dir == "" {
		dir = CacheDir()
	}
	return filepath.Join(dir, cacheFileName)
}

// Load returns the model registry using, in order: a fresh cache, the
// network, a stale cache, and finally the built-in list.
func Load(opts Options) *Registry {
	if opts.URL == "" {
		opts.URL = DefaultURL
	}
	if opts.TTL <= 0 {
		opts.TTL = DefaultTTL
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 20 * time.Second}
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	path := CachePath(opts.CacheDir)

	cached, cacheErr := readCache(path)
	if cacheErr == nil && !opts.Refresh && opts.Now().Sub(cached.FetchedAt) < opts.TTL {
		return NewRegistry(cached.Models, SourceCache, cached.FetchedAt, "")
	}

	list, fetchErr := fetch(opts.Client, opts.URL)
	if fetchErr == nil {
		now := opts.Now()
		// A failed cache write only costs the next run a re-download.
		_ = writeCache(path, cacheFile{FetchedAt: now, Source: opts.URL, Models: list})
		return NewRegistry(list, SourceNetwork, now, "")
	}

	if cacheErr == nil {
		return NewRegistry(cached.Models, SourceStaleCache, cached.FetchedAt,
			fmt.Sprintf("could not refresh model list (%v); using cached data from %s",
				fetchErr, cached.FetchedAt.Format(time.RFC3339)))
	}
	return NewRegistry(builtin(), SourceBuiltin, time.Time{},
		fmt.Sprintf("could not download model list (%v); using built-in list", fetchErr))
}

// ClearCache removes the cache file. A missing file is not an error.
func ClearCache(dir string) error {
	err := os.Remove(CachePath(dir))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// NewRegistry builds a Registry from a model list.
func NewRegistry(list []Model, source string, fetchedAt time.Time, warning string) *Registry {
	sorted := make([]Model, len(list))
	copy(sorted, list)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Provider != sorted[j].Provider {
			return sorted[i].Provider < sorted[j].Provider
		}
		return sorted[i].ID < sorted[j].ID
	})
	byID := make(map[string]Model, len(sorted))
	for _, m := range sorted {
		byID[strings.ToLower(m.ID)] = m
	}
	return &Registry{Models: sorted, Source: source, FetchedAt: fetchedAt, Warning: warning, byID: byID}
}

// Lookup resolves a user supplied model name. It accepts an exact id
// (case-insensitive) or a prefix that identifies exactly one model.
func (r *Registry) Lookup(name string) (Model, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return Model{}, errors.New("empty model name")
	}
	if m, ok := r.byID[key]; ok {
		return m, nil
	}

	var prefixed []Model
	for _, m := range r.Models {
		if strings.HasPrefix(strings.ToLower(m.ID), key) {
			prefixed = append(prefixed, m)
		}
	}
	if len(prefixed) == 1 {
		return prefixed[0], nil
	}
	if len(prefixed) > 1 {
		return Model{}, fmt.Errorf("model %q is ambiguous, did you mean: %s", name, ids(prefixed, 6))
	}

	if similar := r.Search(key); len(similar) > 0 {
		return Model{}, fmt.Errorf("unknown model %q, did you mean: %s (run `tokenix models` to list all)", name, ids(similar, 6))
	}
	return Model{}, fmt.Errorf("unknown model %q (run `tokenix models` to list all)", name)
}

// Search returns models whose id or provider contains substr.
func (r *Registry) Search(substr string) []Model {
	substr = strings.ToLower(strings.TrimSpace(substr))
	var out []Model
	for _, m := range r.Models {
		if substr == "" || strings.Contains(strings.ToLower(m.ID), substr) || strings.Contains(m.Provider, substr) {
			out = append(out, m)
		}
	}
	return out
}

func ids(list []Model, limit int) string {
	parts := make([]string, 0, limit)
	for i, m := range list {
		if i == limit {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, m.ID)
	}
	return strings.Join(parts, ", ")
}

func readCache(path string) (cacheFile, error) {
	var c cacheFile
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if len(c.Models) == 0 {
		return c, errors.New("cache is empty")
	}
	return c, nil
}

func writeCache(path string, c cacheFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".models-*.tmp")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(b)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(tmp.Name())
		return errors.Join(werr, cerr)
	}
	return os.Rename(tmp.Name(), path)
}

func fetch(client *http.Client, url string) ([]Model, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	return parseCatalog(body)
}

// providers kept from the upstream catalog (first-party APIs only).
var providers = map[string]struct{}{
	"openai": {}, "anthropic": {}, "gemini": {}, "mistral": {}, "xai": {}, "deepseek": {},
}

type catalogEntry struct {
	Provider       string  `json:"litellm_provider"`
	Mode           string  `json:"mode"`
	InputPerToken  float64 `json:"input_cost_per_token"`
	OutputPerToken float64 `json:"output_cost_per_token"`
	MaxInputTokens int     `json:"max_input_tokens"`
}

func parseCatalog(data []byte) ([]Model, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid catalog: %w", err)
	}

	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	byID := map[string]Model{}
	for _, key := range keys {
		var e catalogEntry
		// Entries such as "sample_spec" hold strings in numeric fields; skip them.
		if json.Unmarshal(raw[key], &e) != nil {
			continue
		}
		if _, ok := providers[e.Provider]; !ok {
			continue
		}
		if e.Mode != "chat" && e.Mode != "responses" && e.Mode != "completion" {
			continue
		}
		if e.InputPerToken <= 0 || strings.HasPrefix(key, "ft:") {
			continue
		}
		id := strings.TrimPrefix(key, e.Provider+"/")
		if existing, dup := byID[id]; dup && existing.ID == id && key != id {
			continue
		}
		byID[id] = Model{
			ID:             id,
			Provider:       e.Provider,
			InputPerToken:  e.InputPerToken,
			OutputPerToken: e.OutputPerToken,
			MaxInputTokens: e.MaxInputTokens,
		}
	}
	if len(byID) == 0 {
		return nil, errors.New("catalog contained no usable models")
	}

	out := make([]Model, 0, len(byID))
	for _, m := range byID {
		out = append(out, m)
	}
	return out, nil
}

// builtin is the offline fallback with a few popular models.
func builtin() []Model {
	return []Model{
		{ID: "gpt-4o", Provider: "openai", InputPerToken: 2.5e-6, OutputPerToken: 10e-6, MaxInputTokens: 128000},
		{ID: "gpt-4o-mini", Provider: "openai", InputPerToken: 0.15e-6, OutputPerToken: 0.6e-6, MaxInputTokens: 128000},
		{ID: "claude-sonnet-4-5", Provider: "anthropic", InputPerToken: 3e-6, OutputPerToken: 15e-6, MaxInputTokens: 200000},
		{ID: "claude-haiku-4-5", Provider: "anthropic", InputPerToken: 1e-6, OutputPerToken: 5e-6, MaxInputTokens: 200000},
		{ID: "gemini-2.5-pro", Provider: "gemini", InputPerToken: 1.25e-6, OutputPerToken: 10e-6, MaxInputTokens: 1048576},
		{ID: "gemini-2.5-flash", Provider: "gemini", InputPerToken: 0.3e-6, OutputPerToken: 2.5e-6, MaxInputTokens: 1048576},
	}
}
