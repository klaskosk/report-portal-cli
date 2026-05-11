package client

import (
	"context"
	"log/slog"
	"strings"

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
) (*types.PageDashboardResource, error) {
	logger := d.project.operationLogger("ListDashboards")

	params := &api.GetAllDashboardsParams{}
	for _, option := range options {
		option.applyListDashboards(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := d.project.client.raw.GetAllDashboardsWithResponse(
		ctx,
		d.project.projectName,
		params,
	)

	return execute(ctx, logger, "list dashboards", resp, err)
}

// GetByID returns a single dashboard by its numeric ID.
func (d *DashboardScope) GetByID(
	ctx context.Context,
	dashboardID int64,
) (*types.DashboardResource, error) {
	logger := d.project.operationLogger("GetDashboardByID", slog.Int64("id", dashboardID))
	logger.InfoContext(ctx, "API request")

	resp, err := d.project.client.raw.GetDashboardWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
	)

	return execute(ctx, logger, "get dashboard by id", resp, err)
}

// Create creates a new dashboard in the project.
func (d *DashboardScope) Create(
	ctx context.Context,
	request types.CreateDashboardRQ,
) (*types.EntryCreatedRS, error) {
	logger := d.project.operationLogger("CreateDashboard", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	resp, err := d.project.client.raw.CreateDashboardWithResponse(
		ctx,
		d.project.projectName,
		request,
	)

	return execute(ctx, logger, "create dashboard", resp, err)
}

// Update modifies an existing dashboard.
func (d *DashboardScope) Update(
	ctx context.Context,
	dashboardID int64,
	request types.UpdateDashboardRQ,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger("UpdateDashboard", slog.Int64("id", dashboardID))
	logger.InfoContext(ctx, "API request")

	resp, err := d.project.client.raw.UpdateDashboardWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
		request,
	)

	return execute(ctx, logger, "update dashboard", resp, err)
}

// Delete removes a dashboard by its numeric ID.
func (d *DashboardScope) Delete(
	ctx context.Context,
	dashboardID int64,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger("DeleteDashboard", slog.Int64("id", dashboardID))
	logger.InfoContext(ctx, "API request")

	resp, err := d.project.client.raw.DeleteDashboardWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
	)

	return execute(ctx, logger, "delete dashboard", resp, err)
}

// AddWidget adds a widget to a dashboard.
func (d *DashboardScope) AddWidget(
	ctx context.Context,
	dashboardID int64,
	request types.AddWidgetRq,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger(
		"AddWidgetToDashboard",
		slog.Int64("dashboardId", dashboardID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := d.project.client.raw.AddWidgetWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
		request,
	)

	return execute(ctx, logger, "add widget to dashboard", resp, err)
}

// GetConfig retrieves the widget configuration for a specific dashboard.
// The returned DashboardConfigResource always has a non-nil Widgets slice
// to avoid nil-pointer surprises for callers.
func (d *DashboardScope) GetConfig(
	ctx context.Context,
	dashboardID int64,
) (*types.DashboardConfigResource, error) {
	logger := d.project.operationLogger("GetDashboardConfig", slog.Int64("id", dashboardID))
	logger.InfoContext(ctx, "API request")

	resp, err := d.project.client.raw.GetDashboardConfigWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
	)

	result, err := execute(ctx, logger, "get dashboard config", resp, err)
	if err != nil {
		return nil, err
	}

	if result.Widgets == nil {
		result.Widgets = &[]types.WidgetConfigResource{}
	}

	return result, nil
}

// CreatePreconfigured creates a dashboard from a preconfigured template.
func (d *DashboardScope) CreatePreconfigured(
	ctx context.Context,
	request types.DashboardPreconfiguredRq,
) (*types.EntryCreatedRS, error) {
	logger := d.project.operationLogger(
		"CreatePreconfiguredDashboard",
		slog.String("name", request.Name),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := d.project.client.raw.CreatePreconfiguredWithResponse(
		ctx,
		d.project.projectName,
		request,
	)

	return execute(ctx, logger, "create preconfigured dashboard", resp, err)
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

	resp, err := d.project.client.raw.RemoveWidgetWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
		widgetID,
	)

	return execute(ctx, logger, "remove widget from dashboard", resp, err)
}

// ListDashboardsOption configures the filter and pagination parameters for listing dashboards.
type ListDashboardsOption interface {
	applyListDashboards(params *api.GetAllDashboardsParams)
}

type listDashboardsOptionFunc func(*api.GetAllDashboardsParams)

func (f listDashboardsOptionFunc) applyListDashboards(params *api.GetAllDashboardsParams) {
	f(params)
}

// WithDashboardPageNumber sets the page number (0-based) for dashboard listing.
func WithDashboardPageNumber(n int32) listDashboardsOptionFunc {
	return listDashboardsOptionFunc(func(params *api.GetAllDashboardsParams) {
		params.PagePage = &n
	})
}

// WithDashboardPageSize sets the number of results per page for dashboard listing.
func WithDashboardPageSize(size int32) listDashboardsOptionFunc {
	return listDashboardsOptionFunc(func(params *api.GetAllDashboardsParams) {
		params.PageSize = &size
	})
}

// WithDashboardSort sets the sorting criteria for dashboard listing (e.g. "name,asc").
func WithDashboardSort(fields ...string) listDashboardsOptionFunc {
	return listDashboardsOptionFunc(func(params *api.GetAllDashboardsParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}
