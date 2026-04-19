# Prompt 38: Migrate Projects, Dashboards, and Widgets

## Goal

Migrate all standard `handleResponse` call sites in `projects.go`, `dashboard.go`, and `widget.go` to use `execute`.

## Prerequisites

- Prompt 34 is merged: `execute` and `executeVoid` exist with passing tests.

## Context

This is PR 6 of 9 in the codegen boilerplate reduction series. See prompt 33 for the full series overview.

This batch covers the project-facing CRUD and UI-oriented surfaces. It is deliberately narrower than the original catch-all prompt so the large `projects.go` refactor can be reviewed alongside two smaller, straightforward files instead of being buried in a much larger PR.

## Scope

### Files and approximate method counts

| File | `handleResponse` sites | Notes |
|---|---|---|
| `projects.go` | 22 | Largest file in this batch |
| `dashboard.go` | 9 | |
| `widget.go` | 5 | |

**Total: ~36 methods**

### No excluded call sites

All methods in these files use the standard `handleResponse` pattern. None use `handleV2Response`, `handleVoidResponse`, or binary response extraction.

## Migration pattern

Same transformation as prompt 35. Keep `logger.InfoContext(ctx, "API request")` at the wrapper call site.

### `projects.go` notes

`projects.go` has the broadest mix of return shapes in this batch:

- plain typed success structs
- `JSON201` create responses
- methods that return dereferenced slices or maps after `handleResponse`

Preserve the small post-processing blocks after `execute`, just as in prompts 35 and 36:

- `execute` wraps and classifies errors
- callers still dereference `*[]T` or `*map[...]...` results and return empty values where the existing code does so

### Validation cadence

During the inner loop, prefer the narrowest relevant `go test` invocation for the `pkg/client` tests that exercise the file you just touched. Use `make test` and `make lint` once at the end of the PR as the full-repo gate.

## Implementation steps

1. Migrate `dashboard.go` and `widget.go` first — they are compact and make a good smoke test for this batch
2. Run focused `go test` coverage for the `pkg/client` tests that exercise `dashboard.go` and `widget.go`
3. Migrate `projects.go`
4. Run focused `go test` coverage for the `pkg/client` tests that exercise `projects.go`
5. Run `make test`
6. `make lint` — fix any unused imports

## Verification

- `make lint` — clean
- `make test` — all unit tests pass

## Files touched

| File | Change |
|---|---|
| `projects.go` | Migrate 22 methods to `execute` |
| `dashboard.go` | Migrate 9 methods to `execute` |
| `widget.go` | Migrate 5 methods to `execute` |
