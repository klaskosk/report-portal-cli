# Prompt 24: Test Item Sync Gaps

## Goal

Complete the `TestItemScope` in `pkg/client` by adding the `/item/search` and `/item/v2` list endpoints.

## Context

`TestItemScope` covers extensive test item operations but is missing two list-style endpoints:

| Wrapped? | Operation | Generated method | HTTP | Path |
| --- | --- | --- | --- | --- |
| No | Search test items | `GetTestItems2WithResponse` | `GET` | `/v1/{projectName}/item/search` |
| No | List test items (v2) | `GetTestItemsV2WithResponse` | `GET` | `/v1/{projectName}/item/v2` |

Both return `*types.PageTestItemResource` — the same paginated response type used by the existing `List` method.

Already implemented: `StartRoot`, `StartChild`, `Finish`, `GetByID`, `GetByUUID`, `List`, `GetByIDs`, `Update`, `Delete`, `DefineIssueType`, `DeleteBulk`, `BulkUpdate`, attribute helpers, `GetHistory`, `GetStatistics`, `GetSuggestedItems`, `GetSuggestedClusterItems`, `HandleSuggestChoice`, `LinkExternalIssues`, `UnlinkExternalIssues`.

## Files to modify

- `pkg/client/item.go` — add new methods and functional options
- `pkg/client/item_test.go` — add tests for new methods

## Implementation plan

### 1) Add `Search` method

Wrap `GetTestItems2WithResponse` to search test items with simplified filter parameters.

Key details:

- **Generated method**: `i.project.client.raw.GetTestItems2WithResponse(ctx, i.project.projectName, params)`
- **Params struct**: `api.GetTestItems2Params` — a slim struct with:
  - `FilterCntName *string` — contains-match on item name
  - `FilterHasCompositeAttribute *string` — attribute filter
  - `FilterInStatus *string` — status filter
  - `PagePage *int32` — page number
  - `PageSize *int32` — page size
  - `PageSort *string` — sort criteria
- **Response**: `JSON200` is `*types.PageTestItemResource`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `GET /v1/{projectName}/item/search`
- **Logger**: `i.project.operationLogger("SearchTestItems")`
- **Error wrapping**: `fmt.Errorf("search test items: %w", err)`

Signature:

```go
func (i *TestItemScope) Search(
    ctx context.Context,
    options ...SearchTestItemsOption,
) (*types.PageTestItemResource, error)
```

Define a `SearchTestItemsOption` interface and option functions:
- `WithSearchItemName(name string)` — sets `FilterCntName`
- `WithSearchItemAttribute(attr string)` — sets `FilterHasCompositeAttribute`
- `WithSearchItemStatus(status string)` — sets `FilterInStatus`
- `WithSearchItemPageNumber(page int32)` — sets `PagePage`
- `WithSearchItemPageSize(size int32)` — sets `PageSize`
- `WithSearchItemSort(sort string)` — sets `PageSort`

### 2) Add `ListV2` method

Wrap `GetTestItemsV2WithResponse` to list test items using the v2 endpoint with extended filter parameters.

Key details:

- **Generated method**: `i.project.client.raw.GetTestItemsV2WithResponse(ctx, i.project.projectName, params)`
- **Params struct**: `api.GetTestItemsV2Params` — a large struct with a required `Params map[string]string` plus many `filter.eq.*` / `filter.has.*` fields and pagination. This is structurally identical to the existing `GetTestItemsParams` used by `List`, but routed to the `/item/v2` path.
- **Response**: `JSON200` is `*types.PageTestItemResource`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `GET /v1/{projectName}/item/v2`
- **Logger**: `i.project.operationLogger("ListTestItemsV2")`
- **Error wrapping**: `fmt.Errorf("list test items v2: %w", err)`

Signature:

```go
func (i *TestItemScope) ListV2(
    ctx context.Context,
    options ...ListTestItemsV2Option,
) (*types.PageTestItemResource, error)
```

Define a `ListTestItemsV2Option` interface and option functions. The `GetTestItemsV2Params` struct has a very large number of filter fields. Follow the pattern established by the existing `ListTestItemsOption` — expose the most useful subset:
- `WithItemV2PageNumber(page int32)` — sets `PagePage`
- `WithItemV2PageSize(size int32)` — sets `PageSize`
- `WithItemV2Sort(sort string)` — sets `PageSort`
- `WithItemV2Status(status string)` — sets `FilterEqStatus`
- `WithItemV2Name(name string)` — sets `FilterEqName`
- `WithItemV2LaunchID(id int32)` — sets `FilterEqLaunchId`
- `WithItemV2ParentID(id int32)` — sets `FilterEqParentId`

The `Params` field is required — initialize to an empty map by default.

Examine the existing `List` method and its `ListTestItemsOption` options to understand which fields are already exposed and mirror a similar level of coverage for `ListV2`.

### 3) Write tests

Follow the existing patterns in `item_test.go`. For each method add:

- **Success test**: Verify path (`.../item/search` or `.../item/v2`), HTTP method (`GET`), bearer token, and response parsing
- **Options test**: Verify functional options set expected query parameters
- **Error test**: At minimum an unauthorized case
- **Empty page test**: Verify behavior when the API returns an empty page (`{"content": [], "page": {...}}`)

Test naming examples:
- `TestTestItemScope_Search_Success`
- `TestTestItemScope_Search_WithOptions`
- `TestTestItemScope_Search_Unauthorized`
- `TestTestItemScope_ListV2_Success`
- `TestTestItemScope_ListV2_WithFilterOptions`
- `TestTestItemScope_ListV2_Unauthorized`

## Test requirements

- All tests in `package client_test` (black-box)
- `t.Parallel()` on every test function and subtest
- Use `newTestClient` + `http.HandlerFunc` for success paths; access via `c.Project("my_project").Items().Method(...)`
- Use `jsonResponse` helper for error paths
- Assert with `require.NoError`/`require.Error` + `assert.Equal`/`assert.True`

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] `Search` method added to `TestItemScope` with functional options
- [ ] `ListV2` method added to `TestItemScope` with functional options
- [ ] Unit tests added for success, error, and options paths of both methods
- [ ] `docs/api-client-implementation-status.md` updated to reflect new test item coverage (note that `Test Item Async` wrappers remain out of scope)
- [ ] Lint and tests pass
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
