package client

import (
	"context"
	"fmt"
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

	response, err := w.project.client.raw.GetWidgetUsingGET1WithResponse(
		ctx,
		w.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get widget by id: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get widget by id: %w", err)
	}

	return result, nil
}

// Create creates a new widget in the project.
func (w *WidgetScope) Create(
	ctx context.Context,
	request types.WidgetRQ,
) (*types.EntryCreatedRS, error) {
	logger := w.project.operationLogger("CreateWidget", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	response, err := w.project.client.raw.CreateWidgetUsingPOSTWithResponse(
		ctx,
		w.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create widget: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201)
	if err != nil {
		return nil, fmt.Errorf("create widget: %w", err)
	}

	return result, nil
}

// Update modifies an existing widget.
func (w *WidgetScope) Update(
	ctx context.Context,
	id int64,
	request types.WidgetRQ,
) (*types.OperationCompletionRS, error) {
	logger := w.project.operationLogger("UpdateWidget", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := w.project.client.raw.UpdateWidgetUsingPUTWithResponse(
		ctx,
		w.project.projectName,
		id,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update widget: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("update widget: %w", err)
	}

	return result, nil
}

// GetData returns a widget's computed data by its numeric ID.
// The attributes and params arguments allow configuring how the widget data is computed.
func (w *WidgetScope) GetData(
	ctx context.Context,
	id int64,
	options ...GetWidgetDataOption,
) (*types.WidgetResource, error) {
	logger := w.project.operationLogger("GetWidgetData", slog.Int64("id", id))

	params := &api.GetWidgetUsingGETParams{
		Params: map[string]any{},
	}
	for _, option := range options {
		option(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := w.project.client.raw.GetWidgetUsingGETWithResponse(
		ctx,
		w.project.projectName,
		id,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get widget data: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get widget data: %w", err)
	}

	return result, nil
}

// GetWidgetDataOption configures the parameters for getting widget data.
type GetWidgetDataOption func(*api.GetWidgetUsingGETParams)

// WithWidgetAttributes sets the attributes for widget data retrieval.
func WithWidgetAttributes(attributes ...string) GetWidgetDataOption {
	return func(params *api.GetWidgetUsingGETParams) {
		params.Attributes = &attributes
	}
}

// GetNames returns widget names matching the filter (for autocomplete).
func (w *WidgetScope) GetNames(
	ctx context.Context,
	filter string,
) (*types.WidgetNamesIterable, error) {
	logger := w.project.operationLogger("GetWidgetNames", slog.String("filter", filter))

	params := &api.GetWidgetNamesUsingGETParams{}
	if filter != "" {
		params.FilterEqName = &filter
	}

	logger.InfoContext(ctx, "API request")

	response, err := w.project.client.raw.GetWidgetNamesUsingGETWithResponse(
		ctx,
		w.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get widget names: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200)
	if err != nil {
		return nil, fmt.Errorf("get widget names: %w", err)
	}

	if result == nil {
		return &types.WidgetNamesIterable{
			Content: []string{},
			Page:    types.PageMetadata{},
		}, nil
	}

	return result, nil
}
