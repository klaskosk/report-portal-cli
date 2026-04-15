package client

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/internal/api"
	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// TestItemScope provides read operations on test items within a project.
type TestItemScope struct {
	project *ProjectScope
}

// StartRoot creates a root test item under an existing launch.
func (i *TestItemScope) StartRoot(
	ctx context.Context,
	request types.StartTestItemRQ,
) (*types.EntryCreatedAsyncRS, error) {
	logger := i.project.operationLogger("StartRootTestItem", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.StartRootItemWithResponse(ctx, i.project.projectName, request)

	return execute[types.EntryCreatedAsyncRS](ctx, logger, "start root test item", resp, err)
}

// StartChild creates a child test item under the given parent item's UUID as
// returned by StartRoot or StartChild.
func (i *TestItemScope) StartChild(
	ctx context.Context,
	parentItemUUID string,
	request types.StartTestItemRQ,
) (*types.EntryCreatedAsyncRS, error) {
	logger := i.project.operationLogger(
		"StartChildTestItem",
		slog.String("parentItemUuid", parentItemUUID),
		slog.String("name", request.Name),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.StartChildItem2WithResponse(ctx, i.project.projectName, parentItemUUID, request)

	return execute[types.EntryCreatedAsyncRS](ctx, logger, "start child test item", resp, err)
}

// Finish marks an existing test item as finished using the item UUID returned by
// StartRoot or StartChild.
func (i *TestItemScope) Finish(
	ctx context.Context,
	itemUUID string,
	request types.FinishTestItemRQ,
) (*types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("FinishTestItem", slog.String("itemUuid", itemUUID))
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.FinishTestItem2WithResponse(ctx, i.project.projectName, itemUUID, request)

	return execute[types.OperationCompletionRS](ctx, logger, "finish test item", resp, err)
}

// StartRootAsync creates a root test item under an existing launch using the
// asynchronous v2 API endpoint. The server accepts the request immediately and
// processes it in the background.
func (i *TestItemScope) StartRootAsync(
	ctx context.Context,
	request types.StartTestItemRQ,
) (*types.EntryCreatedAsyncRS, error) {
	logger := i.project.operationLogger("StartRootTestItemAsync", slog.String("name", request.Name))
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.StartRootItem1WithResponse(ctx, i.project.projectName, request)

	return execute[types.EntryCreatedAsyncRS](ctx, logger, "start root test item async", resp, err)
}

// StartChildAsync creates a child test item under the given parent item's UUID
// using the asynchronous v2 API endpoint. The server accepts the request
// immediately and processes it in the background.
func (i *TestItemScope) StartChildAsync(
	ctx context.Context,
	parentItemUUID string,
	request types.StartTestItemRQ,
) (*types.EntryCreatedAsyncRS, error) {
	logger := i.project.operationLogger(
		"StartChildTestItemAsync",
		slog.String("parentItemUuid", parentItemUUID),
		slog.String("name", request.Name),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.StartChildItemWithResponse(ctx, i.project.projectName, parentItemUUID, request)

	return execute[types.EntryCreatedAsyncRS](ctx, logger, "start child test item async", resp, err)
}

// FinishAsync marks an existing test item as finished using the asynchronous v2
// API endpoint. The server accepts the request immediately and processes it in
// the background.
func (i *TestItemScope) FinishAsync(
	ctx context.Context,
	itemUUID string,
	request types.FinishTestItemRQ,
) (*types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("FinishTestItemAsync", slog.String("itemUuid", itemUUID))
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.FinishTestItemWithResponse(ctx, i.project.projectName, itemUUID, request)

	return execute[types.OperationCompletionRS](ctx, logger, "finish test item async", resp, err)
}

// GetByID returns a single test item by its numeric ID.
func (i *TestItemScope) GetByID(
	ctx context.Context,
	id int64,
) (*types.TestItemResource, error) {
	logger := i.project.operationLogger("GetTestItemByID", slog.Int64("id", id))
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetTestItemWithResponse(ctx, i.project.projectName, id)

	return execute[types.TestItemResource](ctx, logger, "get test item by id", resp, err)
}

// GetByUUID returns a single test item by its UUID. It prefers the versioned v2
// content type so timestamp fields arrive as ISO-8601 strings, but still accepts
// the legacy JSON shape as a fallback if the server ignores the Accept header.
func (i *TestItemScope) GetByUUID(
	ctx context.Context,
	uuid string,
) (*types.TestItemResource, error) {
	logger := i.project.operationLogger("GetTestItemByUUID", slog.String("uuid", uuid))
	logger.InfoContext(ctx, "API request")

	response, err := i.project.client.raw.GetTestItemByUuidTimestampWithResponse(
		ctx,
		i.project.projectName,
		uuid,
		withAcceptHeader(acceptReportPortalTestV2),
	)
	if err != nil {
		return nil, fmt.Errorf("get test item by uuid: %w", err)
	}

	logger.DebugContext(ctx, "API response", "status", response.StatusCode())

	result, err := handleV2Response(
		response,
		response.JSONDefault,
		response.ApplicationXReportportalTestV2JSON200,
		response.JSON200,
		response.Body,
	)
	if err != nil {
		return nil, fmt.Errorf("get test item by uuid: %w", err)
	}

	return result, nil
}

// ListTestItemsOption configures filter and pagination parameters for listing test items.
type ListTestItemsOption interface {
	applyListTestItems(*api.GetTestItemsParams)
}

type listTestItemsOptionFunc func(*api.GetTestItemsParams)

func (f listTestItemsOptionFunc) applyListTestItems(params *api.GetTestItemsParams) {
	f(params)
}

// List returns a paginated list of test items matching the given filter options.
func (i *TestItemScope) List(
	ctx context.Context,
	options ...ListTestItemsOption,
) (*types.PageTestItemResource, error) {
	logger := i.project.operationLogger("ListTestItems")

	params := &api.GetTestItemsParams{}
	for _, option := range options {
		option.applyListTestItems(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetTestItemsWithResponse(ctx, i.project.projectName, params)

	return execute[types.PageTestItemResource](ctx, logger, "list test items", resp, err)
}

// GetByIDs returns test items for the given numeric IDs.
func (i *TestItemScope) GetByIDs(
	ctx context.Context,
	ids ...int64,
) ([]types.TestItemResource, error) {
	logger := i.project.operationLogger("GetTestItemsByIDs")

	if len(ids) == 0 {
		return []types.TestItemResource{}, nil
	}

	params := &api.GetTestItems3Params{Ids: ids}

	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetTestItems3WithResponse(ctx, i.project.projectName, params)

	items, err := execute[[]types.TestItemResource](ctx, logger, "get test items by ids", resp, err)
	if err != nil {
		return nil, err
	}

	if items == nil {
		return []types.TestItemResource{}, nil
	}

	return *items, nil
}

// Update modifies a single test item's attributes, description, or status.
func (i *TestItemScope) Update(
	ctx context.Context,
	itemID int64,
	request types.UpdateTestItemRQ,
) (*types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("UpdateTestItem", slog.Int64("itemId", itemID))
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.UpdateTestItemWithResponse(ctx, i.project.projectName, itemID, request)

	return execute[types.OperationCompletionRS](ctx, logger, "update test item", resp, err)
}

// Delete removes a single test item by its numeric ID.
func (i *TestItemScope) Delete(
	ctx context.Context,
	itemID int64,
) (*types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("DeleteTestItem", slog.Int64("itemId", itemID))
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.DeleteTestItemWithResponse(ctx, i.project.projectName, itemID)

	return execute[types.OperationCompletionRS](ctx, logger, "delete test item", resp, err)
}

// DefineIssueType bulk-updates the defect classification on multiple test items.
func (i *TestItemScope) DefineIssueType(
	ctx context.Context,
	request types.DefineIssueRQ,
) ([]types.Issue, error) {
	logger := i.project.operationLogger("DefineTestItemIssueType")
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.DefineTestItemIssueTypeWithResponse(ctx, i.project.projectName, request)

	issues, err := execute[[]types.Issue](ctx, logger, "define test item issue type", resp, err)
	if err != nil {
		return nil, err
	}

	if issues == nil {
		return []types.Issue{}, nil
	}

	return *issues, nil
}

// DeleteBulk removes multiple test items by their numeric IDs.
func (i *TestItemScope) DeleteBulk(
	ctx context.Context,
	ids []int64,
) ([]types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("DeleteTestItemsBulk")
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.DeleteTestItemsWithResponse(ctx, i.project.projectName, &api.DeleteTestItemsParams{Ids: ids})

	results, err := execute[[]types.OperationCompletionRS](ctx, logger, "delete test items bulk", resp, err)
	if err != nil {
		return nil, err
	}

	if results == nil {
		return []types.OperationCompletionRS{}, nil
	}

	return *results, nil
}

// BulkUpdate modifies attributes and/or description for multiple test items at once.
func (i *TestItemScope) BulkUpdate(
	ctx context.Context,
	request types.BulkInfoUpdateRQ,
) (*types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("BulkUpdateTestItems")
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.BulkUpdate1WithResponse(ctx, i.project.projectName, request)

	return execute[types.OperationCompletionRS](ctx, logger, "bulk update test items", resp, err)
}

// GetAttributeKeys returns unique test item attribute keys within a specific launch.
// keySubstring is sent as filter.cnt.attributeKey (substring match for autocomplete).
// Use an empty string when no substring filter is needed.
func (i *TestItemScope) GetAttributeKeys(
	ctx context.Context,
	launchID int64,
	keySubstring string,
) ([]string, error) {
	logger := i.project.operationLogger(
		"GetItemAttributeKeys",
		slog.Int64("launchId", launchID),
		slog.String("filter", keySubstring),
	)
	logger.InfoContext(ctx, "API request")

	launch := launchID
	params := &api.GetUniqueAttributeKeysParams{
		FilterCntAttributeKey: keySubstring,
		Launch:                &launch,
	}

	resp, err := i.project.client.raw.GetUniqueAttributeKeysWithResponse(ctx, i.project.projectName, params)

	keys, err := execute[[]string](ctx, logger, "get item attribute keys", resp, err)
	if err != nil {
		return nil, err
	}

	if keys == nil {
		return []string{}, nil
	}

	return *keys, nil
}

// GetAttributeKeysForProjectOption configures optional parameters for
// GetAttributeKeysForProject.
type GetAttributeKeysForProjectOption interface {
	applyGetAttributeKeysForProject(*api.GetAttributeKeysForProjectParams)
}

type getAttributeKeysForProjectOptionFunc func(*api.GetAttributeKeysForProjectParams)

func (f getAttributeKeysForProjectOptionFunc) applyGetAttributeKeysForProject(
	params *api.GetAttributeKeysForProjectParams,
) {
	f(params)
}

// WithProjectKeysLatest restricts results to the latest launches only.
func WithProjectKeysLatest(isLatest bool) GetAttributeKeysForProjectOption {
	return getAttributeKeysForProjectOptionFunc(func(params *api.GetAttributeKeysForProjectParams) {
		params.IsLatest = &isLatest
	})
}

// WithProjectKeysLaunchesLimit sets the maximum number of launches to scan.
func WithProjectKeysLaunchesLimit(limit int32) GetAttributeKeysForProjectOption {
	return getAttributeKeysForProjectOptionFunc(func(params *api.GetAttributeKeysForProjectParams) {
		params.LaunchesLimit = &limit
	})
}

// GetAttributeKeysForProject returns unique test item attribute keys across the project,
// scoped to a saved user filter. keySubstring is sent as filter.cnt.attributeKey (substring
// match for autocomplete). Use an empty string when no substring filter is needed.
func (i *TestItemScope) GetAttributeKeysForProject(
	ctx context.Context,
	filterID int64,
	keySubstring string,
	options ...GetAttributeKeysForProjectOption,
) ([]string, error) {
	logger := i.project.operationLogger(
		"GetItemAttributeKeysForProject",
		slog.Int64("filterId", filterID),
		slog.String("filter", keySubstring),
	)
	logger.InfoContext(ctx, "API request")

	params := &api.GetAttributeKeysForProjectParams{
		FilterCntAttributeKey: keySubstring,
		FilterId:              filterID,
	}
	for _, option := range options {
		option.applyGetAttributeKeysForProject(params)
	}

	resp, err := i.project.client.raw.GetAttributeKeysForProjectWithResponse(ctx, i.project.projectName, params)

	keys, err := execute[[]string](ctx, logger, "get item attribute keys for project", resp, err)
	if err != nil {
		return nil, err
	}

	if keys == nil {
		return []string{}, nil
	}

	return *keys, nil
}

// GetAttributeValues returns unique test item attribute values within a specific launch.
// valueSubstring is sent as filter.cnt.attributeValue (substring match for autocomplete).
// When attributeKey is non-nil, results are limited to values for that key
// (filter.eq.attributeKey).
func (i *TestItemScope) GetAttributeValues(
	ctx context.Context,
	launchID int64,
	valueSubstring string,
	attributeKey *string,
) ([]string, error) {
	logger := i.project.operationLogger(
		"GetItemAttributeValues",
		slog.Int64("launchId", launchID),
		slog.String("valueFilter", valueSubstring),
	)
	if attributeKey != nil {
		logger = logger.With(slog.String("attributeKey", *attributeKey))
	}

	logger.InfoContext(ctx, "API request")

	launch := launchID
	params := &api.GetAttributeValues2Params{
		FilterCntAttributeValue: valueSubstring,
		FilterEqAttributeKey:    attributeKey,
		Launch:                  &launch,
	}

	resp, err := i.project.client.raw.GetAttributeValues2WithResponse(ctx, i.project.projectName, params)

	values, err := execute[[]string](ctx, logger, "get item attribute values", resp, err)
	if err != nil {
		return nil, err
	}

	if values == nil {
		return []string{}, nil
	}

	return *values, nil
}

// GetStepAttributeKeys returns unique step-level attribute keys. keySubstring is sent as
// filter.cnt.attributeKey (substring match for autocomplete). When stepName is non-nil,
// results are limited to steps with that name (filter.eq.name).
func (i *TestItemScope) GetStepAttributeKeys(
	ctx context.Context,
	keySubstring string,
	stepName *string,
) ([]string, error) {
	logger := i.project.operationLogger(
		"GetStepAttributeKeys",
		slog.String("filter", keySubstring),
	)
	if stepName != nil {
		logger = logger.With(slog.String("stepName", *stepName))
	}

	logger.InfoContext(ctx, "API request")

	params := &api.GetAttributeKeys1Params{
		FilterCntAttributeKey: keySubstring,
		FilterEqName:          stepName,
	}

	resp, err := i.project.client.raw.GetAttributeKeys1WithResponse(ctx, i.project.projectName, params)

	keys, err := execute[[]string](ctx, logger, "get step attribute keys", resp, err)
	if err != nil {
		return nil, err
	}

	if keys == nil {
		return []string{}, nil
	}

	return *keys, nil
}

// GetStepAttributeValues returns unique step-level attribute values. valueSubstring is
// sent as filter.cnt.attributeValue (substring match for autocomplete). When attributeKey
// is non-nil, results are limited to values for that key (filter.eq.attributeKey). When
// stepName is non-nil, results are limited to steps with that name (filter.eq.name).
func (i *TestItemScope) GetStepAttributeValues(
	ctx context.Context,
	valueSubstring string,
	attributeKey *string,
	stepName *string,
) ([]string, error) {
	logger := i.project.operationLogger(
		"GetStepAttributeValues",
		slog.String("valueFilter", valueSubstring),
	)
	if attributeKey != nil {
		logger = logger.With(slog.String("attributeKey", *attributeKey))
	}

	if stepName != nil {
		logger = logger.With(slog.String("stepName", *stepName))
	}

	logger.InfoContext(ctx, "API request")

	params := &api.GetAttributeValues1Params{
		FilterCntAttributeValue: valueSubstring,
		FilterEqAttributeKey:    attributeKey,
		FilterEqName:            stepName,
	}

	resp, err := i.project.client.raw.GetAttributeValues1WithResponse(ctx, i.project.projectName, params)

	values, err := execute[[]string](ctx, logger, "get step attribute values", resp, err)
	if err != nil {
		return nil, err
	}

	if values == nil {
		return []string{}, nil
	}

	return *values, nil
}

// SearchTestItemsOption configures filter and pagination parameters for searching test items.
type SearchTestItemsOption interface {
	applySearchTestItems(*api.GetTestItems2Params)
}

type searchTestItemsOptionFunc func(*api.GetTestItems2Params)

func (f searchTestItemsOptionFunc) applySearchTestItems(params *api.GetTestItems2Params) {
	f(params)
}

// Search returns a paginated list of test items matching the given search options.
// It uses the /item/search endpoint which supports simplified filter parameters.
func (i *TestItemScope) Search(
	ctx context.Context,
	options ...SearchTestItemsOption,
) (*types.PageTestItemResource, error) {
	logger := i.project.operationLogger("SearchTestItems")

	params := &api.GetTestItems2Params{}
	for _, option := range options {
		option.applySearchTestItems(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetTestItems2WithResponse(ctx, i.project.projectName, params)

	return execute[types.PageTestItemResource](ctx, logger, "search test items", resp, err)
}

// WithSearchItemName filters search results by a contains-match on item name.
func WithSearchItemName(name string) SearchTestItemsOption {
	return searchTestItemsOptionFunc(func(params *api.GetTestItems2Params) {
		params.FilterCntName = &name
	})
}

// WithSearchItemAttribute filters search results by a composite attribute.
func WithSearchItemAttribute(attr string) SearchTestItemsOption {
	return searchTestItemsOptionFunc(func(params *api.GetTestItems2Params) {
		params.FilterHasCompositeAttribute = &attr
	})
}

// WithSearchItemStatus filters search results by status (using an "in" match).
func WithSearchItemStatus(status string) SearchTestItemsOption {
	return searchTestItemsOptionFunc(func(params *api.GetTestItems2Params) {
		params.FilterInStatus = &status
	})
}

// WithSearchItemPageNumber sets the page number (0-based) for search results.
func WithSearchItemPageNumber(page int32) SearchTestItemsOption {
	return searchTestItemsOptionFunc(func(params *api.GetTestItems2Params) {
		params.PagePage = &page
	})
}

// WithSearchItemPageSize sets the number of results per page for search results.
func WithSearchItemPageSize(size int32) SearchTestItemsOption {
	return searchTestItemsOptionFunc(func(params *api.GetTestItems2Params) {
		params.PageSize = &size
	})
}

// WithSearchItemSort sets the sorting criteria for search results (e.g. "startTime,desc").
func WithSearchItemSort(sort string) SearchTestItemsOption {
	return searchTestItemsOptionFunc(func(params *api.GetTestItems2Params) {
		params.PageSort = &sort
	})
}

// ListTestItemsV2Option configures filter and pagination parameters for listing
// test items via the v2 endpoint.
type ListTestItemsV2Option interface {
	applyListTestItemsV2(*api.GetTestItemsV2Params)
}

type listTestItemsV2OptionFunc func(*api.GetTestItemsV2Params)

func (f listTestItemsV2OptionFunc) applyListTestItemsV2(params *api.GetTestItemsV2Params) {
	f(params)
}

// ListV2 returns a paginated list of test items using the v2 endpoint with
// extended filter parameters.
func (i *TestItemScope) ListV2(
	ctx context.Context,
	options ...ListTestItemsV2Option,
) (*types.PageTestItemResource, error) {
	logger := i.project.operationLogger("ListTestItemsV2")

	params := &api.GetTestItemsV2Params{Params: map[string]string{}}
	for _, option := range options {
		option.applyListTestItemsV2(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetTestItemsV2WithResponse(ctx, i.project.projectName, params)

	return execute[types.PageTestItemResource](ctx, logger, "list test items v2", resp, err)
}

// WithItemV2PageNumber sets the page number (0-based) for v2 list results.
func WithItemV2PageNumber(page int32) ListTestItemsV2Option {
	return listTestItemsV2OptionFunc(func(params *api.GetTestItemsV2Params) {
		params.PagePage = &page
	})
}

// WithItemV2PageSize sets the number of results per page for v2 list results.
func WithItemV2PageSize(size int32) ListTestItemsV2Option {
	return listTestItemsV2OptionFunc(func(params *api.GetTestItemsV2Params) {
		params.PageSize = &size
	})
}

// WithItemV2Sort sets the sorting criteria for v2 list results (e.g. "startTime,desc").
func WithItemV2Sort(sort string) ListTestItemsV2Option {
	return listTestItemsV2OptionFunc(func(params *api.GetTestItemsV2Params) {
		params.PageSort = &sort
	})
}

// WithItemV2Status filters v2 list results by status.
func WithItemV2Status(status string) ListTestItemsV2Option {
	return listTestItemsV2OptionFunc(func(params *api.GetTestItemsV2Params) {
		params.FilterEqStatus = &status
	})
}

// WithItemV2Name filters v2 list results by name.
func WithItemV2Name(name string) ListTestItemsV2Option {
	return listTestItemsV2OptionFunc(func(params *api.GetTestItemsV2Params) {
		params.FilterEqName = &name
	})
}

// WithItemV2LaunchID filters v2 list results by launch ID.
func WithItemV2LaunchID(id int32) ListTestItemsV2Option {
	return listTestItemsV2OptionFunc(func(params *api.GetTestItemsV2Params) {
		params.FilterEqLaunchId = &id
	})
}

// WithItemV2ParentID filters v2 list results by parent item ID.
func WithItemV2ParentID(id int32) ListTestItemsV2Option {
	return listTestItemsV2OptionFunc(func(params *api.GetTestItemsV2Params) {
		params.FilterEqParentId = &id
	})
}

// WithItemPageNumber sets the page number (0-based).
func WithItemPageNumber(n int32) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		params.PagePage = &n
	})
}

// WithItemPageSize sets the number of results per page.
func WithItemPageSize(size int32) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		params.PageSize = &size
	})
}

// WithItemSort sets the sorting criteria (e.g. "startTime,desc").
func WithItemSort(fields ...string) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		if len(fields) == 0 {
			params.PageSort = nil

			return
		}

		s := strings.Join(fields, ",")
		params.PageSort = &s
	})
}

