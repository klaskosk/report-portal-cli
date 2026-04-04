# Prompt 19: Dashboard Completion

## Goal

Complete the `DashboardScope` in `pkg/client` by adding dashboard config retrieval and preconfigured dashboard creation.

## Context

`DashboardScope` covers core CRUD and widget management but is missing two endpoints:

| Wrapped? | Operation | Generated method | HTTP | Path |
| --- | --- | --- | --- | --- |
| No | Get dashboard config | `GetDashboardConfigWithResponse` | `GET` | `/v1/{projectName}/dashboard/{dashboardId}/config` |
| No | Create preconfigured dashboard | `CreatePreconfiguredWithResponse` | `POST` | `/v1/{projectName}/dashboard/preconfigured` |

Already implemented: `List`, `GetByID`, `Create`, `Update`, `Delete`, `AddWidget`, `RemoveWidget`.

## Files to modify

- `pkg/client/dashboard.go` — add new methods
- `pkg/client/dashboard_test.go` — add tests for new methods

## Implementation plan

### 1) Add `GetConfig` method

Wrap `GetDashboardConfigWithResponse` to retrieve the widget configuration for a specific dashboard.

Key details:

- **Generated method**: `d.project.client.raw.GetDashboardConfigWithResponse(ctx, d.project.projectName, dashboardID)`
- **Response**: `JSON200` is `*types.DashboardConfigResource`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `GET /v1/{projectName}/dashboard/{dashboardId}/config`
- **Logger**: Use `d.project.operationLogger("GetDashboardConfig", slog.Int64("id", dashboardID))`
- **Error wrapping**: `fmt.Errorf("get dashboard config: %w", err)`

Signature:

```go
func (d *DashboardScope) GetConfig(
    ctx context.Context,
    dashboardID int64,
) (*types.DashboardConfigResource, error)
```

The `DashboardConfigResource` type has a single field: `Widgets *[]WidgetConfigResource`. Each `WidgetConfigResource` pairs a `WidgetObjectModel` with a `WidgetResource`. If the response has a nil `Widgets` field, consider returning an empty struct rather than nil to avoid nil-pointer surprises for callers.

### 2) Add `CreatePreconfigured` method

Wrap `CreatePreconfiguredWithResponse` to create a dashboard from a preconfigured template.

Key details:

- **Generated method**: `d.project.client.raw.CreatePreconfiguredWithResponse(ctx, d.project.projectName, body)`
- **Request body type**: `types.DashboardPreconfiguredRq` — fields: `Config DashboardConfigResource`, `Description *string`, `Name string`
- **Response**: `JSON200` is `*types.EntryCreatedRS`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `POST /v1/{projectName}/dashboard/preconfigured`
- **Logger**: Use `d.project.operationLogger("CreatePreconfiguredDashboard", slog.String("name", request.Name))`
- **Error wrapping**: `fmt.Errorf("create preconfigured dashboard: %w", err)`

Signature:

```go
func (d *DashboardScope) CreatePreconfigured(
    ctx context.Context,
    request types.DashboardPreconfiguredRq,
) (*types.EntryCreatedRS, error)
```

### 3) Write tests

Follow the existing patterns in `dashboard_test.go`. For each method add:

- **Success test**: Verify path, HTTP method, bearer token, and response parsing
- **Error test(s)**: At minimum an unauthorized case using `client.IsUnauthorized`; optionally a not-found case for `GetConfig`

Test naming examples:
- `TestDashboardScope_GetConfig_Success`
- `TestDashboardScope_GetConfig_NotFound`
- `TestDashboardScope_CreatePreconfigured_Success`
- `TestDashboardScope_CreatePreconfigured_Unauthorized`

## Test requirements

- All tests in `package client_test` (black-box)
- `t.Parallel()` on every test function and subtest
- Use `newTestClient` + `http.HandlerFunc` for success paths; access via `c.Project("my_project").Dashboards().Method(...)`
- Use `jsonResponse` helper for error paths
- Assert with `require.NoError`/`require.Error` + `assert.Equal`/`assert.True`

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] `GetConfig` method added to `DashboardScope`
- [ ] `CreatePreconfigured` method added to `DashboardScope`
- [ ] Unit tests added for success and error paths of both methods
- [ ] `docs/api-client-implementation-status.md` updated to mark Dashboard as fully implemented
- [ ] Lint and tests pass
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
