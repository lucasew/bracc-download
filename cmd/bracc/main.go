package main

import (
	"log/slog"
	"os"
)

func main() {
	if err := Command.Execute(); err != nil {
		slog.Error("error", "error", err)
		os.Exit(1)
	}
}
