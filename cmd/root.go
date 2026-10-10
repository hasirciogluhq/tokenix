package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hasirciogluhq/tokenix/internal/input"
	"github.com/hasirciogluhq/tokenix/internal/models"
	"github.com/hasirciogluhq/tokenix/internal/token"
	"github.com/spf13/cobra"
)

type options struct {
	models   []string
	all      bool
	refresh  bool
	data     string
	file     string
	dir      string
	include  []string
	exclude  []string
	format   string
	cacheDir string // empty = user cache dir; set by tests
	registry func(models.Options) *models.Registry
}

// Execute runs the root command.
func Execute(version string) error {
	opts := options{}

	rootCmd := &cobra.Command{
		Use:   "tokenix",
		Short: "Estimate token counts and input cost for prompts, files and directories",
		Long: `tokenix estimates token counts offline from text heuristics.

By default it prints one model-agnostic "global" estimate. Pick models with
--model (single or comma-separated) or use --all; those include input pricing
from a model catalog that is cached locally for one day (--refresh to re-fetch).`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEstimate(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
		},
	}

	f := rootCmd.Flags()
	f.StringSliceVarP(&opts.models, "model", "m", nil, "Model id(s), comma-separated or repeated (e.g. gpt-4o,claude-sonnet-4-5)")
	f.BoolVarP(&opts.all, "all", "a", false, "Estimate for every known model")
	f.BoolVar(&opts.refresh, "refresh", false, "Ignore the cache and re-download the model list")
	f.StringVarP(&opts.data, "data", "d", "", "Raw text input to estimate")
	f.StringVarP(&opts.file, "file", "f", "", "Path to file input")
	f.StringVarP(&opts.dir, "dir", "D", "", "Directory to scan recursively (all files below it)")
	f.StringSliceVarP(&opts.include, "include", "i", nil, "Only count files matching these glob patterns (e.g. '*.go', 'src/**')")
	f.StringSliceVarP(&opts.exclude, "exclude", "x", nil, "Skip files/dirs matching these glob patterns (e.g. '*_test.go', 'vendor')")
	f.StringVar(&opts.format, "format", "text", "Output format: text or json")

	rootCmd.AddCommand(newModelsCmd())
	rootCmd.AddCommand(newCacheCmd())

	return rootCmd.Execute()
}

func (o options) loadRegistry() *models.Registry {
	load := o.registry
	if load == nil {
		load = models.Load
	}
	return load(models.Options{CacheDir: o.cacheDir, Refresh: o.refresh})
}

type modelResult struct {
	Model           string      `json:"model"`
	Provider        string      `json:"provider,omitempty"`
	InputPricePer1M *float64    `json:"input_price_per_1m_usd,omitempty"`
	CostUSD         *token.Cost `json:"cost_usd,omitempty"`
	token.Estimate
}

type report struct {
	Source   string         `json:"source"`
	Files    []string       `json:"files,omitempty"`
	Counts   token.Counts   `json:"counts"`
	Results  []modelResult  `json:"results"`
	Summary  *token.Summary `json:"summary,omitempty"`
	Catalog  *catalogInfo   `json:"catalog,omitempty"`
	Warnings []string       `json:"warnings,omitempty"`
}

type catalogInfo struct {
	Source    string     `json:"source"`
	FetchedAt *time.Time `json:"fetched_at,omitempty"`
}

