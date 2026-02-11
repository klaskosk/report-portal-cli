# Prompt 04: Identity + Discovery (Users + Projects)

## Goal

Add user identity and project discovery to both the Go client and CLI. This corresponds to Milestone 7 (Project discovery + identity) and the "add identity + discovery follow-up" PR from the suggested next steps.

## Context

Currently the CLI requires users to know their project name upfront (via `--project` flag or config profile). There's no way to discover available projects or verify credentials. Adding `rp whoami` and `rp get project` enables a better onboarding experience and supports automation scripts that need to enumerate projects.

## Generated API endpoints available

The generated client in `pkg/internal/api/api.gen.go` has these relevant methods:

### User endpoints (deprecated-user-controller, v5.11 current)

| Generated method | What it does |
|---|---|
| `GetUserUsingGET1WithResponse(ctx, login)` | Get a specific user by login name |
| `GetUsersUsingGET1WithResponse(ctx, params)` | List users with filters |
| `GetUserProjectsUsingGET1WithResponse(ctx, userName)` | List projects for a user |

Note: there is also a non-deprecated `user-controller` set. The `1` suffix methods are from the deprecated controller but are the ones that work with v5.11's token-authenticated API. Investigate which set of endpoints the bearer token works against — the standard user endpoints may require admin credentials.

### Project endpoints (project-controller)

| Generated method | What it does |
|---|---|
| `GetProjectUsingGETWithResponse(ctx, projectName)` | Get full project details by name |
| `GetAllProjectsInfoUsingGETWithResponse(ctx, params)` | List all projects (admin-level) |
| `GetProjectInfoUsingGETWithResponse(ctx, projectName, params)` | Get project summary info |
| `GetAllProjectNamesUsingGETWithResponse(ctx)` | Get all project names (lightweight) |
| `SearchProjectNamesUsingGETWithResponse(ctx, params)` | Search project names |

For MVP, focus on `GetProjectUsingGET` (get by name) and determine the best "list my projects" endpoint. `GetUserProjectsUsingGET1` may be the most practical since it returns only projects the authenticated user has access to, but it requires knowing the username first — which creates a chicken-and-egg problem that `whoami` solves.

## Client requirements

### Users scope

Create `pkg/client/users.go` with a `UsersScope` struct:

- `UsersScope` should be accessible from `Client` directly (instance-scoped, not project-scoped): `client.Users()`
- Add a `Users()` method to `Client` in `pkg/client/client.go`
- Methods to implement:
  - `GetMyself(ctx) (*types.UserResource, error)` — get the currently authenticated user. This is a convenience that calls the underlying user endpoint with the authenticated user's login. Note: Report Portal doesn't have a dedicated `/me` endpoint, so you'll need to either:
    - Parse the JWT token to extract the username, then call `GetUserUsingGET1`, OR
    - Use another mechanism to determine the current user (investigate what the API offers)
  - `GetByLogin(ctx, login) (*types.UserResource, error)` — get a user by login name
  - `ListProjects(ctx, username) ([]types.OwnedEntityResource, error)` — list projects accessible to a user

### Projects scope

Create `pkg/client/projects.go` with a `ProjectsScope` struct:

- `ProjectsScope` should be accessible from `Client` directly: `client.Projects()`
- Add a `Projects()` method to `Client` in `pkg/client/client.go`
- Methods to implement:
  - `Get(ctx, projectName) (*types.ProjectResource, error)` — get project details by name
  - `ListNames(ctx) ([]string, error)` — list all project names (lightweight)

## Client tests

Create `pkg/client/users_test.go` and `pkg/client/projects_test.go` (black-box, `package client_test`):

- `GetMyself`: success, unauthorized
- `GetByLogin`: success, not found, unauthorized
- `ListProjects`: success, empty list
- `Get` (project): success, not found, unauthorized
- `ListNames`: success, empty list
- All tests must use `t.Parallel()`

## CLI command requirements

### `rp whoami`

Create `internal/cli/whoami.go`:

- Register in `internal/cli/root.go` as a top-level command (not under `get` — it's a convenience verb)
- No required flags beyond the global connection flags
- Output:
  - Table mode: user info in a key-value format (Login, Full Name, Email, Role, Projects)
  - JSON mode: full `UserResource` as JSON
- On success, also list the user's accessible projects (call `ListProjects`)

### `rp get project`

Create `internal/cli/project.go`:

- Register in `internal/cli/get.go` via `cmd.AddCommand(newGetProjectCmd())`
- Command shape: `rp get project [name]`
  - No args → list projects accessible to the current user
  - Positional arg → get project details by name
- Flags:
  - Standard list flags are not needed for the names-only list, but include them in case `ListAll` is added later
- Table output columns: for list mode, just project names; for get mode, Name, ID, Organization, Created, Entry Type
- JSON output: full resource structure

## Integration tests

Add to `test/integration/`:

- `client_users_test.go`:
  - `GetMyself` — verify returns the authenticated user (superadmin in the test stack)
  - `GetByLogin` — verify for known user, not-found for unknown
  - `ListProjects` — verify returns at least the default project
- `client_projects_test.go`:
  - `Get` — verify for the default project, not-found for unknown
  - `ListNames` — verify returns at least the default project
- `cli_whoami_test.go`:
  - `rp whoami` — table output contains expected username
  - `rp whoami -o json` — JSON output has expected fields
  - Error: invalid credentials
- `cli_project_test.go`:
  - `rp get project` — lists projects
  - `rp get project <name>` — gets project details
  - `rp get project <name> -o json` — JSON output

## Design considerations

- **Token parsing**: If `GetMyself` needs to parse a JWT, keep this as a private utility in the client package. Do not expose JWT parsing as public API.
- **Admin vs user endpoints**: Some project list endpoints require admin access. Prefer endpoints that work with regular user tokens, falling back to admin-only endpoints only when needed.
- **`rp whoami` is a diagnostic tool**: It should work even when the project flag is missing or wrong. Don't require `--project` for `whoami`.

## Definition of done

- [ ] `pkg/client/users.go` with `UsersScope` (GetMyself, GetByLogin, ListProjects)
- [ ] `pkg/client/projects.go` with `ProjectsScope` (Get, ListNames)
- [ ] `pkg/client/users_test.go` and `pkg/client/projects_test.go` with unit tests
- [ ] `internal/cli/whoami.go` with `rp whoami` command
- [ ] `internal/cli/project.go` with `rp get project` command
- [ ] Integration tests for both client and CLI
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-integration` passes
