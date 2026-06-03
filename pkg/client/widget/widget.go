package widget

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// WidgetScope provides operations on widgets within a project.
type WidgetScope struct {
	scope core.Scope
}

// GetByID returns a single widget by its numeric ID.
func (w *WidgetScope) GetByID(
	ctx context.Context,
	widgetID int64,
) (*types.WidgetResource, error) {
	logger := w.scope.OperationLogger("GetWidgetByID", slog.Int64("id", widgetID))
	logger.InfoContext(ctx, "API request")

	resp, err := w.scope.Client.GetWidgetWithResponse(
		ctx,
		w.scope.ProjectName,
		widgetID,
	)

	return core.Execute(ctx, logger, "get widget by id", resp, err)
}

// Create creates a new widget in the project.
func (w *WidgetScope) Create(
	ctx context.Context,
	request types.WidgetRQ,
) (*types.EntryCreatedRS, error) {
	logger := w.scope.OperationLogger("CreateWidget", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	resp, err := w.scope.Client.CreateWidgetWithResponse(
		ctx,
		w.scope.ProjectName,
		request,
	)

	return core.Execute(ctx, logger, "create widget", resp, err)
}

// Update modifies an existing widget.
func (w *WidgetScope) Update(
	ctx context.Context,
	widgetID int64,
	request types.WidgetRQ,
) (*types.OperationCompletionRS, error) {
	logger := w.scope.OperationLogger("UpdateWidget", slog.Int64("id", widgetID))
	logger.InfoContext(ctx, "API request")

	resp, err := w.scope.Client.UpdateWidgetWithResponse(
		ctx,
		w.scope.ProjectName,
		widgetID,
		request,
	)

	return core.Execute(ctx, logger, "update widget", resp, err)
}

// GetData returns a widget's computed data by its numeric ID.
// The attributes and params arguments allow configuring how the widget data is computed.
func (w *WidgetScope) GetData(
	ctx context.Context,
	widgetID int64,
	options ...GetWidgetDataOption,
) (*types.WidgetResource, error) {
	logger := w.scope.OperationLogger("GetWidgetData", slog.Int64("id", widgetID))

	params := &api.GetWidget1Params{
		Params: types.MultiValueMapStringString{},
	}
	for _, option := range options {
		option.applyGetWidgetData(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := w.scope.Client.GetWidget1WithResponse(
		ctx,
		w.scope.ProjectName,
		widgetID,
		params,
	)

	return core.Execute(ctx, logger, "get widget data", resp, err)
}

// GetWidgetDataOption configures the parameters for getting widget data.
type GetWidgetDataOption interface {
	applyGetWidgetData(params *api.GetWidget1Params)
}

type getWidgetDataOptionFunc func(*api.GetWidget1Params)

func (f getWidgetDataOptionFunc) applyGetWidgetData(params *api.GetWidget1Params) {
	f(params)
}

// WithWidgetAttributes sets the attributes for widget data retrieval.
func WithWidgetAttributes(attributes ...string) getWidgetDataOptionFunc {
	return getWidgetDataOptionFunc(func(params *api.GetWidget1Params) {
		params.Attributes = &attributes
	})
}

// GetNames returns widget names matching the filter (for autocomplete).
func (w *WidgetScope) GetNames(
	ctx context.Context,
	filter string,
) (*types.WidgetNamesIterable, error) {
	logger := w.scope.OperationLogger("GetWidgetNames", slog.String("filter", filter))

	params := &api.GetWidgetNamesParams{}
	if filter != "" {
		params.FilterEqName = &filter
	}

	logger.InfoContext(ctx, "API request")

	resp, err := w.scope.Client.GetWidgetNamesWithResponse(
		ctx,
		w.scope.ProjectName,
		params,
	)

	result, err := core.Execute(ctx, logger, "get widget names", resp, err)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return &types.WidgetNamesIterable{
			Content: []string{},
			Page:    types.PageMetadata{},
		}, nil
	}

	return result, nil
}

// NewWidgetScope constructs a WidgetScope for wiring from the facade.
func NewWidgetScope(s core.Scope) *WidgetScope {
	return &WidgetScope{scope: s}
}
