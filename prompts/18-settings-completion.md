# Prompt 18: Settings Completion

## Goal

Complete the `SettingsScope` in `pkg/client` by adding the remaining server settings operations: updating server settings by key and resolving the duplicate analytics write variant in the spec.

## Context

`SettingsScope` currently wraps two of the four settings operations from the OpenAPI spec:

| Wrapped? | Operation | Generated method | HTTP | Path |
| --- | --- | --- | --- | --- |
| Yes | Get server settings | `GetServerSettingsWithResponse` | `GET` | `/v1/settings` |
| Yes | Update analytics (POST) | `SaveAnalyticsSettingsWithResponse` | `POST` | `/v1/settings/analytics` |
| No | Update server settings | `UpdateServerSettingsWithResponse` | `PUT` | `/v1/settings` |
| No | Update analytics (PUT) | `SaveAnalyticsSettings1WithResponse` | `PUT` | `/v1/settings/analytics` |

The spec defines two analytics write variants — `POST` and `PUT` to the same path — with the same body type (`AnalyticsResource`) and same response type (`OperationCompletionRS`). The `POST` variant is already wrapped as `UpdateAnalytics`. The `PUT` variant (`SaveAnalyticsSettings1WithResponse`) is functionally identical and needs a decision: either expose it as a separate method or document that the existing `POST`-based wrapper covers this intent.

## Files to modify

- `pkg/client/settings.go` — add new methods
- `pkg/client/settings_test.go` — add tests for new methods

## Implementation plan

### 1) Add `UpdateServerSettings` method

Add a method to `SettingsScope` that wraps `UpdateServerSettingsWithResponse`. Follow the exact pattern used by `GetServerSettings` and `UpdateAnalytics`.

Key details:

- **Generated method**: `s.client.raw.UpdateServerSettingsWithResponse(ctx, body)`
- **Request body type**: `types.UpdateSettingsRq` — a struct with `Key string` and `Value string`
- **Response**: `JSON200` is `*types.OperationCompletionRS`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `PUT /v1/settings`
- **Error wrapping**: `fmt.Errorf("update server settings: %w", err)`
- **Logger operation name**: `"UpdateServerSettings"`

Signature should be:

```go
func (s *SettingsScope) UpdateServerSettings(
    ctx context.Context,
    request types.UpdateSettingsRq,
) (*types.OperationCompletionRS, error)
```

### 2) Handle the duplicate analytics write variant

The spec exposes `POST /v1/settings/analytics` (`SaveAnalyticsSettings`) and `PUT /v1/settings/analytics` (`SaveAnalyticsSettings1`). Both accept `AnalyticsResource` and return `OperationCompletionRS`.

The existing `UpdateAnalytics` method uses the `POST` variant. The `PUT` variant is the same logical operation. Do **not** add a second method — this would confuse consumers. Instead, leave `UpdateAnalytics` as-is (POST-based) and do not wrap the PUT variant. No code change needed here, but ensure the implementation status document reflects this decision.

### 3) Write tests

Follow the existing test patterns in `settings_test.go`. For each new method add:

- **Success test**: Verify path (`/api/v1/settings`), HTTP method (`PUT`), bearer token, request body, and response parsing
- **Error test**: Verify `client.IsUnauthorized` or other predicate with a JSON error response

Use table-driven tests (`[]struct{}` + `t.Run`) if there are multiple error scenarios; otherwise use individual test functions matching the existing style (e.g., `TestSettingsScope_UpdateServerSettings_Success`, `TestSettingsScope_UpdateServerSettings_Unauthorized`).

## Test requirements

- All tests in `package client_test` (black-box)
- `t.Parallel()` on every test function and subtest
- Use `newTestClient` + `http.HandlerFunc` for success paths
- Use `jsonResponse` helper for error paths
- Assert with `require.NoError`/`require.Error` + `assert.Equal`/`assert.True`

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] `UpdateServerSettings` method added to `SettingsScope`
- [ ] Unit tests added for success and error paths
- [ ] Duplicate analytics PUT variant is intentionally not wrapped (no second analytics method)
- [ ] `docs/api-client-implementation-status.md` updated to mark Settings as fully implemented, noting the PUT analytics variant is deliberately excluded
- [ ] Lint and tests pass
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