// WithItemLaunchID filters test items by launch ID.
func WithItemLaunchID(launchID int64) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		id := int32(launchID)
		params.FilterEqLaunchId = &id
	})
}

// WithItemParentID filters test items by parent item ID.
func WithItemParentID(parentID int64) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		id := int32(parentID)
		params.FilterEqParentId = &id
	})
}

// WithItemType filters test items by type (e.g. "SUITE", "TEST", "STEP").
func WithItemType(itemType string) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		params.FilterEqType = &itemType
	})
}

// WithItemName filters test items by name.
func WithItemName(name string) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		params.FilterEqName = &name
	})
}

// WithItemStatus filters test items by status.
func WithItemStatus(status string) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		params.FilterEqStatus = &status
	})
}

// WithItemHasChildren filters test items by whether they have children.
func WithItemHasChildren(hasChildren bool) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		params.FilterEqHasChildren = &hasChildren
	})
}

// WithItemPath filters test items by path identifier.
func WithItemPath(path int64) ListTestItemsOption {
	return listTestItemsOptionFunc(func(params *api.GetTestItemsParams) {
		p := int32(path)
		params.FilterEqPath = &p
	})
}

// HistoryOption configures the filter and pagination parameters for retrieving
// test item execution history.
type HistoryOption interface {
	applyHistory(*api.GetItemsHistoryParams)
}

