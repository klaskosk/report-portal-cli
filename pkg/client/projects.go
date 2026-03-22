package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ProjectsScope provides operations on projects at the instance level.
type ProjectsScope struct {
	client *Client
}

// Get returns the full details of a project by name.
func (p *ProjectsScope) Get(
	ctx context.Context,
	projectName string,
) (*types.ProjectResource, error) {
	logger := p.client.logger.With(
		slog.String("operation", "GetProject"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetProjectUsingGETWithResponse(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	return result, nil
}

// ListNames returns all project names in the Report Portal instance. The underlying
// endpoint (GetAllProjectNames) may return all projects regardless of the caller's
// access, depending on deployment configuration.
func (p *ProjectsScope) ListNames(ctx context.Context) ([]string, error) {
	logger := p.client.logger.With(slog.String("operation", "ListProjectNames"))
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetAllProjectNamesUsingGETWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project names: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(
		response,
		response.JSONDefault,
		response.JSON200,
	)
	if err != nil {
		return nil, fmt.Errorf("list project names: %w", err)
	}

	return []string(*result), nil
}

// ListProjectsOption configures the filter and pagination parameters for listing projects.
type ListProjectsOption func(*api.GetAllProjectsInfoUsingGETParams)

// WithProjectName filters the project list by exact project name.
func WithProjectName(name string) ListProjectsOption {
	return func(params *api.GetAllProjectsInfoUsingGETParams) {
		params.FilterEqName = &name
	}
}

// WithProjectType filters the project list by project type (e.g. "INTERNAL").
func WithProjectType(projectType string) ListProjectsOption {
	return func(params *api.GetAllProjectsInfoUsingGETParams) {
		params.FilterEqType = &projectType
	}
}

// WithProjectPageNumber sets the page number (0-based) for project listing.
func WithProjectPageNumber(n int32) ListProjectsOption {
	return func(params *api.GetAllProjectsInfoUsingGETParams) {
		params.PagePage = &n
	}
}

// WithProjectPageSize sets the number of results per page for project listing.
func WithProjectPageSize(size int32) ListProjectsOption {
	return func(params *api.GetAllProjectsInfoUsingGETParams) {
		params.PageSize = &size
	}
}

// WithProjectSort sets the sorting criteria for project listing (e.g. "name,asc").
func WithProjectSort(fields ...string) ListProjectsOption {
	return func(params *api.GetAllProjectsInfoUsingGETParams) {
		params.PageSort = &fields
	}
}

// List returns a paginated list of project info summaries.
func (p *ProjectsScope) List(
	ctx context.Context,
	options ...ListProjectsOption,
) (*types.IterableProjectInfoResource, error) {
	logger := p.client.logger.With(slog.String("operation", "ListProjects"))

	params := &api.GetAllProjectsInfoUsingGETParams{}
	for _, option := range options {
		option(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetAllProjectsInfoUsingGETWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	return result, nil
}

// SearchNames searches for project names matching the given term.
func (p *ProjectsScope) SearchNames(
	ctx context.Context,
	term string,
) ([]string, error) {
	logger := p.client.logger.With(
		slog.String("operation", "SearchProjectNames"),
		slog.String("term", term),
	)
	logger.InfoContext(ctx, "API request")

	params := &api.SearchProjectNamesUsingGETParams{Term: term}

	response, err := p.client.raw.SearchProjectNamesUsingGETWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search project names: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("search project names: %w", err)
	}

	return []string(*result), nil
}

// GetInfo returns aggregated project statistics for the given project.
func (p *ProjectsScope) GetInfo(
	ctx context.Context,
	projectName string,
	options ...GetProjectInfoOption,
) (*types.ProjectInfoResource, error) {
	logger := p.client.logger.With(
		slog.String("operation", "GetProjectInfo"),
		slog.String("projectName", projectName),
	)

	params := &api.GetProjectInfoUsingGETParams{}
	for _, option := range options {
		option(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetProjectInfoUsingGETWithResponse(ctx, projectName, params)
	if err != nil {
		return nil, fmt.Errorf("get project info: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get project info: %w", err)
	}

	return result, nil
}

// GetProjectInfoOption configures the parameters for getting project info.
type GetProjectInfoOption func(*api.GetProjectInfoUsingGETParams)

// WithProjectInfoInterval sets the interval for project info statistics (e.g. "1M", "3M").
func WithProjectInfoInterval(interval string) GetProjectInfoOption {
	return func(params *api.GetProjectInfoUsingGETParams) {
		params.Interval = &interval
	}
}

// Create creates a new project in the Report Portal instance.
func (p *ProjectsScope) Create(
	ctx context.Context,
	request types.CreateProjectRQ,
) (*types.EntryCreatedRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "CreateProject"),
		slog.String("projectName", request.ProjectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.CreateProjectUsingPOSTWithResponse(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	return result, nil
}

// Delete deletes a project by its numeric ID.
func (p *ProjectsScope) Delete(
	ctx context.Context,
	projectID int64,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "DeleteProject"),
		slog.Int64("projectId", projectID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.DeleteProjectUsingDELETEWithResponse(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}

	return result, nil
}

// Update modifies the configuration of an existing project.
func (p *ProjectsScope) Update(
	ctx context.Context,
	projectName string,
	request types.UpdateProjectRQ,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "UpdateProject"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.UpdateProjectUsingPUTWithResponse(ctx, projectName, request)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	return result, nil
}

// AssignUsers assigns users to a project with specified roles.
func (p *ProjectsScope) AssignUsers(
	ctx context.Context,
	projectName string,
	request types.AssignUsersRQ,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "AssignProjectUsers"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.AssignProjectUsersUsingPUTWithResponse(ctx, projectName, request)
	if err != nil {
		return nil, fmt.Errorf("assign project users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("assign project users: %w", err)
	}

	return result, nil
}

// UnassignUsers removes users from a project.
func (p *ProjectsScope) UnassignUsers(
	ctx context.Context,
	projectName string,
	request types.UnassignUsersRQ,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "UnassignProjectUsers"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.UnassignProjectUsersUsingPUTWithResponse(
		ctx,
		projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("unassign project users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("unassign project users: %w", err)
	}

	return result, nil
}

// ListUsers returns a paginated list of users assigned to a project.
func (p *ProjectsScope) ListUsers(
	ctx context.Context,
	projectName string,
	options ...ListProjectUsersOption,
) (*types.IterableUserResource, error) {
	logger := p.client.logger.With(
		slog.String("operation", "ListProjectUsers"),
		slog.String("projectName", projectName),
	)

	params := &api.GetProjectUsersUsingGET1Params{}
	for _, option := range options {
		option(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetProjectUsersUsingGET1WithResponse(ctx, projectName, params)
	if err != nil {
		return nil, fmt.Errorf("list project users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("list project users: %w", err)
	}

	return result, nil
}

// ListProjectUsersOption configures the filter and pagination parameters for listing project users.
type ListProjectUsersOption func(*api.GetProjectUsersUsingGET1Params)

// WithUserPageNumber sets the page number (0-based) for project users listing.
func WithUserPageNumber(n int32) ListProjectUsersOption {
	return func(params *api.GetProjectUsersUsingGET1Params) {
		params.PagePage = &n
	}
}

// WithUserPageSize sets the number of results per page for project users listing.
func WithUserPageSize(size int32) ListProjectUsersOption {
	return func(params *api.GetProjectUsersUsingGET1Params) {
		params.PageSize = &size
	}
}

// WithUserSort sets the sorting criteria for project users listing.
func WithUserSort(fields ...string) ListProjectUsersOption {
	return func(params *api.GetProjectUsersUsingGET1Params) {
		params.PageSort = &fields
	}
}
