package errorreporter

import "log/slog"

// ReportError centralizes error reporting for the application.
// All unexpected errors should be funneled through this function.
func ReportError(msg string, args ...any) {
	slog.Error(msg, args...)
}
