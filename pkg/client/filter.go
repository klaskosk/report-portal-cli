package client

import (
	"context"
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

	resp, err := f.project.client.raw.GetAllFiltersWithResponse(ctx, f.project.projectName, params)

	return execute[types.PageUserFilterResource](ctx, logger, "list filters", resp, err)
}

// GetByID returns a single filter by its numeric ID.
func (f *UserFilterScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.UserFilterResource, error) {
	logger := f.project.operationLogger("GetFilterByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	resp, err := f.project.client.raw.GetFilterWithResponse(ctx, f.project.projectName, id)

	return execute[types.UserFilterResource](ctx, logger, "get filter by id", resp, err)
}

// Create creates a new filter in the project. The API uses UpdateUserFilterRQ as the
// create request type.
func (f *UserFilterScope) Create(
	ctx context.Context,
	request types.UpdateUserFilterRQ,
) (*types.EntryCreatedRS, error) {
	logger := f.project.operationLogger("CreateFilter")
	logger.InfoContext(ctx, "API request")

	resp, err := f.project.client.raw.CreateFilterWithResponse(ctx, f.project.projectName, request)

	return execute[types.EntryCreatedRS](ctx, logger, "create filter", resp, err)
}

// Update modifies an existing filter.
func (f *UserFilterScope) Update(
	ctx context.Context,
	id int64,
	request types.UpdateUserFilterRQ,
) (*types.OperationCompletionRS, error) {
	logger := f.project.operationLogger("UpdateFilter", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	resp, err := f.project.client.raw.UpdateUserFilterWithResponse(ctx, f.project.projectName, id, request)

	return execute[types.OperationCompletionRS](ctx, logger, "update filter", resp, err)
}

// Delete removes a filter by its numeric ID.
func (f *UserFilterScope) Delete(
	ctx context.Context,
	id int64,
) (*types.OperationCompletionRS, error) {
	logger := f.project.operationLogger("DeleteFilter", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	resp, err := f.project.client.raw.DeleteFilterWithResponse(ctx, f.project.projectName, id)

	return execute[types.OperationCompletionRS](ctx, logger, "delete filter", resp, err)
}

// GetByIds returns multiple filters by their numeric IDs.
func (f *UserFilterScope) GetByIds(
	ctx context.Context,
	ids []int64,
) ([]types.UserFilterResource, error) {
	logger := f.project.operationLogger("GetFiltersByIds")
	logger.InfoContext(ctx, "API request")

	resp, err := f.project.client.raw.GetUserFiltersWithResponse(
		ctx,
		f.project.projectName,
		&api.GetUserFiltersParams{
			Ids: ids,
		},
	)

	result, err := execute[[]types.UserFilterResource](ctx, logger, "get filters by ids", resp, err)
	if err != nil {
		return nil, err
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

	resp, err := f.project.client.raw.GetAllFiltersNamesWithResponse(ctx, f.project.projectName, params)

	result, err := execute[types.PageOwnedEntityResource](ctx, logger, "get filter names", resp, err)
	if err != nil {
		return nil, err
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

// BulkUpdate updates multiple filters in a single API call.
func (f *UserFilterScope) BulkUpdate(
	ctx context.Context,
	request types.CollectionsRQBulkUpdateFilterRQ,
) ([]types.OperationCompletionRS, error) {
	logger := f.project.operationLogger("BulkUpdateFilters")
	logger.InfoContext(ctx, "API request")

	resp, err := f.project.client.raw.UpdateUserFiltersWithResponse(ctx, f.project.projectName, request)

	result, err := execute[[]types.OperationCompletionRS](ctx, logger, "bulk update filters", resp, err)
	if err != nil {
		return nil, err
	}

	if result == nil || *result == nil {
		return []types.OperationCompletionRS{}, nil
	}

	return *result, nil
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
