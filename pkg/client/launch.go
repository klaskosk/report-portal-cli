package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

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

	response, err := l.project.client.raw.StartLaunch1WithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("start launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
	if err != nil {
		return nil, fmt.Errorf("start launch: %w", err)
	}

	return result, nil
}

// Finish marks an existing launch as finished using the launch UUID returned by
// Start. Numeric launch IDs are handled by GetByID, Update, Delete, and
// ForceFinish.
func (l *LaunchScope) Finish(
	ctx context.Context,
	launchUUID string,
	request types.FinishExecutionRQ,
) (*types.FinishLaunchRS, error) {
	logger := l.project.operationLogger("FinishLaunch", slog.String("launchUuid", launchUUID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.FinishLaunch1WithResponse(
		ctx,
		l.project.projectName,
		launchUUID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("finish launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("finish launch: %w", err)
	}

	return result, nil
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
	logger := l.project.operationLogger("GetLaunchByUUID", slog.String("uuid", uuid))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLaunchByUuidOldTimestampWithResponse(
		ctx,
		l.project.projectName,
		uuid,
		withAcceptHeader(acceptReportPortalLaunchV2),
	)
	if err != nil {
		return nil, fmt.Errorf("get launch by uuid: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleV2Response(
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
	logger := l.project.operationLogger("GetLaunchByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLaunchWithResponse(
		ctx,
		l.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch by id: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get launch by id: %w", err)
	}

	return result, nil
}

// ListLaunchesOption configures the filter and pagination parameters for listing launches.
type ListLaunchesOption interface {
	applyListLaunches(*api.GetProjectLaunchesParams)
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
	logger := l.project.operationLogger("ListLaunches")

	params := &api.GetProjectLaunchesParams{}
	for _, option := range options {
		option.applyListLaunches(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetProjectLaunchesWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("list launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list launches: %w", err)
	}

	return result, nil
}

// WithLaunchName filters launches by name.
func WithLaunchName(name string) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqName = &name
	})
}

// WithLaunchStatus filters launches by status.
func WithLaunchStatus(status string) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqStatus = &status
	})
}

// WithLaunchPageNumber sets the page number (0-based).
func WithLaunchPageNumber(n int32) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.PagePage = &n
	})
}

// WithLaunchPageSize sets the number of results per page.
func WithLaunchPageSize(size int32) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.PageSize = &size
	})
}

// WithLaunchSort sets the sorting criteria (e.g. "startTime,desc").
func WithLaunchSort(fields ...string) ListLaunchesOption {
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
func WithLaunchUser(username string) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqUser = &username
	})
}

// WithLaunchUserID filters launches by user ID. The parameter type is int32
// because the upstream 5.15 spec defines userId as a 32-bit integer.
func WithLaunchUserID(userID int32) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqUserId = &userID
	})
}

// WithLaunchMode filters launches by mode (e.g. "DEFAULT", "DEBUG").
func WithLaunchMode(mode string) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqMode = &mode
	})
}

// WithLaunchAttribute filters launches by a composite attribute key:value pair.
func WithLaunchAttribute(key, value string) ListLaunchesOption {
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
func WithLaunchAttributeKey(key string) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqAttributeKey = &key
	})
}

// WithLaunchDescription filters launches by description.
func WithLaunchDescription(description string) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqDescription = &description
	})
}

// WithLaunchNumber filters launches by launch number.
func WithLaunchNumber(number int32) ListLaunchesOption {
	return listLaunchesOptionFunc(func(params *api.GetProjectLaunchesParams) {
		params.FilterEqNumber = &number
	})
}

// WithLaunchHasRetries filters launches by retry status.
func WithLaunchHasRetries(hasRetries bool) ListLaunchesOption {
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
	logger := l.project.operationLogger("GetLatestLaunches")

	baseParams := &api.GetProjectLaunchesParams{}
	for _, option := range options {
		option.applyListLaunches(baseParams)
	}

	converted := api.GetLatestLaunchesParams(*baseParams)
	params := &converted

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLatestLaunchesWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get latest launches: %w", err)
	}

	return result, nil
}