type historyOptionFunc func(*api.GetItemsHistoryParams)

func (f historyOptionFunc) applyHistory(params *api.GetItemsHistoryParams) {
	f(params)
}

// GetHistory returns the execution history for test items matching the given
// options. Each element groups a test item's runs across launches.
func (i *TestItemScope) GetHistory(
	ctx context.Context,
	options ...HistoryOption,
) (*types.PageTestItemHistoryElement, error) {
	logger := i.project.operationLogger("GetItemsHistory")

	params := &api.GetItemsHistoryParams{}
	for _, option := range options {
		option.applyHistory(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetItemsHistoryWithResponse(ctx, i.project.projectName, params)

	return execute[types.PageTestItemHistoryElement](ctx, logger, "get items history", resp, err)
}

// WithHistoryDepth sets the number of previous launches to include in history.
func WithHistoryDepth(depth int32) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.HistoryDepth = &depth
	})
}

// WithHistoryLaunchID filters history by launch ID.
func WithHistoryLaunchID(launchID int64) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		id := int32(launchID)
		params.FilterEqLaunchId = &id
	})
}

// WithHistoryFilterID sets the saved filter ID to scope the history query.
func WithHistoryFilterID(filterID int64) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.FilterId = &filterID
	})
}

// WithHistoryLaunchesLimit sets the maximum number of launches to scan.
func WithHistoryLaunchesLimit(limit int32) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.LaunchesLimit = &limit
	})
}

