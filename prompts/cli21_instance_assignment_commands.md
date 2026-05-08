# PR 21: Instance-Scoped Assignment Commands

## Goal

Implement the `assign` and `unassign` commands for group↔project, group↔user, and project↔user relationships. These are association/relationship commands that mutate memberships between resources at the instance scope.

## Context

Assignment commands map to three different client scopes:
- **Group↔project** and **Group↔user**: `GroupsScope` (`client.Groups()`)
- **Project↔user**: `ProjectsScope` (`client.Projects()`)

All operations are **instance-scoped** — no project resolution via `--project` / `-p` is needed. These commands were classified as association commands in the CLI plan and use the `assign` / `unassign` verb pair (scaffolded in PR 2).

**Prerequisite PRs and dependencies:**
- PR 2 (verb group scaffolding: `assign` and `unassign` parent commands)
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `parseResourceID`, `mustGetStringFlag`, `messageResultHeaders`/`operationResultRows`)

### Client surface

#### Group↔project

| Method | Parameters | Returns |
|--------|------------|---------|
| `AddProject` | `ctx`, `groupID int64`, `projectID int64`, `request types.AddProjectToGroupByIdRequest` | `(*types.SuccessfulUpdate, error)` |
| `RemoveProject` | `ctx`, `groupID int64`, `projectID int64` | `error` |

#### Group↔user

| Method | Parameters | Returns |
|--------|------------|---------|
| `AddUser` | `ctx`, `groupID int64`, `userID int64` | `(*types.SuccessfulUpdate, error)` |
| `RemoveUser` | `ctx`, `groupID int64`, `userID int64` | `error` |

#### Project↔user

| Method | Parameters | Returns |
|--------|------------|---------|
| `AssignUsers` | `ctx`, `projectName string`, `request types.AssignUsersRQ` | `(*types.OperationCompletionRS, error)` |
| `UnassignUsers` | `ctx`, `projectName string`, `request types.UnassignUsersRQ` | `(*types.OperationCompletionRS, error)` |

### Key types

```go
type AddProjectToGroupByIdRequest struct {
    Permission *AddProjectToGroupByIdRequestPermission `json:"permission,omitempty"`
}

// AddProjectToGroupByIdRequestPermission values:
//   "CUSTOMER", "MEMBER", "OPERATOR", "PROJECT_MANAGER"

type SuccessfulUpdate struct {
    Message *string `json:"message,omitempty"`
}

type AssignUsersRQ struct {
    UserNames map[string]string `json:"userNames"`
}

type UnassignUsersRQ struct {
    UserNames []string `json:"userNames"`
}

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

## Scope

Create `internal/cli/assign_commands.go` and `internal/cli/assign_commands_test.go`. Register the new commands in `internal/cli/commands.go`.

**File naming:** Use `assign_commands.go` (not `assign.go`) because `assign.go` already exists as the verb scaffold from PR 2.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp assign project <project-id> --group <group-id> [--permission <perm>]` | Add a project to a group |
| `rp unassign project <project-id> --group <group-id>` | Remove a project from a group |
| `rp assign user <user-id> --group <group-id>` | Add a user to a group |
| `rp unassign user <user-id> --group <group-id>` | Remove a user from a group |
| `rp assign user <login> --project-name <name> --role <role>` | Assign a user to a project with a role |
| `rp unassign user <login> --project-name <name>` | Remove a user from a project |

All commands are instance-scoped (no project resolution).

## Implementation

### 1. Create `internal/cli/assign_commands.go`

This file contains all assignment command constructors and their run functions.

#### `newAssignProjectCmd()`

```go
func newAssignProjectCmd() *cobra.Command
```

- `Use`: `"project <project-id>"`
- `Short`: `"Add a project to a group"`
- `Long`: Include examples with and without `--permission`. List valid permission values (CUSTOMER, MEMBER, OPERATOR, PROJECT_MANAGER).
- `Args`: `cobra.ExactArgs(1)` (the numeric project ID)
- `RunE`: `withPrintedErrors(runAssignProject)`
- Flags:
  - `--group` (int64, required): the group ID to add the project to
  - `--permission` (string, optional): the permission level for the project in the group

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "project")`
2. Validate `--group` is non-zero
3. Build request: if `--permission` is explicitly set, create `types.AddProjectToGroupByIdRequest{Permission: &perm}` (cast string to `types.AddProjectToGroupByIdRequestPermission`); otherwise `types.AddProjectToGroupByIdRequest{}`
4. Resolve connection and client
5. Call `client.Groups().AddProject(ctx, groupID, projectID, request)`
6. Print the `SuccessfulUpdate` result using `messageResultHeaders()` and `successfulUpdateRows(result)`

**Permission validation:** If `--permission` is provided, validate it against the known values (CUSTOMER, MEMBER, OPERATOR, PROJECT_MANAGER). Return a descriptive error listing valid values if invalid. Use case-insensitive comparison and convert to uppercase before sending.

#### `newUnassignProjectCmd()`

```go
func newUnassignProjectCmd() *cobra.Command
```

- `Use`: `"project <project-id>"`
- `Short`: `"Remove a project from a group"`
- `Long`: Include examples.
- `Args`: `cobra.ExactArgs(1)` (the numeric project ID)
- `RunE`: `withPrintedErrors(runUnassignProject)`
- Flags:
  - `--group` (int64, required): the group ID to remove the project from

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "project")`
2. Validate `--group` is non-zero
3. Resolve connection and client
4. Call `client.Groups().RemoveProject(ctx, groupID, projectID)` — returns only `error`, no response body
5. Print a synthesized success message. Since `RemoveProject` returns no body, construct a local result: print using `messageResultHeaders()` with a single row `[][]string{{"Project removed from group successfully"}}`

