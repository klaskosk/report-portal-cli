# Prompt 36: Migrate Items, Activity, and Filters

## Goal

Migrate all standard `handleResponse` call sites in `item.go`, `activity.go`, `activity_events.go`, and `filter.go` to use the `execute` helper.

## Prerequisites

- Prompt 34 is merged: `execute` and `executeVoid` exist with passing tests.

## Context

This is PR 4 of 9 in the codegen boilerplate reduction series. See prompt 33 for the full series overview.

## Scope

### Files and approximate method counts

| File | `handleResponse` sites | Excluded (manual) | Migratable |
|---|---|---|---|
| `item.go` | 25 | GetByUUID (v2) | ~24 |
| `activity.go` | 2 | None | 2 |
| `activity_events.go` | 2 | None | 2 |
| `filter.go` | 8 | None | 8 |

**Total: ~36 methods**

### Excluded call sites

- **`GetByUUID`** in `item.go` — uses `handleV2Response` for the `ApplicationXReportportalTestV2JSON200` content-negotiated response

## Migration pattern

Same transformation as prompt 35. See that prompt for the before/after examples of the standard `handleResponse` → `execute` pattern, the requirement to keep `logger.InfoContext(ctx, "API request")` at the call site, and the slice/map dereference post-processing pattern.

### `item.go` specific notes

`item.go` is the largest file in this batch. Several methods return dereferenced slices or maps — follow the same pattern as `launch.go`'s `GetNames`:

```go
// execute returns *T, then dereference
result, err := execute[[]types.SomeType](ctx, logger, "op name",
    l.project.client.raw.SomeOpWithResponse(ctx, ...),
)
if err != nil {
    return nil, err
}
if result == nil {
    return []types.SomeType{}, nil
}
return *result, nil
```

### Validation cadence

During the inner loop, prefer the narrowest relevant `go test` invocation for the `pkg/client` tests that exercise the file you just touched. Use `make test` and `make lint` once at the end of the PR as the full-repo gate.

## Implementation steps

1. Migrate `item.go` — rewrite all ~24 standard methods, skip GetByUUID (v2)
2. Run focused `go test` coverage for the `pkg/client` tests that exercise `item.go`
3. Migrate `activity.go` — rewrite both methods
4. Migrate `activity_events.go` — rewrite both methods
5. Run focused `go test` coverage for the `pkg/client` tests that exercise `activity.go` and `activity_events.go`
6. Migrate `filter.go` — rewrite all 8 methods
7. Run focused `go test` coverage for the `pkg/client` tests that exercise `filter.go`
8. Run `make test`
9. `make lint` — fix any unused imports

## Verification

- `make lint` — clean
- `make test` — all unit tests pass

## Files touched

| File | Change |
|---|---|
| `item.go` | Migrate ~24 methods to `execute` |
| `activity.go` | Migrate 2 methods to `execute` |
| `activity_events.go` | Migrate 2 methods to `execute` |
| `filter.go` | Migrate 8 methods to `execute` |
