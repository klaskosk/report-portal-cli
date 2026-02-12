# Prompt 10: Activity / Audit Event Search Scopes (Milestone 10)

## Goal

Implement client scopes for querying activity logs and audit events in `pkg/client`. These endpoints let operators answer "what changed and when" questions — essential for debugging configuration drift, tracking who modified launches/items, and audit compliance. This slice is client-first with an optional CLI follow-up.

## Context

Report Portal tracks two kinds of activity:

- **Activity (per-project)**: tracks modifications to project resources (launches, items, dashboards, filters, etc.) scoped to a project. Queryable by object type, time range, and user.
- **Activity events (instance-level)**: a newer, cross-project event log that includes project-level events plus instance-level events (user management, integration changes, etc.). Supports POST-based search with structured criteria.

Both are read-only from the client's perspective — there's no API to create or modify activity records.

## Generated API endpoints available

### Activity controller (project-scoped)

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetTestItemActivitiesUsingGETWithResponse(ctx, projectName, itemId int64, GetTestItemActivitiesUsingGETParams)` | Get activities for a specific test item | -> `IterableObject` (activity list) |
| `GetActivityUsingGETWithResponse(ctx, projectName, activityId int64)` | Get a single activity by ID | -> `ActivityResource` |

`GetTestItemActivitiesUsingGETParams` has pagination fields (`PagePage`, `PageSize`, `PageSort`).

### Activity event controller (instance-scoped)

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetActivitiesUsingPOSTWithResponse(ctx, GetActivitiesUsingPOSTParams, SearchCriteriaRQ)` | Search activity events with structured criteria | `SearchCriteriaRQ` -> `IterableActivityEventResource` or `PagedResponseActivityEventResource` |
| `GetProjectSubjectNameUsingGETWithResponse(ctx, projectName, GetProjectSubjectNameUsingGETParams)` | Get subject names for a project (for autocomplete in UI) | -> name list |

`GetActivitiesUsingPOSTParams` has pagination fields (`PagePage`, `PageSize`, `PageSort`).

### Key types (from `pkg/types/types.gen.go`)

```go
type ActivityResource struct {
    ActionType     string                  `json:"actionType"`
    Details        *map[string]interface{} `json:"details,omitempty"`
    Id             int64                   `json:"id"`
    LastModified   EpochMillis             `json:"lastModified"`
    LoggedObjectId int64                   `json:"loggedObjectId"`
    ObjectName     *string                 `json:"objectName,omitempty"`
    ObjectType     string                  `json:"objectType"`
    ProjectId      int64                   `json:"projectId"`
    ProjectName    *string                 `json:"projectName,omitempty"`
    User           string                  `json:"user"`
}

type ActivityEventResource struct {
    CreatedAt   *string                 `json:"created_at,omitempty"`
    Details     *map[string]interface{} `json:"details,omitempty"`
    EventName   string                  `json:"event_name"`
    Id          int64                   `json:"id"`
    ObjectId    int64                   `json:"object_id"`
    ObjectName  string                  `json:"object_name"`
    ObjectType  string                  `json:"object_type"`
    ProjectId   int64                   `json:"project_id"`
    ProjectName string                  `json:"project_name"`
    SubjectId   string                  `json:"subject_id"`
    SubjectName string                  `json:"subject_name"`
    SubjectType string                  `json:"subject_type"`
}

type IterableActivityEventResource struct {
    Content *[]ActivityEventResource `json:"content,omitempty"`
    Page    *PageMetadata            `json:"page,omitempty"`
}

type SearchCriteriaRQ struct {
    FilterCriteria *[]SearchCriteria `json:"filterCriteria,omitempty"`
}

type SearchCriteria struct {
    FilterKey *string `json:"filterKey,omitempty"`
    Operation *string `json:"operation,omitempty"`
    Value     *string `json:"value,omitempty"`
}

type PagedResponseActivityEventResource struct {
    Content    *[]ActivityEventResource `json:"content,omitempty"`
    Page       *PageMetadata            `json:"page,omitempty"`
}
```

