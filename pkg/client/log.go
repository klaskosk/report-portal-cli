package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// BatchLogEntry represents a single log entry for batch creation. It holds the
// log payload fields and an optional file attachment.
type BatchLogEntry struct {
	LaunchUUID     string
	ItemUUID       string
	Level          string
	Message        string
	Time           types.EpochMillis
	AttachmentName string
	AttachmentData []byte
}

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

// CreateBatch creates multiple log entries in a single multipart request,
// optionally including file attachments.
func (l *LogScope) CreateBatch(
	ctx context.Context,
	entries ...BatchLogEntry,
) (*types.BatchSaveOperatingRS, error) {
	logger := l.project.operationLogger("CreateLogBatch")
	logger.InfoContext(ctx, "API request")

	body, contentType, err := buildBatchLogMultipart(entries)
	if err != nil {
		return nil, fmt.Errorf("create log batch: %w", err)
	}

	response, err := l.project.client.raw.CreateLogUsingPOST1WithBodyWithResponse(
		ctx,
		l.project.projectName,
		contentType,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("create log batch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.StatusCode() == http.StatusCreated {
		var created types.BatchSaveOperatingRS
		if err := json.Unmarshal(response.Body, &created); err == nil {
			return &created, nil
		}
	}

	return handleResponse(response, "create log batch", response.JSONDefault, response.JSON200)
}

// batchLogJSON is the JSON representation of a single log entry within the
// multipart batch payload.
type batchLogJSON struct {
	LaunchUUID string            `json:"launchUuid"`
	ItemUUID   string            `json:"itemUuid,omitempty"`
	Level      string            `json:"level"`
	Message    string            `json:"message"`
	Time       types.EpochMillis `json:"time"`
	File       *batchLogFile     `json:"file,omitempty"`
}

// batchLogFile is the file metadata embedded in the JSON part of a batch log entry.
type batchLogFile struct {
	Name string `json:"name"`
}

// buildBatchLogMultipart assembles the multipart body for a batch log request.
// It returns the body buffer and the full Content-Type header value including
// the boundary.
func buildBatchLogMultipart(entries []BatchLogEntry) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	jsonEntries := make([]batchLogJSON, len(entries))
	for i, entry := range entries {
		jsonEntries[i] = batchLogJSON{
			LaunchUUID: entry.LaunchUUID,
			ItemUUID:   entry.ItemUUID,
			Level:      entry.Level,
			Message:    entry.Message,
			Time:       entry.Time,
		}
		if entry.AttachmentName != "" {
			jsonEntries[i].File = &batchLogFile{Name: entry.AttachmentName}
		}
	}

	jsonData, err := json.Marshal(jsonEntries)
	if err != nil {
		return nil, "", fmt.Errorf("marshaling log entries: %w", err)
	}

	jsonHeader := make(textproto.MIMEHeader)
	jsonHeader.Set("Content-Disposition", `form-data; name="json_request_part"`)
	jsonHeader.Set("Content-Type", "application/json")

	jsonPart, err := writer.CreatePart(jsonHeader)
	if err != nil {
		return nil, "", fmt.Errorf("creating json part: %w", err)
	}

	if _, err := jsonPart.Write(jsonData); err != nil {
		return nil, "", fmt.Errorf("writing json part: %w", err)
	}

	for _, entry := range entries {
		if entry.AttachmentName == "" {
			continue
		}

		filePart, err := writer.CreateFormFile("file", entry.AttachmentName)
		if err != nil {
			return nil, "", fmt.Errorf("creating file part: %w", err)
		}

		if _, err := filePart.Write(entry.AttachmentData); err != nil {
			return nil, "", fmt.Errorf("writing file part: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
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

// Search finds log entries related to the given test item that match similar
// failure patterns. This is useful for triage workflows where an operator
// needs to find analogous failures across the project.
func (l *LogScope) Search(
	ctx context.Context,
	itemID int64,
	request types.SearchLogRq,
) (*types.IterableSearchLogRs, error) {
	logger := l.project.operationLogger("SearchLogs", slog.Int64("itemId", itemID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.SearchLogsUsingPOSTWithResponse(
		ctx,
		l.project.projectName,
		itemID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("search logs: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "search logs", response.JSONDefault, response.JSON200)
}

// GetLogsUnder retrieves log entries grouped by item ID for the specified
// items and optional log level filter. The returned map is keyed by item ID
// (as a string) with each value containing that item's log entries.
func (l *LogScope) GetLogsUnder(
	ctx context.Context,
	request types.GetLogsUnderRq,
) (map[string][]types.LogResource, error) {
	logger := l.project.operationLogger("GetLogsUnder")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLogsUnderUsingPOSTWithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("get logs under: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, "get logs under", response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string][]types.LogResource{}, nil
	}

	return *result, nil
}
