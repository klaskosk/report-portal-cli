package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// PluginScope provides operations on installed plugins at the instance level.
// Most methods require admin privileges.
type PluginScope struct {
	client *Client
}

// List returns all installed plugins. Requires admin privileges.
func (s *PluginScope) List(ctx context.Context) ([]types.IntegrationTypeResource, error) {
	logger := s.client.logger.With(slog.String("operation", "ListPlugins"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetPluginsUsingGETWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, "list plugins", response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []types.IntegrationTypeResource{}, nil
	}

	return *result, nil
}

// ListPublic returns all plugins via the public endpoint. Some deployments
// allow this endpoint without authentication.
func (s *PluginScope) ListPublic(ctx context.Context) ([]types.IntegrationTypeResource, error) {
	logger := s.client.logger.With(slog.String("operation", "ListPublicPlugins"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetPluginsUsingGET1WithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("list public plugins: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, "list public plugins", response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []types.IntegrationTypeResource{}, nil
	}

	return *result, nil
}

// UpdateState enables or disables a plugin. Requires admin privileges.
func (s *PluginScope) UpdateState(ctx context.Context, pluginID int64, stateUpdate types.UpdatePluginStateRQ) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(
		slog.String("operation", "UpdatePluginState"),
		slog.Int64("pluginId", pluginID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.UpdatePluginStateUsingPUTWithResponse(ctx, pluginID, stateUpdate)
	if err != nil {
		return nil, fmt.Errorf("update plugin state: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "update plugin state", response.JSONDefault, response.JSON200)
}

// Delete removes a plugin. Requires admin privileges.
func (s *PluginScope) Delete(ctx context.Context, pluginID int64) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(
		slog.String("operation", "DeletePlugin"),
		slog.Int64("pluginId", pluginID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.DeletePluginUsingDELETEWithResponse(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("delete plugin: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "delete plugin", response.JSONDefault, response.JSON200)
}

// GetPublicFile downloads a public file from a plugin (e.g., plugin icon).
// Returns the raw file bytes. Some deployments allow this without authentication.
func (s *PluginScope) GetPublicFile(ctx context.Context, pluginName, fileName string) ([]byte, error) {
	logger := s.client.logger.With(
		slog.String("operation", "GetPublicPluginFile"),
		slog.String("pluginName", pluginName),
		slog.String("fileName", fileName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetPublicFileUsingGETWithResponse(ctx, pluginName, fileName)
	if err != nil {
		return nil, fmt.Errorf("get public plugin file: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, newAPIError("get public plugin file", response.StatusCode(), response.JSONDefault)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, newUnexpectedStatusError("get public plugin file", code)
	}

	return response.Body, nil
}
