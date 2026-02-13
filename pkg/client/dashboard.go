package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// DashboardScope provides operations on dashboards within a project.
type DashboardScope struct {
	project *ProjectScope
}

// List returns a paginated list of dashboards matching the given options.
func (d *DashboardScope) List(
	ctx context.Context,
	options ...ListDashboardsOption,
) (*types.IterableDashboardResource, error) {
	logger := d.project.operationLogger("ListDashboards")

	params := &api.GetAllDashboardsUsingGETParams{}
	for _, option := range options {
		option(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.GetAllDashboardsUsingGETWithResponse(
		ctx,
		d.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("list dashboards: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "list dashboards", response.JSONDefault, response.JSON200)
}

// GetByID returns a single dashboard by its numeric ID.
func (d *DashboardScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.DashboardResource, error) {
	logger := d.project.operationLogger("GetDashboardByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.GetDashboardUsingGETWithResponse(
		ctx,
		d.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get dashboard by id: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get dashboard by id", response.JSONDefault, response.JSON200)
}

// Create creates a new dashboard in the project.
func (d *DashboardScope) Create(
	ctx context.Context,
	request types.CreateDashboardRQ,
) (*types.EntryCreatedRS, error) {
	logger := d.project.operationLogger("CreateDashboard", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.CreateDashboardUsingPOSTWithResponse(
		ctx,
		d.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "create dashboard", response.JSONDefault, response.JSON201)
}

// Update modifies an existing dashboard.
func (d *DashboardScope) Update(
	ctx context.Context,
	id int64,
	request types.UpdateDashboardRQ,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger("UpdateDashboard", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.UpdateDashboardUsingPUTWithResponse(
		ctx,
		d.project.projectName,
		id,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "update dashboard", response.JSONDefault, response.JSON200)
}

// Delete removes a dashboard by its numeric ID.
func (d *DashboardScope) Delete(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger("DeleteDashboard", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.DeleteDashboardUsingDELETEWithResponse(
		ctx,
		d.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("delete dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "delete dashboard", response.JSONDefault, response.JSON200)
}

// AddWidget adds a widget to a dashboard.
func (d *DashboardScope) AddWidget(
	ctx context.Context,
	dashboardID int64,
	request types.AddWidgetRq,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger("AddWidgetToDashboard", slog.Int64("dashboardId", dashboardID))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.AddWidgetUsingPUTWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("add widget to dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "add widget to dashboard", response.JSONDefault, response.JSON200)
}

// RemoveWidget removes a widget from a dashboard.
func (d *DashboardScope) RemoveWidget(
	ctx context.Context,
	dashboardID int64,
	widgetID int64,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger("RemoveWidgetFromDashboard",
		slog.Int64("dashboardId", dashboardID),
		slog.Int64("widgetId", widgetID),
	)
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.RemoveWidgetUsingDELETEWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
		widgetID,
	)
	if err != nil {
		return nil, fmt.Errorf("remove widget from dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "remove widget from dashboard", response.JSONDefault, response.JSON200)
}

// ListDashboardsOption configures the filter and pagination parameters for listing dashboards.
type ListDashboardsOption func(*api.GetAllDashboardsUsingGETParams)

// WithDashboardPageNumber sets the page number (0-based) for dashboard listing.
func WithDashboardPageNumber(n int32) ListDashboardsOption {
	return func(params *api.GetAllDashboardsUsingGETParams) {
		params.PagePage = &n
	}
}

// WithDashboardPageSize sets the number of results per page for dashboard listing.
func WithDashboardPageSize(size int32) ListDashboardsOption {
	return func(params *api.GetAllDashboardsUsingGETParams) {
		params.PageSize = &size
	}
}

// WithDashboardSort sets the sorting criteria for dashboard listing (e.g. "name,asc").
func WithDashboardSort(fields ...string) ListDashboardsOption {
	return func(params *api.GetAllDashboardsUsingGETParams) {
		params.PageSort = &fields
	}
}
