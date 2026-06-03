package log

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

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
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
	scope core.Scope
}

// GetByID returns a single log entry by its numeric ID.
func (l *LogScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.LogResource, error) {
	logger := l.scope.OperationLogger("GetLogByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetLogWithResponse(ctx, l.scope.ProjectName, id)

	return core.Execute[types.LogResource](ctx, logger, "get log by id", response, err)
}

// GetByUUID returns a single log entry by its UUID.
func (l *LogScope) GetByUUID(
	ctx context.Context,
	uuid string,
) (*types.LogResource, error) {
	logger := l.scope.OperationLogger("GetLogByUUID", slog.String("uuid", uuid))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetLogByUuidWithResponse(ctx, l.scope.ProjectName, uuid)

	return core.Execute[types.LogResource](ctx, logger, "get log by uuid", response, err)
}

// ListLogsOption configures filter and pagination parameters for listing logs.
type ListLogsOption interface {
	applyListLogs(params *api.GetLogsParams)
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
	logger := l.scope.OperationLogger("ListLogs")

	params := &api.GetLogsParams{}
	for _, option := range options {
		option.applyListLogs(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetLogsWithResponse(ctx, l.scope.ProjectName, params)

	return core.Execute[types.PageLogResource](ctx, logger, "list logs", response, err)
}

// Create creates a single log entry (JSON body, no file attachment).
func (l *LogScope) Create(
	ctx context.Context,
	request types.SaveLogRQ,
) (*types.EntryCreatedAsyncRS, error) {
	logger := l.scope.OperationLogger("CreateLogEntry")
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.CreateLogEntry1WithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	return core.Execute[types.EntryCreatedAsyncRS](ctx, logger, "create log entry", response, err)
}

// CreateAsync creates a single log entry using the asynchronous v2 API
// endpoint. The server accepts the request immediately and processes it in the
// background.
func (l *LogScope) CreateAsync(
	ctx context.Context,
	request types.SaveLogRQ,
) (*types.EntryCreatedAsyncRS, error) {
	logger := l.scope.OperationLogger("CreateLogEntryAsync")
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.CreateLogEntryWithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	return core.Execute[types.EntryCreatedAsyncRS](
		ctx,
		logger,
		"create log entry async",
		response,
		err,
	)
}

// decodeBatchSaveOperatingBody interprets batch log create responses shared by sync and async endpoints.
func decodeBatchSaveOperatingBody(
	opName string,
	statusCode int,
	rawBody []byte,
	jsonDefault *types.ErrorRS,
	json200 *types.BatchSaveOperatingRS,
) (*types.BatchSaveOperatingRS, error) {
	if statusCode == http.StatusCreated {
		var created types.BatchSaveOperatingRS

		err := json.Unmarshal(rawBody, &created)
		if err == nil {
			return &created, nil
		}
	}

	if jsonDefault != nil {
		return nil, fmt.Errorf("%s: %w", opName, core.NewAPIError(statusCode, jsonDefault))
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("%s: %w", opName, core.NewUnexpectedStatusError(statusCode, rawBody))
	}

	if json200 != nil {
		return json200, nil
	}

	return nil, fmt.Errorf(
		"%s: API returned success status (%d) but no valid response body was parsed",
		opName,
		statusCode,
	)
}

// CreateBatchAsync creates multiple log entries in a single multipart request
// using the asynchronous v2 API endpoint, optionally including file
// attachments.
func (l *LogScope) CreateBatchAsync(
	ctx context.Context,
	entries ...BatchLogEntry,
) (*types.BatchSaveOperatingRS, error) {
	logger := l.scope.OperationLogger("CreateLogBatchAsync")
	logger.InfoContext(ctx, "API request")

	body, contentType, err := buildBatchLogMultipart(entries)
	if err != nil {
		return nil, fmt.Errorf("create log batch async: %w", err)
	}

	response, err := l.scope.Client.CreateLogWithBodyWithResponse(
		ctx,
		l.scope.ProjectName,
		contentType,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("create log batch async: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return decodeBatchSaveOperatingBody(
		"create log batch async",
		response.StatusCode(),
		response.Body,
		response.JSONDefault,
		response.JSON200,
	)
}

// Delete removes a single log entry by its numeric ID.
func (l *LogScope) Delete(
	ctx context.Context,
	logID int64,
) (*types.OperationCompletionRS, error) {
	logger := l.scope.OperationLogger("DeleteLog", slog.Int64("logId", logID))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.DeleteLogWithResponse(ctx, l.scope.ProjectName, logID)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "delete log", response, err)
}

// CreateBatch creates multiple log entries in a single multipart request,
// optionally including file attachments.
func (l *LogScope) CreateBatch(
	ctx context.Context,
	entries ...BatchLogEntry,
) (*types.BatchSaveOperatingRS, error) {
	logger := l.scope.OperationLogger("CreateLogBatch")
	logger.InfoContext(ctx, "API request")

	body, contentType, err := buildBatchLogMultipart(entries)
	if err != nil {
		return nil, fmt.Errorf("create log batch: %w", err)
	}

	response, err := l.scope.Client.CreateLog1WithBodyWithResponse(
		ctx,
		l.scope.ProjectName,
		contentType,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("create log batch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return decodeBatchSaveOperatingBody(
		"create log batch",
		response.StatusCode(),
		response.Body,
		response.JSONDefault,
		response.JSON200,
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

// appendBatchLogFileParts writes optional attachment parts for batch log uploads.
func appendBatchLogFileParts(writer *multipart.Writer, entries []BatchLogEntry) error {
	for _, entry := range entries {
		if entry.AttachmentName == "" {
			continue
		}

		filePart, err := writer.CreateFormFile("file", entry.AttachmentName)
		if err != nil {
			return fmt.Errorf("creating file part: %w", err)
		}

		_, werr := filePart.Write(entry.AttachmentData)
		if werr != nil {
			return fmt.Errorf("writing file part: %w", werr)
		}
	}

	return nil
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

	_, werr := jsonPart.Write(jsonData)
	if werr != nil {
		return nil, "", fmt.Errorf("writing json part: %w", werr)
	}

	appendErr := appendBatchLogFileParts(writer, entries)
	if appendErr != nil {
		return nil, "", appendErr
	}

	closeErr := writer.Close()
	if closeErr != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", closeErr)
	}

	return &buf, writer.FormDataContentType(), nil
}

// WithLogItemID filters logs by test item ID.
func WithLogItemID(itemID int64) listLogsOptionFunc {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		if id := core.Int32PtrFromInt64(itemID); id != nil {
			params.FilterEqItem = id
		}
	})
}

// WithLogLaunchID filters logs by launch ID.
func WithLogLaunchID(launchID int64) listLogsOptionFunc {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		if id := core.Int32PtrFromInt64(launchID); id != nil {
			params.FilterEqLaunchId = id
		}
	})
}

// WithLogLevel filters logs by level (e.g. "ERROR", "INFO", "DEBUG").
func WithLogLevel(level string) listLogsOptionFunc {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		params.FilterEqLevel = &level
	})
}