// WithHistoryType filters history by a specific item execution type
// (e.g. "SUITE", "TEST", "STEP").
func WithHistoryType(historyType string) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.Type = &historyType
	})
}

// WithHistoryItemName filters history entries by test item name.
func WithHistoryItemName(name string) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.FilterEqName = &name
	})
}

// WithHistoryItemStatus filters history entries by test item status.
func WithHistoryItemStatus(status string) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.FilterEqStatus = &status
	})
}

// WithHistoryPageNumber sets the page number (0-based) for history results.
func WithHistoryPageNumber(n int32) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.PagePage = &n
	})
}

// WithHistoryPageSize sets the number of results per page for history results.
func WithHistoryPageSize(size int32) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.PageSize = &size
	})
}

// WithHistoryLatest filters history to only the latest results.
func WithHistoryLatest(isLatest bool) HistoryOption {
	return historyOptionFunc(func(params *api.GetItemsHistoryParams) {
		params.IsLatest = &isLatest
	})
}

// StatisticsOption configures filter parameters for fetching accumulated test item statistics.
type StatisticsOption interface {
	applyStatistics(*api.GetTestItems1Params)
}

type statisticsOptionFunc func(*api.GetTestItems1Params)

func (f statisticsOptionFunc) applyStatistics(params *api.GetTestItems1Params) {
	f(params)
}

