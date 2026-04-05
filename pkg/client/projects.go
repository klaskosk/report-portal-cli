package client

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"

	openapi_types "github.com/oapi-codegen/runtime/types"
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

	response, err := p.client.raw.GetProjectWithResponse(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
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

	response, err := p.client.raw.GetAllProjectNamesWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project names: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(
		response,
		response.JSONDefault,
		response.JSON200,
		response.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("list project names: %w", err)
	}

	return []string(*result), nil
}

// ListProjectsOption configures the filter and pagination parameters for listing projects.
type ListProjectsOption interface {
	applyListProjects(*api.GetAllProjectsInfoParams)
}

type listProjectsOptionFunc func(*api.GetAllProjectsInfoParams)

func (f listProjectsOptionFunc) applyListProjects(params *api.GetAllProjectsInfoParams) {
	f(params)
}

// WithProjectName filters the project list by exact project name.
func WithProjectName(name string) ListProjectsOption {
	return listProjectsOptionFunc(func(params *api.GetAllProjectsInfoParams) {
		params.FilterEqName = &name
	})
}

// WithProjectType filters the project list by project type (e.g. "INTERNAL").
func WithProjectType(projectType string) ListProjectsOption {
	return listProjectsOptionFunc(func(params *api.GetAllProjectsInfoParams) {
		params.FilterEqType = &projectType
	})
}

// WithProjectPageNumber sets the page number (0-based) for project listing.
func WithProjectPageNumber(n int32) ListProjectsOption {
	return listProjectsOptionFunc(func(params *api.GetAllProjectsInfoParams) {
		params.PagePage = &n
	})
}

// WithProjectPageSize sets the number of results per page for project listing.
func WithProjectPageSize(size int32) ListProjectsOption {
	return listProjectsOptionFunc(func(params *api.GetAllProjectsInfoParams) {
		params.PageSize = &size
	})
}

