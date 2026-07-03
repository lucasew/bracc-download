package main

import (
	"fmt"
	"os"

	"bracc/pkg/provider"

	"github.com/spf13/cobra"
)

var downloadCommand = &cobra.Command{
	Use:   "download DESTINATION",
	Short: "Download jobs from all configured providers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(provider.Providers) == 0 {
			return fmt.Errorf("no provider configured")
		}
		destination := args[0]
		if err := os.MkdirAll(destination, 0750); err != nil {
			return err
		}

		progressFactory, err := newMultibarFactory()
		if err != nil {
			return err
		}

		runtime := provider.NewJobRuntime(provider.Providers).WithURLFilters(urlFilters)
		ctx := provider.WithProgressFactory(cmd.Context(), progressFactory)
		return runtime.Run(ctx, destination)
	},
}

func init() {
	Command.AddCommand(downloadCommand)
}
