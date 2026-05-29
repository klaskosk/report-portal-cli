package core

import (
	"context"
	"log/slog"
	"math"
	"os"

	"github.com/spf13/cobra"
)

type loggerKey struct{}

// NewLogger creates a *slog.Logger that writes to stderr at the level implied
// by the given kubectl-style verbosity.
func NewLogger(verbosity int) *slog.Logger {
	level := verbosityToLevel(verbosity)

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler)
}

func verbosityToLevel(verbosity int) slog.Level {
	switch {
	case verbosity <= 0:
		return slog.Level(math.MaxInt)
	case verbosity == 1:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

// LoggerFromCmd extracts the logger stored in the command's context.
func LoggerFromCmd(cmd *cobra.Command) *slog.Logger {
	return LoggerFromContext(cmd.Context())
}

// LoggerFromContext extracts the logger from ctx, returning a discard logger if none is set.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}

	return slog.New(slog.DiscardHandler)
}

// WithLogger stores logger in ctx.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}
