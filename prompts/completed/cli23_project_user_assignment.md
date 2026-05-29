# PR 23: Project User Assignment

## Goal

Add `rp project user {assign,unassign}` as sub-resource commands for managing user membership in projects.

## Context

Phase 7. The project command already has list/get/create/update/delete/export. The `ProjectsScope` client already has `AssignUsers` and `UnassignUsers`.

### Client API surface

**ProjectsScope** (via `client.Projects()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `AssignUsers` | `AssignUsers(ctx, projectName string, types.AssignUsersRQ)` | `*types.OperationCompletionRS` |
| `UnassignUsers` | `UnassignUsers(ctx, projectName string, types.UnassignUsersRQ)` | `*types.OperationCompletionRS` |

### Relevant types

```go
type AssignUsersRQ struct {
    UserNames map[string]string `json:"userNames"`
}
// Map of username -> projectRole (e.g., "john": "MEMBER", "admin": "PROJECT_MANAGER")

type UnassignUsersRQ struct {
    UserNames []string `json:"userNames"`
}
```

Look up exact types in `pkg/types/`.

## Implementation plan

### 1. Create `internal/cli/project_user.go`

#### `rp project user assign`

- No positional args
- Required flag: `--user` (string slice, repeatable) — format: `username:role` (e.g., `john:MEMBER`)
- Alternative: `--from-file` (string) — reads full `AssignUsersRQ` from JSON
- The project name comes from `-p`/`--project` (inherited from project parent or resolved from config)
- Parse `--user` values into `map[string]string` (username → role)
- Valid roles: MEMBER, PROJECT_MANAGER, CUSTOMER, OPERATOR (validate case-insensitively)
- Call `scope.AssignUsers(ctx, projectName, request)`
- Output: `OperationCompletionRS` message

#### `rp project user unassign`

- No positional args
- Required flag: `--user` (string slice, repeatable) — just usernames to remove
- Alternative: `--from-file` (string) — reads full `UnassignUsersRQ` from JSON
- Call `scope.UnassignUsers(ctx, projectName, request)`
- Output: `OperationCompletionRS` message

### 2. Register in project.go

Add to `newProjectCmd()`:
```go
cmd.AddCommand(newProjectUserCmd())
```

Note: The project command does NOT have a `--project` persistent flag like launch/item do. You will need to add a project name resolution mechanism. Look at how the existing `project update` and `project delete` commands work — they may take the project name as a positional arg or via the config default. Check the existing code pattern.

Actually, looking at the existing code: `project create/update/delete` take the project name as a positional arg or flag. For `project user assign/unassign`, use a required `--project-name` flag or the first positional arg. Check how other project mutation commands handle this.

## Test requirements

### Unit tests (`internal/cli/project_user_test.go`)

- **Assign:** `--user` required (unless `--from-file`), format validation (must contain `:`)
- **Assign role validation:** Valid roles accepted, invalid rejected
- **Unassign:** `--user` required (unless `--from-file`)
- **From-file:** overrides flags
- **Command wiring:** project has `user` subcommand with `assign` and `unassign`

## Edge cases

- **Project name resolution:** Figure out how the existing project commands resolve the target project name. Follow the same pattern.
- **Role case-insensitivity:** Accept "member", "Member", "MEMBER" — normalize to uppercase
- **Multiple users:** `--user` flag is repeatable for bulk assignment
- **`--from-file` takes precedence**

## Example commands

```bash
# Assign users to a project
$ rp project user assign --user "john:MEMBER" --user "jane:PROJECT_MANAGER" my_project
MESSAGE
Users have been assigned to project successfully.

# Unassign users from a project
$ rp project user unassign --user john --user jane my_project
MESSAGE
Users have been unassigned from project successfully.
```

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [x] `internal/cli/project_user.go` with assign and unassign commands
- [x] Registered in `newProjectCmd()`
- [x] Role validation (case-insensitive)
- [x] Unit tests cover flag/format validation and command wiring
- [x] Existing project subcommands unchanged
- [x] `make lint` and `make test` pass
- [x] This file moved to `prompts/completed/`
