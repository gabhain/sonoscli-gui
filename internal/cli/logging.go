package cli

import (
	"io"
	"log/slog"
	"os"
)

// enableDebugLogging configures the global slog logger to emit debug logs to stderr.
//
// It is intentionally a no-op unless the user passes --debug, to avoid affecting
// library/test consumers.
func enableDebugLogging() {
	enableDebugLoggingTo(os.Stderr)
}

func enableDebugLoggingTo(w io.Writer) {
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})))
}
