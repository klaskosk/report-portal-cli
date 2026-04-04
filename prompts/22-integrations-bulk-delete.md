# Prompt 22: Integrations Bulk Delete-by-Type

## Goal

Complete the `IntegrationsScope` and `ProjectIntegrationsScope` in `pkg/client` by adding delete-all-by-type endpoints for both global and project-level integrations.

## Context

The integration scopes cover most CRUD but are missing the bulk delete-by-type operations:

| Wrapped? | Scope | Operation | Generated method | HTTP | Path |
| --- | --- | --- | --- | --- | --- |
| No | Instance | Delete all by type | `DeleteAllIntegrationsWithResponse` | `DELETE` | `/v1/integration/all/{type}` |
| No | Project | Delete all by type | `DeleteAllProjectIntegrationsWithResponse` | `DELETE` | `/v1/integration/{projectName}/all/{type}` |

Already implemented on `IntegrationsScope`: `ListGlobal`, `ListGlobalByPlugin`, `GetGlobal`, `TestGlobalConnection`, `CreateGlobal`, `UpdateGlobal`, `DeleteGlobal`.

Already implemented on `ProjectIntegrationsScope`: `List`, `ListByPlugin`, `Get`, `TestConnection`, `Create`, `Update`, `Delete`, `ExecuteCommand`.

## Files to modify

- `pkg/client/integrations.go` — add new methods to both scopes
- `pkg/client/integrations_test.go` — add tests for new methods

## Implementation plan

### 1) Add `DeleteAllByType` to `IntegrationsScope`

Wrap `DeleteAllIntegrationsWithResponse` to delete all global integrations of a given type.

Key details:

- **Generated method**: `s.client.raw.DeleteAllIntegrationsWithResponse(ctx, pluginType)`
- **Path parameter**: `pType string` — the integration type name (e.g., `"jira"`, `"rally"`)
- **Response**: `JSON200` is `*types.OperationCompletionRS`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `DELETE /v1/integration/all/{type}`
- **Logger**: `s.client.logger.With(slog.String("operation", "DeleteAllIntegrationsByType"), slog.String("type", pluginType))`
- **Error wrapping**: `fmt.Errorf("delete all integrations by type: %w", err)`

Signature:

```go
func (s *IntegrationsScope) DeleteAllByType(
    ctx context.Context,
    pluginType string,
) (*types.OperationCompletionRS, error)
```

### 2) Add `DeleteAllByType` to `ProjectIntegrationsScope`

Wrap `DeleteAllProjectIntegrationsWithResponse` to delete all project-level integrations of a given type.

Key details:

- **Generated method**: `s.project.client.raw.DeleteAllProjectIntegrationsWithResponse(ctx, s.project.projectName, pluginType)`
- **Path parameters**: `projectName` (from scope) + `pType string`
- **Response**: `JSON200` is `*types.OperationCompletionRS`; `JSONDefault` is `*types.ErrorRS`
- **HTTP**: `DELETE /v1/integration/{projectName}/all/{type}`
- **Logger**: Use `s.project.operationLogger("DeleteAllProjectIntegrationsByType", slog.String("type", pluginType))`
- **Error wrapping**: `fmt.Errorf("delete all project integrations by type: %w", err)`

Signature:

```go
func (s *ProjectIntegrationsScope) DeleteAllByType(
    ctx context.Context,
    pluginType string,
) (*types.OperationCompletionRS, error)
```

### 3) Write tests

Follow existing patterns in `integrations_test.go`, which has separate test blocks for global vs project scopes.

For each scope's `DeleteAllByType`, add:

- **Success test**: Verify path (global: `/api/v1/integration/all/jira`; project: `/api/v1/integration/my_project/all/jira`), HTTP method (`DELETE`), bearer token, and response parsing
- **Error test**: Verify `client.IsUnauthorized` with a JSON error response

Test naming:
- `TestIntegrationsScope_DeleteAllByType_Success`
- `TestIntegrationsScope_DeleteAllByType_Unauthorized`
- `TestProjectIntegrationsScope_DeleteAllByType_Success`
- `TestProjectIntegrationsScope_DeleteAllByType_Unauthorized`

## Test requirements

- All tests in `package client_test` (black-box)
- `t.Parallel()` on every test function and subtest
- Global scope: access via `c.Integrations().DeleteAllByType(...)`
- Project scope: access via `c.Project("my_project").Integrations().DeleteAllByType(...)`
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

- [ ] `DeleteAllByType` method added to `IntegrationsScope`
- [ ] `DeleteAllByType` method added to `ProjectIntegrationsScope`
- [ ] Unit tests added for success and error paths of both methods
- [ ] `docs/api-client-implementation-status.md` updated to mark Integration as fully implemented
- [ ] Lint and tests pass
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
