package main

import (
	"fmt"

	"bracc/pkg/provider"

	"github.com/spf13/cobra"
)

var listCommand = &cobra.Command{
	Use:   "list",
	Short: "List jobs grouped by provider",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(provider.Providers) == 0 {
			return fmt.Errorf("no provider configured")
		}

		for i, p := range provider.Providers {
			if !matchURLFilters(p.GetURL().String(), urlFilters) {
				continue
			}
			fmt.Printf("provider[%d]: %#v\n", i, p)
			runtime := provider.NewJobRuntime(nil).WithURLFilters(urlFilters)
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

func init() {
	Command.AddCommand(listCommand)
}
