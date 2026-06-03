package activity

import (
	"context"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client/internal/core"
	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ActivityScope provides read-only access to project-scoped activity logs.
// Activities track user actions on test items within a specific project.
type ActivityScope struct {
	scope core.Scope
}

// GetByID returns a single activity record by its numeric ID.
func (a *ActivityScope) GetByID(
	ctx context.Context,
	activityID int64,
) (*types.ActivityResource, error) {
	logger := a.scope.OperationLogger("GetActivityByID", slog.Int64("activityId", activityID))
	logger.InfoContext(ctx, "API request")

	resp, err := a.scope.Client.GetActivityWithResponse(
		ctx,
		a.scope.ProjectName,
		activityID,
	)

	return core.Execute[types.ActivityResource](ctx, logger, "get activity by id", resp, err)
}

// ListForItem returns a paginated list of activity events for a specific test item.
func (a *ActivityScope) ListForItem(
	ctx context.Context,
	itemID int64,
	options ...ListActivitiesOption,
) (*types.PageActivityEventResource, error) {
	logger := a.scope.OperationLogger("ListActivitiesForItem", slog.Int64("itemId", itemID))

	params := &api.GetTestItemActivitiesParams{}
	for _, option := range options {
		option.applyListActivities(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := a.scope.Client.GetTestItemActivitiesWithResponse(
		ctx,
		a.scope.ProjectName,
		itemID,
		params,
	)

	return core.Execute[types.PageActivityEventResource](
		ctx,
		logger,
		"list activities for item",
		resp,
		err,
	)
}

// ListActivitiesOption configures the filter and pagination parameters for listing item activities.
type ListActivitiesOption interface {
	applyListActivities(params *api.GetTestItemActivitiesParams)
}

type listActivitiesOptionFunc func(*api.GetTestItemActivitiesParams)

func (f listActivitiesOptionFunc) applyListActivities(params *api.GetTestItemActivitiesParams) {
	f(params)
}

// WithActivityPageNumber sets the page number (0-based) for activity listing.
func WithActivityPageNumber(n int32) listActivitiesOptionFunc {
	return listActivitiesOptionFunc(func(params *api.GetTestItemActivitiesParams) {
		params.PagePage = &n
	})
}

// WithActivityPageSize sets the number of results per page for activity listing.
func WithActivityPageSize(size int32) listActivitiesOptionFunc {
	return listActivitiesOptionFunc(func(params *api.GetTestItemActivitiesParams) {
		params.PageSize = &size
	})
}

// WithActivitySort sets the sorting criteria for activity listing (e.g. "lastModified,desc").
func WithActivitySort(fields ...string) listActivitiesOptionFunc {
	return listActivitiesOptionFunc(func(params *api.GetTestItemActivitiesParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}
