package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"strconv"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// LaunchScope provides read operations on launches within a project.
type LaunchScope struct {
	project *ProjectScope
}

// Start creates a new launch in the project.
func (l *LaunchScope) Start(
	ctx context.Context,
	request types.StartLaunchRQ,
) (*types.StartLaunchRS, error) {
	logger := l.project.operationLogger("StartLaunch", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.StartLaunchUsingPOST1WithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("start launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "start launch", response.JSONDefault, response.JSON201)
}

// Finish marks an existing launch as finished.
func (l *LaunchScope) Finish(
	ctx context.Context,
	launchID string,
	request types.FinishExecutionRQ,
) (*types.FinishLaunchRS, error) {
	logger := l.project.operationLogger("FinishLaunch", slog.String("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.FinishLaunchUsingPUT1WithResponse(
		ctx,
		l.project.projectName,
		launchID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("finish launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "finish launch", response.JSONDefault, response.JSON200)
}

// GetByUUID returns a single launch by its UUID.
func (l *LaunchScope) GetByUUID(
	ctx context.Context,
	uuid string,
) (*types.LaunchResource, error) {
	logger := l.project.operationLogger("GetLaunchByUUID", slog.String("uuid", uuid))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLaunchByUuidUsingGETWithResponse(
		ctx,
		l.project.projectName,
		uuid,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch by uuid: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get launch by uuid", response.JSONDefault, response.JSON200)
}

// GetByID returns a single launch by its numeric ID.
func (l *LaunchScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.LaunchResource, error) {
	logger := l.project.operationLogger("GetLaunchByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLaunchUsingGETWithResponse(
		ctx,
		l.project.projectName,
		strconv.FormatInt(id, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("get launch by id: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get launch by id", response.JSONDefault, response.JSON200)
}

// ListLaunchesOption configures the filter and pagination parameters for listing launches.
type ListLaunchesOption func(*api.GetProjectLaunchesUsingGETParams)

// List returns a paginated list of launches matching the given filter options.
func (l *LaunchScope) List(
	ctx context.Context,
	options ...ListLaunchesOption,
) (*types.IterableLaunchResource, error) {
	logger := l.project.operationLogger("ListLaunches")

	params := &api.GetProjectLaunchesUsingGETParams{}
	for _, option := range options {
		option(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetProjectLaunchesUsingGETWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("list launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "list launches", response.JSONDefault, response.JSON200)
}

// WithName filters launches by name.
func WithName(name string) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqName = &name
	}
}

// WithStatus filters launches by status.
func WithStatus(status string) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqStatus = &status
	}
}

// WithPageNumber sets the page number (0-based).
func WithPageNumber(n int32) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.PagePage = &n
	}
}

// WithPageSize sets the number of results per page.
func WithPageSize(size int32) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.PageSize = &size
	}
}

// WithSort sets the sorting criteria (e.g. "startTime,desc").
func WithSort(fields ...string) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.PageSort = &fields
	}
}

// WithUser filters launches by username.
func WithUser(username string) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqUser = &username
	}
}

// WithUserID filters launches by user ID.
func WithUserID(userID int64) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqUserId = &userID
	}
}

// WithMode filters launches by mode (e.g. "DEFAULT", "DEBUG").
func WithMode(mode string) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqMode = &mode
	}
}

// WithAttribute filters launches by a composite attribute key:value pair.
func WithAttribute(key, value string) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		composite := key + ":" + value
		if params.FilterEqCompositeAttribute == nil {
			params.FilterEqCompositeAttribute = &[]string{composite}
		} else {
			*params.FilterEqCompositeAttribute = append(
				*params.FilterEqCompositeAttribute,
				composite,
			)
		}
	}
}

// WithAttributeKey filters launches by attribute key only.
func WithAttributeKey(key string) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqAttributeKey = &key
	}
}

// WithDescription filters launches by description.
func WithDescription(description string) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqDescription = &description
	}
}

// WithNumber filters launches by launch number.
func WithNumber(number int32) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqNumber = &number
	}
}

// WithHasRetries filters launches by retry status.
func WithHasRetries(hasRetries bool) ListLaunchesOption {
	return func(params *api.GetProjectLaunchesUsingGETParams) {
		params.FilterEqHasRetries = &hasRetries
	}
}

