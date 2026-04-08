package client

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// WidgetScope provides operations on widgets within a project.
type WidgetScope struct {
	project *ProjectScope
}

// GetByID returns a single widget by its numeric ID.
func (w *WidgetScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.WidgetResource, error) {
	logger := w.project.operationLogger("GetWidgetByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	resp, err := w.project.client.raw.GetWidgetWithResponse(
		ctx,
		w.project.projectName,
		id,
	)

	return execute(ctx, logger, "get widget by id", resp, err)
}

// Create creates a new widget in the project.
func (w *WidgetScope) Create(
	ctx context.Context,
	request types.WidgetRQ,
) (*types.EntryCreatedRS, error) {
	logger := w.project.operationLogger("CreateWidget", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	resp, err := w.project.client.raw.CreateWidgetWithResponse(
		ctx,
		w.project.projectName,
		request,
	)

	return execute(ctx, logger, "create widget", resp, err)
}

// Update modifies an existing widget.
func (w *WidgetScope) Update(
	ctx context.Context,
	id int64,
	request types.WidgetRQ,
) (*types.OperationCompletionRS, error) {
	logger := w.project.operationLogger("UpdateWidget", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	resp, err := w.project.client.raw.UpdateWidgetWithResponse(
		ctx,
		w.project.projectName,
		id,
		request,
	)

	return execute(ctx, logger, "update widget", resp, err)
}

// GetData returns a widget's computed data by its numeric ID.
// The attributes and params arguments allow configuring how the widget data is computed.
func (w *WidgetScope) GetData(
	ctx context.Context,
	id int64,
	options ...GetWidgetDataOption,
) (*types.WidgetResource, error) {
	logger := w.project.operationLogger("GetWidgetData", slog.Int64("id", id))

	params := &api.GetWidget1Params{
		Params: types.MultiValueMapStringString{},
	}
	for _, option := range options {
		option.applyGetWidgetData(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := w.project.client.raw.GetWidget1WithResponse(
		ctx,
		w.project.projectName,
		id,
		params,
	)

	return execute(ctx, logger, "get widget data", resp, err)
}

// GetWidgetDataOption configures the parameters for getting widget data.
type GetWidgetDataOption interface {
	applyGetWidgetData(*api.GetWidget1Params)
}

type getWidgetDataOptionFunc func(*api.GetWidget1Params)

func (f getWidgetDataOptionFunc) applyGetWidgetData(params *api.GetWidget1Params) {
	f(params)
}

// WithWidgetAttributes sets the attributes for widget data retrieval.
func WithWidgetAttributes(attributes ...string) GetWidgetDataOption {
	return getWidgetDataOptionFunc(func(params *api.GetWidget1Params) {
		params.Attributes = &attributes
	})
}

// GetNames returns widget names matching the filter (for autocomplete).
func (w *WidgetScope) GetNames(
	ctx context.Context,
	filter string,
) (*types.WidgetNamesIterable, error) {
	logger := w.project.operationLogger("GetWidgetNames", slog.String("filter", filter))

	params := &api.GetWidgetNamesParams{}
	if filter != "" {
		params.FilterEqName = &filter
	}

	logger.InfoContext(ctx, "API request")

	resp, err := w.project.client.raw.GetWidgetNamesWithResponse(
		ctx,
		w.project.projectName,
		params,
	)

	result, err := execute(ctx, logger, "get widget names", resp, err)
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