// GetStatistics returns accumulated statistics for test items matching the given filter options.
func (i *TestItemScope) GetStatistics(
	ctx context.Context,
	options ...StatisticsOption,
) (*types.StatisticsResource, error) {
	logger := i.project.operationLogger("GetTestItemStatistics")

	params := &api.GetTestItems1Params{Params: map[string]string{}}
	for _, option := range options {
		option.applyStatistics(params)
	}

	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetTestItems1WithResponse(ctx, i.project.projectName, params)

	return execute[types.StatisticsResource](ctx, logger, "get test item statistics", resp, err)
}

// WithStatisticsLaunchID filters statistics by launch ID.
func WithStatisticsLaunchID(launchID int64) StatisticsOption {
	return statisticsOptionFunc(func(params *api.GetTestItems1Params) {
		id := int32(launchID)
		params.FilterEqLaunchId = &id
	})
}

// WithStatisticsParentID filters statistics by parent item ID.
func WithStatisticsParentID(parentID int64) StatisticsOption {
	return statisticsOptionFunc(func(params *api.GetTestItems1Params) {
		id := int32(parentID)
		params.FilterEqParentId = &id
	})
}

// WithStatisticsStatus filters statistics by item status.
func WithStatisticsStatus(status string) StatisticsOption {
	return statisticsOptionFunc(func(params *api.GetTestItems1Params) {
		params.FilterEqStatus = &status
	})
}

