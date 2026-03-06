package main

import (
	"strings"

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

func matchURLFilters(u string, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if strings.Contains(u, filter) {
			return true
		}
	}
	return false
}
