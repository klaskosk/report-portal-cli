package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// SettingsScope provides operations on server-level settings.
// Most methods require admin privileges.
type SettingsScope struct {
	client *Client
}

// GetServerSettings returns the current server settings (analytics configuration).
// Requires admin privileges.
func (s *SettingsScope) GetServerSettings(ctx context.Context) (map[string]string, error) {
	logger := s.client.logger.With(slog.String("operation", "GetServerSettings"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetServerSettingsUsingGETWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get server settings: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, "get server settings", response.JSONDefault, response.JSON200)
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
func (s *SettingsScope) UpdateAnalytics(ctx context.Context, analyticsConfig types.AnalyticsResource) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(slog.String("operation", "UpdateAnalytics"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.SaveAnalyticsSettingsUsingPUTWithResponse(ctx, analyticsConfig)
	if err != nil {
		return nil, fmt.Errorf("update analytics settings: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "update analytics settings", response.JSONDefault, response.JSON200)
}
