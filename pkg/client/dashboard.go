package client

import (
	"context"
	"fmt"
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

	response, err := d.project.client.raw.GetAllDashboardsWithResponse(
		ctx,
		d.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("list dashboards: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list dashboards: %w", err)
	}

	return result, nil
}

// GetByID returns a single dashboard by its numeric ID.
func (d *DashboardScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.DashboardResource, error) {
	logger := d.project.operationLogger("GetDashboardByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.GetDashboardWithResponse(
		ctx,
		d.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get dashboard by id: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get dashboard by id: %w", err)
	}

	return result, nil
}

// Create creates a new dashboard in the project.
func (d *DashboardScope) Create(
	ctx context.Context,
	request types.CreateDashboardRQ,
) (*types.EntryCreatedRS, error) {
	logger := d.project.operationLogger("CreateDashboard", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.CreateDashboardWithResponse(
		ctx,
		d.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create dashboard: %w", err)
	}

	return result, nil
}

// Update modifies an existing dashboard.
func (d *DashboardScope) Update(
	ctx context.Context,
	id int64,
	request types.UpdateDashboardRQ,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger("UpdateDashboard", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.UpdateDashboardWithResponse(
		ctx,
		d.project.projectName,
		id,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update dashboard: %w", err)
	}

	return result, nil
}

// Delete removes a dashboard by its numeric ID.
func (d *DashboardScope) Delete(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := d.project.operationLogger("DeleteDashboard", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.DeleteDashboardWithResponse(
		ctx,
		d.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("delete dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete dashboard: %w", err)
	}

	return result, nil
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

	response, err := d.project.client.raw.AddWidgetWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("add widget to dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("add widget to dashboard: %w", err)
	}

	return result, nil
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

	response, err := d.project.client.raw.GetDashboardConfigWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
	)
	if err != nil {
		return nil, fmt.Errorf("get dashboard config: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get dashboard config: %w", err)
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
	logger := d.project.operationLogger("CreatePreconfiguredDashboard", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	response, err := d.project.client.raw.CreatePreconfiguredWithResponse(
		ctx,
		d.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create preconfigured dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create preconfigured dashboard: %w", err)
	}

	return result, nil
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

	response, err := d.project.client.raw.RemoveWidgetWithResponse(
		ctx,
		d.project.projectName,
		dashboardID,
		widgetID,
	)
	if err != nil {
		return nil, fmt.Errorf("remove widget from dashboard: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("remove widget from dashboard: %w", err)
	}

	return result, nil
}

// ListDashboardsOption configures the filter and pagination parameters for listing dashboards.
type ListDashboardsOption interface {
	applyListDashboards(*api.GetAllDashboardsParams)
}

type listDashboardsOptionFunc func(*api.GetAllDashboardsParams)

func (f listDashboardsOptionFunc) applyListDashboards(params *api.GetAllDashboardsParams) {
	f(params)
}

// WithDashboardPageNumber sets the page number (0-based) for dashboard listing.
func WithDashboardPageNumber(n int32) ListDashboardsOption {
	return listDashboardsOptionFunc(func(params *api.GetAllDashboardsParams) {
		params.PagePage = &n
	})
}

// WithDashboardPageSize sets the number of results per page for dashboard listing.
func WithDashboardPageSize(size int32) ListDashboardsOption {
	return listDashboardsOptionFunc(func(params *api.GetAllDashboardsParams) {
		params.PageSize = &size
	})
}

// WithDashboardSort sets the sorting criteria for dashboard listing (e.g. "name,asc").
func WithDashboardSort(fields ...string) ListDashboardsOption {
	return listDashboardsOptionFunc(func(params *api.GetAllDashboardsParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}
