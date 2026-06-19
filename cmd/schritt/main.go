package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/tui"
)

// These are overwritten at release time via -ldflags.
var (
	version = "0.1.0-dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "schritt",
		Short:         "A terminal UI scaffold built with Bubble Tea",
		SilenceUsage:  true,
		SilenceErrors: false,
		// With no subcommand, launch the TUI.
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run(tui.Config{
				Items: model.SampleItems(),
			})
		},
	}
	cmd.AddCommand(newVersionCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print schritt version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "schritt %s (commit %s, built %s)\n", version, commit, date)
			return nil
		},
	}
}
