package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

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

	return handleResponse(response, "get widget by id", response.JSONDefault, response.JSON200)
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

	return handleResponse(response, "create widget", response.JSONDefault, response.JSON201)
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

	return handleResponse(response, "update widget", response.JSONDefault, response.JSON200)
}

// GetNames returns widget names matching the filter (for autocomplete). The returned
// IterableObject contains maps with name/owner keys.
func (w *WidgetScope) GetNames(
	ctx context.Context,
	filter string,
) (*types.IterableObject, error) {
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
		logger.DebugContext(ctx, "typed widget names decode failed; retrying raw", "error", err)

		return w.getNamesRaw(ctx, params)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(
		response,
		"get widget names",
		response.JSONDefault,
		response.JSON200,
	)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return &types.IterableObject{}, nil
	}

	return result, nil
}

// getNamesRaw retries widget-name lookup and normalizes deployments that return string items.
func (w *WidgetScope) getNamesRaw(
	ctx context.Context,
	params *api.GetWidgetNamesUsingGETParams,
) (*types.IterableObject, error) {
	rawClient, ok := w.project.client.raw.(*api.ClientWithResponses)
	if !ok {
		return nil, fmt.Errorf(
			"get widget names: raw fallback unavailable for client type %T",
			w.project.client.raw,
		)
	}

	rawResponse, err := rawClient.GetWidgetNamesUsingGET(
		ctx,
		w.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get widget names: %w", err)
	}
	defer rawResponse.Body.Close()

	body, err := io.ReadAll(rawResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("reading widget names response: %w", err)
	}

	if rawResponse.StatusCode != http.StatusOK {
		var errorResponse types.ErrorRS
		if json.Unmarshal(body, &errorResponse) == nil && errorResponse.ErrorCode != 0 {
			return nil, newAPIError("get widget names", rawResponse.StatusCode, &errorResponse)
		}

		return nil, newUnexpectedStatusError("get widget names", rawResponse.StatusCode)
	}

	var rawIterable struct {
		Content []json.RawMessage `json:"content"`
		Page    types.PageMetadata `json:"page"`
	}
	if err := json.Unmarshal(body, &rawIterable); err != nil {
		return nil, fmt.Errorf("get widget names: %w", err)
	}

	result := &types.IterableObject{
		Content: make([]map[string]interface{}, 0, len(rawIterable.Content)),
		Page:    rawIterable.Page,
	}

	for _, rawItem := range rawIterable.Content {
		var objectItem map[string]interface{}
		if err := json.Unmarshal(rawItem, &objectItem); err == nil {
			result.Content = append(result.Content, objectItem)

			continue
		}

		var stringItem string
		if err := json.Unmarshal(rawItem, &stringItem); err == nil {
			result.Content = append(result.Content, map[string]interface{}{"name": stringItem})

			continue
		}

		return nil, fmt.Errorf("get widget names: unsupported content item %s", string(rawItem))
	}

	return result, nil
}