// GetLatest returns the latest launches (uses /launch/latest endpoint).
func (l *LaunchScope) GetLatest(
	ctx context.Context,
	options ...ListLaunchesOption,
) (*types.IterableLaunchResource, error) {
	logger := l.project.operationLogger("GetLatestLaunches")

	// Create params using the same type structure as List
	baseParams := &api.GetProjectLaunchesUsingGETParams{}
	for _, option := range options {
		option(baseParams)
	}

	// Convert to GetLatestLaunchesUsingGETParams
	params := convertToLatestParams(baseParams)

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLatestLaunchesUsingGETWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get latest launches", response.JSONDefault, response.JSON200)
}

// GetDebug returns debug-mode launches (uses /launch/mode endpoint).
func (l *LaunchScope) GetDebug(
	ctx context.Context,
	options ...ListLaunchesOption,
) (*types.IterableLaunchResource, error) {
	logger := l.project.operationLogger("GetDebugLaunches")

	// Create params using the same type structure as List
	baseParams := &api.GetProjectLaunchesUsingGETParams{}
	for _, option := range options {
		option(baseParams)
	}

	// Convert to GetDebugLaunchesUsingGETParams
	params := convertToDebugParams(baseParams)

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetDebugLaunchesUsingGETWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get debug launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get debug launches", response.JSONDefault, response.JSON200)
}

// convertToLatestParams converts GetProjectLaunchesUsingGETParams to GetLatestLaunchesUsingGETParams. NOTE: This
// function must be kept in sync with convertToDebugParams -- they map identical field sets to structurally identical
// (but distinct) generated param types.
func convertToLatestParams(
	src *api.GetProjectLaunchesUsingGETParams,
) *api.GetLatestLaunchesUsingGETParams {
	return &api.GetLatestLaunchesUsingGETParams{
		FilterEqAttributeKey:       src.FilterEqAttributeKey,
		FilterEqAttributeValue:     src.FilterEqAttributeValue,
		FilterEqCompositeAttribute: src.FilterEqCompositeAttribute,
		FilterEqDescription:        src.FilterEqDescription,
		FilterEqEndTime:            src.FilterEqEndTime,
		FilterEqHasRetries:         src.FilterEqHasRetries,
		FilterEqId:                 src.FilterEqId,
		FilterEqLastModified:       src.FilterEqLastModified,
		FilterEqMode:               src.FilterEqMode,
		FilterEqName:               src.FilterEqName,
		FilterEqNumber:             src.FilterEqNumber,
		FilterEqProjectId:          src.FilterEqProjectId,
		FilterEqStartTime:          src.FilterEqStartTime,
		FilterEqStatisticsDefectsAutomationBugAb001: src.FilterEqStatisticsDefectsAutomationBugAb001,
		FilterEqStatisticsDefectsNoDefectNd001:      src.FilterEqStatisticsDefectsNoDefectNd001,
		FilterEqStatisticsDefectsProductBugPb001:    src.FilterEqStatisticsDefectsProductBugPb001,
		FilterEqStatisticsDefectsSystemIssueSi001:   src.FilterEqStatisticsDefectsSystemIssueSi001,
		FilterEqStatisticsDefectsToInvestigateTi001: src.FilterEqStatisticsDefectsToInvestigateTi001,
		FilterEqStatisticsExecutionsFailed:          src.FilterEqStatisticsExecutionsFailed,
		FilterEqStatisticsExecutionsPassed:          src.FilterEqStatisticsExecutionsPassed,
		FilterEqStatisticsExecutionsSkipped:         src.FilterEqStatisticsExecutionsSkipped,
		FilterEqStatisticsExecutionsTotal:           src.FilterEqStatisticsExecutionsTotal,
		FilterEqStatus:                              src.FilterEqStatus,
		FilterEqUser:                                src.FilterEqUser,
		FilterEqUserId:                              src.FilterEqUserId,
		FilterEqUuid:                                src.FilterEqUuid,
		PagePage:                                    src.PagePage,
		PageSize:                                    src.PageSize,
		PageSort:                                    src.PageSort,
	}
}

