package launch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// LaunchScope provides read operations on launches within a project.
type LaunchScope struct {
	scope core.Scope
}

// Start creates a new launch in the project.
func (l *LaunchScope) Start(
	ctx context.Context,
	request types.StartLaunchRQ,
) (*types.StartLaunchRS, error) {
	logger := l.scope.OperationLogger("StartLaunch", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.StartLaunch1WithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	return core.Execute[types.StartLaunchRS](ctx, logger, "start launch", response, err)
}

// Finish marks an existing launch as finished using the launch UUID returned by
// Start. Numeric launch IDs are handled by GetByID, Update, Delete, and
// ForceFinish.
func (l *LaunchScope) Finish(
	ctx context.Context,
	launchUUID string,
	request types.FinishExecutionRQ,
) (*types.FinishLaunchRS, error) {
	logger := l.scope.OperationLogger("FinishLaunch", slog.String("launchUuid", launchUUID))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.FinishLaunch1WithResponse(
		ctx,
		l.scope.ProjectName,
		launchUUID,
		request,
	)

	return core.Execute[types.FinishLaunchRS](ctx, logger, "finish launch", response, err)
}

// StartAsync creates a new launch in the project using the asynchronous v2
// API endpoint. The server accepts the request immediately and processes it
// in the background.
func (l *LaunchScope) StartAsync(
	ctx context.Context,
	request types.StartLaunchRQ,
) (*types.StartLaunchRS, error) {
	logger := l.scope.OperationLogger("StartLaunchAsync", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.StartLaunchWithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	return core.Execute[types.StartLaunchRS](ctx, logger, "start launch async", response, err)
}

// FinishAsync marks an existing launch as finished using the asynchronous v2
// API endpoint. The server accepts the request immediately and processes it
// in the background.
func (l *LaunchScope) FinishAsync(
	ctx context.Context,
	launchUUID string,
	request types.FinishExecutionRQ,
) (*types.FinishLaunchRS, error) {
	logger := l.scope.OperationLogger("FinishLaunchAsync", slog.String("launchUuid", launchUUID))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.FinishLaunchWithResponse(
		ctx,
		l.scope.ProjectName,
		launchUUID,
		request,
	)

	return core.Execute[types.FinishLaunchRS](ctx, logger, "finish launch async", response, err)
}

// GetByUUID returns a single launch by its UUID. It sends the versioned v2
// Accept header (application/x.reportportal.launch.v2+json) so timestamp
// fields arrive as ISO-8601 strings instead of epoch milliseconds. The
// underlying API method is named GetLaunchByUuidOldTimestamp in the generated
// client because the spec path predates the v2 content negotiation; the "old
// timestamp" refers to the endpoint's default response shape, not the format
// we request.
func (l *LaunchScope) GetByUUID(
	ctx context.Context,
	uuid string,
) (*types.LaunchResource, error) {
	logger := l.scope.OperationLogger("GetLaunchByUUID", slog.String("uuid", uuid))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetLaunchByUuidOldTimestampWithResponse(
		ctx,
		l.scope.ProjectName,
		uuid,
		core.WithAcceptHeader(core.AcceptReportPortalLaunchV2),
	)
	if err != nil {
		return nil, fmt.Errorf("get launch by uuid: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := core.HandleV2Response(
		response,
		response.JSONDefault,
		response.ApplicationXReportportalLaunchV2JSON200,
		response.JSON200,
		response.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch by uuid: %w", err)
	}

	return result, nil
}

// GetByID returns a single launch by its numeric ID.
func (l *LaunchScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.LaunchResource, error) {
	logger := l.scope.OperationLogger("GetLaunchByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetLaunchWithResponse(ctx, l.scope.ProjectName, id)

	return core.Execute[types.LaunchResource](ctx, logger, "get launch by id", response, err)
}

// ListLaunchesOption configures the filter and pagination parameters for listing launches.
type ListLaunchesOption interface {
	applyListLaunches(params *api.GetProjectLaunchesParams)
}

type listLaunchesOptionFunc func(*api.GetProjectLaunchesParams)

func (f listLaunchesOptionFunc) applyListLaunches(params *api.GetProjectLaunchesParams) {
	f(params)
}

// List returns a paginated list of launches matching the given filter options.
func (l *LaunchScope) List(
	ctx context.Context,
	options ...ListLaunchesOption,
) (*types.PageLaunchResource, error) {
	logger := l.scope.OperationLogger("ListLaunches")

	params := &api.GetProjectLaunchesParams{}
	for _, option := range options {
		option.applyListLaunches(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetProjectLaunchesWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	return core.Execute[types.PageLaunchResource](ctx, logger, "list launches", response, err)
}

// WithLaunchName filters launches by name.
func WithLaunchName(name string) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqName = &name
	})
}

// WithLaunchStatus filters launches by status.
func WithLaunchStatus(status string) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqStatus = &status
	})
}

