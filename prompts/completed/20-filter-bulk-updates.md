# Prompt 20: User Filter Bulk Updates

## Goal

Complete the `UserFilterScope` in `pkg/client` by adding bulk filter update support.

## Context

`UserFilterScope` covers single-filter CRUD, get-by-IDs, and name lookup, but is missing the bulk update endpoint:

| Wrapped? | Operation | Generated method | HTTP | Path |
| --- | --- | --- | --- | --- |
| No | Bulk update filters | `UpdateUserFiltersWithResponse` | `PUT` | `/v1/{projectName}/filter` |

The single-filter `Update` method uses `PUT /v1/{projectName}/filter/{filterId}` with `UpdateUserFilterRQ`. The bulk variant uses `PUT /v1/{projectName}/filter` (no ID in the path) with `CollectionsRQBulkUpdateFilterRQ` and returns an array of `OperationCompletionRS`.

Already implemented: `Create`, `GetByID`, `List`, `Update` (single), `Delete`, `GetByIds`, `GetNames`.

## Files to modify

- `pkg/client/filter.go` — add new method
- `pkg/client/filter_test.go` — add tests for the new method

## Implementation plan

### 1) Add `BulkUpdate` method

Wrap `UpdateUserFiltersWithResponse` to update multiple filters in a single call.

Key details:

- **Generated method**: `f.project.client.raw.UpdateUserFiltersWithResponse(ctx, f.project.projectName, body)`
- **Request body type**: `types.CollectionsRQBulkUpdateFilterRQ` — a wrapper containing `Elements []BulkUpdateFilterRQ`
- Each `BulkUpdateFilterRQ` has: `Id string`, `Name string`, `Type BulkUpdateFilterRQType` (enum: `"launch"`, `"testitem"`, `"log"`), `Conditions []UserFilterCondition`, `Orders []Order`, `Description *string`
- **Response**: `JSON200` is `*[]types.OperationCompletionRS`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `PUT /v1/{projectName}/filter`
- **Logger**: Use `f.project.operationLogger("BulkUpdateFilters")`
- **Error wrapping**: `fmt.Errorf("bulk update filters: %w", err)`

Signature:

```go
func (f *UserFilterScope) BulkUpdate(
    ctx context.Context,
    request types.CollectionsRQBulkUpdateFilterRQ,
) ([]types.OperationCompletionRS, error)
```

Note: the response type is `*[]OperationCompletionRS` (pointer to slice). Return a non-nil empty slice if the pointer is nil, following the project's nil-to-empty convention seen in other scopes (e.g., `GetByIds` in the same file).

### 2) Write tests

Follow the existing patterns in `filter_test.go`. Add:

- **Success test**: Verify path (`/api/v1/my_project/filter`), HTTP method (`PUT`), bearer token, request body structure, and response parsing. Return a JSON array of `OperationCompletionRS` objects and verify each element.
- **Error test**: Verify `client.IsUnauthorized` with a JSON error response.
- **Empty response test**: Verify behavior when the API returns an empty array or null body.

Test naming:
- `TestUserFilterScope_BulkUpdate_Success`
- `TestUserFilterScope_BulkUpdate_Unauthorized`

## Test requirements

- All tests in `package client_test` (black-box)
- `t.Parallel()` on every test function and subtest
- Use `newTestClient` + `http.HandlerFunc` for success paths; access via `c.Project("my_project").Filters().BulkUpdate(...)`
- Use `jsonResponse` helper for error paths
- Assert with `require.NoError`/`require.Error` + `assert.Equal`/`assert.True`

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] `BulkUpdate` method added to `UserFilterScope`
- [ ] Unit tests added for success and error paths
- [ ] `docs/api-client-implementation-status.md` updated to mark User Filter as fully implemented
- [ ] Lint and tests pass
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
