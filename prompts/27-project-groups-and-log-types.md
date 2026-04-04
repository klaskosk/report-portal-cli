# Prompt 18: Project Groups and Log Types (Backlog PR 10)

## Goal

Add two new project-scoped child scopes to `ProjectScope`:

1. **`ProjectGroupsScope`** — CRUD for group-project associations under `/projects/{project_name}/groups/...`
2. **`LogTypesScope`** — CRUD for project log types under `/projects/{project_name}/log-types/...`

Both scopes are entirely absent from `pkg/client` today. The generated client already contains all needed operations.

## Context

The 5.15 OpenAPI spec introduced the `projects` tag, which defines project-group linking and log-type management endpoints. These are project-scoped resources accessed via `Client.Project(name)`, unlike the instance-level `GroupsScope` (PR 9) which hangs off `Client` directly. This PR adds two new accessors on `ProjectScope` and two new scope files.

## Endpoints to wrap

### Project groups (`/projects/{project_name}/groups/...`)

| HTTP | Path | operationId | Generated method | Response type |
|------|------|-------------|-----------------|---------------|
| GET | `/projects/{project_name}/groups` | `getGroupsOfProject` | `GetGroupsOfProjectWithResponse` | `ProjectGroupsPage` (200) |
| GET | `/projects/{project_name}/groups/{group_id}` | `getProjectGroupById` | `GetProjectGroupByIdWithResponse` | `ProjectGroupInfo` (200) |
| PUT | `/projects/{project_name}/groups/{group_id}` | `addGroupToProjectById` | `AddGroupToProjectByIdWithResponse` | `SuccessfulUpdate` (200) |
| DELETE | `/projects/{project_name}/groups/{group_id}` | `deleteGroupFromProjectById` | `DeleteGroupFromProjectByIdWithResponse` | 204 no-content |

**Pagination**: The list endpoint accepts `GetGroupsOfProjectParams` with `Limit` (default 300, max 1000) and `Offset`. Expose these as functional options following the project conventions (e.g. `WithGroupPageSize`, `WithGroupPageOffset`).

### Log types (`/projects/{project_name}/log-types/...`)

| HTTP | Path | operationId | Generated method | Response type |
|------|------|-------------|-----------------|---------------|
| GET | `/projects/{project_name}/log-types` | `getLogTypes` | `GetLogTypesWithResponse` | `GetLogTypes200Response` (200) |
| POST | `/projects/{project_name}/log-types` | `createLogType` | `CreateLogTypeWithResponse` | `LogTypeResponse` (200) |
| PUT | `/projects/{project_name}/log-types/{log_type_id}` | `updateLogTypeById` | `UpdateLogTypeByIdWithResponse` | `SuccessfulUpdate` (200) |
| DELETE | `/projects/{project_name}/log-types/{log_type_id}` | `deleteLogTypeById` | `DeleteLogTypeByIdWithResponse` | 204 no-content |

## Key types (from `pkg/types`)

- `ProjectGroupsPage` — paginated list with `Items *[]ProjectGroupInfo`, `Limit`, `Offset`, `TotalCount`, `Sort`, `Order`
- `ProjectGroupInfo` — `Id`, `Name`, `Slug`, `Permission`, `AddedAt`, `Stats`
- `AddProjectToGroupByIdRequest` — `Permission` field (enum: see `AddProjectToGroupByIdRequestPermission`)
- `GetLogTypes200Response` — `Items *[]LogTypeResponse`
- `LogTypeRequest` — `Name`, `Level` (int32), `IsFilterable` (bool), `Style` (optional)
- `LogTypeResponse` — `Id`, `Name`, `Level`, `IsFilterable`, `IsSystem`, plus `Style` fields
- `SuccessfulUpdate` — `Message *string`

## Implementation plan

### 1) Handle 204 no-content responses

Both delete operations return HTTP 204 with no body. The existing `handleResponse` rejects nil success pointers. You need a way to handle these. Options:

- Add a `handleNoContentResponse` helper in `response.go` that checks `JSONDefault` for error bodies and validates the status is 2xx, but does not expect a success payload.
- Or return the `error` directly after checking `JSONDefault` and the status code inline.

Whichever approach you choose, make sure it is consistent and testable. If you add a helper, add unit tests for it in `response_test.go`. Prefer a helper if it avoids duplicating the same inline logic in multiple delete methods.

### 2) Create `pkg/client/project_groups.go`

Define `ProjectGroupsScope`:

```go
type ProjectGroupsScope struct {
    project *ProjectScope
}
```

Methods to implement:

- `List(ctx, ...ListOption) (*types.ProjectGroupsPage, error)` — wraps `GetGroupsOfProjectWithResponse`. Define functional options for `Limit` and `Offset`.
- `Get(ctx, groupID int64) (*types.ProjectGroupInfo, error)` — wraps `GetProjectGroupByIdWithResponse`.
- `AddOrUpdate(ctx, groupID int64, request types.AddProjectToGroupByIdRequest) (*types.SuccessfulUpdate, error)` — wraps `AddGroupToProjectByIdWithResponse`. This is a PUT that either adds or updates the group-project association.
- `Delete(ctx, groupID int64) error` — wraps `DeleteGroupFromProjectByIdWithResponse`. Returns only `error` since the API returns 204 no-content.

Follow the standard pattern from existing scopes:

