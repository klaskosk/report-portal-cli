# Prompt 21: Widget Multilevel Retrieval

## Goal

Complete the `WidgetScope` in `pkg/client` by exposing the multilevel widget retrieval endpoint as a clearly named public method.

## Context

The OpenAPI spec defines a multilevel widget data endpoint:

| Wrapped? | Operation | Generated method | HTTP | Path |
| --- | --- | --- | --- | --- |
| Partially | Get multilevel widget data | `GetWidget1WithResponse` | `GET` | `/v1/{projectName}/widget/multilevel/{widgetId}` |

**Important**: The existing `GetData` method in `widget.go` already calls `GetWidget1WithResponse`, which is the generated method for the `/widget/multilevel/{widgetId}` path. The implementation status document lists "multilevel widget retrieval" as not implemented, but `GetData` may already satisfy this. Before writing new code, verify whether `GetData` fully covers the multilevel use case.

The generated method signature is:

```
GetWidget1WithResponse(ctx, projectName, widgetId, *GetWidget1Params) (*GetWidget1Response, error)
```

Where `GetWidget1Params` has:
- `Params MultiValueMapStringString` (required) — a map of string to string
- `Attributes *[]string` (optional)

The existing `GetData` method already accepts `GetWidgetDataOption` functional options, with `WithWidgetAttributes` defined.

## Files to modify

- `pkg/client/widget.go` — potentially add or rename methods
- `pkg/client/widget_test.go` — add or update tests

## Implementation plan

### 1) Verify existing coverage

Read `widget.go` and confirm that `GetData` calls `GetWidget1WithResponse` (the multilevel endpoint). If it does, the multilevel endpoint is already wrapped. Decide whether:

- **Option A**: `GetData` is sufficient as-is and just needs the status doc updated. The name "GetData" is appropriate since it returns the computed/rendered widget data from the multilevel endpoint.
- **Option B**: A `GetMultilevelData` alias is needed alongside `GetData` for clarity. Only do this if there is a separate non-multilevel data preview endpoint that `GetData` could be confused with.

Given that the generated client has no separate "preview" vs "multilevel" distinction — `GetWidget1` IS the multilevel endpoint — Option A is almost certainly correct. `GetData` already wraps the multilevel endpoint; it just wasn't recognized as such in the status document.

### 2) If `GetData` already covers this

If `GetData` is confirmed to wrap the multilevel endpoint:

- Update `docs/api-client-implementation-status.md` to move "multilevel widget retrieval" from "Not implemented" to "Implemented" (covered by `GetData`)
- Verify existing tests in `widget_test.go` cover the multilevel path adequately (they should, since they test `GetData`)
- No new code is needed

### 3) If additional coverage is needed

If inspection reveals that `GetData` does NOT fully cover the multilevel use case (e.g., it doesn't pass all required parameters), add the missing functionality. Follow the patterns in existing widget methods:

- Logger: `w.project.operationLogger("GetMultilevelWidgetData", slog.Int64("id", id))`
- Error wrapping: `fmt.Errorf("get multilevel widget data: %w", err)`
- Use `handleResponse(response, response.JSONDefault, response.JSON200, response.Body)`

## Test requirements

If new methods are added:

- All tests in `package client_test` (black-box)
- `t.Parallel()` on every test function and subtest
- Use `newTestClient` + `http.HandlerFunc` for success paths; access via `c.Project("my_project").Widgets().Method(...)`
- Verify the request path includes `/widget/multilevel/`
- Use `jsonResponse` helper for error paths
- Assert with `require.NoError`/`require.Error` + `assert.Equal`/`assert.True`

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] Multilevel widget retrieval is confirmed covered (either by existing `GetData` or a new method)
- [ ] Tests exist for the multilevel endpoint path
- [ ] `docs/api-client-implementation-status.md` updated to mark Widget as fully implemented
- [ ] Lint and tests pass
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
