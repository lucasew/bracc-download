package main

import (
	"bracc/pkg/errorreporter"
	"os"
)

func main() {
	if err := Command.Execute(); err != nil {
		errorreporter.ReportError(err, "msg", "error")
		os.Exit(1)
	}
}
