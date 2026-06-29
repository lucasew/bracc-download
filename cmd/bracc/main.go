package main

import (
	"os"

	"bracc/pkg/errorreporter"
)

func main() {
	if err := Command.Execute(); err != nil {
		errorreporter.ReportError(err, "error")
		os.Exit(1)
	}
}