// WithLaunchPageNumber sets the page number (0-based).
func WithLaunchPageNumber(n int32) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.PagePage = &n
	})
}

// WithLaunchPageSize sets the number of results per page.
func WithLaunchPageSize(size int32) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.PageSize = &size
	})
}

// WithLaunchSort sets the sorting criteria (e.g. "startTime,desc").
func WithLaunchSort(fields ...string) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// WithLaunchUser filters launches by username.
func WithLaunchUser(username string) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqUser = &username
	})
}

// WithLaunchUserID filters launches by user ID. The parameter type is int32
// because the upstream 5.15 spec defines userId as a 32-bit integer.
func WithLaunchUserID(userID int32) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqUserId = &userID
	})
}

// WithLaunchMode filters launches by mode (e.g. "DEFAULT", "DEBUG").
func WithLaunchMode(mode string) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqMode = &mode
	})
}

// WithLaunchAttribute filters launches by a composite attribute key:value pair.
func WithLaunchAttribute(key, value string) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		composite := key + ":" + value
		if params.FilterHasCompositeAttribute == nil {
			params.FilterHasCompositeAttribute = &composite
		} else {
			joined := *params.FilterHasCompositeAttribute + "," + composite
			params.FilterHasCompositeAttribute = &joined
		}
	})
}

// WithLaunchAttributeKey filters launches by attribute key only.
func WithLaunchAttributeKey(key string) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqAttributeKey = &key
	})
}

// WithLaunchDescription filters launches by description.
func WithLaunchDescription(description string) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqDescription = &description
	})
}

// WithLaunchNumber filters launches by launch number.
func WithLaunchNumber(number int32) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqNumber = &number
	})
}

// WithLaunchHasRetries filters launches by retry status.
func WithLaunchHasRetries(hasRetries bool) listLaunchesOptionFunc {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqHasRetries = &hasRetries
	})
}

// GetLatest returns the latest launches (uses /launch/latest endpoint).
// Unlike GetByUUID and Merge, this method does not send the v2 Accept header
// because the /launch/latest endpoint does not support content negotiation for
// the v2 response shape. Timestamps arrive in the server's default format.
func (l *LaunchScope) GetLatest(
	ctx context.Context,
	options ...ListLaunchesOption,
) (*types.PageLaunchResource, error) {
	logger := l.scope.OperationLogger("GetLatestLaunches")

	baseParams := &api.GetProjectLaunchesParams{}
	for _, option := range options {
		option.applyListLaunches(baseParams)
	}

	converted := api.GetLatestLaunchesParams(*baseParams)
	params := &converted

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetLatestLaunchesWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	return core.Execute[types.PageLaunchResource](ctx, logger, "get latest launches", response, err)
}

// GetDebug returns debug-mode launches (uses /launch/mode endpoint).
func (l *LaunchScope) GetDebug(
	ctx context.Context,
	options ...ListLaunchesOption,
) (*types.PageLaunchResource, error) {
	logger := l.scope.OperationLogger("GetDebugLaunches")

	baseParams := &api.GetProjectLaunchesParams{}
	for _, option := range options {
		option.applyListLaunches(baseParams)
	}

	converted := api.GetDebugLaunchesParams(*baseParams)
	params := &converted

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetDebugLaunchesWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	return core.Execute[types.PageLaunchResource](ctx, logger, "get debug launches", response, err)
}

// Update modifies a single launch's attributes, description, or mode.
func (l *LaunchScope) Update(
	ctx context.Context,
	launchID int64,
	request types.UpdateLaunchRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.scope.OperationLogger("UpdateLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.UpdateLaunchWithResponse(
		ctx,
		l.scope.ProjectName,
		launchID,
		request,
	)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "update launch", response, err)
}

