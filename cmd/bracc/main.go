package main

import (
	"bracc/pkg/errorreporter"
	"os"
)

func main() {
	if err := Command.Execute(); err != nil {
		errorreporter.ReportError("error", "error", err)
		os.Exit(1)
	}
}
