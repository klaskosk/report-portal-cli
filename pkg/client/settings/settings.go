package settings

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// SettingsScope provides operations on server-level settings.
// Most methods require admin privileges.
type SettingsScope struct {
	scope core.Scope
}

// GetServerSettings returns the current server settings (analytics configuration).
// Requires admin privileges.
func (s *SettingsScope) GetServerSettings(ctx context.Context) (map[string]string, error) {
	logger := s.scope.Logger.With(slog.String("operation", "GetServerSettings"))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetServerSettingsWithResponse(ctx)

	result, err := core.Execute[map[string]string](ctx, logger, "get server settings", resp, err)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string]string{}, nil
	}

	return *result, nil
}

// UpdateAnalytics updates the server analytics configuration.
// Requires admin privileges.
func (s *SettingsScope) UpdateAnalytics(
	ctx context.Context,
	analyticsConfig types.AnalyticsResource,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.Logger.With(slog.String("operation", "UpdateAnalytics"))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.SaveAnalyticsSettingsWithResponse(ctx, analyticsConfig)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"update analytics settings",
		resp,
		err,
	)
}

// UpdateServerSettings updates a single server setting by key.
// Requires admin privileges.
func (s *SettingsScope) UpdateServerSettings(
	ctx context.Context,
	request types.UpdateSettingsRq,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.Logger.With(slog.String("operation", "UpdateServerSettings"))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.UpdateServerSettingsWithResponse(ctx, request)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"update server settings",
		resp,
		err,
	)
}
