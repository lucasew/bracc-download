package main

import (
	_ "bracc/prelude"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var urlFilters []string
var verbose bool

var Command = &cobra.Command{
	Use:   "bracc",
	Short: "BRACC download utility",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		level := slog.LevelInfo
		if verbose {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})))
	},
}

func init() {
	Command.PersistentFlags().StringSliceVar(&urlFilters, "url-filter", nil, "Only include jobs whose URL contains one of these substrings")
	Command.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable debug logging")
}