// WithProjectSort sets the sorting criteria for project listing (e.g. "name,asc").
func WithProjectSort(fields ...string) ListProjectsOption {
	return listProjectsOptionFunc(func(params *api.GetAllProjectsInfoParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// List returns a paginated list of project info summaries.
func (p *ProjectsScope) List(
	ctx context.Context,
	options ...ListProjectsOption,
) (*types.PageProjectInfoResource, error) {
	logger := p.client.logger.With(slog.String("operation", "ListProjects"))

	params := &api.GetAllProjectsInfoParams{}
	for _, option := range options {
		option.applyListProjects(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetAllProjectsInfoWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
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

	params := &api.SearchProjectNamesParams{Term: term}

	response, err := p.client.raw.SearchProjectNamesWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search project names: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
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

	params := &api.GetProjectInfoParams{}
	for _, option := range options {
		option.applyGetProjectInfo(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetProjectInfoWithResponse(ctx, projectName, params)
	if err != nil {
		return nil, fmt.Errorf("get project info: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get project info: %w", err)
	}

	return result, nil
}

// GetProjectInfoOption configures the parameters for getting project info.
type GetProjectInfoOption interface {
	applyGetProjectInfo(*api.GetProjectInfoParams)
}

type getProjectInfoOptionFunc func(*api.GetProjectInfoParams)

func (f getProjectInfoOptionFunc) applyGetProjectInfo(params *api.GetProjectInfoParams) {
	f(params)
}

// WithProjectInfoInterval sets the interval for project info statistics (e.g. "1M", "3M").
func WithProjectInfoInterval(interval string) GetProjectInfoOption {
	return getProjectInfoOptionFunc(func(params *api.GetProjectInfoParams) {
		params.Interval = &interval
	})
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

	response, err := p.client.raw.CreateProjectWithResponse(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	return result, nil
}

// Delete deletes a project by its numeric ID.
func (p *ProjectsScope) Delete(
	ctx context.Context,
	projectID int64,
) (*types.DeleteBulkRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "DeleteProject"),
		slog.Int64("projectId", projectID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.DeleteProjectWithResponse(
		ctx,
		&api.DeleteProjectParams{Ids: []int64{projectID}},
	)
	if err != nil {
		return nil, fmt.Errorf("delete project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
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

	response, err := p.client.raw.UpdateProjectWithResponse(ctx, projectName, request)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
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

	response, err := p.client.raw.AssignProjectUsersWithResponse(ctx, projectName, request)
	if err != nil {
		return nil, fmt.Errorf("assign project users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
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

	response, err := p.client.raw.UnassignProjectUsersWithResponse(
		ctx,
		projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("unassign project users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
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
) (*types.PageUserResource, error) {
	logger := p.client.logger.With(
		slog.String("operation", "ListProjectUsers"),
		slog.String("projectName", projectName),
	)

	params := &api.GetProjectUsersParams{}
	for _, option := range options {
		option.applyListProjectUsers(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetProjectUsersWithResponse(ctx, projectName, params)
	if err != nil {
		return nil, fmt.Errorf("list project users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list project users: %w", err)
	}

	return result, nil
}

// ListProjectUsersOption configures the filter and pagination parameters for listing project users.
type ListProjectUsersOption interface {
	applyListProjectUsers(*api.GetProjectUsersParams)
}

type listProjectUsersOptionFunc func(*api.GetProjectUsersParams)

func (f listProjectUsersOptionFunc) applyListProjectUsers(params *api.GetProjectUsersParams) {
	f(params)
}

// WithUserPageNumber sets the page number (0-based) for project users listing.
func WithUserPageNumber(n int32) ListProjectUsersOption {
	return listProjectUsersOptionFunc(func(params *api.GetProjectUsersParams) {
		params.PagePage = &n
	})
}

// WithUserPageSize sets the number of results per page for project users listing.
func WithUserPageSize(size int32) ListProjectUsersOption {
	return listProjectUsersOptionFunc(func(params *api.GetProjectUsersParams) {
		params.PageSize = &size
	})
}

// WithUserSort sets the sorting criteria for project users listing.
func WithUserSort(fields ...string) ListProjectUsersOption {
	return listProjectUsersOptionFunc(func(params *api.GetProjectUsersParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// ExportProjectsOption configures the filter parameters for exporting projects.
type ExportProjectsOption interface {
	applyExportProjects(*api.ExportProjectsParams)
}

type exportProjectsOptionFunc func(*api.ExportProjectsParams)

func (f exportProjectsOptionFunc) applyExportProjects(params *api.ExportProjectsParams) {
	f(params)
}

// WithExportView sets the export format view (e.g. "csv").
func WithExportView(view string) ExportProjectsOption {
	return exportProjectsOptionFunc(func(params *api.ExportProjectsParams) {
		v := api.ExportProjectsParamsView(view)
		params.View = &v
	})
}

// WithExportFilterName filters exported projects by exact name.
func WithExportFilterName(name string) ExportProjectsOption {
	return exportProjectsOptionFunc(func(params *api.ExportProjectsParams) {
		params.FilterEqName = &name
	})
}

// WithExportFilterType filters exported projects by type (e.g. "INTERNAL").
func WithExportFilterType(projectType string) ExportProjectsOption {
	return exportProjectsOptionFunc(func(params *api.ExportProjectsParams) {
		params.FilterEqType = &projectType
	})
}

// WithExportFilterLastRun filters exported projects by last run date.
func WithExportFilterLastRun(date openapi_types.Date) ExportProjectsOption {
	return exportProjectsOptionFunc(func(params *api.ExportProjectsParams) {
		params.FilterEqLastRun = &date
	})
}

// Export returns a Jasper report of projects as raw bytes.
func (p *ProjectsScope) Export(
	ctx context.Context,
	options ...ExportProjectsOption,
) ([]byte, error) {
	logger := p.client.logger.With(slog.String("operation", "ExportProjects"))

	params := &api.ExportProjectsParams{}
	for _, option := range options {
		option.applyExportProjects(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.ExportProjectsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("export projects: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	if response.JSONDefault != nil {
		return nil, fmt.Errorf(
			"export projects: %w",
			newAPIError(response.StatusCode(), response.JSONDefault),
		)
	}

	code := response.StatusCode()
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf(
			"export projects: %w",
			newUnexpectedStatusError(code, response.Body),
		)
	}

	return response.Body, nil
}

// GetAnalyzerStatus returns the analyzer indexing status for all projects.
func (p *ProjectsScope) GetAnalyzerStatus(
	ctx context.Context,
) (map[string]bool, error) {
	logger := p.client.logger.With(slog.String("operation", "GetAnalyzerIndexingStatus"))
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetAnalyzerIndexingStatusWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get analyzer status: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get analyzer status: %w", err)
	}

	if result == nil {
		return map[string]bool{}, nil
	}

	return *result, nil
}

// UpdateNotificationConfig updates the notification configuration for a project.
func (p *ProjectsScope) UpdateNotificationConfig(
	ctx context.Context,
	projectName string,
	request types.ProjectNotificationConfigDTO,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "UpdateProjectNotificationConfig"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.UpdateProjectNotificationConfigWithResponse(
		ctx,
		projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update notification config: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update notification config: %w", err)
	}

	return result, nil
}

// GetPreference returns the current user's preference for the specified project.
func (p *ProjectsScope) GetPreference(
	ctx context.Context,
	projectName string,
) (*types.PreferenceResource, error) {
	logger := p.client.logger.With(
		slog.String("operation", "GetUserPreference"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetUserPreferenceWithResponse(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("get user preference: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get user preference: %w", err)
	}

	return result, nil
}

// AddPreference adds a filter to the current user's preferences for the specified project.
func (p *ProjectsScope) AddPreference(
	ctx context.Context,
	projectName string,
	filterID int64,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "AddUserPreference"),
		slog.String("projectName", projectName),
		slog.Int64("filterId", filterID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.AddUserPreferenceWithResponse(ctx, projectName, filterID)
	if err != nil {
		return nil, fmt.Errorf("add user preference: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("add user preference: %w", err)
	}

	return result, nil
}

// RemovePreference removes a filter from the current user's preferences for the specified project.
func (p *ProjectsScope) RemovePreference(
	ctx context.Context,
	projectName string,
	filterID int64,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "RemoveUserPreference"),
		slog.String("projectName", projectName),
		slog.Int64("filterId", filterID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.RemoveUserPreferenceWithResponse(ctx, projectName, filterID)
	if err != nil {
		return nil, fmt.Errorf("remove user preference: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("remove user preference: %w", err)
	}

	return result, nil
}

// DeleteIndex deletes the analyzer index for the specified project.
func (p *ProjectsScope) DeleteIndex(
	ctx context.Context,
	projectName string,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "DeleteProjectIndex"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.DeleteProjectIndexWithResponse(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("delete project index: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete project index: %w", err)
	}

	return result, nil
}

// IndexData triggers re-indexing of the specified project's data.
func (p *ProjectsScope) IndexData(
	ctx context.Context,
	projectName string,
) (*types.OperationCompletionRS, error) {
	logger := p.client.logger.With(
		slog.String("operation", "IndexProjectData"),
		slog.String("projectName", projectName),
	)
	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.IndexProjectDataWithResponse(ctx, projectName)
	if err != nil {
		return nil, fmt.Errorf("index project data: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("index project data: %w", err)
	}

	return result, nil
}

// ListAssignableUsersOption configures the filter and pagination parameters
// for listing assignable users.
type ListAssignableUsersOption interface {
	applyListAssignableUsers(*api.GetUsersForAssignParams)
}

type listAssignableUsersOptionFunc func(*api.GetUsersForAssignParams)

func (f listAssignableUsersOptionFunc) applyListAssignableUsers(params *api.GetUsersForAssignParams) {
	f(params)
}

// WithAssignableUserPageNumber sets the page number (0-based) for the assignable users listing.
func WithAssignableUserPageNumber(n int32) ListAssignableUsersOption {
	return listAssignableUsersOptionFunc(func(params *api.GetUsersForAssignParams) {
		params.PagePage = &n
	})
}

// WithAssignableUserPageSize sets the number of results per page for the assignable users listing.
func WithAssignableUserPageSize(size int32) ListAssignableUsersOption {
	return listAssignableUsersOptionFunc(func(params *api.GetUsersForAssignParams) {
		params.PageSize = &size
	})
}

// WithAssignableUserSort sets the sorting criteria for the assignable users listing.
func WithAssignableUserSort(fields ...string) ListAssignableUsersOption {
	return listAssignableUsersOptionFunc(func(params *api.GetUsersForAssignParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// ListAssignableUsers returns a paginated list of users available for assignment to the project.
func (p *ProjectsScope) ListAssignableUsers(
	ctx context.Context,
	projectName string,
	options ...ListAssignableUsersOption,
) (*types.PageUserResource, error) {
	logger := p.client.logger.With(
		slog.String("operation", "ListAssignableUsers"),
		slog.String("projectName", projectName),
	)

	params := &api.GetUsersForAssignParams{}
	for _, option := range options {
		option.applyListAssignableUsers(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetUsersForAssignWithResponse(ctx, projectName, params)
	if err != nil {
		return nil, fmt.Errorf("list assignable users: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list assignable users: %w", err)
	}

	return result, nil
}

// ListUsernames returns usernames assigned to a project, filtered by the given term.
func (p *ProjectsScope) ListUsernames(
	ctx context.Context,
	projectName string,
	term string,
) ([]string, error) {
	logger := p.client.logger.With(
		slog.String("operation", "ListProjectUsernames"),
		slog.String("projectName", projectName),
		slog.String("term", term),
	)
	logger.InfoContext(ctx, "API request")

	params := &api.GetProjectUsers1Params{FilterCntUsers: term}

	response, err := p.client.raw.GetProjectUsers1WithResponse(ctx, projectName, params)
	if err != nil {
		return nil, fmt.Errorf("list project usernames: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list project usernames: %w", err)
	}

	if result == nil {
		return nil, nil
	}

	return *result, nil
}

// SearchUsernamesOption configures the pagination parameters for searching usernames.
type SearchUsernamesOption interface {
	applySearchUsernames(*api.SearchForUserParams)
}

type searchUsernamesOptionFunc func(*api.SearchForUserParams)

func (f searchUsernamesOptionFunc) applySearchUsernames(params *api.SearchForUserParams) {
	f(params)
}

// WithSearchUsernamesPageNumber sets the page number (0-based) for the username search.
func WithSearchUsernamesPageNumber(n int32) SearchUsernamesOption {
	return searchUsernamesOptionFunc(func(params *api.SearchForUserParams) {
		params.PagePage = &n
	})
}

// WithSearchUsernamesPageSize sets the number of results per page for the username search.
func WithSearchUsernamesPageSize(size int32) SearchUsernamesOption {
	return searchUsernamesOptionFunc(func(params *api.SearchForUserParams) {
		params.PageSize = &size
	})
}

// WithSearchUsernamesSort sets the sorting criteria for the username search.
func WithSearchUsernamesSort(fields ...string) SearchUsernamesOption {
	return searchUsernamesOptionFunc(func(params *api.SearchForUserParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// SearchUsernames searches for usernames within a project by term.
func (p *ProjectsScope) SearchUsernames(
	ctx context.Context,
	projectName string,
	term string,
	options ...SearchUsernamesOption,
) (*types.PageSearchUserResource, error) {
	logger := p.client.logger.With(
		slog.String("operation", "SearchProjectUsernames"),
		slog.String("projectName", projectName),
		slog.String("term", term),
	)

	params := &api.SearchForUserParams{Term: term}
	for _, option := range options {
		option.applySearchUsernames(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.SearchForUserWithResponse(ctx, projectName, params)
	if err != nil {
		return nil, fmt.Errorf("search project usernames: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("search project usernames: %w", err)
	}

	return result, nil
}

// GetWidgetByCodeOption configures the query parameters for getting a project widget.
type GetWidgetByCodeOption interface {
	applyGetWidgetByCode(*api.GetProjectWidgetParams)
}

type getWidgetByCodeOptionFunc func(*api.GetProjectWidgetParams)

func (f getWidgetByCodeOptionFunc) applyGetWidgetByCode(params *api.GetProjectWidgetParams) {
	f(params)
}

// WithWidgetInterval sets the interval for the project widget data (e.g. "1M", "3M").
func WithWidgetInterval(interval string) GetWidgetByCodeOption {
	return getWidgetByCodeOptionFunc(func(params *api.GetProjectWidgetParams) {
		params.Interval = &interval
	})
}

// GetWidgetByCode returns a preconfigured project widget by its code.
func (p *ProjectsScope) GetWidgetByCode(
	ctx context.Context,
	projectName string,
	widgetCode string,
	options ...GetWidgetByCodeOption,
) (map[string]any, error) {
	logger := p.client.logger.With(
		slog.String("operation", "GetProjectWidget"),
		slog.String("projectName", projectName),
		slog.String("widgetCode", widgetCode),
	)

	params := &api.GetProjectWidgetParams{}
	for _, option := range options {
		option.applyGetWidgetByCode(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := p.client.raw.GetProjectWidgetWithResponse(
		ctx,
		projectName,
		widgetCode,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get project widget: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get project widget: %w", err)
	}

	if result == nil {
		return map[string]any{}, nil
	}

	return *result, nil
}
