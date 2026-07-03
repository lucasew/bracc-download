package errorreporter

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestReportError(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	err := errors.New("test error")
	ReportError(err, "context", "something bad happened")

	output := buf.String()
	if !strings.Contains(output, "level=ERROR") {
		t.Errorf("expected level=ERROR, got: %s", output)
	}
	if !strings.Contains(output, "msg=\"error reported\"") {
		t.Errorf("expected msg=\"error reported\", got: %s", output)
	}
	if !strings.Contains(output, "error=\"test error\"") {
		t.Errorf("expected error=\"test error\", got: %s", output)
	}
	if !strings.Contains(output, "context=\"something bad happened\"") {
		t.Errorf("expected context=\"something bad happened\", got: %s", output)
	}
}

func TestReportError_NilError(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	slog.SetDefault(slog.New(handler))

	ReportError(nil, "context", "this should not be logged")

	output := buf.String()
	if output != "" {
		t.Errorf("expected no output for nil error, got: %s", output)
	}
}