// WithStatisticsName filters statistics by item name.
func WithStatisticsName(name string) StatisticsOption {
	return statisticsOptionFunc(func(params *api.GetTestItems1Params) {
		params.FilterEqName = &name
	})
}

// WithStatisticsType filters statistics by item type (e.g. "SUITE", "TEST", "STEP").
func WithStatisticsType(itemType string) StatisticsOption {
	return statisticsOptionFunc(func(params *api.GetTestItems1Params) {
		params.FilterEqType = &itemType
	})
}

// WithStatisticsHasChildren filters statistics by whether items have children.
func WithStatisticsHasChildren(hasChildren bool) StatisticsOption {
	return statisticsOptionFunc(func(params *api.GetTestItems1Params) {
		params.FilterEqHasChildren = &hasChildren
	})
}

// GetSuggestedItems returns ML-suggested test items for the given item ID.
func (i *TestItemScope) GetSuggestedItems(
	ctx context.Context,
	itemID int64,
) ([]types.SuggestedItem, error) {
	logger := i.project.operationLogger("GetSuggestedItems", slog.Int64("itemId", itemID))
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetSuggestedItemsWithResponse(ctx, i.project.projectName, itemID)

	items, err := execute[[]types.SuggestedItem](ctx, logger, "get suggested items", resp, err)
	if err != nil {
		return nil, err
	}

	if items == nil {
		return []types.SuggestedItem{}, nil
	}

	return *items, nil
}

