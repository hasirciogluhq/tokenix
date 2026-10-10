package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/hasirciogluhq/tokenix/internal/models"
	"github.com/spf13/cobra"
)

func newModelsCmd() *cobra.Command {
	var refresh, asJSON bool
	c := &cobra.Command{
		Use:   "models [filter]",
		Short: "List known models and their pricing",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := models.Load(models.Options{Refresh: refresh})
			if reg.Warning != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", reg.Warning)
			}
			filter := ""
			if len(args) == 1 {
				filter = args[0]
			}
			list := reg.Search(filter)

			out := cmd.OutOrStdout()
			if asJSON {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(list)
			}
			tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "MODEL\tPROVIDER\tIN $/1M\tOUT $/1M\tCONTEXT")
			for _, m := range list {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n", m.ID, m.Provider,
					formatUSD(m.InputPerToken*1e6), formatUSD(m.OutputPerToken*1e6), m.MaxInputTokens)
			}
			return tw.Flush()
		},
	}
	c.Flags().BoolVar(&refresh, "refresh", false, "Ignore the cache and re-download the model list")
	c.Flags().BoolVar(&asJSON, "json", false, "Output JSON")
	return c
}

func newCacheCmd() *cobra.Command {
	c := &cobra.Command{Use: "cache", Short: "Manage the local model cache"}
	c.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the cache file location",
		Args:  cobra.NoArgs,
		Run:   func(cmd *cobra.Command, _ []string) { fmt.Fprintln(cmd.OutOrStdout(), models.CachePath("")) },
	})
	c.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Delete the cached model list",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := models.ClearCache(""); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "cache cleared:", models.CachePath(""))
			return nil
		},
	})
	return c
}
