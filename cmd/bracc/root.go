package main

import (
	"bracc/pkg/httpcontext"
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
		cmd.SetContext(httpcontext.WithClient(cmd.Context(), httpcontext.NewClient("Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")))
	},
}

func init() {
	Command.PersistentFlags().StringSliceVar(&urlFilters, "url-filter", nil, "Only include jobs whose URL contains one of these substrings")
	Command.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable debug logging")
}
