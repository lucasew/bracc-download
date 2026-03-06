package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

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
	root := &cobra.Command{
		Use:   "bracc",
		Short: "BRACC download utility",
	}

	root.AddCommand(newListCommand())
	root.AddCommand(newDownloadCommand())
	return root
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List jobs grouped by provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(provider.Providers) == 0 {
				return fmt.Errorf("no provider configured")
			}

			for i, p := range provider.Providers {
				fmt.Printf("provider[%d]: %v\n", i, p)
				jobs, err := p.Jobs()
				if err != nil {
					return fmt.Errorf("provider %v: %w", p, err)
				}
				count := 0
				for job := range jobs {
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

func newDownloadCommand() *cobra.Command {
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

			runtime := provider.NewJobRuntime(provider.Providers)
			return runtime.Run(context.Background(), destination)
		},
	}
}