// GetDebug returns debug-mode launches (uses /launch/mode endpoint).
func (l *LaunchScope) GetDebug(
	ctx context.Context,
	options ...ListLaunchesOption,
) (*types.PageLaunchResource, error) {
	logger := l.project.operationLogger("GetDebugLaunches")

	baseParams := &api.GetProjectLaunchesParams{}
	for _, option := range options {
		option.applyListLaunches(baseParams)
	}

	converted := api.GetDebugLaunchesParams(*baseParams)
	params := &converted

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetDebugLaunchesWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get debug launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get debug launches: %w", err)
	}

	return result, nil
}

// Update modifies a single launch's attributes, description, or mode.
func (l *LaunchScope) Update(
	ctx context.Context,
	launchID int64,
	request types.UpdateLaunchRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("UpdateLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.UpdateLaunchWithResponse(
		ctx,
		l.project.projectName,
		launchID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update launch: %w", err)
	}

	return result, nil
}

// Delete removes a single launch by its numeric ID.
func (l *LaunchScope) Delete(
	ctx context.Context,
	launchID int64,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("DeleteLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.DeleteLaunchWithResponse(
		ctx,
		l.project.projectName,
		launchID,
	)
	if err != nil {
		return nil, fmt.Errorf("delete launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete launch: %w", err)
	}

	return result, nil
}

// DeleteBulk removes multiple launches by their numeric IDs.
func (l *LaunchScope) DeleteBulk(
	ctx context.Context,
	ids []int64,
) (*types.DeleteBulkRS, error) {
	logger := l.project.operationLogger("DeleteLaunchesBulk")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.DeleteLaunchesWithResponse(
		ctx,
		l.project.projectName,
		&api.DeleteLaunchesParams{Ids: ids},
	)
	if err != nil {
		return nil, fmt.Errorf("delete launches bulk: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete launches bulk: %w", err)
	}

	return result, nil
}