// Delete removes a single launch by its numeric ID.
func (l *LaunchScope) Delete(
	ctx context.Context,
	launchID int64,
) (*types.OperationCompletionRS, error) {
	logger := l.scope.OperationLogger("DeleteLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.DeleteLaunchWithResponse(
		ctx,
		l.scope.ProjectName,
		launchID,
	)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "delete launch", response, err)
}

// DeleteBulk removes multiple launches by their numeric IDs.
func (l *LaunchScope) DeleteBulk(
	ctx context.Context,
	ids []int64,
) (*types.DeleteBulkRS, error) {
	logger := l.scope.OperationLogger("DeleteLaunchesBulk")
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.DeleteLaunchesWithResponse(
		ctx,
		l.scope.ProjectName,
		&api.DeleteLaunchesParams{Ids: ids},
	)

	return core.Execute[types.DeleteBulkRS](ctx, logger, "delete launches bulk", response, err)
}

// ForceFinish stops a stuck launch that cannot be finished normally.
func (l *LaunchScope) ForceFinish(
	ctx context.Context,
	launchID int64,
	request types.FinishExecutionRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.scope.OperationLogger("ForceFinishLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.ForceFinishLaunchWithResponse(
		ctx,
		l.scope.ProjectName,
		launchID,
		request,
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"force finish launch",
		response,
		err,
	)
}

