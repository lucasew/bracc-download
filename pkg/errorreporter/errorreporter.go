package errorreporter

import "log/slog"

func ReportError(err error, args ...any) {
	if err == nil {
		return
	}
	slog.Error("error reported", "error", err, "args", args)
}