// ForceFinish stops a stuck launch that cannot be finished normally.
func (l *LaunchScope) ForceFinish(
	ctx context.Context,
	launchID int64,
	request types.FinishExecutionRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("ForceFinishLaunch", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.ForceFinishLaunchWithResponse(
		ctx,
		l.project.projectName,
		launchID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("force finish launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("force finish launch: %w", err)
	}

	return result, nil
}

// Merge combines multiple launches into a single launch. If MergeType is not
// set, it defaults to BASIC.
func (l *LaunchScope) Merge(
	ctx context.Context,
	request types.MergeLaunchesRQ,
) (*types.LaunchResource, error) {
	logger := l.project.operationLogger("MergeLaunches")
	logger.InfoContext(ctx, "API request")

	if request.MergeType == "" {
		request.MergeType = types.MergeLaunchesRQMergeTypeBASIC
	}

	response, err := l.project.client.raw.MergeLaunchesOldUuid1WithResponse(
		ctx,
		l.project.projectName,
		request,
		withAcceptHeader(acceptReportPortalLaunchV2),
	)
	if err != nil {
		return nil, fmt.Errorf("merge launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleV2Response(
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

// Analyze triggers auto-analysis on a launch.
func (l *LaunchScope) Analyze(
	ctx context.Context,
	request types.AnalyzeLaunchRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("AnalyzeLaunch")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.StartLaunchAnalyzerWithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("analyze launch: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("analyze launch: %w", err)
	}

	return result, nil
}

// BulkUpdate modifies attributes and/or description for multiple launches at once.
func (l *LaunchScope) BulkUpdate(
	ctx context.Context,
	request types.BulkInfoUpdateRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("BulkUpdateLaunches")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.BulkUpdateWithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("bulk update launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("bulk update launches: %w", err)
	}

	return result, nil
}

// BulkForceFinish stops multiple stuck launches that cannot be finished normally.
func (l *LaunchScope) BulkForceFinish(
	ctx context.Context,
	request types.BulkRQLongFinishExecutionRQ,
) ([]types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("BulkForceFinishLaunches")
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.BulkForceFinishWithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("bulk force finish launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	results, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("bulk force finish launches: %w", err)
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

	params := &api.GetAllLaunchNamesParams{}
	if filter != "" {
		params.FilterCntName = &filter
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetAllLaunchNamesWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch names: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	names, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get launch names: %w", err)
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
	logger := l.project.operationLogger(
		"GetLaunchAttributeKeys",
		slog.String("filter", keySubstring),
	)
	logger.InfoContext(ctx, "API request")

	params := &api.GetAttributeKeysParams{
		FilterCntAttributeKey: keySubstring,
	}

	response, err := l.project.client.raw.GetAttributeKeysWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch attribute keys: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	keys, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get launch attribute keys: %w", err)
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
	logger := l.project.operationLogger(
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

	response, err := l.project.client.raw.GetAttributeValuesWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch attribute values: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	values, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get launch attribute values: %w", err)
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
	logger := l.project.operationLogger("CompareLaunches")
	logger.InfoContext(ctx, "API request")

	params := &api.CompareLaunchesParams{Ids: ids}

	response, err := l.project.client.raw.CompareLaunchesWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("compare launches: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("compare launches: %w", err)
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
	logger := l.project.operationLogger("ImportLaunch", slog.String("filename", filename))
	logger.InfoContext(ctx, "API request")

	return nil, errors.New(
		"import launch: not supported by generated API client (no import operation in OpenAPI spec)",
	)
}

// ListClustersOption configures the pagination parameters for listing ML clusters.
type ListClustersOption interface {
	applyListClusters(*api.GetClustersParams)
}

type listClustersOptionFunc func(*api.GetClustersParams)

func (f listClustersOptionFunc) applyListClusters(params *api.GetClustersParams) {
	f(params)
}

// WithClusterPageNumber sets the page number (0-based) for cluster listing.
func WithClusterPageNumber(n int32) ListClustersOption {
	return listClustersOptionFunc(func(params *api.GetClustersParams) {
		params.PagePage = &n
	})
}

// WithClusterPageSize sets the number of results per page for cluster listing.
func WithClusterPageSize(size int32) ListClustersOption {
	return listClustersOptionFunc(func(params *api.GetClustersParams) {
		params.PageSize = &size
	})
}

// WithClusterSort sets the sorting criteria for cluster listing (e.g. "index,asc").
func WithClusterSort(fields ...string) ListClustersOption {
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
	logger := l.project.operationLogger("GetClusters", slog.String("launchId", launchID))

	params := &api.GetClustersParams{}
	for _, option := range options {
		option.applyListClusters(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetClustersWithResponse(
		ctx,
		l.project.projectName,
		launchID,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get clusters: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get clusters: %w", err)
	}

	return result, nil
}

// CreateClusters triggers ML cluster creation for a launch.
func (l *LaunchScope) CreateClusters(
	ctx context.Context,
	request types.CreateClustersRQ,
) (*types.OperationCompletionRS, error) {
	logger := l.project.operationLogger("CreateClusters", slog.Int64("launchId", request.LaunchId))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.CreateClustersWithResponse(
		ctx,
		l.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create clusters: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create clusters: %w", err)
	}

	return result, nil
}

// GetOwners returns unique launch owner usernames matching the filter.
func (l *LaunchScope) GetOwners(
	ctx context.Context,
	userFilter string,
) ([]string, error) {
	logger := l.project.operationLogger("GetLaunchOwners", slog.String("filter", userFilter))
	logger.InfoContext(ctx, "API request")

	params := &api.GetAllOwnersParams{
		FilterCntUser: userFilter,
	}

	response, err := l.project.client.raw.GetAllOwnersWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch owners: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	owners, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get launch owners: %w", err)
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
	logger := l.project.operationLogger("GetLaunchStatuses")
	logger.InfoContext(ctx, "API request")

	params := &api.GetStatusesParams{Ids: ids}

	response, err := l.project.client.raw.GetStatusesWithResponse(
		ctx,
		l.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get launch statuses: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	statuses, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get launch statuses: %w", err)
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
	logger := l.project.operationLogger("GetLaunchReport", slog.Int64("launchId", launchID))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLaunchReportWithResponse(
		ctx,
		l.project.projectName,
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
			newAPIError(response.StatusCode(), response.JSONDefault),
		)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf(
			"get launch report: %w",
			newUnexpectedStatusError(code, response.Body),
		)
	}

	return response.Body, nil
}