// WithLogPageNumber sets the page number (0-based).
func WithLogPageNumber(n int32) listLogsOptionFunc {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		params.PagePage = &n
	})
}

// WithLogPageSize sets the number of results per page.
func WithLogPageSize(size int32) listLogsOptionFunc {
	return listLogsOptionFunc(func(params *api.GetLogsParams) {
		params.PageSize = &size
	})
}

// WithLogSort sets the sorting criteria (e.g. "logTime,desc").
func WithLogSort(fields ...string) listLogsOptionFunc {
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
	logger := l.scope.OperationLogger("SearchLogs", slog.Int64("itemId", itemID))
	logger.InfoContext(ctx, "API request")

	if request.SearchMode == "" {
		request.SearchMode = types.CurrentLaunch
	}

	response, err := l.scope.Client.SearchLogsWithResponse(
		ctx,
		l.scope.ProjectName,
		itemID,
		request,
	)

	result, err := core.Execute[[]types.SearchLogRs](ctx, logger, "search logs", response, err)
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
	applyGetLogPageNumber(params *api.GetPageNumberParams)
}

type getLogPageNumberOptionFunc func(*api.GetPageNumberParams)

func (f getLogPageNumberOptionFunc) applyGetLogPageNumber(params *api.GetPageNumberParams) {
	f(params)
}

