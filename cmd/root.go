package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hasirciogluhq/tokenix/internal/input"
	"github.com/hasirciogluhq/tokenix/internal/token"
	"github.com/spf13/cobra"
)

type options struct {
	model   string
	data    string
	file    string
	dir     string
	include []string
	exclude []string
	format  string
}

// Execute runs the root command.
func Execute(version string) error {
	opts := options{}

	rootCmd := &cobra.Command{
		Use:     "tokenix",
		Short:   "Estimate prompt token usage across popular models",
		Long:    "tokenix estimates token counts with min/max/average and a practical net estimate.",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEstimate(cmd.InOrStdin(), cmd.OutOrStdout(), opts)
		},
	}

	rootCmd.Flags().StringVarP(&opts.model, "model", "m", "all", "Model key (e.g. gpt-4o, claude-3.5-sonnet, gemini-1.5-pro) or all")
	rootCmd.Flags().StringVarP(&opts.data, "data", "d", "", "Raw text input to estimate")
	rootCmd.Flags().StringVarP(&opts.file, "file", "f", "", "Path to file input")
	rootCmd.Flags().StringVarP(&opts.dir, "dir", "D", "", "Directory to scan recursively (all files below it)")
	rootCmd.Flags().StringSliceVarP(&opts.include, "include", "i", nil, "Only count files matching these glob patterns (e.g. '*.go', 'src/**'); repeatable or comma-separated")
	rootCmd.Flags().StringSliceVarP(&opts.exclude, "exclude", "x", nil, "Skip files/dirs matching these glob patterns (e.g. '*_test.go', 'vendor'); repeatable or comma-separated")
	rootCmd.Flags().StringVarP(&opts.format, "format", "", "text", "Output format: text or json")

	return rootCmd.Execute()
}

func runEstimate(stdin io.Reader, stdout io.Writer, opts options) error {
	var (
		text   string
		source string
		files  []string
		err    error
	)
	if opts.dir != "" {
		var res input.DirResult
		res, err = input.ReadDir(opts.dir, opts.include, opts.exclude)
		if err != nil {
			return err
		}
		text, files = res.Content, res.Files
		source = fmt.Sprintf("dir:%s (%d files)", opts.dir, len(files))
	} else {
		text, source, err = input.ResolveInput(stdin, opts.data, opts.file)
		if err != nil {
			return err
		}
	}

	resultSet, err := token.Estimate(text, opts.model)
	if err != nil {
		return err
	}

	if opts.format == "json" {
		out := struct {
			Source string   `json:"source"`
			Files  []string `json:"files,omitempty"`
			token.ResultSet
		}{
			Source:    source,
			Files:     files,
			ResultSet: resultSet,
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	printResult(stdout, source, resultSet)
	return nil
}

func printResult(stdout io.Writer, source string, resultSet token.ResultSet) {
	fmt.Fprintf(stdout, "source: %s\n", source)
	fmt.Fprintf(stdout, "characters: %d\n", resultSet.CharCount)
	fmt.Fprintf(stdout, "words: %d\n", resultSet.WordCount)
	fmt.Fprintln(stdout)

	for _, r := range resultSet.Results {
		fmt.Fprintf(stdout, "model: %s\n", r.Model)
		fmt.Fprintf(stdout, "  min_tokens: %d\n", r.Min)
		fmt.Fprintf(stdout, "  max_tokens: %d\n", r.Max)
		fmt.Fprintf(stdout, "  average_tokens: %d\n", r.Average)
		fmt.Fprintf(stdout, "  net_estimate: %d\n", r.NetEstimate)
		fmt.Fprintln(stdout)
	}

	fmt.Fprintln(stdout, "overall:")
	fmt.Fprintf(stdout, "  min_tokens: %d\n", resultSet.Summary.Min)
	fmt.Fprintf(stdout, "  max_tokens: %d\n", resultSet.Summary.Max)
	fmt.Fprintf(stdout, "  average_tokens: %d\n", resultSet.Summary.Average)
	fmt.Fprintf(stdout, "  net_estimate: %d\n", resultSet.Summary.NetEstimate)
	fmt.Fprintf(stdout, "  spread: %d\n", resultSet.Summary.Spread)
}
