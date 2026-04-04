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

	response, err := s.client.raw.GetServerSettingsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get server settings: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get server settings: %w", err)
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
	logger := s.client.logger.With(slog.String("operation", "UpdateAnalytics"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.SaveAnalyticsSettingsWithResponse(ctx, analyticsConfig)
	if err != nil {
		return nil, fmt.Errorf("update analytics settings: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update analytics settings: %w", err)
	}

	return result, nil
}
