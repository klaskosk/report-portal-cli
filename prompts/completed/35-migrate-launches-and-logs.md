# Prompt 35: Migrate Launches and Logs

## Goal

Migrate all standard `handleResponse` call sites in `launch.go` and `log.go` to use the `execute` helper from prompt 34. This is the core reporting pipeline — the write/finish/read cycle.

## Prerequisites

- Prompt 34 is merged: `execute` and `executeVoid` exist with passing tests.

## Context

This is PR 3 of 9 in the codegen boilerplate reduction series. See prompt 33 for the full series overview.

## Scope

### Files and approximate method counts

| File | `handleResponse` sites | Excluded (manual) | Migratable |
|---|---|---|---|
| `launch.go` | 21 | GetByUUID (v2), Merge (v2), GetReport (binary body), Import (stub) | ~17 |
| `log.go` | 12 | CreateBatch (spec mismatch), GetPageNumber (empty-body 2xx), GetLogLocations (empty-body 2xx) | ~9 |

**Total: ~26 standard methods + 3 inline rewrites**

### Excluded call sites

#### `launch.go` — remain unchanged

- **`GetByUUID`** and **`Merge`** — use `handleV2Response` for content-negotiated v2 responses
- **`GetReport`** — returns raw `response.Body` bytes (HTML/PDF), not a typed JSON response
- **`Import`** — returns a hardcoded unsupported-operation error, no API call

#### `log.go` — rewrite to inline response handling

These methods cannot use `execute` due to spec/runtime mismatches or empty-body success responses. They must be rewritten to remove their `handleResponse` dependency (so prompt 41 can delete `handleResponse` cleanly), but the replacement is inline error handling rather than `execute`.

- **`CreateBatch`** — the generated response struct declares `JSON200` but the API actually returns HTTP 201 for batch creation. The current code works around this by manually unmarshaling `response.Body` when it sees 201, then falls through to `handleResponse`. Replace the `handleResponse` fallback with inline checks: `JSONDefault` → `newAPIError`, non-2xx → `newUnexpectedStatusError`, then keep the manual `json.Unmarshal` path for the 201 body.
- **`GetPageNumber`** — the endpoint can return a 2xx with no parsed body (`JSON200 == nil && JSONDefault == nil`). `execute` would reject this as "no valid response body was parsed". Rewrite to inline error checks and return `map[string]any{}` for the empty-body 2xx case.
- **`GetLogLocations`** — same pattern as `GetPageNumber`. Return `[]types.PagedLogResource{}` for the empty-body 2xx case.

## Migration pattern

### Standard `handleResponse` → `execute`

Before:

```go
func (l *LaunchScope) Start(ctx context.Context, request types.StartLaunchRQ) (*types.StartLaunchRS, error) {
    logger := l.project.operationLogger("StartLaunch", slog.String("name", request.Name))
    logger.InfoContext(ctx, "API request")

    response, err := l.project.client.raw.StartLaunch1WithResponse(ctx, l.project.projectName, request)
    if err != nil {
        return nil, fmt.Errorf("start launch: %w", err)
    }

    logger.DebugContext(ctx, "API response", "status", response.StatusCode())

    result, err := handleResponse(response, response.JSONDefault, response.JSON201, response.Body)
    if err != nil {
        return nil, fmt.Errorf("start launch: %w", err)
    }

    return result, nil
}
```

After:

```go
func (l *LaunchScope) Start(ctx context.Context, request types.StartLaunchRQ) (*types.StartLaunchRS, error) {
    logger := l.project.operationLogger("StartLaunch", slog.String("name", request.Name))
    logger.InfoContext(ctx, "API request")

    return execute[types.StartLaunchRS](ctx, logger, "start launch",
        l.project.client.raw.StartLaunch1WithResponse(ctx, l.project.projectName, request),
    )
}
```

Note: keep the existing `logger.InfoContext(ctx, "API request")` call at the wrapper call site. `execute` owns response handling and debug logging, but it intentionally does not move the request log.

### Methods with post-processing (slice/map dereference)

Several methods dereference the result pointer or provide a default empty value. These still use `execute` but keep a small post-processing step:

```go
func (l *LaunchScope) GetNames(ctx context.Context, filter string) ([]string, error) {
    logger := l.project.operationLogger("GetLaunchNames", slog.String("filter", filter))

    params := &api.GetAllLaunchNamesParams{}
    if filter != "" {
        params.FilterCntName = &filter
    }

    logger.InfoContext(ctx, "API request")

    names, err := execute[[]string](ctx, logger, "get launch names",
        l.project.client.raw.GetAllLaunchNamesWithResponse(ctx, l.project.projectName, params),
    )
    if err != nil {
        return nil, err
    }

    if names == nil {
        return []string{}, nil
    }

    return *names, nil
}
```

Note: callers do NOT re-wrap the error (`return nil, err` not `return nil, fmt.Errorf(..., err)`). `execute` already wraps with the operation name.

### Methods with bulk result unwrapping

`BulkForceFinish` returns `[]types.OperationCompletionRS` (not a pointer). Same pattern — `execute` then dereference:

```go
results, err := execute[[]types.OperationCompletionRS](ctx, logger, "bulk force finish launches",
    l.project.client.raw.BulkForceFinishWithResponse(ctx, l.project.projectName, request),
)
if err != nil {
    return nil, err
}
if results == nil {
    return []types.OperationCompletionRS{}, nil
}
return *results, nil
```

### Validation cadence

During the inner loop, prefer the narrowest relevant `go test` invocation for the `pkg/client` tests that exercise the file you just touched. Use `make test` and `make lint` once at the end of the PR as the full-repo gate.

## Implementation steps

1. Migrate `launch.go` — rewrite all ~17 standard methods, skip the 4 excluded call sites
2. Run focused `go test` coverage for the `pkg/client` tests that exercise `launch.go`
3. Migrate `log.go` — rewrite ~9 standard methods to `execute`, rewrite `CreateBatch`, `GetPageNumber`, and `GetLogLocations` to inline response handling (see excluded call sites above)
4. Run focused `go test` coverage for the `pkg/client` tests that exercise `log.go`
5. Run `make test`
6. `make lint` — fix any unused imports (`"fmt"` may become unused in some files)

## Verification

- `make lint` — clean
- `make test` — all unit tests pass
- Error predicates (`IsNotFound`, etc.) still work in tests — verify via existing test assertions

## Files touched

| File | Change |
|---|---|
| `launch.go` | Migrate ~17 methods to `execute` |
| `log.go` | Migrate ~9 methods to `execute`; rewrite 3 bespoke methods to inline handling |
