package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// LogTypesScope provides operations on log types within a specific project.
type LogTypesScope struct {
	project *ProjectScope
}

// List returns all log types for the project.
func (s *LogTypesScope) List(ctx context.Context) ([]types.LogTypeResponse, error) {
	logger := s.project.operationLogger("ListLogTypes")
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.GetLogTypesWithResponse(ctx, s.project.projectName)
	if err != nil {
		return nil, fmt.Errorf("list log types: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list log types: %w", err)
	}

	if result.Items == nil || len(*result.Items) == 0 {
		return []types.LogTypeResponse{}, nil
	}

	return *result.Items, nil
}

// Create creates a new log type in the project.
func (s *LogTypesScope) Create(ctx context.Context, request types.LogTypeRequest) (*types.LogTypeResponse, error) {
	logger := s.project.operationLogger("CreateLogType", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.CreateLogTypeWithResponse(ctx, s.project.projectName, request)
	if err != nil {
		return nil, fmt.Errorf("create log type: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create log type: %w", err)
	}

	return result, nil
}

// Update modifies an existing log type by its ID.
func (s *LogTypesScope) Update(
	ctx context.Context,
	logTypeID int64,
	request types.LogTypeRequest,
) (*types.SuccessfulUpdate, error) {
	logger := s.project.operationLogger("UpdateLogType", slog.Int64("logTypeId", logTypeID))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.UpdateLogTypeByIdWithResponse(
		ctx, s.project.projectName, logTypeID, request,
	)
	if err != nil {
		return nil, fmt.Errorf("update log type: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update log type: %w", err)
	}

	return result, nil
}

// Delete removes a log type by its ID.
func (s *LogTypesScope) Delete(ctx context.Context, logTypeID int64) error {
	logger := s.project.operationLogger("DeleteLogType", slog.Int64("logTypeId", logTypeID))
	logger.InfoContext(ctx, "API request")

	response, err := s.project.client.raw.DeleteLogTypeByIdWithResponse(ctx, s.project.projectName, logTypeID)
	if err != nil {
		return fmt.Errorf("delete log type: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if err := handleVoidResponse(response, response.JSONDefault, response.Body); err != nil {
		return fmt.Errorf("delete log type: %w", err)
	}

	return nil
}
