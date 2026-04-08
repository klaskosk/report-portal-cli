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
	"strings"
	"time"

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
	Time           time.Time
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

	response, err := l.project.client.raw.GetLogWithResponse(ctx, l.project.projectName, id)

	return execute[types.LogResource](ctx, logger, "get log by id", response, err)
}

// GetByUUID returns a single log entry by its UUID.
func (l *LogScope) GetByUUID(
	ctx context.Context,
	uuid string,
) (*types.LogResource, error) {
	logger := l.project.operationLogger("GetLogByUUID", slog.String("uuid", uuid))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLogByUuidWithResponse(ctx, l.project.projectName, uuid)

	return execute[types.LogResource](ctx, logger, "get log by uuid", response, err)
}

// ListLogsOption configures filter and pagination parameters for listing logs.
type ListLogsOption interface {
	applyListLogs(*api.GetLogsParams)
}

type listLogsOptionFunc func(*api.GetLogsParams)

func (f listLogsOptionFunc) applyListLogs(params *api.GetLogsParams) {
	f(params)
}

// List returns a paginated list of log entries matching the given filter options.
func (l *LogScope) List(
	ctx context.Context,
	options ...ListLogsOption,
) (*types.PageLogResource, error) {
	logger := l.project.operationLogger("ListLogs")

	params := &api.GetLogsParams{}
	for _, option := range options {
		option.applyListLogs(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLogsWithResponse(ctx, l.project.projectName, params)

	return execute[types.PageLogResource](ctx, logger, "list logs", response, err)
}

// Create creates a single log entry (JSON body, no file attachment).
func (l *LogScope) Create(
	ctx context.Context,
	request types.SaveLogRQ,
) (*types.EntryCreatedAsyncRS, error) {
	logger := l.project.operationLogger("CreateLogEntry")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.CreateLogEntry1WithResponse(ctx, l.project.projectName, request)

	return execute[types.EntryCreatedAsyncRS](ctx, logger, "create log entry", response, err)
}

// Delete removes a single log entry by its numeric ID.
func (l *LogScope) Delete(
	ctx context.Context,
	logID int64,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("DeleteLog", slog.Int64("logId", logID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.DeleteLogWithResponse(ctx, l.project.projectName, logID)

	return execute[types.OperationCompletionRS](ctx, logger, "delete log", response, err)
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

	response, err := l.project.client.raw.CreateLog1WithBodyWithResponse(
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

		err := json.Unmarshal(response.Body, &created)
		if err == nil {
			return &created, nil
		}
	}

	if response.JSONDefault != nil {
		return nil, fmt.Errorf("create log batch: %w", newAPIError(response.StatusCode(), response.JSONDefault))
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("create log batch: %w", newUnexpectedStatusError(code, response.Body))
	}

	if response.JSON200 != nil {
		return response.JSON200, nil
	}

	return nil, fmt.Errorf(
		"create log batch: API returned success status (%d) but no valid response body was parsed",
		code,
	)
}

// batchLogJSON is the JSON representation of a single log entry within the
// multipart batch payload.
type batchLogJSON struct {
	LaunchUUID string        `json:"launchUuid"`
	ItemUUID   string        `json:"itemUuid,omitempty"`
	Level      string        `json:"level"`
	Message    string        `json:"message"`
	Time       time.Time     `json:"time"`
	File       *batchLogFile `json:"file,omitempty"`
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
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		id := int32(itemID)
		params.FilterEqItem = &id
	})
}

// WithLogLaunchID filters logs by launch ID.
func WithLogLaunchID(launchID int64) ListLogsOption {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		id := int32(launchID)
		params.FilterEqLaunchId = &id
	})
}

// WithLogLevel filters logs by level (e.g. "ERROR", "INFO", "DEBUG").
func WithLogLevel(level string) ListLogsOption {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		params.FilterEqLevel = &level
	})
}

// WithLogPageNumber sets the page number (0-based).
func WithLogPageNumber(n int32) ListLogsOption {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		params.PagePage = &n
	})
}

// WithLogPageSize sets the number of results per page.
func WithLogPageSize(size int32) ListLogsOption {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		params.PageSize = &size
	})
}

