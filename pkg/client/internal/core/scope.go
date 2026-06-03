package core

import (
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
)

// Scope carries shared execution context for API calls.
type Scope struct {
	Client      api.ClientWithResponsesInterface
	RawClient   api.ClientInterface
	Logger      *slog.Logger
	ProjectName string // empty for instance-level scopes
}

// NewScope constructs a root scope with the given clients and logger.
func NewScope(
	client api.ClientWithResponsesInterface,
	rawClient api.ClientInterface,
	logger *slog.Logger,
) Scope {
	return Scope{
		Client:    client,
		RawClient: rawClient,
		Logger:    logger,
	}
}

// WithProject returns a copy of the scope scoped to the given project name.
func (s Scope) WithProject(name string) Scope {
	s.ProjectName = name

	return s
}

// OperationLogger returns a logger scoped to this scope and the given operation.
func (s Scope) OperationLogger(operation string, attrs ...slog.Attr) *slog.Logger {
	logger := s.Logger.With(slog.String("operation", operation))
	if s.ProjectName != "" {
		logger = logger.With(slog.String("project", s.ProjectName))
	}

	for _, attr := range attrs {
		logger = logger.With(attr)
	}

	return logger
}
