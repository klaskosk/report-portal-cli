package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// IntegrationsScope provides operations on global integrations at the instance level.
type IntegrationsScope struct {
	client *Client
}

// ListGlobal returns all global integrations.
func (s *IntegrationsScope) ListGlobal(ctx context.Context) ([]types.IntegrationResource, error) {
	logger := s.client.logger.With(slog.String("operation", "ListGlobalIntegrations"))
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetGlobalIntegrations1WithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("list global integrations: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list global integrations: %w", err)
	}

	if result == nil {
		return []types.IntegrationResource{}, nil
	}

	return *result, nil
}

// ListGlobalByPlugin returns global integrations filtered by plugin name.
func (s *IntegrationsScope) ListGlobalByPlugin(
	ctx context.Context,
	pluginName string,
) ([]types.IntegrationResource, error) {
	logger := s.client.logger.With(
		slog.String("operation", "ListGlobalIntegrationsByPlugin"),
		slog.String("pluginName", pluginName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetGlobalIntegrationsWithResponse(ctx, pluginName)
	if err != nil {
		return nil, fmt.Errorf("list global integrations by plugin: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list global integrations by plugin: %w", err)
	}

	if result == nil {
		return []types.IntegrationResource{}, nil
	}

	return *result, nil
}

// GetGlobal returns a single global integration by its numeric ID.
func (s *IntegrationsScope) GetGlobal(
	ctx context.Context,
	id int64,
) (*types.IntegrationResource, error) {
	logger := s.client.logger.With(
		slog.String("operation", "GetGlobalIntegration"),
		slog.Int64("id", id),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.GetGlobalIntegrationWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get global integration: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get global integration: %w", err)
	}

	return result, nil
}

// TestGlobalConnection tests connectivity for a global integration.
func (s *IntegrationsScope) TestGlobalConnection(ctx context.Context, id int64) (bool, error) {
	logger := s.client.logger.With(
		slog.String("operation", "TestGlobalIntegrationConnection"),
		slog.Int64("id", id),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.TestIntegrationConnection1WithResponse(ctx, id)
	if err != nil {
		return false, fmt.Errorf("test global integration connection: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return false, fmt.Errorf("test global integration connection: %w", err)
	}

	if result == nil {
		return false, nil
	}

	return *result, nil
}

// CreateGlobal creates a new global integration for the given plugin.
func (s *IntegrationsScope) CreateGlobal(
	ctx context.Context,
	pluginName string,
	request types.IntegrationRQ,
) (*types.EntryCreatedRS, error) {
	logger := s.client.logger.With(
		slog.String("operation", "CreateGlobalIntegration"),
		slog.String("pluginName", pluginName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.CreateGlobalIntegrationWithResponse(
		ctx,
		pluginName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create global integration: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create global integration: %w", err)
	}

	return result, nil
}

// UpdateGlobal updates an existing global integration.
func (s *IntegrationsScope) UpdateGlobal(
	ctx context.Context,
	id int64,
	request types.IntegrationRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(
		slog.String("operation", "UpdateGlobalIntegration"),
		slog.Int64("id", id),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.UpdateGlobalIntegrationWithResponse(ctx, id, request)
	if err != nil {
		return nil, fmt.Errorf("update global integration: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update global integration: %w", err)
	}

	return result, nil
}

// DeleteGlobal removes a global integration by its numeric ID.
func (s *IntegrationsScope) DeleteGlobal(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := s.client.logger.With(
		slog.String("operation", "DeleteGlobalIntegration"),
		slog.Int64("id", id),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.client.raw.DeleteGlobalIntegrationWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("delete global integration: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete global integration: %w", err)
	}

	return result, nil
}

// ProjectIntegrationsScope provides operations on integrations within a specific project.
type ProjectIntegrationsScope struct {
	project *ProjectScope
}

// List returns all integrations for the project.
func (s *ProjectIntegrationsScope) List(ctx context.Context) ([]types.IntegrationResource, error) {
	logger := s.project.operationLogger("ListProjectIntegrations")
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetProjectIntegrationsWithResponse(
		ctx,
		s.project.projectName,
	)
	if err != nil {
		return nil, fmt.Errorf("list project integrations: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list project integrations: %w", err)
	}

	if result == nil {
		return []types.IntegrationResource{}, nil
	}

	return *result, nil
}

// ListByPlugin returns project integrations filtered by plugin name.
func (s *ProjectIntegrationsScope) ListByPlugin(
	ctx context.Context,
	pluginName string,
) ([]types.IntegrationResource, error) {
	logger := s.project.operationLogger("ListProjectIntegrationsByPlugin",
		slog.String("pluginName", pluginName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetProjectIntegrations1WithResponse(
		ctx,
		s.project.projectName,
		pluginName,
	)
	if err != nil {
		return nil, fmt.Errorf("list project integrations by plugin: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list project integrations by plugin: %w", err)
	}

	if result == nil {
		return []types.IntegrationResource{}, nil
	}

	return *result, nil
}

// Get returns a single project integration by its numeric ID.
func (s *ProjectIntegrationsScope) Get(
	ctx context.Context,
	id int64,
) (*types.IntegrationResource, error) {
	logger := s.project.operationLogger("GetProjectIntegration", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetProjectIntegrationWithResponse(
		ctx,
		s.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get project integration: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get project integration: %w", err)
	}

	return result, nil
}

// TestConnection tests connectivity for a project integration.
func (s *ProjectIntegrationsScope) TestConnection(ctx context.Context, id int64) (bool, error) {
	logger := s.project.operationLogger("TestProjectIntegrationConnection", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.TestIntegrationConnectionWithResponse(
		ctx,
		s.project.projectName,
		id,
	)
	if err != nil {
		return false, fmt.Errorf("test project integration connection: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return false, fmt.Errorf("test project integration connection: %w", err)
	}

	if result == nil {
		return false, nil
	}

	return *result, nil
}

// Create creates a new project integration for the given plugin.
func (s *ProjectIntegrationsScope) Create(
	ctx context.Context,
	pluginName string,
	request types.IntegrationRQ,
) (*types.EntryCreatedRS, error) {
	logger := s.project.operationLogger("CreateProjectIntegration",
		slog.String("pluginName", pluginName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.CreateProjectIntegrationWithResponse(
		ctx,
		s.project.projectName,
		pluginName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create project integration: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create project integration: %w", err)
	}

	return result, nil
}

// Update modifies an existing project integration.
func (s *ProjectIntegrationsScope) Update(
	ctx context.Context,
	id int64,
	request types.IntegrationRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("UpdateProjectIntegration", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.UpdateProjectIntegrationWithResponse(
		ctx,
		s.project.projectName,
		id,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update project integration: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update project integration: %w", err)
	}

	return result, nil
}

// Delete removes a project integration by its numeric ID.
func (s *ProjectIntegrationsScope) Delete(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := s.project.operationLogger("DeleteProjectIntegration", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.DeleteProjectIntegrationWithResponse(
		ctx,
		s.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("delete project integration: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete project integration: %w", err)
	}

	return result, nil
}

// ExecuteCommand executes a command on a project integration.
func (s *ProjectIntegrationsScope) ExecuteCommand(
	ctx context.Context,
	integrationID int64,
	command string,
	body map[string]any,
) (map[string]any, error) {
	logger := s.project.operationLogger("ExecuteIntegrationCommand",
		slog.Int64("integrationId", integrationID),
		slog.String("command", command),
	)
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.ExecuteIntegrationCommandWithResponse(
		ctx,
		s.project.projectName,
		integrationID,
		command,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("execute integration command: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("execute integration command: %w", err)
	}

	if result == nil {
		return map[string]any{}, nil
	}

	return *result, nil
}
