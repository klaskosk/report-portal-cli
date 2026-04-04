package client

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// UserFilterScope provides operations on user filters (saved filters) within a project.
type UserFilterScope struct {
	project *ProjectScope
}

// List returns a paginated list of filters matching the given options.
func (f *UserFilterScope) List(
	ctx context.Context,
	options ...ListFiltersOption,
) (*types.PageUserFilterResource, error) {
	logger := f.project.operationLogger("ListFilters")

	params := &api.GetAllFiltersParams{}
	for _, option := range options {
		option.applyListFilters(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := f.project.client.raw.GetAllFiltersWithResponse(
		ctx,
		f.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("list filters: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("list filters: %w", err)
	}

	return result, nil
}

// GetByID returns a single filter by its numeric ID.
func (f *UserFilterScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.UserFilterResource, error) {
	logger := f.project.operationLogger("GetFilterByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := f.project.client.raw.GetFilterWithResponse(
		ctx,
		f.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get filter by id: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get filter by id: %w", err)
	}

	return result, nil
}

// Create creates a new filter in the project. The API uses UpdateUserFilterRQ as the
// create request type.
func (f *UserFilterScope) Create(
	ctx context.Context,
	request types.UpdateUserFilterRQ,
) (*types.EntryCreatedRS, error) {
	logger := f.project.operationLogger("CreateFilter")
	logger.InfoContext(ctx, "API request")

	response, err := f.project.client.raw.CreateFilterWithResponse(
		ctx,
		f.project.projectName,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("create filter: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
	if err != nil {
		return nil, fmt.Errorf("create filter: %w", err)
	}

	return result, nil
}

// Update modifies an existing filter.
func (f *UserFilterScope) Update(
	ctx context.Context,
	id int64,
	request types.UpdateUserFilterRQ,
) (*types.OperationCompletionRS, error) {
	logger := f.project.operationLogger("UpdateFilter", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := f.project.client.raw.UpdateUserFilterWithResponse(
		ctx,
		f.project.projectName,
		id,
		request,
	)
	if err != nil {
		return nil, fmt.Errorf("update filter: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("update filter: %w", err)
	}

	return result, nil
}

// Delete removes a filter by its numeric ID.
func (f *UserFilterScope) Delete(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := f.project.operationLogger("DeleteFilter", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	response, err := f.project.client.raw.DeleteFilterWithResponse(
		ctx,
		f.project.projectName,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("delete filter: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("delete filter: %w", err)
	}

	return result, nil
}

// GetByIds returns multiple filters by their numeric IDs.
func (f *UserFilterScope) GetByIds(
	ctx context.Context,
	ids []int64,
) ([]types.UserFilterResource, error) {
	logger := f.project.operationLogger("GetFiltersByIds")
	logger.InfoContext(ctx, "API request")

	response, err := f.project.client.raw.GetUserFiltersWithResponse(
		ctx,
		f.project.projectName,
		&api.GetUserFiltersParams{
			Ids: ids,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("get filters by ids: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get filters by ids: %w", err)
	}

	if result == nil {
		return []types.UserFilterResource{}, nil
	}

	return *result, nil
}

// GetNames returns filter names (for autocomplete). The returned PageOwnedEntityResource
// contains OwnedEntityResource items with name, owner, and description fields.
func (f *UserFilterScope) GetNames(
	ctx context.Context,
	filter string,
) (*types.PageOwnedEntityResource, error) {
	logger := f.project.operationLogger("GetFilterNames", slog.String("filter", filter))

	params := &api.GetAllFiltersNamesParams{}
	if filter != "" {
		params.FilterEqName = &filter
	}

	logger.InfoContext(ctx, "API request")

	response, err := f.project.client.raw.GetAllFiltersNamesWithResponse(
		ctx,
		f.project.projectName,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("get filter names: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
	if err != nil {
		return nil, fmt.Errorf("get filter names: %w", err)
	}

	if result == nil {
		empty := []types.OwnedEntityResource{}

		return &types.PageOwnedEntityResource{
			Content: &empty,
			Page:    &types.PageMetadata{},
		}, nil
	}

	return result, nil
}

// ListFiltersOption configures the filter and pagination parameters for listing user filters.
type ListFiltersOption interface {
	applyListFilters(*api.GetAllFiltersParams)
}

type listFiltersOptionFunc func(*api.GetAllFiltersParams)

func (f listFiltersOptionFunc) applyListFilters(params *api.GetAllFiltersParams) {
	f(params)
}

// WithFilterPageNumber sets the page number (0-based) for filter listing.
func WithFilterPageNumber(n int32) ListFiltersOption {
	return listFiltersOptionFunc(func(params *api.GetAllFiltersParams) {
		params.PagePage = &n
	})
}

// WithFilterPageSize sets the number of results per page for filter listing.
func WithFilterPageSize(size int32) ListFiltersOption {
	return listFiltersOptionFunc(func(params *api.GetAllFiltersParams) {
		params.PageSize = &size
	})
}

// WithFilterSort sets the sorting criteria for filter listing (e.g. "name,asc").
func WithFilterSort(fields ...string) ListFiltersOption {
	return listFiltersOptionFunc(func(params *api.GetAllFiltersParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}
