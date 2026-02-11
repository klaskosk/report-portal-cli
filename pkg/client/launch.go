package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// LaunchScope provides read operations on launches within a project.
type LaunchScope struct {
	project *ProjectScope
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
	id string,
) (*types.LaunchResource, error) {
	logger := l.project.operationLogger("GetLaunchByID", slog.String("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := l.project.client.raw.GetLaunchUsingGETWithResponse(
		ctx,
		l.project.projectName,
		id,
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
			*params.FilterEqCompositeAttribute = append(*params.FilterEqCompositeAttribute, composite)
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

// convertToLatestParams converts GetProjectLaunchesUsingGETParams to GetLatestLaunchesUsingGETParams.
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
		FilterEqStatus:             src.FilterEqStatus,
		FilterEqUser:               src.FilterEqUser,
		FilterEqUserId:             src.FilterEqUserId,
		FilterEqUuid:               src.FilterEqUuid,
		PagePage:                   src.PagePage,
		PageSize:                   src.PageSize,
		PageSort:                   src.PageSort,
	}
}

// convertToDebugParams converts GetProjectLaunchesUsingGETParams to GetDebugLaunchesUsingGETParams.
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
		FilterEqStatus:             src.FilterEqStatus,
		FilterEqUser:               src.FilterEqUser,
		FilterEqUserId:             src.FilterEqUserId,
		FilterEqUuid:               src.FilterEqUuid,
		PagePage:                   src.PagePage,
		PageSize:                   src.PageSize,
		PageSort:                   src.PageSort,
	}
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