// Merge combines multiple launches into a single launch. If MergeType is not
// set, it defaults to BASIC.
func (l *LaunchScope) Merge(
	ctx context.Context,
	request types.MergeLaunchesRQ,
) (*types.LaunchResource, error) {
	logger := l.scope.OperationLogger("MergeLaunches")
	logger.InfoContext(ctx, "API request")

	if request.MergeType == "" {
		request.MergeType = types.MergeLaunchesRQMergeTypeBASIC
	}

	response, err := l.scope.Client.MergeLaunchesOldUuid1WithResponse(
		ctx,
		l.scope.ProjectName,
		request,
		core.WithAcceptHeader(core.AcceptReportPortalLaunchV2),
	)
	if err != nil {
		return nil, fmt.Errorf("merge launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := core.HandleV2Response(
		response,
		response.JSONDefault,
		response.ApplicationXReportportalLaunchV2JSON200,
		response.JSON200,
		response.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("merge launches: %w", err)
	}

	return result, nil
}

// MergeAsync combines multiple launches into a single launch using the
// asynchronous v2 API endpoint. If MergeType is not set, it defaults to BASIC.
func (l *LaunchScope) MergeAsync(
	ctx context.Context,
	request types.MergeLaunchesRQ,
) (*types.LaunchResource, error) {
	logger := l.scope.OperationLogger("MergeLaunchesAsync")
	logger.InfoContext(ctx, "API request")

	if request.MergeType == "" {
		request.MergeType = types.MergeLaunchesRQMergeTypeBASIC
	}

	response, err := l.scope.Client.MergeLaunchesOldUuidWithResponse(
		ctx,
		l.scope.ProjectName,
		request,
		core.WithAcceptHeader(core.AcceptReportPortalLaunchV2),
	)
	if err != nil {
		return nil, fmt.Errorf("merge launches async: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := core.HandleV2Response(
		response,
		response.JSONDefault,
		response.ApplicationXReportportalLaunchV2JSON200,
		response.JSON200,
		response.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("merge launches async: %w", err)
	}

	return result, nil
}

// Analyze triggers auto-analysis on a launch.
func (l *LaunchScope) Analyze(
	ctx context.Context,
	request types.AnalyzeLaunchRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.scope.OperationLogger("AnalyzeLaunch")
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.StartLaunchAnalyzerWithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "analyze launch", response, err)
}

// BulkUpdate modifies attributes and/or description for multiple launches at once.
func (l *LaunchScope) BulkUpdate(
	ctx context.Context,
	request types.BulkInfoUpdateRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.scope.OperationLogger("BulkUpdateLaunches")
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.BulkUpdateWithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	return core.Execute[types.OperationCompletionRS](
		ctx,
		logger,
		"bulk update launches",
		response,
		err,
	)
}

// BulkForceFinish stops multiple stuck launches that cannot be finished normally.
func (l *LaunchScope) BulkForceFinish(
	ctx context.Context,
	request types.BulkRQLongFinishExecutionRQ,
) ([]types.OperationCompletionRS, error) {
	logger := l.scope.OperationLogger("BulkForceFinishLaunches")
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.BulkForceFinishWithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	results, err := core.Execute[[]types.OperationCompletionRS](
		ctx,
		logger,
		"bulk force finish launches",
		response,
		err,
	)
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
	logger := l.scope.OperationLogger("GetLaunchNames", slog.String("filter", filter))

	params := &api.GetAllLaunchNamesParams{}
	if filter != "" {
		params.FilterCntName = &filter
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetAllLaunchNamesWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	names, err := core.Execute[[]string](ctx, logger, "get launch names", response, err)
	if err != nil {
		return nil, err
	}

	if names == nil {
		return []string{}, nil
	}

	return *names, nil
}

// GetAttributeKeys returns unique launch attribute keys in the project. keySubstring is sent as
// filter.cnt.attributeKey (substring match for autocomplete). Use an empty string when no
// substring filter is needed.
func (l *LaunchScope) GetAttributeKeys(
	ctx context.Context,
	keySubstring string,
) ([]string, error) {
	logger := l.scope.OperationLogger(
		"GetLaunchAttributeKeys",
		slog.String("filter", keySubstring),
	)
	logger.InfoContext(ctx, "API request")

	params := &api.GetAttributeKeysParams{
		FilterCntAttributeKey: keySubstring,
	}

	response, err := l.scope.Client.GetAttributeKeysWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	keys, err := core.Execute[[]string](ctx, logger, "get launch attribute keys", response, err)
	if err != nil {
		return nil, err
	}

	if keys == nil {
		return []string{}, nil
	}

	return *keys, nil
}

// GetAttributeValues returns unique launch attribute values in the project. valueSubstring is
// sent as filter.cnt.attributeValue. When attributeKey is non-nil, results are limited to values
// for that key (filter.eq.attributeKey).
func (l *LaunchScope) GetAttributeValues(
	ctx context.Context,
	valueSubstring string,
	attributeKey *string,
) ([]string, error) {
	logger := l.scope.OperationLogger(
		"GetLaunchAttributeValues",
		slog.String("valueFilter", valueSubstring),
	)
	if attributeKey != nil {
		logger = logger.With(slog.String("attributeKey", *attributeKey))
	}

	logger.InfoContext(ctx, "API request")

	params := &api.GetAttributeValuesParams{
		FilterCntAttributeValue: valueSubstring,
		FilterEqAttributeKey:    attributeKey,
	}

	response, err := l.scope.Client.GetAttributeValuesWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	values, err := core.Execute[[]string](ctx, logger, "get launch attribute values", response, err)
	if err != nil {
		return nil, err
	}

	if values == nil {
		return []string{}, nil
	}

	return *values, nil
}

// Compare returns statistics for the given launch IDs, keyed by metric name.
// Each metric maps to a slice of chart data points across the compared launches.
func (l *LaunchScope) Compare(
	ctx context.Context,
	ids []int64,
) (map[string][]types.ChartStatisticsContent, error) {
	logger := l.scope.OperationLogger("CompareLaunches")
	logger.InfoContext(ctx, "API request")

	params := &api.CompareLaunchesParams{Ids: ids}

	response, err := l.scope.Client.CompareLaunchesWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	result, err := core.Execute[map[string][]types.ChartStatisticsContent](
		ctx,
		logger,
		"compare launches",
		response,
		err,
	)
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
	_ = data
	logger := l.scope.OperationLogger("ImportLaunch", slog.String("filename", filename))
	logger.InfoContext(ctx, "API request")

	return nil, errors.New(
		"import launch: not supported by generated API client (no import operation in OpenAPI spec)",
	)
}

// ListClustersOption configures the pagination parameters for listing ML clusters.
type ListClustersOption interface {
	applyListClusters(params *api.GetClustersParams)
}

type listClustersOptionFunc func(*api.GetClustersParams)

func (f listClustersOptionFunc) applyListClusters(params *api.GetClustersParams) {
	f(params)
}

// WithClusterPageNumber sets the page number (0-based) for cluster listing.
func WithClusterPageNumber(n int32) listClustersOptionFunc {
	return listClustersOptionFunc(func(params *api.GetClustersParams) {
		params.PagePage = &n
	})
}

// WithClusterPageSize sets the number of results per page for cluster listing.
func WithClusterPageSize(size int32) listClustersOptionFunc {
	return listClustersOptionFunc(func(params *api.GetClustersParams) {
		params.PageSize = &size
	})
}

// WithClusterSort sets the sorting criteria for cluster listing (e.g. "index,asc").
func WithClusterSort(fields ...string) listClustersOptionFunc {
	return listClustersOptionFunc(func(params *api.GetClustersParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// GetClusters returns ML clusters for the given launch.
func (l *LaunchScope) GetClusters(
	ctx context.Context,
	launchID string,
	options ...ListClustersOption,
) (*types.PageClusterInfoResource, error) {
	logger := l.scope.OperationLogger("GetClusters", slog.String("launchId", launchID))

	params := &api.GetClustersParams{}
	for _, option := range options {
		option.applyListClusters(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetClustersWithResponse(
		ctx,
		l.scope.ProjectName,
		launchID,
		params,
	)

	return core.Execute[types.PageClusterInfoResource](ctx, logger, "get clusters", response, err)
}

// CreateClusters triggers ML cluster creation for a launch.
func (l *LaunchScope) CreateClusters(
	ctx context.Context,
	request types.CreateClustersRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.scope.OperationLogger("CreateClusters", slog.Int64("launchId", request.LaunchId))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.CreateClustersWithResponse(
		ctx,
		l.scope.ProjectName,
		request,
	)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "create clusters", response, err)
}

// GetOwners returns unique launch owner usernames matching the filter.
func (l *LaunchScope) GetOwners(
	ctx context.Context,
	userFilter string,
) ([]string, error) {
	logger := l.scope.OperationLogger("GetLaunchOwners", slog.String("filter", userFilter))
	logger.InfoContext(ctx, "API request")

	params := &api.GetAllOwnersParams{
		FilterCntUser: userFilter,
	}

	response, err := l.scope.Client.GetAllOwnersWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	owners, err := core.Execute[[]string](ctx, logger, "get launch owners", response, err)
	if err != nil {
		return nil, err
	}

	if owners == nil {
		return []string{}, nil
	}

	return *owners, nil
}

// GetStatuses returns the statuses of launches by their IDs. The returned map
// keys are launch IDs (as strings) and values are status strings.
func (l *LaunchScope) GetStatuses(
	ctx context.Context,
	ids []int64,
) (map[string]string, error) {
	logger := l.scope.OperationLogger("GetLaunchStatuses")
	logger.InfoContext(ctx, "API request")

	params := &api.GetStatusesParams{Ids: ids}

	response, err := l.scope.Client.GetStatusesWithResponse(
		ctx,
		l.scope.ProjectName,
		params,
	)

	statuses, err := core.Execute[map[string]string](
		ctx,
		logger,
		"get launch statuses",
		response,
		err,
	)
	if err != nil {
		return nil, err
	}

	if statuses == nil {
		return map[string]string{}, nil
	}

	return *statuses, nil
}

// GetReport retrieves the raw launch report for the given launch ID. The
// response is typically an HTML document.
func (l *LaunchScope) GetReport(
	ctx context.Context,
	launchID int64,
) ([]byte, error) {
	logger := l.scope.OperationLogger("GetLaunchReport", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.scope.Client.GetLaunchReportWithResponse(
		ctx,
		l.scope.ProjectName,
		launchID,
		&api.GetLaunchReportParams{},
	)
	if err != nil {
		return nil, fmt.Errorf("get launch report: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf(
			"get launch report: %w",
			core.NewAPIError(response.StatusCode(), response.JSONDefault),
		)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf(
			"get launch report: %w",
			core.NewUnexpectedStatusError(code, response.Body),
		)
	}

	return response.Body, nil
}

// NewLaunchScope constructs a LaunchScope for wiring from the facade.
func NewLaunchScope(s core.Scope) *LaunchScope {
	return &LaunchScope{scope: s}
}
