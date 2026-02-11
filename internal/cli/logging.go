package cli

import (
	"context"
	"log/slog"
	"math"
	"os"

	"github.com/spf13/cobra"
)

// LevelTrace is a custom slog level for high-verbosity tracing (V=3+).
const LevelTrace = slog.LevelDebug - 4

type loggerKey struct{}

// newLogger creates a *slog.Logger that writes to stderr at the level implied
// by the given kubectl-style verbosity:
//
//	0 → effectively off (only CLI output)
//	1 → Info
//	2 → Debug
//	3+→ Trace (custom level below Debug)
func newLogger(verbosity int) *slog.Logger {
	level := verbosityToLevel(verbosity)

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceLevel,
	})

	return slog.New(handler)
}

// verbosityToLevel converts a kubectl-style verbosity (0-3+) to an slog.Level.
func verbosityToLevel(verbosity int) slog.Level {
	switch {
	case verbosity <= 0:
		return slog.Level(math.MaxInt)
	case verbosity == 1:
		return slog.LevelInfo
	case verbosity == 2:
		return slog.LevelDebug
	default:
		return LevelTrace
	}
}

// replaceLevel renders the custom LevelTrace as "TRACE" in text output.
func replaceLevel(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key != slog.LevelKey {
		return attr
	}

	level, ok := attr.Value.Any().(slog.Level)
	if !ok {
		return attr
	}

	if level <= LevelTrace {
		attr.Value = slog.StringValue("TRACE")
	}

	return attr
}

// loggerFromCmd extracts the logger stored in the command's context.
func loggerFromCmd(cmd *cobra.Command) *slog.Logger {
	return loggerFromContext(cmd.Context())
}

// loggerFromContext extracts the logger from ctx, returning a discard logger if none is set.
func loggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}

	return slog.New(slog.DiscardHandler)
}
