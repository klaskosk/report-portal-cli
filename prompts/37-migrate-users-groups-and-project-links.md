# Prompt 37: Migrate Users, Groups, and Project Links

## Goal

Migrate all `handleResponse` and `handleVoidResponse` call sites in `users.go`, `groups.go`, `project_groups.go`, and `log_types.go` to use `execute` and `executeVoid`.

## Prerequisites

- Prompt 34 is merged: `execute` and `executeVoid` exist with passing tests.

## Context

This is PR 5 of 9 in the codegen boilerplate reduction series. See prompt 33 for the full series overview.

This batch covers instance-level user management plus the groups, project-groups, and log-types resources. It includes the first uses of `executeVoid` in the series (for delete operations that return 204 No Content), but stays small enough to review as a mostly mechanical refactor.

## Scope

### Files and approximate method counts

| File | `handleResponse` | `handleVoidResponse` | Migratable |
|---|---|---|---|
| `users.go` | 19 | 0 | 19 |
| `groups.go` | 10 | 3 | 13 |
| `project_groups.go` | 3 | 1 | 4 |
| `log_types.go` | 3 | 1 | 4 |

**Total: ~40 methods** (35 `execute` + 5 `executeVoid`)

### No excluded call sites

All methods in these files use the standard `handleResponse` or `handleVoidResponse` pattern. None use `handleV2Response` or custom binary handling.

## Migration patterns

### Standard `handleResponse` → `execute`

Same transformation as prompt 35. Keep `logger.InfoContext(ctx, "API request")` at the wrapper call site.

### `handleVoidResponse` → `executeVoid`

Before:

```go
func (g *GroupsScope) Delete(ctx context.Context, groupID int64) error {
    logger := g.client.logger.With(
        slog.String("operation", "DeleteGroup"),
        slog.Int64("groupId", groupID),
    )
    logger.InfoContext(ctx, "API request")

    response, err := g.client.raw.DeleteGroupWithResponse(ctx, groupID)
    if err != nil {
        return fmt.Errorf("delete group: %w", err)
    }

    logger.DebugContext(ctx, "API response", "status", response.StatusCode())

    if err := handleVoidResponse(response, response.JSONDefault, response.Body); err != nil {
        return fmt.Errorf("delete group: %w", err)
    }

    return nil
}
```

After:

```go
func (g *GroupsScope) Delete(ctx context.Context, groupID int64) error {
    logger := g.client.logger.With(
        slog.String("operation", "DeleteGroup"),
        slog.Int64("groupId", groupID),
    )
    logger.InfoContext(ctx, "API request")

    return executeVoid(ctx, logger, "delete group",
        g.client.raw.DeleteGroupWithResponse(ctx, groupID),
    )
}
```

### `users.go` logger pattern

Note that `users.go` methods use a different logger construction than project-scoped resources. Instead of `l.project.operationLogger(...)`, they use `u.client.logger.With(slog.String("operation", ...))`. Preserve this pattern.

The request log also stays in the wrapper method:

- keep `logger.InfoContext(ctx, "API request")`
- remove the explicit `logger.DebugContext(...)`
- replace the old `handleResponse` / `handleVoidResponse` block with `execute` / `executeVoid`

### Validation cadence

During the inner loop, prefer the narrowest relevant `go test` invocation for the `pkg/client` tests that exercise the file you just touched. Use `make test` and `make lint` once at the end of the PR as the full-repo gate.

## Implementation steps

1. Migrate `groups.go` first — it uses both `execute` and `executeVoid`, making it a good proof-of-concept for this batch
2. Run focused `go test` coverage for the `pkg/client` tests that exercise `groups.go`
3. Migrate `project_groups.go` and `log_types.go` — they are small and also exercise `executeVoid`
4. Run focused `go test` coverage for the `pkg/client` tests that exercise `project_groups.go` and `log_types.go`
5. Migrate `users.go`
6. Run focused `go test` coverage for the `pkg/client` tests that exercise `users.go`
7. Run `make test`
8. `make lint` — fix any unused imports

## Verification

- `make lint` — clean
- `make test` — all unit tests pass

## Files touched

| File | Change |
|---|---|
| `users.go` | Migrate 19 methods to `execute` |
| `groups.go` | Migrate 10 methods to `execute`, 3 to `executeVoid` |
| `project_groups.go` | Migrate 3 methods to `execute`, 1 to `executeVoid` |
| `log_types.go` | Migrate 3 methods to `execute`, 1 to `executeVoid` |
