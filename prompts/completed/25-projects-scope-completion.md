# PR 8: ProjectsScope Completion

## Goal

Complete the `ProjectsScope` in `pkg/client/projects.go` by wrapping every remaining `/v1/project` endpoint that the generated client already supports. After this PR, every synchronous project-management operation in the OpenAPI spec will be accessible through the high-level `pkg/client` API.

## Context

`ProjectsScope` already covers the most common project CRUD and user-assignment flows (11 methods). The generated client in `pkg/internal/api/` exposes additional operations that have no high-level wrapper yet. This PR fills those gaps without changing any existing method signatures.

### Already implemented (do not modify signatures)

| Method | Generated operation | Path |
|--------|-------------------|------|
| `Get` | `GetProjectWithResponse` | `GET /v1/project/{projectName}` |
| `ListNames` | `GetAllProjectNamesWithResponse` | `GET /v1/project/names` |
| `List` | `GetAllProjectsInfoWithResponse` | `GET /v1/project/list` |
| `SearchNames` | `SearchProjectNamesWithResponse` | `GET /v1/project/names/search` |
| `GetInfo` | `GetProjectInfoWithResponse` | `GET /v1/project/list/{projectName}` |
| `Create` | `CreateProjectWithResponse` | `POST /v1/project` |
| `Delete` | `DeleteProjectWithResponse` | `DELETE /v1/project?ids=` |
| `Update` | `UpdateProjectWithResponse` | `PUT /v1/project/{projectName}` |
| `AssignUsers` | `AssignProjectUsersWithResponse` | `PUT /v1/project/{projectName}/assign` |
| `UnassignUsers` | `UnassignProjectUsersWithResponse` | `PUT /v1/project/{projectName}/unassign` |
| `ListUsers` | `GetProjectUsersWithResponse` | `GET /v1/project/{projectName}/users` |

### Methods to add

| Proposed method | Generated operation | Path | Notes |
|----------------|-------------------|------|-------|
| `Export` | `ExportProjectsWithResponse` | `GET /v1/project/export` | Returns a Jasper report (raw bytes, not JSON). Follow the same non-JSON response pattern used by `LaunchScope.GetLaunchReport` and `PluginScope.GetPublicFile`. Accept functional options for query filters. |
| `GetAnalyzerStatus` | `GetAnalyzerIndexingStatusWithResponse` | `GET /v1/project/analyzer/status` | Returns analyzer indexing status. Simple no-arg call. |
| `UpdateNotificationConfig` | `UpdateProjectNotificationConfigWithResponse` | `PUT /v1/project/{projectName}/notification` | Accepts a `projectName` and a request body. |
| `GetPreference` | `GetUserPreferenceWithResponse` | `GET /v1/project/{projectName}/preference` | Returns the current user's preference for a project. |
| `AddPreference` | `AddUserPreferenceWithResponse` | `PUT /v1/project/{projectName}/preference/{filterId}` | Adds a filter to the user's project preferences. |
| `RemovePreference` | `RemoveUserPreferenceWithResponse` | `DELETE /v1/project/{projectName}/preference/{filterId}` | Removes a filter from the user's project preferences. |
| `DeleteIndex` | `DeleteProjectIndexWithResponse` | `DELETE /v1/project/{projectName}/index` | Deletes the project's analyzer index. |
| `IndexData` | `IndexProjectDataWithResponse` | `PUT /v1/project/{projectName}/index` | Triggers re-indexing of the project's data. |
| `ListAssignableUsers` | `GetUsersForAssignWithResponse` | `GET /v1/project/{projectName}/assignable` | Returns users available for assignment. Accept functional options for query/pagination. |
| `ListUsernames` | `GetProjectUsers1WithResponse` | `GET /v1/project/{projectName}/usernames` | Returns usernames assigned to a project (lightweight string list). Accept functional options. |
| `SearchUsernames` | `SearchForUserWithResponse` | `GET /v1/project/{projectName}/usernames/search` | Searches usernames within a project by term. Accept functional options. |
| `GetWidgetByCode` | `GetProjectWidgetWithResponse` | `GET /v1/project/{projectName}/widget/{widgetCode}` | Returns a preconfigured project widget by its code. Accept functional options for query params. |

## Implementation plan

### 1) Discover response types and params