## Client requirements

### ActivityScope (project-scoped)

Create `pkg/client/activity.go` with an `ActivityScope` struct:

- **Project-scoped**: accessible via `client.Project(name).Activities()`
- Add an `Activities()` method to `ProjectScope` in `pkg/client/project.go`
- Struct holds `project *ProjectScope`

Methods to implement:

1. **`GetByID(ctx, activityId int64) (*types.ActivityResource, error)`** — get a single activity by ID
2. **`ListForItem(ctx, itemId int64, ...ListActivitiesOption) (*types.IterableObject, error)`** — list activities for a specific test item. Note: The return type may be `IterableObject` (a generic paginated wrapper) — investigate the generated response type and use the appropriate one. If needed, the wrapper can return the raw response and let callers decode.

Functional options for `ListForItem`:

- `WithActivityPageNumber(n int32)`
- `WithActivityPageSize(size int32)`
- `WithActivitySort(fields ...string)`

### ActivityEventsScope (instance-scoped)

Create `pkg/client/activity_events.go` with an `ActivityEventsScope` struct:

- **Instance-scoped**: accessible via `client.ActivityEvents()`
- Add an `ActivityEvents()` method to `Client` in `pkg/client/client.go`
- Struct holds `client *Client`

Methods to implement:

1. **`Search(ctx, criteria SearchCriteriaRQ, ...SearchActivityEventsOption) (*types.IterableActivityEventResource, error)`** — search activity events with structured criteria and pagination. If the response type is `PagedResponseActivityEventResource`, use that instead.

2. **`GetProjectSubjectNames(ctx, projectName string) ([]string, error)`** — get subject names for a project (useful for building search UIs or autocomplete). Note: Check the `GetProjectSubjectNameUsingGETParams` for any required query params.

Functional options for `Search`:

- `WithEventPageNumber(n int32)`
- `WithEventPageSize(size int32)`
- `WithEventSort(fields ...string)`

### Convenience helpers for search criteria

Add builder functions to help construct `SearchCriteriaRQ` for common search patterns:

```go
// NewSearchCriteria creates a new search criteria with the given filter conditions.
func NewSearchCriteria(criteria ...SearchCriteria) types.SearchCriteriaRQ {
    return types.SearchCriteriaRQ{
        FilterCriteria: &criteria,
    }
}

// FilterByUser creates a search filter for a specific user.
func FilterByUser(username string) types.SearchCriteria {
    return types.SearchCriteria{
        FilterKey: ptr("subject_name"),
        Operation: ptr("eq"),
        Value:     ptr(username),
    }
}

// FilterByObjectType creates a search filter for a specific object type.
func FilterByObjectType(objectType string) types.SearchCriteria {
    return types.SearchCriteria{
        FilterKey: ptr("object_type"),
        Operation: ptr("eq"),
        Value:     ptr(objectType),
    }
}

// FilterByEventName creates a search filter for a specific event name.
func FilterByEventName(eventName string) types.SearchCriteria {
    return types.SearchCriteria{
        FilterKey: ptr("event_name"),
        Operation: ptr("eq"),
        Value:     ptr(eventName),
    }
}

// FilterByProjectName creates a search filter for a specific project.
func FilterByProjectName(projectName string) types.SearchCriteria {
    return types.SearchCriteria{
        FilterKey: ptr("project_name"),
        Operation: ptr("eq"),
        Value:     ptr(projectName),
    }
}
```

Where `ptr` is a small generic helper (`func ptr[T any](v T) *T { return &v }`) — add it as a private utility if it doesn't already exist, or use the existing pattern in the codebase.

## Client tests

### `pkg/client/activity_test.go` (new, `package client_test`)

