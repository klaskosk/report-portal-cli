# Prompt 23: Log Sync Navigation Gaps

## Goal

Complete the `LogScope` in `pkg/client` by adding page lookup, location navigation, location search, and nested log retrieval.

## Context

`LogScope` covers core log CRUD and search but is missing four navigation-related endpoints:

| Wrapped? | Operation | Generated method | HTTP | Path |
| --- | --- | --- | --- | --- |
| No | Get page number | `GetPageNumberWithResponse` | `GET` | `/v1/{projectName}/log/{logId}/page` |
| No | Get error page (location navigation) | `GetErrorPageWithResponse` | `GET` | `/v1/{projectName}/log/locations/{parentId}` |
| No | Search logs with location | `GetLogsWithLocationBySearchWithResponse` | `GET` | `/v1/{projectName}/log/locations/search/{parentId}` |
| No | Get nested items | `GetNestedItemsWithResponse` | `GET` | `/v1/{projectName}/log/nested/{parentId}` |

Already implemented: `GetByID`, `GetByUUID`, `List`, `Create`, `Delete`, `CreateBatch`, `Search`, `GetLogsUnder`.

These four endpoints form a cohesive log-navigation slice and should be kept together in one PR.

## Files to modify

- `pkg/client/log.go` — add new methods and functional options
- `pkg/client/log_test.go` — add tests for new methods

## Implementation plan

### 1) Add `GetPageNumber` method

Wrap `GetPageNumberWithResponse` to look up which page a specific log entry appears on.

Key details:

- **Generated method**: `l.project.client.raw.GetPageNumberWithResponse(ctx, l.project.projectName, logID, params)`
- **Params struct**: `api.GetPageNumberParams` — contains numerous `filter.eq.*` fields (log ID, launch ID, status, level, item, etc.) and pagination fields (`page.page`, `page.size`, `page.sort`)
- **Response**: `JSON200` is `*map[string]interface{}`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `GET /v1/{projectName}/log/{logId}/page`
- **Logger**: `l.project.operationLogger("GetLogPageNumber", slog.Int64("logId", logID))`
- **Error wrapping**: `fmt.Errorf("get log page number: %w", err)`

Signature:

```go
func (l *LogScope) GetPageNumber(
    ctx context.Context,
    logID int64,
    options ...GetLogPageNumberOption,
) (map[string]interface{}, error)
```

Define a `GetLogPageNumberOption` interface and option functions for the most useful filter params. At minimum expose:
- `WithLogPageItemID(id int32)` — sets `FilterEqItem`
- `WithLogPageSize(size int32)` — sets `PageSize`

Return a non-nil empty map if `JSON200` is nil.

### 2) Add `GetLogLocations` method

Wrap `GetErrorPageWithResponse` to retrieve paginated log entries at error locations for a parent item.

Key details:

- **Generated method**: `l.project.client.raw.GetErrorPageWithResponse(ctx, l.project.projectName, parentID, params)`
- **Params struct**: `api.GetErrorPageParams` — has a required `Params map[string]string` plus `filter.eq.*` and pagination fields
- **Response**: `JSON200` is `*[]types.PagedLogResource`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `GET /v1/{projectName}/log/locations/{parentId}`
- **Logger**: `l.project.operationLogger("GetLogLocations", slog.Int64("parentId", parentID))`
- **Error wrapping**: `fmt.Errorf("get log locations: %w", err)`

Signature:

```go
func (l *LogScope) GetLogLocations(
    ctx context.Context,
    parentID int64,
    options ...GetLogLocationsOption,
) ([]types.PagedLogResource, error)
```

The `PagedLogResource` type includes: `Id`, `ItemId`, `LaunchId`, `Level`, `Message`, `BinaryContent`, and `PagesLocation` (slice of key-value pairs).

Define a `GetLogLocationsOption` interface. At minimum expose:
- `WithLogLocationsPageSize(size int32)` — sets `PageSize`
- `WithLogLocationsPageNumber(page int32)` — sets `PagePage`

