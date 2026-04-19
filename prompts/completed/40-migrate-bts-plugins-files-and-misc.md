# Prompt 40: Migrate BTS, Plugins, Files, and Misc

## Goal

Migrate the remaining standard `handleResponse` call sites in `bts.go`, `instance_bts.go`, `plugin.go`, `file.go`, `onboarding.go`, and `demo_data.go` to use `execute`.

## Prerequisites

- Prompt 34 is merged: `execute` and `executeVoid` exist with passing tests.

## Context

This is PR 8 of 9 in the codegen boilerplate reduction series. See prompt 33 for the full series overview.

This batch sweeps up the remaining non-`v2`, non-void, non-binary wrapper methods after prompts 35-39. After this PR, the only remaining callers of the old response helpers should be the helper implementations and tests themselves.

## Scope

### Files and approximate method counts

| File | `handleResponse` sites | Notes |
|---|---|---|
| `bts.go` | 6 | Project-scoped bug tracking |
| `instance_bts.go` | 2 | Instance-scoped BTS lookups |
| `plugin.go` | 8 | Plugin management and command execution |
| `file.go` | 3 | Only the typed JSON operations |
| `onboarding.go` | 1 | |
| `demo_data.go` | 1 | |

**Total: ~21 methods**

### Excluded call sites (remain unchanged)

- `GetMyPhoto` in `file.go` — uses `extractFileContent` for binary response handling
- `GetUserPhoto` in `file.go` — uses `extractFileContent` for binary response handling
- `DownloadFile` in `file.go` — uses `extractFileContent` for binary response handling

## Migration pattern

Same transformation as prompt 35. Keep `logger.InfoContext(ctx, "API request")` at the wrapper call site.

### `file.go` notes

Only migrate the three typed JSON operations in `file.go`:

- `UploadPhoto`
- `DeletePhoto`
- `CleanAttachments`

Do not refactor `extractFileContent` or the binary download paths in this series. They are intentionally outside the `execute` abstraction.

### Validation cadence

During the inner loop, prefer the narrowest relevant `go test` invocation for the `pkg/client` tests that exercise the file you just touched. Use `make test` and `make lint` once at the end of the PR as the full-repo gate.

## Implementation steps

1. Migrate `bts.go` and `instance_bts.go`
2. Run focused `go test` coverage for the `pkg/client` tests that exercise `bts.go` and `instance_bts.go`
3. Migrate `plugin.go`
4. Run focused `go test` coverage for the `pkg/client` tests that exercise `plugin.go`
5. Migrate the three typed response methods in `file.go`, plus `onboarding.go` and `demo_data.go`
6. Run focused `go test` coverage for the `pkg/client` tests that exercise `file.go`, `onboarding.go`, and `demo_data.go`
7. Run `make test`
8. `make lint` — fix any unused imports

## Verification

- `make lint` — clean
- `make test` — all unit tests pass
- Confirm no remaining `handleResponse(` calls in `pkg/client` outside `response.go` and `response_test.go`
- Confirm no remaining `handleVoidResponse(` calls in `pkg/client` outside `response.go` and `response_test.go`

## Files touched

| File | Change |
|---|---|
| `bts.go` | Migrate 6 methods to `execute` |
| `instance_bts.go` | Migrate 2 methods to `execute` |
| `plugin.go` | Migrate 8 methods to `execute` |
| `file.go` | Migrate 3 methods to `execute` |
| `onboarding.go` | Migrate 1 method to `execute` |
| `demo_data.go` | Migrate 1 method to `execute` |