func runEstimate(stdin io.Reader, stdout, stderr io.Writer, opts options) error {
	if opts.format != "text" && opts.format != "json" {
		return fmt.Errorf("unknown format %q (use text or json)", opts.format)
	}

	text, source, files, err := readInput(stdin, opts)
	if err != nil {
		return err
	}
	counts, err := token.Count(text)
	if err != nil {
		return err
	}

	rep := report{Source: source, Files: files, Counts: counts}
	selection := selectedModels(opts)

	if opts.all || len(selection) > 0 {
		reg := opts.loadRegistry()
		if reg.Warning != "" {
			rep.Warnings = append(rep.Warnings, reg.Warning)
		}
		rep.Catalog = &catalogInfo{Source: reg.Source}
		if !reg.FetchedAt.IsZero() {
			t := reg.FetchedAt
			rep.Catalog.FetchedAt = &t
		}

		chosen, err := resolveModels(reg, selection, opts.all)
		if err != nil {
			return err
		}
		estimates := make([]token.Estimate, 0, len(chosen))
		for _, m := range chosen {
			e := token.ProfileFor(m.Provider).Estimate(counts)
			cost := e.Cost(m.InputPerToken)
			price := m.InputPerToken * 1e6
			rep.Results = append(rep.Results, modelResult{
				Model: m.ID, Provider: m.Provider, InputPricePer1M: &price, CostUSD: &cost, Estimate: e,
			})
			estimates = append(estimates, e)
		}
		if len(estimates) > 1 {
			s := token.Summarize(estimates)
			rep.Summary = &s
		}
	} else {
		rep.Results = []modelResult{{Model: "global", Estimate: token.Global.Estimate(counts)}}
	}

	for _, w := range rep.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}

	if opts.format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	}
	printReport(stdout, rep)
	return nil
}

func readInput(stdin io.Reader, opts options) (text, source string, files []string, err error) {
	if opts.dir != "" {
		res, err := input.ReadDir(opts.dir, opts.include, opts.exclude)
		if err != nil {
			return "", "", nil, err
		}
		return res.Content, fmt.Sprintf("dir:%s (%d files)", opts.dir, len(res.Files)), res.Files, nil
	}
	text, source, err = input.ResolveInput(stdin, opts.data, opts.file)
	return text, source, nil, err
}

// selectedModels flattens --model values, accepting the legacy "all" keyword.
func selectedModels(opts options) []string {
	var out []string
	for _, v := range opts.models {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func resolveModels(reg *models.Registry, names []string, all bool) ([]models.Model, error) {
	seen := map[string]struct{}{}
	var out []models.Model
	add := func(m models.Model) {
		if _, dup := seen[m.ID]; !dup {
			seen[m.ID] = struct{}{}
			out = append(out, m)
		}
	}

	for _, name := range names {
		if strings.EqualFold(name, "all") {
			all = true
			continue
		}
		m, err := reg.Lookup(name)
		if err != nil {
			return nil, err
		}
		add(m)
	}
	if all {
		for _, m := range reg.Models {
			add(m)
		}
	}
	return out, nil
}

func printReport(w io.Writer, rep report) {
	fmt.Fprintf(w, "source: %s\n", rep.Source)
	fmt.Fprintf(w, "characters: %d\n", rep.Counts.Chars)
	fmt.Fprintf(w, "words: %d\n\n", rep.Counts.Words)

	priced := rep.Catalog != nil
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if priced {
		fmt.Fprintln(tw, "MODEL\tMIN\tMAX\tAVG\tNET\tIN $/1M\tCOST (NET)")
	} else {
		fmt.Fprintln(tw, "MODEL\tMIN\tMAX\tAVG\tNET")
	}
	for _, r := range rep.Results {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d", r.Model, r.Min, r.Max, r.Average, r.Net)
		if priced && r.CostUSD != nil {
			fmt.Fprintf(tw, "\t%s\t%s", formatUSD(*r.InputPricePer1M), formatUSD(r.CostUSD.Net))
		}
		fmt.Fprintln(tw)
	}
	tw.Flush()

	if rep.Summary != nil {
		fmt.Fprintf(w, "\noverall: min %d, max %d, avg %d, net %d, spread %d\n",
			rep.Summary.Min, rep.Summary.Max, rep.Summary.Average, rep.Summary.Net, rep.Summary.Spread)
	}
	if !priced {
		fmt.Fprintln(w, "\ntip: use --model <id> or --all for model-specific estimates with pricing")
	}
}

func formatUSD(v float64) string {
	switch {
	case v == 0:
		return "$0"
	case v < 0.01:
		return fmt.Sprintf("$%.6f", v)
	default:
		return fmt.Sprintf("$%.4f", v)
	}
}
