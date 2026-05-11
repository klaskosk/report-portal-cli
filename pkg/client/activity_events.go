package client

import (
	"context"
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ActivityEventsScope provides read-only access to instance-level activity events.
// Unlike ActivityScope (project-scoped test item activities), this scope searches
// across all projects and supports structured criteria-based queries for auditing
// and administrative purposes.
type ActivityEventsScope struct {
	client *Client
}

// Search queries activity events using structured search criteria and pagination.
func (a *ActivityEventsScope) Search(
	ctx context.Context,
	criteria types.SearchCriteriaRQ,
	options ...SearchActivityEventsOption,
) (*types.PagedResponseActivityEventResource, error) {
	logger := a.client.logger.With(slog.String("operation", "SearchActivityEvents"))

	params := &api.GetActivitiesParams{
		Limit:  20,
		Offset: 0,
		Order:  api.GetActivitiesParamsOrder(EventOrderDesc),
		Sort:   "createdAt",
	}
	for _, option := range options {
		option.applySearchActivityEvents(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := a.client.raw.GetActivitiesWithResponse(ctx, params, criteria)

	return execute[types.PagedResponseActivityEventResource](
		ctx,
		logger,
		"search activity events",
		resp,
		err,
	)
}

// GetProjectSubjectNames returns the subject names (users) that have generated
// activity events for the given project. This is useful for building search UIs
// or autocomplete fields.
func (a *ActivityEventsScope) GetProjectSubjectNames(
	ctx context.Context,
	projectName string,
) ([]string, error) {
	logger := a.client.logger.With(
		slog.String("operation", "GetProjectSubjectNames"),
		slog.String("project", projectName),
	)
	logger.InfoContext(ctx, "API request")

	params := &api.GetProjectSubjectNameParams{}

	resp, err := a.client.raw.GetProjectSubjectNameWithResponse(ctx, projectName, params)

	result, err := execute[[]string](ctx, logger, "get project subject names", resp, err)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []string{}, nil
	}

	return *result, nil
}

// SearchActivityEventsOption configures pagination parameters for searching activity events.
type SearchActivityEventsOption interface {
	applySearchActivityEvents(params *api.GetActivitiesParams)
}

type searchActivityEventsOptionFunc func(*api.GetActivitiesParams)

func (f searchActivityEventsOptionFunc) applySearchActivityEvents(params *api.GetActivitiesParams) {
	f(params)
}

// EventSortOrder defines the sort direction for activity event queries.
type EventSortOrder string

// WithEventLimit sets the maximum number of results to return.
func WithEventLimit(limit int32) searchActivityEventsOptionFunc {
	return searchActivityEventsOptionFunc(func(params *api.GetActivitiesParams) {
		params.Limit = limit
	})
}

// WithEventOffset sets the offset for paginated results.
func WithEventOffset(offset int32) searchActivityEventsOptionFunc {
	return searchActivityEventsOptionFunc(func(params *api.GetActivitiesParams) {
		params.Offset = offset
	})
}

// WithEventSort sets the field to sort results by (e.g. "createdAt").
func WithEventSort(field string) searchActivityEventsOptionFunc {
	return searchActivityEventsOptionFunc(func(params *api.GetActivitiesParams) {
		params.Sort = field
	})
}

// WithEventOrder sets the sort direction for activity event results.
func WithEventOrder(order EventSortOrder) searchActivityEventsOptionFunc {
	return searchActivityEventsOptionFunc(func(params *api.GetActivitiesParams) {
		params.Order = api.GetActivitiesParamsOrder(order)
	})
}

// EventOrderAsc is the ascending sort order for activity event queries.
const EventOrderAsc EventSortOrder = EventSortOrder(api.ASC)

// EventOrderDesc is the descending sort order for activity event queries.
const EventOrderDesc EventSortOrder = EventSortOrder(api.DESC)

// NewEventSearchCriteria creates a SearchCriteriaRQ with the given filter conditions.
func NewEventSearchCriteria(criteria ...types.SearchCriteria) types.SearchCriteriaRQ {
	return types.SearchCriteriaRQ{
		SearchCriterias: criteria,
	}
}

// EventFilterByUser creates a search filter matching events by subject name.
func EventFilterByUser(username string) types.SearchCriteria {
	return newEqualityFilter("subjectName", username)
}

// EventFilterByObjectType creates a search filter matching events by object type.
func EventFilterByObjectType(objectType string) types.SearchCriteria {
	return newEqualityFilter("objectType", objectType)
}

// EventFilterByEventName creates a search filter matching events by event name.
func EventFilterByEventName(eventName string) types.SearchCriteria {
	return newEqualityFilter("eventName", eventName)
}

// EventFilterByProjectName creates a search filter matching events by project name.
func EventFilterByProjectName(projectName string) types.SearchCriteria {
	return newEqualityFilter("projectName", projectName)
}

// newEqualityFilter creates a SearchCriteria with an equality operation.
// This helper eliminates repetitive filter construction code.
func newEqualityFilter(filterKey, value string) types.SearchCriteria {
	equalityOperation := types.EQ

	return types.SearchCriteria{
		FilterKey: filterKey,
		Operation: &equalityOperation,
		Value:     value,
	}
}
