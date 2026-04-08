# PR 9: GroupsScope

## Goal

Create a new `GroupsScope` providing instance-level group CRUD, group-user membership, and group-project linking. This is a brand-new scope — no existing code covers groups today.

## Context

The 5.15 OpenAPI spec introduced a `groups` tag with endpoints under `/api/groups/...` (not under `/v1/...`). The generated client in `pkg/internal/api/` already exposes typed operations for all of these. This PR wraps them in a new `GroupsScope` and wires it into `Client`.

### Generated operations to wrap

| Proposed method | Generated operation | HTTP | Path | Notes |
|----------------|-------------------|------|------|-------|
| `List` | `GetGroupsWithResponse` | GET | `/groups` | Accepts pagination params (`limit`, `offset`, `order`, `sort`). Returns a paginated groups list. |
| `Get` | `GetGroupByIdWithResponse` | GET | `/groups/{group_id}` | Returns a single group by numeric ID. |
| `Create` | `CreateGroupWithResponse` | POST | `/groups` | Accepts a JSON request body. |
| `Update` | `UpdateGroupWithResponse` | PUT | `/groups/{group_id}` | Accepts a group ID and JSON request body. |
| `Delete` | `DeleteGroupWithResponse` | DELETE | `/groups/{group_id}` | Deletes a group by numeric ID. |
| `ListProjects` | `GetGroupProjectsWithResponse` | GET | `/groups/{group_id}/projects` | Accepts pagination params. |
| `GetProject` | `GetGroupProjectByIdWithResponse` | GET | `/groups/{group_id}/projects/{project_id}` | Returns one project-group link. |
| `AddProject` | `AddProjectToGroupByIdWithResponse` | PUT | `/groups/{group_id}/projects/{project_id}` | Accepts a JSON request body with project-group configuration. |
| `RemoveProject` | `DeleteProjectFromGroupByIdWithResponse` | DELETE | `/groups/{group_id}/projects/{project_id}` | Removes a project from a group. |
| `ListUsers` | `GetGroupUsersWithResponse` | GET | `/groups/{group_id}/users` | Accepts pagination params. |
| `GetUser` | `GetGroupUserByIdWithResponse` | GET | `/groups/{group_id}/users/{user_id}` | Returns one user-group link. |
| `AddUser` | `AddUserToGroupByIdWithResponse` | PUT | `/groups/{group_id}/users/{user_id}` | Adds a user to a group. |
| `RemoveUser` | `DeleteUserFromGroupByIdWithResponse` | DELETE | `/groups/{group_id}/users/{user_id}` | Removes a user from a group. |

## Implementation plan

### 1) Discover response types

Before writing any code, inspect `pkg/internal/api/api.gen.go` for each generated operation listed above:

- Find the `*Response` struct and identify which `JSONxxx` fields it provides.
- Find the `*Params` struct if one exists (e.g., `GetGroupsParams`, `GetGroupProjectsParams`, `GetGroupUsersParams`).
- Check what request body types are expected (e.g., `CreateGroupJSONRequestBody`, `UpdateGroupJSONRequestBody`, `AddProjectToGroupByIdJSONRequestBody`).
- Map generated request/response types to their `pkg/types` equivalents (or confirm they alias `externalRef0.*` types).

Pay special attention: the groups API uses `/api/groups/...` (no `/v1/` prefix). The generated client handles this, but verify the paths when writing test assertions.

### 2) Create `pkg/client/groups.go`

Define the scope struct and all methods:

```go
type GroupsScope struct {
    client *Client
}
```

Follow the exact pattern used by `UsersScope` and `ProjectsScope`:

- Each method takes `context.Context` as the first parameter.
- Build a logger with `s.client.logger.With(slog.String("operation", "..."))`.
- Call the generated `s.client.raw.<Operation>WithResponse(...)`.
- Use `handleResponse(response, response.JSONDefault, response.JSONxxx, response.Body)` for JSON responses.
- Wrap errors with `fmt.Errorf("verb noun: %w", err)`.

### 3) Add functional options for paginated endpoints

`List`, `ListProjects`, and `ListUsers` accept pagination parameters. Define functional option types for each:

```go
type ListGroupsOption interface {
    applyListGroups(*api.GetGroupsParams)
}
```

Export option constructors like `WithGroupPageSize`, `WithGroupOffset`, `WithGroupSort`, `WithGroupOrder`, etc. Follow the `ListProjectsOption` pattern exactly.

Similarly for `ListGroupProjectsOption` and `ListGroupUsersOption`.

### 4) Wire into `client.go`

Add the factory method to `Client`:

```go
// Groups returns a GroupsScope for instance-level group operations.
func (c *Client) Groups() *GroupsScope {
    return &GroupsScope{client: c}
}
```

Place this alphabetically among the existing scope factory methods.

### 5) Design considerations

- **Flat scope vs. nested scope**: Keep `GroupsScope` flat (all methods receive `groupID int64` as a parameter) rather than introducing a `GroupScope` intermediary (like `ProjectScope`). The groups API surface is small enough that nesting would add complexity without benefit. If the agent disagrees after examining the API surface, a nested `Group(id int64) *GroupScope` pattern (like `Client.Project(name)`) is acceptable but must be justified.
- **Method naming**: Use `AddProject`/`RemoveProject`/`AddUser`/`RemoveUser` for membership operations. These are clearer than `Assign`/`Unassign` given the group context.

## Test requirements

### Unit tests

Create `pkg/client/groups_test.go` using the existing black-box test pattern (`package client_test`). Every test must call `t.Parallel()`.

For each method, write at minimum:

1. **Success test**: Set up an `httptest` server returning a representative JSON response. Assert the returned value's key fields. Assert the correct HTTP method and path (note: paths are under `/api/groups/...`, not `/api/v1/groups/...`).
2. **Error test**: At least one error case per method. Use table tests for multiple error scenarios when appropriate.

For paginated methods, add a test verifying that functional options produce the expected query parameters.

### Test helpers

Use the existing helpers: `newTestClient`, `assertBearerToken`, `assertMethod`, `assertPath`, `jsonResponse`. Access the new scope via `c.Groups()`.

## Files to create

| File | Content |
|------|---------|
| `pkg/client/groups.go` | `GroupsScope` struct and all 13 methods |
| `pkg/client/groups_test.go` | Corresponding unit tests |

## Files to modify

| File | Change |
|------|--------|
| `pkg/client/client.go` | Add `Groups() *GroupsScope` factory method |

## Files NOT to modify

- `pkg/client/project.go` — project-scoped group endpoints are a separate PR (PR 10)
- `pkg/internal/api/` — generated code, do not hand-edit
- `pkg/types/` — generated types, do not hand-edit

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] `GroupsScope` struct created in `groups.go`
- [ ] All 13 methods from the table above are implemented
- [ ] Functional options defined for `List`, `ListProjects`, and `ListUsers`
- [ ] `Client.Groups()` factory method added to `client.go`
- [ ] Unit tests cover success and error cases for every method
- [ ] Option tests verify query parameter wiring for paginated methods
- [ ] All error messages are lowercase with no trailing punctuation
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