- **GetByID**: success (200), not found (404), unauthorized (401)
- **ListForItem**: success (200 with pagination), empty results, unauthorized (401)

### `pkg/client/activity_events_test.go` (new, `package client_test`)

- **Search**: success (200 with results), empty results (200 with empty content), unauthorized (401)
- **Search with criteria**: verify the request body contains the expected `SearchCriteriaRQ` structure
- **GetProjectSubjectNames**: success (200), unauthorized (401)

### Criteria builder tests

- Test the convenience functions (`FilterByUser`, `FilterByObjectType`, etc.) produce the expected `SearchCriteria` structs

### `export_test.go`

Add test-only constructors for `ActivityScope` and `ActivityEventsScope`.

## Integration tests

### `test/integration/client_activity_test.go` (new)

- Use `Label("client", "activity")`
- **GetByID for project activity**: Seed a launch, then query activities for the project. Find an activity related to launch creation, verify it has expected fields (actionType, objectType, user).
- **ListForItem**: Seed a launch + item + finish, then query activities for the test item. Verify some activity exists.

### `test/integration/client_activity_events_test.go` (new)

- Use `Label("client", "activity-events")`
- **Search all events**: Search with empty criteria, verify results are returned and have expected structure
- **Search by project**: Search with `FilterByProjectName(testProjectName)`, verify only events for that project appear
- **Search by event type**: Search for "startLaunch" or similar events after seeding a launch

### Seeding notes

- Activities are generated automatically when resources are created/modified. Seed a launch (start + finish) and the activity log should contain entries.
- The test should seed data first, then query activities — there may be a short delay before activities appear. Use the existing `waitForSeedResolution` pattern if needed.
- Activity events are instance-level and may include events from other test contexts. Filter by project name or object ID to isolate test-relevant events.

## CLI (optional — defer unless explicitly requested)

If there's time, add `rp activity list` as a CLI command:

- Command shape: `rp activity list [--project <name>] [--user <username>] [--type <event-type>] [--since <timestamp>] [--until <timestamp>]`
- This uses the instance-scoped `ActivityEventsScope.Search` under the hood
- Table output columns: ID, EVENT NAME, OBJECT TYPE, OBJECT NAME, SUBJECT NAME, PROJECT NAME, CREATED AT
- JSON output: full `ActivityEventResource` array

This is explicitly optional for this prompt. The client coverage is the priority.

## Design considerations

- **Two activity systems**: Report Portal has both the older `ActivityResource` (project-scoped, GET-based) and the newer `ActivityEventResource` (instance-scoped, POST-based search). Both should be supported, but the newer event system is more powerful.
- **Search is POST-based**: The activity events search uses POST with a request body (`SearchCriteriaRQ`) rather than GET with query params. This is unusual but follows the generated API. The wrapper method should accept the criteria as a parameter.
- **IterableObject return type**: The `GetTestItemActivitiesUsingGET` endpoint may return a generic `IterableObject` rather than a typed list. Investigate the generated response struct. If it's untyped, the wrapper can return the raw pagination and content for the caller to decode.
- **Timestamp filtering**: Activity events use string timestamps (`created_at`), not `EpochMillis`. The search criteria builder should accept string values. Consider adding a convenience helper that accepts `time.Time` and formats it.
- **Private `ptr` helper**: If the codebase doesn't already have a `ptr[T any](v T) *T` helper, add one as a private function in the appropriate file. It's needed for constructing optional fields in search criteria.

## Definition of done

- [ ] `pkg/client/activity.go` with `ActivityScope` (GetByID, ListForItem)
- [ ] `pkg/client/activity_events.go` with `ActivityEventsScope` (Search, GetProjectSubjectNames)
- [ ] Search criteria convenience helpers
- [ ] Scope accessor methods on `Client` and `ProjectScope`
- [ ] Unit tests for all new scopes and helpers
- [ ] Integration tests for activity querying
- [ ] `code-verifier` subagent says your changes are good to go
