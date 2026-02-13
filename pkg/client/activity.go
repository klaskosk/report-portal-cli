package client

import (
	"context"
	"fmt"
	"log/slog"

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

	response, err := a.project.client.raw.GetActivityUsingGETWithResponse(
		ctx,
		a.project.projectName,
		activityID,
	)
	if err != nil {
		return nil, fmt.Errorf("get activity by id: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "get activity by id", response.JSONDefault, response.JSON200)
}

// ListForItem returns a paginated list of activity events for a specific test item.
func (a *ActivityScope) ListForItem(
	ctx context.Context,
	itemID int64,
	options ...ListActivitiesOption,
) (*types.IterableActivityEventResource, error) {
	logger := a.project.operationLogger("ListActivitiesForItem", slog.Int64("itemId", itemID))

	params := &api.GetTestItemActivitiesUsingGETParams{}
	for _, option := range options {
		option(params)
	}

	logger.InfoContext(ctx, "API request")

	response, err := a.project.client.raw.GetTestItemActivitiesUsingGETWithResponse(
		ctx,
		a.project.projectName,
		itemID,
		params,
	)
	if err != nil {
		return nil, fmt.Errorf("list activities for item: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	return handleResponse(response, "list activities for item", response.JSONDefault, response.JSON200)
}

// ListActivitiesOption configures the filter and pagination parameters for listing item activities.
type ListActivitiesOption func(*api.GetTestItemActivitiesUsingGETParams)

// WithActivityPageNumber sets the page number (0-based) for activity listing.
func WithActivityPageNumber(n int32) ListActivitiesOption {
	return func(params *api.GetTestItemActivitiesUsingGETParams) {
		params.PagePage = &n
	}
}

// WithActivityPageSize sets the number of results per page for activity listing.
func WithActivityPageSize(size int32) ListActivitiesOption {
	return func(params *api.GetTestItemActivitiesUsingGETParams) {
		params.PageSize = &size
	}
}

// WithActivitySort sets the sorting criteria for activity listing (e.g. "lastModified,desc").
func WithActivitySort(fields ...string) ListActivitiesOption {
	return func(params *api.GetTestItemActivitiesUsingGETParams) {
		params.PageSort = &fields
	}
}
