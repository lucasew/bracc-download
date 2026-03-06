package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"bracc/pkg/provider"
	_ "bracc/prelude"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		slog.Error("error", "error", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var urlFilters []string

	root := &cobra.Command{
		Use:   "bracc",
		Short: "BRACC download utility",
	}

	root.PersistentFlags().StringSliceVar(&urlFilters, "url-filter", nil, "Only include jobs whose URL contains one of these substrings")

	root.AddCommand(newListCommand(&urlFilters))
	root.AddCommand(newDownloadCommand(&urlFilters))
	return root
}

func newListCommand(urlFilters *[]string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List jobs grouped by provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(provider.Providers) == 0 {
				return fmt.Errorf("no provider configured")
			}

			for i, p := range provider.Providers {
				if !matchURLFilters(p.GetURL().String(), *urlFilters) {
					continue
				}
				fmt.Printf("provider[%d]: %#v\n", i, p)
				runtime := provider.NewJobRuntime(nil).WithURLFilters(*urlFilters)
				jobs, err := p.Jobs()
				if err != nil {
					return fmt.Errorf("provider %#v: %w", p, err)
				}
				count := 0
				for job := range jobs {
					if !runtime.Match(job) {
						continue
					}
					count++
					fmt.Printf("  - %s\n", job.GetURL().String())
				}
				if count == 0 {
					fmt.Println("  - (no jobs)")
				}
			}
			return nil
		},
	}
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

func newDownloadCommand(urlFilters *[]string) *cobra.Command {
	return &cobra.Command{
		Use:   "download DESTINATION",
		Short: "Download jobs from all configured providers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(provider.Providers) == 0 {
				return fmt.Errorf("no provider configured")
			}
			destination := args[0]
			if err := os.MkdirAll(destination, os.ModePerm); err != nil {
				return err
			}

			runtime := provider.NewJobRuntime(provider.Providers).WithURLFilters(*urlFilters)
			return runtime.Run(context.Background(), destination)
		},
	}
}