// WithLogSort sets the sorting criteria (e.g. "logTime,desc").
func WithLogSort(fields ...string) ListLogsOption {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// Search finds log entries related to the given test item that match similar
// failure patterns. This is useful for triage workflows where an operator
// needs to find analogous failures across the project.
func (l *LogScope) Search(
	ctx context.Context,
	itemID int64,
	request types.SearchLogRq,
) ([]types.SearchLogRs, error) {
	logger := l.project.operationLogger("SearchLogs", slog.Int64("itemId", itemID))
	logger.InfoContext(ctx, "API request")

	if request.SearchMode == "" {
		request.SearchMode = types.CurrentLaunch
	}

	response, err := l.project.client.raw.SearchLogsWithResponse(ctx, l.project.projectName, itemID, request)

	result, err := execute[[]types.SearchLogRs](ctx, logger, "search logs", response, err)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return *result, nil
}

// GetLogPageNumberOption configures parameters for looking up a log's page number.
type GetLogPageNumberOption interface {
	applyGetLogPageNumber(*api.GetPageNumberParams)
}

type getLogPageNumberOptionFunc func(*api.GetPageNumberParams)

func (f getLogPageNumberOptionFunc) applyGetLogPageNumber(params *api.GetPageNumberParams) {
	f(params)
}

// WithLogPageItemID filters the page number lookup by test item ID.
func WithLogPageItemID(id int64) GetLogPageNumberOption {
	return getLogPageNumberOptionFunc(func(params *api.GetPageNumberParams) {
		narrowed := int32(id)
		params.FilterEqItem = &narrowed
	})
}

// WithLogPageNumberPageSize sets the page size for the page number lookup.
func WithLogPageNumberPageSize(size int32) GetLogPageNumberOption {
	return getLogPageNumberOptionFunc(func(params *api.GetPageNumberParams) {
		params.PageSize = &size
	})
}

// GetPageNumber looks up which page a specific log entry appears on.
func (l *LogScope) GetPageNumber(
	ctx context.Context,
	logID int64,
	options ...GetLogPageNumberOption,
) (map[string]any, error) {
	logger := l.project.operationLogger("GetLogPageNumber", slog.Int64("logId", logID))

	params := &api.GetPageNumberParams{}
	for _, option := range options {
		option.applyGetLogPageNumber(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetPageNumberWithResponse(
		ctx,
		l.project.projectName,
		logID,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get log page number: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf("get log page number: %w", newAPIError(response.StatusCode(), response.JSONDefault))
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("get log page number: %w", newUnexpectedStatusError(code, response.Body))
	}

	if response.JSON200 == nil || *response.JSON200 == nil {
		return map[string]any{}, nil
	}

	return *response.JSON200, nil
}

// GetLogLocationsOption configures parameters for retrieving log locations.
type GetLogLocationsOption interface {
	applyGetLogLocations(*api.GetErrorPageParams)
}

type getLogLocationsOptionFunc func(*api.GetErrorPageParams)

func (f getLogLocationsOptionFunc) applyGetLogLocations(params *api.GetErrorPageParams) {
	f(params)
}

// WithLogLocationsPageSize sets the number of results per page.
func WithLogLocationsPageSize(size int32) GetLogLocationsOption {
	return getLogLocationsOptionFunc(func(params *api.GetErrorPageParams) {
		params.PageSize = &size
	})
}

// WithLogLocationsPageNumber sets the page number (0-based).
func WithLogLocationsPageNumber(page int32) GetLogLocationsOption {
	return getLogLocationsOptionFunc(func(params *api.GetErrorPageParams) {
		params.PagePage = &page
	})
}

// GetLogLocations retrieves paginated log entries at error locations for a
// parent item.
func (l *LogScope) GetLogLocations(
	ctx context.Context,
	parentID int64,
	options ...GetLogLocationsOption,
) ([]types.PagedLogResource, error) {
	logger := l.project.operationLogger("GetLogLocations", slog.Int64("parentId", parentID))

	params := &api.GetErrorPageParams{
		Params: map[string]string{},
	}
	for _, option := range options {
		option.applyGetLogLocations(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetErrorPageWithResponse(
		ctx,
		l.project.projectName,
		parentID,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get log locations: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf("get log locations: %w", newAPIError(response.StatusCode(), response.JSONDefault))
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("get log locations: %w", newUnexpectedStatusError(code, response.Body))
	}

	if response.JSON200 == nil || *response.JSON200 == nil {
		return []types.PagedLogResource{}, nil
	}

	return *response.JSON200, nil
}

// SearchLogLocationsOption configures parameters for searching log locations.
type SearchLogLocationsOption interface {
	applySearchLogLocations(*api.GetLogsWithLocationBySearchParams)
}

type searchLogLocationsOptionFunc func(*api.GetLogsWithLocationBySearchParams)

func (f searchLogLocationsOptionFunc) applySearchLogLocations(params *api.GetLogsWithLocationBySearchParams) {
	f(params)
}

// WithSearchLogLocationsPageSize sets the number of results per page.
func WithSearchLogLocationsPageSize(size int32) SearchLogLocationsOption {
	return searchLogLocationsOptionFunc(func(params *api.GetLogsWithLocationBySearchParams) {
		params.PageSize = &size
	})
}

// WithSearchLogLocationsPageNumber sets the page number (0-based).
func WithSearchLogLocationsPageNumber(page int32) SearchLogLocationsOption {
	return searchLogLocationsOptionFunc(func(params *api.GetLogsWithLocationBySearchParams) {
		params.PagePage = &page
	})
}

// SearchLogLocations searches log locations with pagination for a parent item.
func (l *LogScope) SearchLogLocations(
	ctx context.Context,
	parentID int64,
	options ...SearchLogLocationsOption,
) (*types.PagePagedLogResource, error) {
	logger := l.project.operationLogger("SearchLogLocations", slog.Int64("parentId", parentID))

	params := &api.GetLogsWithLocationBySearchParams{
		Params: map[string]string{},
	}
	for _, option := range options {
		option.applySearchLogLocations(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLogsWithLocationBySearchWithResponse(ctx, l.project.projectName, parentID, params)

	return execute[types.PagePagedLogResource](ctx, logger, "search log locations", response, err)
}

// GetNestedLogsOption configures parameters for retrieving nested log entries.
type GetNestedLogsOption interface {
	applyGetNestedLogs(*api.GetNestedItemsParams)
}

type getNestedLogsOptionFunc func(*api.GetNestedItemsParams)

func (f getNestedLogsOptionFunc) applyGetNestedLogs(params *api.GetNestedItemsParams) {
	f(params)
}

// WithNestedLogsPageSize sets the number of results per page.
func WithNestedLogsPageSize(size int32) GetNestedLogsOption {
	return getNestedLogsOptionFunc(func(params *api.GetNestedItemsParams) {
		params.PageSize = &size
	})
}

// WithNestedLogsPageNumber sets the page number (0-based).
func WithNestedLogsPageNumber(page int32) GetNestedLogsOption {
	return getNestedLogsOptionFunc(func(params *api.GetNestedItemsParams) {
		params.PagePage = &page
	})
}

// GetNestedLogs retrieves nested log entries under a parent item.
func (l *LogScope) GetNestedLogs(
	ctx context.Context,
	parentID int64,
	options ...GetNestedLogsOption,
) (*types.PageObject, error) {
	logger := l.project.operationLogger("GetNestedLogs", slog.Int64("parentId", parentID))

	params := &api.GetNestedItemsParams{
		Params: map[string]string{},
	}
	for _, option := range options {
		option.applyGetNestedLogs(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetNestedItemsWithResponse(ctx, l.project.projectName, parentID, params)

	return execute[types.PageObject](ctx, logger, "get nested logs", response, err)
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

	if request.LogLevel == "" {
		request.LogLevel = "INFO"
	}

	response, err := l.project.client.raw.GetLogsUnderWithResponse(ctx, l.project.projectName, request)

	result, err := execute[map[string][]types.LogResource](ctx, logger, "get logs under", response, err)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string][]types.LogResource{}, nil
	}

	return *result, nil
}