The `Params` field is required and must be initialized to an empty map if no caller-supplied values set it.

Return a non-nil empty slice if `JSON200` is nil.

### 3) Add `SearchLogLocations` method

Wrap `GetLogsWithLocationBySearchWithResponse` to search log locations with pagination.

Key details:

- **Generated method**: `l.project.client.raw.GetLogsWithLocationBySearchWithResponse(ctx, l.project.projectName, parentID, params)`
- **Params struct**: `api.GetLogsWithLocationBySearchParams` — same shape as `GetErrorPageParams` (required `Params` map + filters + pagination)
- **Response**: `JSON200` is `*types.PagePagedLogResource`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `GET /v1/{projectName}/log/locations/search/{parentId}`
- **Logger**: `l.project.operationLogger("SearchLogLocations", slog.Int64("parentId", parentID))`
- **Error wrapping**: `fmt.Errorf("search log locations: %w", err)`

Signature:

```go
func (l *LogScope) SearchLogLocations(
    ctx context.Context,
    parentID int64,
    options ...SearchLogLocationsOption,
) (*types.PagePagedLogResource, error)
```

Define a `SearchLogLocationsOption` interface with similar pagination options.

The `Params` field is required — initialize to an empty map by default.

### 4) Add `GetNestedLogs` method

Wrap `GetNestedItemsWithResponse` to retrieve nested log entries under a parent.

Key details:

- **Generated method**: `l.project.client.raw.GetNestedItemsWithResponse(ctx, l.project.projectName, parentID, params)`
- **Params struct**: `api.GetNestedItemsParams` — same shape as above (required `Params` map + filters + pagination)
- **Response**: `JSON200` is `*types.PageObject`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `GET /v1/{projectName}/log/nested/{parentId}`
- **Logger**: `l.project.operationLogger("GetNestedLogs", slog.Int64("parentId", parentID))`
- **Error wrapping**: `fmt.Errorf("get nested logs: %w", err)`

The `PageObject` type has `Content *[]interface{}` and `Page *PageMetadata`. This is an untyped response from the API.

Signature:

```go
func (l *LogScope) GetNestedLogs(
    ctx context.Context,
    parentID int64,
    options ...GetNestedLogsOption,
) (*types.PageObject, error)
```

Define a `GetNestedLogsOption` interface with pagination options.

The `Params` field is required — initialize to an empty map by default.

### 5) Write tests

Follow existing patterns in `log_test.go`. For each method add:

- **Success test**: Verify path, HTTP method, bearer token, query parameters, and response parsing
- **Error test**: At minimum an unauthorized case
- **Options test**: Verify that functional options correctly set query parameters (check `r.URL.Query()` in the handler)
- **Empty response test**: Verify nil-to-empty conversion for slice/map responses

Test naming examples:
- `TestLogScope_GetPageNumber_Success`
- `TestLogScope_GetPageNumber_WithOptions`
- `TestLogScope_GetPageNumber_Unauthorized`
- `TestLogScope_GetLogLocations_Success`
- `TestLogScope_GetLogLocations_EmptyResponse`
- `TestLogScope_SearchLogLocations_Success`
- `TestLogScope_GetNestedLogs_Success`

## Test requirements

- All tests in `package client_test` (black-box)
- `t.Parallel()` on every test function and subtest
- Use `newTestClient` + `http.HandlerFunc` for success paths; access via `c.Project("my_project").Logs().Method(...)`
- Use `jsonResponse` helper for error paths
- Assert with `require.NoError`/`require.Error` + `assert.Equal`/`assert.True`

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] `GetPageNumber` method added to `LogScope` with functional options
- [ ] `GetLogLocations` method added to `LogScope` with functional options
- [ ] `SearchLogLocations` method added to `LogScope` with functional options
- [ ] `GetNestedLogs` method added to `LogScope` with functional options
- [ ] Unit tests added for success, error, and options paths of all four methods
- [ ] `docs/api-client-implementation-status.md` updated to reflect new log coverage
- [ ] Lint and tests pass
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
