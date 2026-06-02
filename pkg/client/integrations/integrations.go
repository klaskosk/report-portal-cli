package integrations

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// IntegrationsScope provides operations on global integrations at the instance level.
type IntegrationsScope struct {
	scope core.Scope
}

// ListGlobal returns all global integrations.
func (s *IntegrationsScope) ListGlobal(ctx context.Context) ([]types.IntegrationResource, error) {
	logger := s.scope.Logger.With(slog.String("operation", "ListGlobalIntegrations"))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetGlobalIntegrations1WithResponse(ctx)

	result, err := core.Execute[[]types.IntegrationResource](
		ctx,
		logger,
		"list global integrations",
		resp,
		err,
	)
	if err != nil {
		return nil, err
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
	logger := s.scope.Logger.With(
		slog.String("operation", "ListGlobalIntegrationsByPlugin"),
		slog.String("pluginName", pluginName),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetGlobalIntegrationsWithResponse(ctx, pluginName)

	result, err := core.Execute[[]types.IntegrationResource](
		ctx,
		logger,
		"list global integrations by plugin",
		resp,
		err,
	)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []types.IntegrationResource{}, nil
	}

	return *result, nil
}

// GetGlobal returns a single global integration by its numeric ID.
func (s *IntegrationsScope) GetGlobal(
	ctx context.Context,
	integrationID int64,
) (*types.IntegrationResource, error) {
	logger := s.scope.Logger.With(
		slog.String("operation", "GetGlobalIntegration"),
		slog.Int64("id", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetGlobalIntegrationWithResponse(ctx, integrationID)

	return core.Execute[types.IntegrationResource](ctx, logger, "get global integration", resp, err)
}

// TestGlobalConnection tests connectivity for a global integration.
func (s *IntegrationsScope) TestGlobalConnection(
	ctx context.Context,
	integrationID int64,
) (bool, error) {
	logger := s.scope.Logger.With(
		slog.String("operation", "TestGlobalIntegrationConnection"),
		slog.Int64("id", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.TestIntegrationConnection1WithResponse(ctx, integrationID)

	result, err := core.Execute[bool](ctx, logger, "test global integration connection", resp, err)
	if err != nil {
		return false, err
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
	logger := s.scope.Logger.With(
		slog.String("operation", "CreateGlobalIntegration"),
		slog.String("pluginName", pluginName),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.CreateGlobalIntegrationWithResponse(ctx, pluginName, request)

	return core.Execute[types.EntryCreatedRS](ctx, logger, "create global integration", resp, err)
}

// UpdateGlobal updates an existing global integration.
func (s *IntegrationsScope) UpdateGlobal(
	ctx context.Context,
	integrationID int64,
	request types.IntegrationRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.Logger.With(
		slog.String("operation", "UpdateGlobalIntegration"),
		slog.Int64("id", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.UpdateGlobalIntegrationWithResponse(ctx, integrationID, request)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"update global integration",
		resp,
		err,
	)
}

// DeleteGlobal removes a global integration by its numeric ID.
func (s *IntegrationsScope) DeleteGlobal(
	ctx context.Context,
	integrationID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.Logger.With(
		slog.String("operation", "DeleteGlobalIntegration"),
		slog.Int64("id", integrationID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeleteGlobalIntegrationWithResponse(ctx, integrationID)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"delete global integration",
		resp,
		err,
	)
}

// DeleteAllByType removes all global integrations of the given type.
func (s *IntegrationsScope) DeleteAllByType(
	ctx context.Context,
	pluginType string,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.Logger.With(
		slog.String("operation", "DeleteAllIntegrationsByType"),
		slog.String("type", pluginType),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeleteAllIntegrationsWithResponse(ctx, pluginType)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"delete all integrations by type",
		resp,
		err,
	)
}

// ProjectIntegrationsScope provides operations on integrations within a specific project.
type ProjectIntegrationsScope struct {
	scope core.Scope
}

// List returns all integrations for the project.
func (s *ProjectIntegrationsScope) List(ctx context.Context) ([]types.IntegrationResource, error) {
	logger := s.scope.OperationLogger("ListProjectIntegrations")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetProjectIntegrationsWithResponse(
		ctx, s.scope.ProjectName,
	)

	result, err := core.Execute[[]types.IntegrationResource](
		ctx,
		logger,
		"list project integrations",
		resp,
		err,
	)
	if err != nil {
		return nil, err
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
	logger := s.scope.OperationLogger("ListProjectIntegrationsByPlugin",
		slog.String("pluginName", pluginName),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetProjectIntegrations1WithResponse(
		ctx, s.scope.ProjectName, pluginName,
	)

	result, err := core.Execute[[]types.IntegrationResource](
		ctx,
		logger,
		"list project integrations by plugin",
		resp,
		err,
	)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []types.IntegrationResource{}, nil
	}

	return *result, nil
}

// Get returns a single project integration by its numeric ID.
func (s *ProjectIntegrationsScope) Get(
	ctx context.Context,
	integrationID int64,
) (*types.IntegrationResource, error) {
	logger := s.scope.OperationLogger("GetProjectIntegration", slog.Int64("id", integrationID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetProjectIntegrationWithResponse(
		ctx, s.scope.ProjectName, integrationID,
	)

	return core.Execute[types.IntegrationResource](
		ctx,
		logger,
		"get project integration",
		resp,
		err,
	)
}

// TestConnection tests connectivity for a project integration.
func (s *ProjectIntegrationsScope) TestConnection(ctx context.Context, id int64) (bool, error) {
	logger := s.scope.OperationLogger("TestProjectIntegrationConnection", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.TestIntegrationConnectionWithResponse(
		ctx, s.scope.ProjectName, id,
	)

	result, err := core.Execute[bool](ctx, logger, "test project integration connection", resp, err)
	if err != nil {
		return false, err
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
	logger := s.scope.OperationLogger("CreateProjectIntegration",
		slog.String("pluginName", pluginName),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.CreateProjectIntegrationWithResponse(
		ctx, s.scope.ProjectName, pluginName, request,
	)

	return core.Execute[types.EntryCreatedRS](ctx, logger, "create project integration", resp, err)
}

// Update modifies an existing project integration.
func (s *ProjectIntegrationsScope) Update(
	ctx context.Context,
	integrationID int64,
	request types.IntegrationRQ,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger("UpdateProjectIntegration", slog.Int64("id", integrationID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.UpdateProjectIntegrationWithResponse(
		ctx, s.scope.ProjectName, integrationID, request,
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"update project integration",
		resp,
		err,
	)
}

// Delete removes a project integration by its numeric ID.
func (s *ProjectIntegrationsScope) Delete(
	ctx context.Context,
	integrationID int64,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger("DeleteProjectIntegration", slog.Int64("id", integrationID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeleteProjectIntegrationWithResponse(
		ctx, s.scope.ProjectName, integrationID,
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"delete project integration",
		resp,
		err,
	)
}

// DeleteAllByType removes all project integrations of the given type.
func (s *ProjectIntegrationsScope) DeleteAllByType(
	ctx context.Context,
	pluginType string,
) (*types.OperationCompletionRS, error) {
	logger := s.scope.OperationLogger("DeleteAllProjectIntegrationsByType",
		slog.String("type", pluginType),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeleteAllProjectIntegrationsWithResponse(
		ctx, s.scope.ProjectName, pluginType,
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"delete all project integrations by type",
		resp,
		err,
	)
}

// ExecuteCommand executes a command on a project integration.
func (s *ProjectIntegrationsScope) ExecuteCommand(
	ctx context.Context,
	integrationID int64,
	command string,
	body map[string]any,
) (map[string]any, error) {
	logger := s.scope.OperationLogger("ExecuteIntegrationCommand",
		slog.Int64("integrationId", integrationID),
		slog.String("command", command),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.ExecuteIntegrationCommandWithResponse(
		ctx, s.scope.ProjectName, integrationID, command, body,
	)

	result, err := core.Execute[map[string]any](
		ctx,
		logger,
		"execute integration command",
		resp,
		err,
	)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string]any{}, nil
	}

	return *result, nil
}

// NewIntegrationsScope constructs a IntegrationsScope for wiring from the facade.
func NewIntegrationsScope(s core.Scope) *IntegrationsScope {
	return &IntegrationsScope{scope: s}
}

// NewProjectIntegrationsScope constructs a ProjectIntegrationsScope for wiring from the facade.
func NewProjectIntegrationsScope(s core.Scope) *ProjectIntegrationsScope {
	return &ProjectIntegrationsScope{scope: s}
}
