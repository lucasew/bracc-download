package main

import (
	_ "bracc/prelude"

	"github.com/spf13/cobra"
)

var urlFilters []string

var Command = &cobra.Command{
	Use:   "bracc",
	Short: "BRACC download utility",
}

func init() {
	Command.PersistentFlags().StringSliceVar(&urlFilters, "url-filter", nil, "Only include jobs whose URL contains one of these substrings")
}
