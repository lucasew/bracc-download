package errorreporter

import "log/slog"

func ReportError(err error, args ...any) {
	if err == nil {
		return
	}
	// Combine err into args and forward to slog.Error
	combined := append([]any{"error", err}, args...)
	slog.Error("error reported", combined...)
}