// WithLogPageItemID filters the page number lookup by test item ID.
func WithLogPageItemID(id int64) getLogPageNumberOptionFunc {
	return getLogPageNumberOptionFunc(func(params *api.GetPageNumberParams) {
		if narrowed := core.Int32PtrFromInt64(id); narrowed != nil {
			params.FilterEqItem = narrowed
		}
	})
}

// WithLogPageNumberPageSize sets the page size for the page number lookup.
func WithLogPageNumberPageSize(size int32) getLogPageNumberOptionFunc {
	return getLogPageNumberOptionFunc(func(params *api.GetPageNumberParams) {
		params.PageSize = &size
	})
}

// WithLogPageNumberSort sets the sorting criteria for the page number lookup (e.g. "logTime,desc").
func WithLogPageNumberSort(sort string) getLogPageNumberOptionFunc {
	return getLogPageNumberOptionFunc(func(params *api.GetPageNumberParams) {
		if sort == "" {
			params.PageSort = nil

			return
		}

		s := sort
		params.PageSort = &s
	})
}

// GetPageNumber looks up which page a specific log entry appears on.
func (l *LogScope) GetPageNumber(
	ctx context.Context,
	logID int64,
	options ...GetLogPageNumberOption,
) (map[string]any, error) {
	logger := l.scope.OperationLogger("GetLogPageNumber", slog.Int64("logId", logID))

	params := &api.GetPageNumberParams{}
	for _, option := range options {
		option.applyGetLogPageNumber(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetPageNumberWithResponse(
		ctx,
		l.scope.ProjectName,
		logID,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get log page number: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf(
			"get log page number: %w",
			core.NewAPIError(response.StatusCode(), response.JSONDefault),
		)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf(
			"get log page number: %w",
			core.NewUnexpectedStatusError(code, response.Body),
		)
	}

	if response.JSON200 == nil || *response.JSON200 == nil {
		return map[string]any{}, nil
	}

	return *response.JSON200, nil
}

// GetLogLocationsOption configures parameters for retrieving log locations.
type GetLogLocationsOption interface {
	applyGetLogLocations(params *api.GetErrorPageParams)
}

type getLogLocationsOptionFunc func(*api.GetErrorPageParams)

func (f getLogLocationsOptionFunc) applyGetLogLocations(params *api.GetErrorPageParams) {
	f(params)
}

// WithLogLocationsPageSize sets the number of results per page.
func WithLogLocationsPageSize(size int32) getLogLocationsOptionFunc {
	return getLogLocationsOptionFunc(func(params *api.GetErrorPageParams) {
		params.PageSize = &size
	})
}

// WithLogLocationsPageNumber sets the page number (0-based).
func WithLogLocationsPageNumber(page int32) getLogLocationsOptionFunc {
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
	logger := l.scope.OperationLogger("GetLogLocations", slog.Int64("parentId", parentID))

	params := &api.GetErrorPageParams{
		Params: map[string]string{},
	}
	for _, option := range options {
		option.applyGetLogLocations(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetErrorPageWithResponse(
		ctx,
		l.scope.ProjectName,
		parentID,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get log locations: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf(
			"get log locations: %w",
			core.NewAPIError(response.StatusCode(), response.JSONDefault),
		)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf(
			"get log locations: %w",
			core.NewUnexpectedStatusError(code, response.Body),
		)
	}

	if response.JSON200 == nil || *response.JSON200 == nil {
		return []types.PagedLogResource{}, nil
	}

	return *response.JSON200, nil
}

// SearchLogLocationsOption configures parameters for searching log locations.
type SearchLogLocationsOption interface {
	applySearchLogLocations(params *api.GetLogsWithLocationBySearchParams)
}

type searchLogLocationsOptionFunc func(*api.GetLogsWithLocationBySearchParams)

func (f searchLogLocationsOptionFunc) applySearchLogLocations(
	params *api.GetLogsWithLocationBySearchParams,
) {
	f(params)
}

// WithSearchLogLocationsPageSize sets the number of results per page.
func WithSearchLogLocationsPageSize(size int32) searchLogLocationsOptionFunc {
	return searchLogLocationsOptionFunc(func(params *api.GetLogsWithLocationBySearchParams) {
		params.PageSize = &size
	})
}

// WithSearchLogLocationsPageNumber sets the page number (0-based).
func WithSearchLogLocationsPageNumber(page int32) searchLogLocationsOptionFunc {
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
	logger := l.scope.OperationLogger("SearchLogLocations", slog.Int64("parentId", parentID))

	params := &api.GetLogsWithLocationBySearchParams{
		Params: map[string]string{},
	}
	for _, option := range options {
		option.applySearchLogLocations(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetLogsWithLocationBySearchWithResponse(
		ctx,
		l.scope.ProjectName,
		parentID,
		params,
	)

	return core.Execute[types.PagePagedLogResource](
		ctx,
		logger,
		"search log locations",
		response,
		err,
	)
}

// GetNestedLogsOption configures parameters for retrieving nested log entries.
type GetNestedLogsOption interface {
	applyGetNestedLogs(params *api.GetNestedItemsParams)
}

type getNestedLogsOptionFunc func(*api.GetNestedItemsParams)

func (f getNestedLogsOptionFunc) applyGetNestedLogs(params *api.GetNestedItemsParams) {
	f(params)
}

// WithNestedLogsPageSize sets the number of results per page.
func WithNestedLogsPageSize(size int32) getNestedLogsOptionFunc {
	return getNestedLogsOptionFunc(func(params *api.GetNestedItemsParams) {
		params.PageSize = &size
	})
}

// WithNestedLogsPageNumber sets the page number (0-based).
func WithNestedLogsPageNumber(page int32) getNestedLogsOptionFunc {
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
	logger := l.scope.OperationLogger("GetNestedLogs", slog.Int64("parentId", parentID))

	params := &api.GetNestedItemsParams{
		Params: map[string]string{},
	}
	for _, option := range options {
		option.applyGetNestedLogs(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetNestedItemsWithResponse(
		ctx,
		l.scope.ProjectName,
		parentID,
		params,
	)

	return core.Execute[types.PageObject](ctx, logger, "get nested logs", response, err)
}

// GetLogsUnder retrieves log entries grouped by item ID for the specified
// items and optional log level filter. The returned map is keyed by item ID
// (as a string) with each value containing that item's log entries.
func (l *LogScope) GetLogsUnder(
	ctx context.Context,
	request types.GetLogsUnderRq,
) (map[string][]types.LogResource, error) {
	logger := l.scope.OperationLogger("GetLogsUnder")
	logger.InfoContext(ctx, "API request")

	if request.LogLevel == "" {
		request.LogLevel = "INFO"
	}

	response, err := l.scope.Client.GetLogsUnderWithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	result, err := core.Execute[map[string][]types.LogResource](
		ctx,
		logger,
		"get logs under",
		response,
		err,
	)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string][]types.LogResource{}, nil
	}

	return *result, nil
}

// NewLogScope constructs a LogScope for wiring from the facade.
func NewLogScope(s core.Scope) *LogScope {
	return &LogScope{scope: s}
}
