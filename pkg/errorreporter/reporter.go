package errorreporter

import "log/slog"

// ReportError centralizes error reporting for the application.
// All code paths that handle unexpected errors MUST funnel through this function.
func ReportError(msg string, args ...any) {
	slog.Error(msg, args...)
}
