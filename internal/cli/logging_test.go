package cli

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestEnableDebugLoggingEmitsDebug(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	enableDebugLoggingTo(&buf)

	slog.Debug("hello", "k", "v")

	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected debug output to contain message, got: %q", out)
	}
}