// convertToDebugParams converts GetProjectLaunchesUsingGETParams to GetDebugLaunchesUsingGETParams. NOTE: This function
// must be kept in sync with convertToLatestParams -- they map identical field sets to structurally identical (but
// distinct) generated param types.
func convertToDebugParams(
	src *api.GetProjectLaunchesUsingGETParams,
) *api.GetDebugLaunchesUsingGETParams {
	return &api.GetDebugLaunchesUsingGETParams{
		FilterEqAttributeKey:       src.FilterEqAttributeKey,
		FilterEqAttributeValue:     src.FilterEqAttributeValue,
		FilterEqCompositeAttribute: src.FilterEqCompositeAttribute,
		FilterEqDescription:        src.FilterEqDescription,
		FilterEqEndTime:            src.FilterEqEndTime,
		FilterEqHasRetries:         src.FilterEqHasRetries,
		FilterEqId:                 src.FilterEqId,
		FilterEqLastModified:       src.FilterEqLastModified,
		FilterEqMode:               src.FilterEqMode,
		FilterEqName:               src.FilterEqName,
		FilterEqNumber:             src.FilterEqNumber,
		FilterEqProjectId:          src.FilterEqProjectId,
		FilterEqStartTime:          src.FilterEqStartTime,
		FilterEqStatisticsDefectsAutomationBugAb001: src.FilterEqStatisticsDefectsAutomationBugAb001,
		FilterEqStatisticsDefectsNoDefectNd001:      src.FilterEqStatisticsDefectsNoDefectNd001,
		FilterEqStatisticsDefectsProductBugPb001:    src.FilterEqStatisticsDefectsProductBugPb001,
		FilterEqStatisticsDefectsSystemIssueSi001:   src.FilterEqStatisticsDefectsSystemIssueSi001,
		FilterEqStatisticsDefectsToInvestigateTi001: src.FilterEqStatisticsDefectsToInvestigateTi001,
		FilterEqStatisticsExecutionsFailed:          src.FilterEqStatisticsExecutionsFailed,
		FilterEqStatisticsExecutionsPassed:          src.FilterEqStatisticsExecutionsPassed,
		FilterEqStatisticsExecutionsSkipped:         src.FilterEqStatisticsExecutionsSkipped,
		FilterEqStatisticsExecutionsTotal:           src.FilterEqStatisticsExecutionsTotal,
		FilterEqStatus:                              src.FilterEqStatus,
		FilterEqUser:                                src.FilterEqUser,
		FilterEqUserId:                              src.FilterEqUserId,
		FilterEqUuid:                                src.FilterEqUuid,
		PagePage:                                    src.PagePage,
		PageSize:                                    src.PageSize,
		PageSort:                                    src.PageSort,
	}
}

// Update modifies a single launch's attributes, description, or mode.
func (l *LaunchScope) Update(
	ctx context.Context,
	launchID int64,
	request types.UpdateLaunchRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("UpdateLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.UpdateLaunchUsingPUTWithResponse(
		ctx,
		l.project.projectName,
		launchID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "update launch", response.JSONDefault, response.JSON200)
}