1. Build logger via `s.project.operationLogger("OperationName", ...attrs)`
2. `InfoContext` before the call
3. Call the generated `WithResponse` method
4. `DebugContext` with status after the call
5. Use `handleResponse` (or `handleNoContentResponse` for deletes)
6. Wrap errors with `fmt.Errorf("operation name: %w", err)`

### 3) Create `pkg/client/log_types.go`

Define `LogTypesScope`:

```go
type LogTypesScope struct {
    project *ProjectScope
}
```

Methods to implement:

- `List(ctx) ([]types.LogTypeResponse, error)` — wraps `GetLogTypesWithResponse`. Unwrap from `GetLogTypes200Response.Items`. Return `[]types.LogTypeResponse{}` (not nil) when the items list is nil/empty.
- `Create(ctx, request types.LogTypeRequest) (*types.LogTypeResponse, error)` — wraps `CreateLogTypeWithResponse`.
- `Update(ctx, logTypeID int64, request types.LogTypeRequest) (*types.SuccessfulUpdate, error)` — wraps `UpdateLogTypeByIdWithResponse`.
- `Delete(ctx, logTypeID int64) error` — wraps `DeleteLogTypeByIdWithResponse`. Returns only `error` (204 no-content).

### 4) Add accessors on `ProjectScope`

In `pkg/client/project.go`, add:

```go
func (p *ProjectScope) Groups() *ProjectGroupsScope {
    return &ProjectGroupsScope{project: p}
}

func (p *ProjectScope) LogTypes() *LogTypesScope {
    return &LogTypesScope{project: p}
}
```

### 5) Add test constructors in `export_test.go`

If needed for external test packages, add test constructors following the existing pattern:

```go
var NewTestProjectGroupsScope = func(c *Client, projectName string) *ProjectGroupsScope {
    return c.Project(projectName).Groups()
}

var NewTestLogTypesScope = func(c *Client, projectName string) *LogTypesScope {
    return c.Project(projectName).LogTypes()
}
```

## Test requirements

### Unit tests

Create `pkg/client/project_groups_test.go` and `pkg/client/log_types_test.go` in package `client_test`.

For **each** method, write at minimum:

- **Success case**: httptest handler that asserts method, path, bearer token, and (for write ops) request body. Return a realistic JSON response and assert the parsed result fields.
- **Error case**: At least one error path per method (not-found, forbidden, or unauthorized as appropriate). Use `jsonResponse` helper and assert with `client.IsNotFound`, `client.IsForbidden`, etc.
- **Edge cases**:
  - `List` returning empty items
  - `Delete` success returning 204 with empty body
  - `Delete` returning an error body on failure

Use `t.Parallel()` on every test. Use `newTestClient(t, handler)` to create the client. Use `assertBearerToken`, `assertMethod`, `assertPath` helpers.

Expected paths to assert:

- `/api/projects/{project_name}/groups` (list)
- `/api/projects/{project_name}/groups/{group_id}` (get/put/delete)
- `/api/projects/{project_name}/log-types` (list/create)
- `/api/projects/{project_name}/log-types/{log_type_id}` (update/delete)

### Test patterns to follow

```go
func TestProjectGroupsScope_List_Success(t *testing.T) {
    t.Parallel()

    c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assertBearerToken(t, r, "test-token")
        assertPath(t, r, "/api/projects/my_project/groups")
        assertMethod(t, r, http.MethodGet)

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{ ... }`))
    }))

    result, err := c.Project("my_project").Groups().List(t.Context())
    require.NoError(t, err)
    // assertions...
}
```

## Files to create or modify

| File | Action |
|------|--------|
| `pkg/client/project_groups.go` | Create — `ProjectGroupsScope` with list/get/add-or-update/delete |
| `pkg/client/project_groups_test.go` | Create — unit tests for all methods |
| `pkg/client/log_types.go` | Create — `LogTypesScope` with list/create/update/delete |
| `pkg/client/log_types_test.go` | Create — unit tests for all methods |
| `pkg/client/project.go` | Modify — add `Groups()` and `LogTypes()` accessors |
| `pkg/client/response.go` | Possibly modify — add `handleNoContentResponse` if needed |
| `pkg/client/response_test.go` | Possibly modify — test the new helper |
| `pkg/client/export_test.go` | Possibly modify — add test constructors |

## Design considerations

- The delete endpoints return 204 no-content. This is the first time the client must handle a successful response with no body. Handle this cleanly and consistently.
- `GetLogTypes200Response` has an `Items *[]LogTypeResponse` field. Unwrap this so the public API returns `[]LogTypeResponse` directly, consistent with other list methods.
- The list groups endpoint uses offset/limit pagination (not the page-based `FilterCondition` pattern from launches). Model the options accordingly.
- `AddGroupToProjectById` is a PUT that performs upsert semantics. Name the method `AddOrUpdate` to make the semantics clear.

## Validation commands

Run before finalizing:

- `make lint`
- `make test`

## Definition of done

- [ ] `ProjectGroupsScope` implemented with List/Get/AddOrUpdate/Delete
- [ ] `LogTypesScope` implemented with List/Create/Update/Delete
- [ ] `ProjectScope.Groups()` and `ProjectScope.LogTypes()` accessors added
- [ ] 204 no-content responses handled correctly for both delete operations
- [ ] Unit tests for every method with success, error, and edge cases
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
