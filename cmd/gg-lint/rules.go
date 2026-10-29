package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cruffinoni/go-guidelines/internal/catalog"
)

// newRulesCommand builds the rules command over cat.
func newRulesCommand(cat *catalog.Catalog) *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "List the rule catalog.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch format {
			case "table":
				return cat.WriteTable(cmd.OutOrStdout())
			case "markdown":
				return cat.WriteMarkdown(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported rules format %q (want table or markdown)", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "output format: table or markdown")
	return cmd
}
