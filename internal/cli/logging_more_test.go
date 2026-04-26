package cli

import (
	"log/slog"
	"testing"
)

func TestEnableDebugLoggingDoesNotPanic(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	enableDebugLogging()
}