// Delete removes a single launch by its numeric ID.
func (l *LaunchScope) Delete(
	ctx context.Context,
	launchID int64,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("DeleteLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.DeleteLaunchUsingDELETEWithResponse(
		ctx,
		l.project.projectName,
		launchID,
	)
	if err != nil {
		return nil, fmt.Errorf("delete launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "delete launch", response.JSONDefault, response.JSON200)
}

// DeleteBulk removes multiple launches by their numeric IDs.
func (l *LaunchScope) DeleteBulk(
	ctx context.Context,
	ids []int64,
) (*types.DeleteBulkRS, error) {
	logger := l.project.operationLogger("DeleteLaunchesBulk")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.DeleteLaunchesUsingDELETEWithResponse(
		ctx,
		l.project.projectName,
		types.DeleteBulkRQ{Ids: &ids},
	)
	if err != nil {
		return nil, fmt.Errorf("delete launches bulk: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "delete launches bulk", response.JSONDefault, response.JSON200)
}

// ForceFinish stops a stuck launch that cannot be finished normally.
func (l *LaunchScope) ForceFinish(
	ctx context.Context,
	launchID int64,
	request types.FinishExecutionRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("ForceFinishLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.ForceFinishLaunchUsingPUTWithResponse(
		ctx,
		l.project.projectName,
		launchID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("force finish launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "force finish launch", response.JSONDefault, response.JSON200)
}

// Merge combines multiple launches into a single launch. If MergeType is not
// set, it defaults to BASIC.
func (l *LaunchScope) Merge(
	ctx context.Context,
	request types.MergeLaunchesRQ,
) (*types.LaunchResource, error) {
	logger := l.project.operationLogger("MergeLaunches")
	logger.InfoContext(ctx, "API request")

	if request.MergeType == nil {
		basic := types.MergeLaunchesRQMergeTypeBASIC
		request.MergeType = &basic
	}

	response, err := l.project.client.raw.MergeLaunchesUsingPOST1WithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("merge launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "merge launches", response.JSONDefault, response.JSON200)
}

// Analyze triggers auto-analysis on a launch.
func (l *LaunchScope) Analyze(
	ctx context.Context,
	request types.AnalyzeLaunchRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("AnalyzeLaunch")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.StartLaunchAnalyzerUsingPOSTWithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("analyze launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "analyze launch", response.JSONDefault, response.JSON200)
}

// BulkUpdate modifies attributes and/or description for multiple launches at once.
func (l *LaunchScope) BulkUpdate(
	ctx context.Context,
	request types.BulkInfoUpdateRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("BulkUpdateLaunches")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.BulkUpdateUsingPUTWithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("bulk update launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "bulk update launches", response.JSONDefault, response.JSON200)
}

// BulkForceFinish stops multiple stuck launches that cannot be finished normally.
func (l *LaunchScope) BulkForceFinish(
	ctx context.Context,
	request types.BulkRQLongFinishExecutionRQ,
) ([]types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("BulkForceFinishLaunches")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.BulkForceFinishUsingPUTWithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("bulk force finish launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	results, err := handleResponse(response, "bulk force finish launches", response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, err
	}

	if results == nil {
		return []types.OperationCompletionRS{}, nil
	}

	return *results, nil
}

// GetNames returns unique launch names matching the filter (for autocomplete).
func (l *LaunchScope) GetNames(
	ctx context.Context,
	filter string,
) ([]string, error) {
	logger := l.project.operationLogger("GetLaunchNames", slog.String("filter", filter))

	params := &api.GetAllLaunchNamesUsingGETParams{}
	if filter != "" {
		params.FilterCntName = &filter
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetAllLaunchNamesUsingGETWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch names: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	names, err := handleResponse(
		response,
		"get launch names",
		response.JSONDefault,
		response.JSON200,
	)
	if err != nil {
		return nil, err
	}

	if names == nil {
		return []string{}, nil
	}

	return *names, nil
}

// Compare returns statistics for the given launch IDs, keyed by metric name.
// Each metric maps to a slice of chart data points across the compared launches.
func (l *LaunchScope) Compare(
	ctx context.Context,
	ids []int64,
) (map[string][]types.ChartStatisticsContent, error) {
	logger := l.project.operationLogger("CompareLaunches")
	logger.InfoContext(ctx, "API request")

	params := &api.CompareLaunchesUsingGETParams{Ids: ids}

	response, err := l.project.client.raw.CompareLaunchesUsingGETWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("compare launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, "compare launches", response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return map[string][]types.ChartStatisticsContent{}, nil
	}

	return *result, nil
}

// Import uploads a launch archive (zip) to the project. The filename is used
// as the form field name in the multipart upload.
func (l *LaunchScope) Import(
	ctx context.Context,
	filename string,
	data io.Reader,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("ImportLaunch", slog.String("filename", filename))
	logger.InfoContext(ctx, "API request")

	body, contentType, err := buildImportMultipart(filename, data)
	if err != nil {
		return nil, fmt.Errorf("import launch: %w", err)
	}

	response, err := l.project.client.raw.ImportLaunchUsingPOSTWithBodyWithResponse(
		ctx,
		l.project.projectName,
		contentType,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("import launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "import launch", response.JSONDefault, response.JSON200)
}

// buildImportMultipart assembles the multipart body for a launch import request.
func buildImportMultipart(filename string, data io.Reader) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("creating file part: %w", err)
	}

	if _, err := io.Copy(part, data); err != nil {
		return nil, "", fmt.Errorf("writing file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("closing multipart writer: %w", err)
	}

	return &buf, writer.FormDataContentType(), nil
}

// GetReport retrieves the raw launch report for the given launch ID. The
// response is typically an HTML document.
func (l *LaunchScope) GetReport(
	ctx context.Context,
	launchID int64,
) ([]byte, error) {
	logger := l.project.operationLogger("GetLaunchReport", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLaunchReportUsingGETWithResponse(
		ctx,
		l.project.projectName,
		launchID,
		&api.GetLaunchReportUsingGETParams{},
	)
	if err != nil {
		return nil, fmt.Errorf("get launch report: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, newAPIError("get launch report", response.StatusCode(), response.JSONDefault)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, newUnexpectedStatusError("get launch report", code)
	}

	return response.Body, nil
}
