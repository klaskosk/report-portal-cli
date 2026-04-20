package client

import (
	"context"
	"encoding/json"
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

	resp, err := s.project.client.raw.GetLogTypesWithResponse(ctx, s.project.projectName)

	result, err := execute(ctx, logger, "list log types", resp, err)
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
func (s *LogTypesScope) Create(ctx context.Context, request types.LogTypeRequest) (*types.LogTypeResponse, error) {
	logger := s.project.operationLogger("CreateLogType", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.CreateLogTypeWithResponse(ctx, s.project.projectName, request)
	if err != nil {
		return nil, fmt.Errorf("create log type: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", resp.StatusCode())

	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}

	if resp.StatusCode() == 201 {
		var result types.LogTypeResponse
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return nil, fmt.Errorf("create log type: %w", err)
		}

		return &result, nil
	}

	if resp.JSONDefault != nil {
		return nil, fmt.Errorf("create log type: %w",
			newAPIError(resp.StatusCode(), resp.JSONDefault))
	}

	return nil, fmt.Errorf("create log type: %w",
		newUnexpectedStatusError(resp.StatusCode(), resp.Body))
}

// Update modifies an existing log type by its ID.
func (s *LogTypesScope) Update(
	ctx context.Context,
	logTypeID int64,
	request types.LogTypeRequest,
) (*types.SuccessfulUpdate, error) {
	logger := s.project.operationLogger("UpdateLogType", slog.Int64("logTypeId", logTypeID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.UpdateLogTypeByIdWithResponse(
		ctx, s.project.projectName, logTypeID, request,
	)

	return execute(ctx, logger, "update log type", resp, err)
}

// Delete removes a log type by its ID.
func (s *LogTypesScope) Delete(ctx context.Context, logTypeID int64) error {
	logger := s.project.operationLogger("DeleteLogType", slog.Int64("logTypeId", logTypeID))
	logger.InfoContext(ctx, "API request")

	resp, err := s.project.client.raw.DeleteLogTypeByIdWithResponse(ctx, s.project.projectName, logTypeID)

	return executeVoid(ctx, logger, "delete log type", resp, err)
}
