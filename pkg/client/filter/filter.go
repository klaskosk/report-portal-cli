package filter

import (
	"context"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// UserFilterScope provides operations on user filters (saved filters) within a project.
type UserFilterScope struct {
	scope core.Scope
}

// List returns a paginated list of filters matching the given options.
func (f *UserFilterScope) List(
	ctx context.Context,
	options ...ListFiltersOption,
) (*types.PageUserFilterResource, error) {
	logger := f.scope.OperationLogger("ListFilters")

	params := &api.GetAllFiltersParams{}
	for _, option := range options {
		option.applyListFilters(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := f.scope.Client.GetAllFiltersWithResponse(ctx, f.scope.ProjectName, params)

	return core.Execute[types.PageUserFilterResource](ctx, logger, "list filters", resp, err)
}

// GetByID returns a single filter by its numeric ID.
func (f *UserFilterScope) GetByID(
	ctx context.Context,
	filterID int64,
) (*types.UserFilterResource, error) {
	logger := f.scope.OperationLogger("GetFilterByID", slog.Int64("id", filterID))
	logger.InfoContext(ctx, "API request")

	resp, err := f.scope.Client.GetFilterWithResponse(ctx, f.scope.ProjectName, filterID)

	return core.Execute[types.UserFilterResource](ctx, logger, "get filter by id", resp, err)
}

// Create creates a new filter in the project. The API uses UpdateUserFilterRQ as the
// create request type.
func (f *UserFilterScope) Create(
	ctx context.Context,
	request types.UpdateUserFilterRQ,
) (*types.EntryCreatedRS, error) {
	logger := f.scope.OperationLogger("CreateFilter")
	logger.InfoContext(ctx, "API request")

	resp, err := f.scope.Client.CreateFilterWithResponse(ctx, f.scope.ProjectName, request)

	return core.Execute[types.EntryCreatedRS](ctx, logger, "create filter", resp, err)
}

// Update modifies an existing filter.
func (f *UserFilterScope) Update(
	ctx context.Context,
	filterID int64,
	request types.UpdateUserFilterRQ,
) (*types.OperationCompletionRS, error) {
	logger := f.scope.OperationLogger("UpdateFilter", slog.Int64("id", filterID))
	logger.InfoContext(ctx, "API request")

	resp, err := f.scope.Client.UpdateUserFilterWithResponse(
		ctx,
		f.scope.ProjectName,
		filterID,
		request,
	)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "update filter", resp, err)
}

// Delete removes a filter by its numeric ID.
func (f *UserFilterScope) Delete(
	ctx context.Context,
	filterID int64,
) (*types.OperationCompletionRS, error) {
	logger := f.scope.OperationLogger("DeleteFilter", slog.Int64("id", filterID))
	logger.InfoContext(ctx, "API request")

	resp, err := f.scope.Client.DeleteFilterWithResponse(ctx, f.scope.ProjectName, filterID)

	return core.Execute[types.OperationCompletionRS](ctx, logger, "delete filter", resp, err)
}

// GetByIDs returns multiple filters by their numeric IDs.
func (f *UserFilterScope) GetByIDs(
	ctx context.Context,
	ids []int64,
) ([]types.UserFilterResource, error) {
	logger := f.scope.OperationLogger("GetFiltersByIds")
	logger.InfoContext(ctx, "API request")

	resp, err := f.scope.Client.GetUserFiltersWithResponse(
		ctx,
		f.scope.ProjectName,
		&api.GetUserFiltersParams{
			Ids: ids,
		},
	)

	result, err := core.Execute[[]types.UserFilterResource](
		ctx,
		logger,
		"get filters by ids",
		resp,
		err,
	)
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
	logger := f.scope.OperationLogger("GetFilterNames", slog.String("filter", filter))

	params := &api.GetAllFiltersNamesParams{}
	if filter != "" {
		params.FilterEqName = &filter
	}

	logger.InfoContext(ctx, "API request")

	resp, err := f.scope.Client.GetAllFiltersNamesWithResponse(
		ctx,
		f.scope.ProjectName,
		params,
	)

	result, err := core.Execute[types.PageOwnedEntityResource](
		ctx,
		logger,
		"get filter names",
		resp,
		err,
	)
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
	logger := f.scope.OperationLogger("BulkUpdateFilters")
	logger.InfoContext(ctx, "API request")

	resp, err := f.scope.Client.UpdateUserFiltersWithResponse(
		ctx,
		f.scope.ProjectName,
		request,
	)

	result, err := core.Execute[[]types.OperationCompletionRS](
		ctx,
		logger,
		"bulk update filters",
		resp,
		err,
	)
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
	applyListFilters(params *api.GetAllFiltersParams)
}

type listFiltersOptionFunc func(*api.GetAllFiltersParams)

func (f listFiltersOptionFunc) applyListFilters(params *api.GetAllFiltersParams) {
	f(params)
}

// WithFilterPageNumber sets the page number (0-based) for filter listing.
func WithFilterPageNumber(n int32) listFiltersOptionFunc {
	return listFiltersOptionFunc(func(params *api.GetAllFiltersParams) {
		params.PagePage = &n
	})
}

// WithFilterPageSize sets the number of results per page for filter listing.
func WithFilterPageSize(size int32) listFiltersOptionFunc {
	return listFiltersOptionFunc(func(params *api.GetAllFiltersParams) {
		params.PageSize = &size
	})
}

// WithFilterSort sets the sorting criteria for filter listing (e.g. "name,asc").
func WithFilterSort(fields ...string) listFiltersOptionFunc {
	return listFiltersOptionFunc(func(params *api.GetAllFiltersParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// NewUserFilterScope constructs a UserFilterScope for wiring from the facade.
func NewUserFilterScope(s core.Scope) *UserFilterScope {
	return &UserFilterScope{scope: s}
}
