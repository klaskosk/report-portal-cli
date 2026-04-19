# Prompt 39: Migrate Settings and Integrations

## Goal

Migrate all standard `handleResponse` call sites in `project_settings.go`, `settings.go`, and `integrations.go` to use `execute`.

## Prerequisites

- Prompt 34 is merged: `execute` and `executeVoid` exist with passing tests.

## Context

This is PR 7 of 9 in the codegen boilerplate reduction series. See prompt 33 for the full series overview.

This batch covers configuration and integration management. These files are a good fit to migrate together because they all use the standard `handleResponse` pattern, but they are still small enough as a set to review comfortably.

## Scope

### Files and approximate method counts

| File | `handleResponse` sites | Notes |
|---|---|---|
| `project_settings.go` | 11 | Project-scoped configuration |
| `settings.go` | 3 | Instance-level server settings |
| `integrations.go` | 17 | Global and project integration flows |

**Total: ~31 methods**

### No excluded call sites

All methods in these files use the standard `handleResponse` pattern. None use `handleV2Response`, `handleVoidResponse`, or binary response extraction.

## Migration pattern

Same transformation as prompt 35. Keep `logger.InfoContext(ctx, "API request")` at the wrapper call site.

### `integrations.go` notes

`integrations.go` mixes instance-scoped and project-scoped methods in the same file. Preserve the existing logger construction and scope boundaries exactly:

- do not normalize logger setup across unrelated methods in this refactor
- do not rename helpers or reorder unrelated code
- only replace the `handleResponse` boilerplate with `execute`

### Validation cadence

During the inner loop, prefer the narrowest relevant `go test` invocation for the `pkg/client` tests that exercise the file you just touched. Use `make test` and `make lint` once at the end of the PR as the full-repo gate.

## Implementation steps

1. Migrate `settings.go` and `project_settings.go` first
2. Run focused `go test` coverage for the `pkg/client` tests that exercise `settings.go` and `project_settings.go`
3. Migrate `integrations.go`
4. Run focused `go test` coverage for the `pkg/client` tests that exercise `integrations.go`
5. Run `make test`
6. `make lint` — fix any unused imports

## Verification

- `make lint` — clean
- `make test` — all unit tests pass

## Files touched

| File | Change |
|---|---|
| `project_settings.go` | Migrate 11 methods to `execute` |
| `settings.go` | Migrate 3 methods to `execute` |
| `integrations.go` | Migrate 17 methods to `execute` |