Before writing any code, inspect the generated client to understand each operation's response struct and parameter types. The generated types live in `pkg/internal/api/api.gen.go`. For each operation above:

- Find the `*Response` struct and check which `JSONxxx` fields it has (e.g., `JSON200`, `JSON201`, `JSONDefault`).
- Find the `*Params` struct if one exists.
- Note whether the response is JSON or raw bytes (e.g., `Export` returns a Jasper report).

### 2) Add methods to `ProjectsScope`

Add each new method to `pkg/client/projects.go`. Follow the established pattern exactly:

```go
func (p *ProjectsScope) MethodName(
    ctx context.Context,
    // ... parameters ...
) (*types.ReturnType, error) {
    logger := p.client.logger.With(
        slog.String("operation", "OperationName"),
        // ... additional structured fields ...
    )
    logger.InfoContext(ctx, "API request")

    response, err := p.client.raw.GeneratedOperationWithResponse(ctx, ...)
    if err != nil {
        return nil, fmt.Errorf("operation name: %w", err)
    }

    logger.DebugContext(ctx, "API response", "status", response.StatusCode())

    result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
    if err != nil {
        return nil, fmt.Errorf("operation name: %w", err)
    }

    return result, nil
}
```

For `Export`, which returns raw bytes instead of JSON, follow the non-JSON pattern from `GetPublicFile` or `GetLaunchReport`: check `response.JSONDefault` for API errors, then check the status code manually, and return `response.Body` directly.

### 3) Add functional options for parameterized endpoints

For methods that accept query parameters (like `Export`, `ListAssignableUsers`, `ListUsernames`, `SearchUsernames`, `GetWidgetByCode`), define functional option types following the existing patterns:

- Define an option interface (e.g., `ListAssignableUsersOption`)
- Define a private `func` adapter type
- Export public option constructors (e.g., `WithAssignableUserPageSize`)

Look at `ListProjectsOption` and `ListProjectUsersOption` in the existing code for the exact pattern.

### 4) Handle `ExportProjectsParams`

`Export` takes the same filter/pagination params as `List` (project name, type, page, size, sort). You may either reuse `ListProjectsOption` (if the generated params struct is compatible) or define a separate `ExportProjectsOption` type. Prefer reuse if the params structs share the same field names and types.

## Test requirements

### Unit tests

Add tests in `pkg/client/projects_test.go` using the existing black-box test pattern (`package client_test`). Every test must call `t.Parallel()`.

For each new method, write at minimum:

1. **Success test**: Set up an `httptest` server returning a representative success response. Assert the returned value's key fields. Assert the correct HTTP method, path, and (where applicable) query parameters.
2. **Error test**: At least one error case using `jsonResponse` helper with an appropriate status code. Assert the correct predicate function (`client.IsNotFound`, `client.IsUnauthorized`, etc.).

For methods with functional options, add an additional test verifying that options produce the expected query parameters.

For `Export`, test the raw-bytes response path (non-JSON), similar to how `TestGetPublicFile_Success` or launch report tests work.

Use the existing test helpers: `newTestClient`, `assertBearerToken`, `assertMethod`, `assertPath`, `jsonResponse`.

### Test style

- Black-box: `package client_test`
- `t.Parallel()` on every test and subtest
- Table tests (`[]struct{}` + `t.Run`) for error-case variants
- No test-only exports needed for `ProjectsScope` (the existing `newTestClient` + `c.Projects()` pattern works)

## Files to modify

| File | Change |
|------|--------|
| `pkg/client/projects.go` | Add 12 new methods and their functional option types |
| `pkg/client/projects_test.go` | Add corresponding unit tests |

## Files NOT to modify

- `pkg/client/client.go` — `Projects()` already exists
- `pkg/client/project.go` — this is `ProjectScope` (singular), not `ProjectsScope`
- `pkg/internal/api/` — generated code, do not hand-edit
- `pkg/types/` — generated types, do not hand-edit

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] All 12 methods from the table above are implemented in `projects.go`
- [ ] Each method follows the standard logging/error-handling pattern
- [ ] Functional options are defined for parameterized endpoints
- [ ] Unit tests cover success and at least one error case per method
- [ ] Option tests verify query parameter wiring for parameterized methods
- [ ] `Export` uses the raw-bytes response pattern (not `handleResponse`)
- [ ] All error messages are lowercase with no trailing punctuation
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
