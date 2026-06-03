package logtypes

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// LogTypesScope provides operations on log types within a specific project.
type LogTypesScope struct {
	scope core.Scope
}

// List returns all log types for the project.
func (s *LogTypesScope) List(ctx context.Context) ([]types.LogTypeResponse, error) {
	logger := s.scope.OperationLogger("ListLogTypes")
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.GetLogTypesWithResponse(ctx, s.scope.ProjectName)

	result, err := core.Execute(ctx, logger, "list log types", resp, err)
	if err != nil {
		return nil, err
	}

	if result.Items == nil || len(*result.Items) == 0 {
		return []types.LogTypeResponse{}, nil
	}

	return *result.Items, nil
}

// Create creates a new log type in the project. The RP API returns 201
// Created, but the spec declares 200, so the generated parser misclassifies
// the response. This method handles both status codes.
func (s *LogTypesScope) Create(
	ctx context.Context,
	request types.LogTypeRequest,
) (*types.LogTypeResponse, error) {
	logger := s.scope.OperationLogger("CreateLogType", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.CreateLogTypeWithResponse(ctx, s.scope.ProjectName, request)
	if err != nil {
		return nil, fmt.Errorf("create log type: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", resp.StatusCode())

	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}

	if resp.StatusCode() == 201 {
		var result types.LogTypeResponse

		err := json.Unmarshal(resp.Body, &result)
		if err != nil {
			return nil, fmt.Errorf("create log type: %w", err)
		}

		return &result, nil
	}

	if resp.JSONDefault != nil {
		return nil, fmt.Errorf("create log type: %w",
			core.NewAPIError(resp.StatusCode(), resp.JSONDefault))
	}

	return nil, fmt.Errorf("create log type: %w",
		core.NewUnexpectedStatusError(resp.StatusCode(), resp.Body))
}

// Update modifies an existing log type by its ID.
func (s *LogTypesScope) Update(
	ctx context.Context,
	logTypeID int64,
	request types.LogTypeRequest,
) (*types.SuccessfulUpdate, error) {
	logger := s.scope.OperationLogger("UpdateLogType", slog.Int64("logTypeId", logTypeID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.UpdateLogTypeByIdWithResponse(
		ctx, s.scope.ProjectName, logTypeID, request,
	)

	return core.Execute(ctx, logger, "update log type", resp, err)
}

// Delete removes a log type by its ID.
func (s *LogTypesScope) Delete(ctx context.Context, logTypeID int64) error {
	logger := s.scope.OperationLogger("DeleteLogType", slog.Int64("logTypeId", logTypeID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.scope.Client.DeleteLogTypeByIdWithResponse(
		ctx,
		s.scope.ProjectName,
		logTypeID,
	)

	return core.ExecuteVoid(ctx, logger, "delete log type", resp, err)
}

// NewLogTypesScope constructs a LogTypesScope for wiring from the facade.
func NewLogTypesScope(s core.Scope) *LogTypesScope {
	return &LogTypesScope{scope: s}
}
