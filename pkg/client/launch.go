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
