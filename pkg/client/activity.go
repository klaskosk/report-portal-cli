package client

import (
	"context"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ActivityScope provides read-only access to project-scoped activity logs.
// Activities track user actions on test items within a specific project.
type ActivityScope struct {
	project *ProjectScope
}

// GetByID returns a single activity record by its numeric ID.
func (a *ActivityScope) GetByID(
	ctx context.Context,
	activityID int64,
) (*types.ActivityResource, error) {
	logger := a.project.operationLogger("GetActivityByID", slog.Int64("activityId", activityID))
	logger.InfoContext(ctx, "API request")

	resp, err := a.project.client.raw.GetActivityWithResponse(ctx, a.project.projectName, activityID)

	return execute[types.ActivityResource](ctx, logger, "get activity by id", resp, err)
}

// ListForItem returns a paginated list of activity events for a specific test item.
func (a *ActivityScope) ListForItem(
	ctx context.Context,
	itemID int64,
	options ...ListActivitiesOption,
) (*types.PageActivityEventResource, error) {
	logger := a.project.operationLogger("ListActivitiesForItem", slog.Int64("itemId", itemID))

	params := &api.GetTestItemActivitiesParams{}
	for _, option := range options {
		option.applyListActivities(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := a.project.client.raw.GetTestItemActivitiesWithResponse(ctx, a.project.projectName, itemID, params)

	return execute[types.PageActivityEventResource](ctx, logger, "list activities for item", resp, err)
}

// ListActivitiesOption configures the filter and pagination parameters for listing item activities.
type ListActivitiesOption interface {
	applyListActivities(*api.GetTestItemActivitiesParams)
}

type listActivitiesOptionFunc func(*api.GetTestItemActivitiesParams)

func (f listActivitiesOptionFunc) applyListActivities(params *api.GetTestItemActivitiesParams) {
	f(params)
}

// WithActivityPageNumber sets the page number (0-based) for activity listing.
func WithActivityPageNumber(n int32) ListActivitiesOption {
	return listActivitiesOptionFunc(func(params *api.GetTestItemActivitiesParams) {
		params.PagePage = &n
	})
}

// WithActivityPageSize sets the number of results per page for activity listing.
func WithActivityPageSize(size int32) ListActivitiesOption {
	return listActivitiesOptionFunc(func(params *api.GetTestItemActivitiesParams) {
		params.PageSize = &size
	})
}

// WithActivitySort sets the sorting criteria for activity listing (e.g. "lastModified,desc").
func WithActivitySort(fields ...string) ListActivitiesOption {
	return listActivitiesOptionFunc(func(params *api.GetTestItemActivitiesParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}
