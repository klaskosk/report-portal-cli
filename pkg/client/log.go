package client

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// LogScope provides read operations on logs within a project.
type LogScope struct {
	project *ProjectScope
}

// GetByID returns a single log entry by its numeric ID.
func (l *LogScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.LogResource, error) {
	logger := l.project.operationLogger("GetLogByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLogUsingGETWithResponse(
		ctx,
		l.project.projectName,
		strconv.FormatInt(id, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("get log by id: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get log by id", response.JSONDefault, response.JSON200)
}

// GetByUUID returns a single log entry by its UUID.
func (l *LogScope) GetByUUID(
	ctx context.Context,
	uuid string,
) (*types.LogResource, error) {
	logger := l.project.operationLogger("GetLogByUUID", slog.String("uuid", uuid))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLogByUuidUsingGETWithResponse(
		ctx,
		l.project.projectName,
		uuid,
	)
	if err != nil {
		return nil, fmt.Errorf("get log by uuid: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get log by uuid", response.JSONDefault, response.JSON200)
}

// ListLogsOption configures filter and pagination parameters for listing logs.
type ListLogsOption func(*api.GetLogsUsingGETParams)

// List returns a paginated list of log entries matching the given filter options.
func (l *LogScope) List(
	ctx context.Context,
	options ...ListLogsOption,
) (*types.IterableLogResource, error) {
	logger := l.project.operationLogger("ListLogs")

	params := &api.GetLogsUsingGETParams{}
	for _, option := range options {
		option(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLogsUsingGETWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("list logs: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "list logs", response.JSONDefault, response.JSON200)
}

// Create creates a single log entry (JSON body, no file attachment).
func (l *LogScope) Create(
	ctx context.Context,
	request types.SaveLogRQ,
) (*types.EntryCreatedAsyncRS, error) {
	logger := l.project.operationLogger("CreateLogEntry")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.CreateLogEntryUsingPOST1WithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create log entry: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "create log entry", response.JSONDefault, response.JSON201)
}

// Delete removes a single log entry by its numeric ID.
func (l *LogScope) Delete(
	ctx context.Context,
	logID int64,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("DeleteLog", slog.Int64("logId", logID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.DeleteLogUsingDELETEWithResponse(
		ctx,
		l.project.projectName,
		logID,
	)
	if err != nil {
		return nil, fmt.Errorf("delete log: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "delete log", response.JSONDefault, response.JSON200)
}

// WithLogItemID filters logs by test item ID.
func WithLogItemID(itemID int64) ListLogsOption {
	return func(params *api.GetLogsUsingGETParams) {
		params.FilterEqItem = &itemID
	}
}

// WithLogLaunchID filters logs by launch ID.
func WithLogLaunchID(launchID int64) ListLogsOption {
	return func(params *api.GetLogsUsingGETParams) {
		params.FilterEqLaunchId = &launchID
	}
}

// WithLogLevel filters logs by level (e.g. "ERROR", "INFO", "DEBUG").
func WithLogLevel(level string) ListLogsOption {
	return func(params *api.GetLogsUsingGETParams) {
		params.FilterEqLevel = &level
	}
}

// WithLogPageNumber sets the page number (0-based).
func WithLogPageNumber(n int32) ListLogsOption {
	return func(params *api.GetLogsUsingGETParams) {
		params.PagePage = &n
	}
}

// WithLogPageSize sets the number of results per page.
func WithLogPageSize(size int32) ListLogsOption {
	return func(params *api.GetLogsUsingGETParams) {
		params.PageSize = &size
	}
}

// WithLogSort sets the sorting criteria (e.g. "logTime,desc").
func WithLogSort(fields ...string) ListLogsOption {
	return func(params *api.GetLogsUsingGETParams) {
		params.PageSort = &fields
	}
}
