package cli

import (
	"context"
	"log/slog"
	"math"
	"os"

	"github.com/spf13/cobra"
)

type loggerKey struct{}

// newLogger creates a *slog.Logger that writes to stderr at the level implied
// by the given kubectl-style verbosity:
//
//	0 → effectively off (only CLI output)
//	1 → Info
//	2+→ Debug
func newLogger(verbosity int) *slog.Logger {
	level := verbosityToLevel(verbosity)

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

// verbosityToLevel converts a kubectl-style verbosity (0-2+) to an slog.Level.
func verbosityToLevel(verbosity int) slog.Level {
	switch {
	case verbosity <= 0:
		// Disable all logging when not explicitly enabled. Errors will still be reported as part of the CLI,
		// just not through the logger.
		return slog.Level(math.MaxInt)
	case verbosity == 1:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
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