#### `newAssignUserCmd()`

```go
func newAssignUserCmd() *cobra.Command
```

- `Use`: `"user <user-id-or-login>"`
- `Short`: `"Add a user to a group or assign a user to a project"`
- `Long`: Include examples for both modes: group↔user (`--group`) and project↔user (`--project-name` + `--role`). Explain that exactly one of `--group` or `--project-name` must be provided. When using `--group`, the positional arg is a numeric user ID. When using `--project-name`, it is a login string.
- `Args`: `cobra.ExactArgs(1)` (user ID or login depending on mode)
- `RunE`: `withPrintedErrors(runAssignUser)`
- Flags:
  - `--group` (int64, optional): the group ID (for group↔user assignment)
  - `--project-name` (string, optional): the project name (for project↔user assignment)
  - `--role` (string, optional): the role to assign (required when `--project-name` is used)

**Why `--project-name` instead of `--project`?** The `--project` / `-p` flag convention throughout the CLI means "resolve the project scope for this command." Here we need a project name as a data parameter for the assign operation, not as a scope selector. Using `--project-name` avoids confusion and does not trigger project resolution. No `-p` shorthand.

The run function:
1. Check mutual exclusivity: exactly one of `--group` or `--project-name` must have been explicitly set (check each flag's `Changed()` state). If neither or both are changed, return error: `"provide exactly one of --group or --project-name"`
2. **Group↔user mode** (when `--group` is set):
   a. Parse `args[0]` as int64 using `parseResourceID(args[0], "user")`
   b. Resolve connection and client
   c. Call `client.Groups().AddUser(ctx, groupID, userID)`
   d. Print the `SuccessfulUpdate` result using `messageResultHeaders()` and `successfulUpdateRows(result)`
3. **Project↔user mode** (when `--project-name` is set):
   a. Validate `--role` is non-empty; return error if missing: `"--role is required when using --project-name"`
   b. Build `types.AssignUsersRQ{UserNames: map[string]string{args[0]: role}}`
   c. Resolve connection and client
   d. Call `client.Projects().AssignUsers(ctx, projectName, request)`
   e. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newUnassignUserCmd()`

```go
func newUnassignUserCmd() *cobra.Command
```

- `Use`: `"user <user-id-or-login>"`
- `Short`: `"Remove a user from a group or unassign a user from a project"`
- `Long`: Include examples for both modes. Explain the same mutual exclusivity as `assign user`.
- `Args`: `cobra.ExactArgs(1)` (user ID or login depending on mode)
- `RunE`: `withPrintedErrors(runUnassignUser)`
- Flags:
  - `--group` (int64, optional): the group ID (for group↔user removal)
  - `--project-name` (string, optional): the project name (for project↔user removal)

The run function:
1. Check mutual exclusivity: exactly one of `--group` or `--project-name` must be provided. Return descriptive error if neither or both are set.
2. **Group↔user mode** (when `--group` is set):
   a. Parse `args[0]` as int64 using `parseResourceID(args[0], "user")`
   b. Resolve connection and client
   c. Call `client.Groups().RemoveUser(ctx, groupID, userID)` — returns only `error`
   d. Print a synthesized success message: `"User removed from group successfully"`
3. **Project↔user mode** (when `--project-name` is set):
   a. Build `types.UnassignUsersRQ{UserNames: []string{args[0]}}`
   b. Resolve connection and client
   c. Call `client.Projects().UnassignUsers(ctx, projectName, request)`
   d. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// assign
assign.AddCommand(newAssignProjectCmd())
assign.AddCommand(newAssignUserCmd())

// unassign
unassign.AddCommand(newUnassignProjectCmd())
unassign.AddCommand(newUnassignUserCmd())
```

### 3. Create `internal/cli/assign_commands_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewAssignProjectCmd_Flags`** — Verify `--group` and `--permission` flags are registered. Verify it accepts exactly 1 arg.

2. **`TestNewUnassignProjectCmd_Flags`** — Verify `--group` flag is registered. Verify it accepts exactly 1 arg.

3. **`TestNewAssignUserCmd_Flags`** — Verify `--group`, `--project-name`, and `--role` flags are registered. Verify it accepts exactly 1 arg.

4. **`TestNewAssignUserCmd_NoProjectShorthand`** — Verify `--project-name` does NOT have a `-p` shorthand.

5. **`TestNewUnassignUserCmd_Flags`** — Verify `--group` and `--project-name` flags are registered. Verify it accepts exactly 1 arg.

6. **`TestNewUnassignUserCmd_NoRoleFlag`** — Verify unassign user has no `--role` flag (only assign needs it).

7. **`TestAssignProjectCmd_PermissionValidation`** — Test that the permission validation helper accepts all valid values case-insensitively and rejects invalid ones.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **One `user` subcommand for two operations.** Rather than `assign group-user` and `assign project-user` (which are ugly and non-intuitive), a single `assign user` command dispatches on `--group` vs `--project-name`. This matches the conceptual model: "assign this user somewhere." The `--group` / `--project-name` flag tells the CLI where.
- **Mutual exclusivity validation.** Exactly one of `--group` or `--project-name` must be set. This is validated in the run function with a clear error message listing the two options. Cobra's built-in mutual exclusivity could be used (`cmd.MarkFlagsMutuallyExclusive`), but manual validation gives better error messages.
- **`--project-name` not `--project`.** This avoids collision with the conventional `--project` / `-p` project-resolution flag used throughout the CLI. The assign commands are instance-scoped; there is no project resolution happening. The project name here is a data parameter identifying the target project.
- **`--permission` validation.** The permission string is validated client-side against the known enum values. This provides an immediate, clear error instead of a cryptic 400 from the API.
- **Synthesized success messages for void returns.** `RemoveProject` and `RemoveUser` return no response body (just `error`). The CLI synthesizes a human-friendly success message for table/JSON output consistency. The synthesized message is a static string, not a formatted template.
- **`AssignUsersRQ` maps a single user.** The API supports batch assignment (`map[string]string` with multiple entries), but the CLI assigns one user at a time for simplicity and clear UX. Batch assignment can be added as a stretch goal later.
- **Reuse shared helpers.** Use `parseResourceID`, `messageResultHeaders`/`operationResultRows`, `successfulUpdateRows`, `withPrintedErrors`.
- **Route all output through `printer.Print`.** Use `printer.Print(result, headers, rows)` for all results. This ensures JSON, Go-template, and table modes work correctly.
- **Use client error predicates.** Handle API errors with predicate functions (`IsNotFound`, `IsUnauthorized`, etc.) rather than type assertions.
- **Command registration follows PR 3's centralized command-tree layout.** Shared mutation/result helpers assume PR 7's naming conventions.

## What NOT to do

- **Do not implement `ListProjects`, `GetProject`, `ListUsers`, or `GetUser` from the Groups scope.** Those are read operations that could appear under `get` in a future PR (e.g., `rp get group-project --group <id>`), but they are not part of the assignment commands scope.
- **Do not add batch user assignment.** One user per command invocation. Batch is a stretch goal.
- **Do not use `--project` / `-p` for project resolution.** These are instance-scoped commands.
- **Do not add integration tests.** Unit tests are sufficient for this PR.
- **Do not implement `ProjectGroupsScope` commands.** The same relationships are accessible from the groups side; duplicating them from the project side is unnecessary.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp assign --help` shows `project` and `user` as subcommands
- `rp unassign --help` shows `project` and `user` as subcommands
- `rp assign project --help` shows `--group` and `--permission` flags
- `rp assign user --help` shows `--group`, `--project-name`, and `--role` flags, explains both modes
- `rp unassign project --help` shows `--group` flag
- `rp unassign user --help` shows `--group` and `--project-name` flags
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/assign_commands.go` created with all four command constructors and run functions
- [ ] `internal/cli/assign_commands_test.go` created with unit tests
- [ ] Commands registered in `commands.go` (assign project, assign user, unassign project, unassign user)
- [ ] All commands are instance-scoped (no project resolution)
- [ ] `assign user` / `unassign user` dispatch on `--group` vs `--project-name` with mutual exclusivity validation
- [ ] `--project-name` has no `-p` shorthand
- [ ] `--role` is required when `--project-name` is used for assign, not present for unassign
- [ ] `--permission` on `assign project` is validated against known enum values (case-insensitive)
- [ ] Void-returning operations (`RemoveProject`, `RemoveUser`) print synthesized success messages
- [ ] Shared helpers used: `parseResourceID`, `messageResultHeaders`/`operationResultRows`, `successfulUpdateRows`, `withPrintedErrors`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