// GetSuggestedClusterItems returns ML-suggested test items for the given cluster ID.
func (i *TestItemScope) GetSuggestedClusterItems(
	ctx context.Context,
	clusterID int64,
) ([]types.SuggestedItem, error) {
	logger := i.project.operationLogger(
		"GetSuggestedClusterItems",
		slog.Int64("clusterId", clusterID),
	)
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.GetSuggestedClusterItemsWithResponse(ctx, i.project.projectName, clusterID)

	items, err := execute[[]types.SuggestedItem](ctx, logger, "get suggested cluster items", resp, err)
	if err != nil {
		return nil, err
	}

	if items == nil {
		return []types.SuggestedItem{}, nil
	}

	return *items, nil
}

// HandleSuggestChoice accepts or rejects ML-suggested items.
func (i *TestItemScope) HandleSuggestChoice(
	ctx context.Context,
	choices []types.SuggestInfo,
) (*types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("HandleSuggestChoice")
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.HandleSuggestChooseWithResponse(ctx, i.project.projectName, choices)

	return execute[types.OperationCompletionRS](ctx, logger, "handle suggest choice", resp, err)
}

// LinkExternalIssues links external bug tracking system issues to test items.
func (i *TestItemScope) LinkExternalIssues(
	ctx context.Context,
	request types.LinkExternalIssueRQ,
) ([]types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("LinkExternalIssues")
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.LinkExternalIssuesWithResponse(ctx, i.project.projectName, request)

	results, err := execute[[]types.OperationCompletionRS](ctx, logger, "link external issues", resp, err)
	if err != nil {
		return nil, err
	}

	if results == nil {
		return []types.OperationCompletionRS{}, nil
	}

	return *results, nil
}

// UnlinkExternalIssues removes external bug tracking system issue links from test items.
func (i *TestItemScope) UnlinkExternalIssues(
	ctx context.Context,
	request types.UnlinkExternalIssueRQ,
) ([]types.OperationCompletionRS, error) {
	logger := i.project.operationLogger("UnlinkExternalIssues")
	logger.InfoContext(ctx, "API request")

	resp, err := i.project.client.raw.UnlinkExternalIssuesWithResponse(ctx, i.project.projectName, request)

	results, err := execute[[]types.OperationCompletionRS](ctx, logger, "unlink external issues", resp, err)
	if err != nil {
		return nil, err
	}

	if results == nil {
		return []types.OperationCompletionRS{}, nil
	}

	return *results, nil
}
