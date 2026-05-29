# PR 19: Group Associations

## Goal

Add `rp group user {list,get,add,remove}` and `rp group project {list,get,add,remove}` as sub-resources of the existing `group` command.

## Context

Phase 7. The `group` command already has list/get/create/update/delete. The `GroupsScope` client already has `ListUsers`, `GetUser`, `AddUser`, `RemoveUser`, `ListProjects`, `GetProject`, `AddProject`, `RemoveProject`.

### Client API surface

**GroupsScope** (via `client.Groups()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `ListUsers` | `ListUsers(ctx, groupID int64, ...ListGroupUsersOption)` | `*types.GroupUsersPage` |
| `GetUser` | `GetUser(ctx, groupID, userID int64)` | `*types.GroupUserInfo` |
| `AddUser` | `AddUser(ctx, groupID, userID int64)` | `*types.SuccessfulUpdate` |
| `RemoveUser` | `RemoveUser(ctx, groupID, userID int64)` | `error` |
| `ListProjects` | `ListProjects(ctx, groupID int64, ...ListGroupProjectsOption)` | `*types.GroupProjectsPage` |
| `GetProject` | `GetProject(ctx, groupID, projectID int64)` | `*types.GroupProjectInfo` |
| `AddProject` | `AddProject(ctx, groupID, projectID int64, types.AddProjectToGroupByIdRequest)` | `*types.SuccessfulUpdate` |
| `RemoveProject` | `RemoveProject(ctx, groupID, projectID int64)` | `error` |

Pagination options: `WithGroupUserPageSize`, `WithGroupUserOffset`, `WithGroupProjectPageSize`, `WithGroupProjectOffset`.

### Relevant types

Look up in `pkg/types/`:
- `GroupUsersPage` — has Content (list of user info), Page metadata
- `GroupUserInfo` — user-in-group details
- `GroupProjectsPage` — has Content (list of project info), Page metadata  
- `GroupProjectInfo` — project-in-group details with permissions
- `AddProjectToGroupByIdRequest` — has Permission field
- `SuccessfulUpdate` — has Message field

## Implementation plan

### 1. Create `internal/cli/group_associations.go`

This sub-file keeps association commands separate from the main group CRUD.

#### Command structure

```
rp group user list <group-id>
rp group user get <group-id> <user-id>
rp group user add <group-id> <user-id>
rp group user remove <group-id> <user-id>
rp group project list <group-id>
rp group project get <group-id> <project-id>
rp group project add <group-id> <project-id> [--permission ROLE]
rp group project remove <group-id> <project-id>
```

#### `rp group user list <group-id>`
- One positional arg: numeric group ID
- Pagination flags (page-size, offset) using the group user pagination options
- Table columns based on `GroupUserInfo` fields (ID, LOGIN/NAME, etc.)
- JSON output: raw page response

#### `rp group user get <group-id> <user-id>`
- Two positional args: group ID, user ID (both numeric)
- Table/JSON: single `GroupUserInfo`

#### `rp group user add <group-id> <user-id>`
- Two positional args
- Call `AddUser(ctx, groupID, userID)`
- Output: `SuccessfulUpdate` message

#### `rp group user remove <group-id> <user-id>`
- Two positional args
- Call `RemoveUser(ctx, groupID, userID)`
- Output: success message (void return, print confirmation)

#### `rp group project list <group-id>`
- One positional arg: numeric group ID
- Pagination flags
- Table columns based on `GroupProjectInfo` fields
- JSON output: raw page response

#### `rp group project get <group-id> <project-id>`
- Two positional args
- Table/JSON: single `GroupProjectInfo`

#### `rp group project add <group-id> <project-id>`
- Two positional args
- Optional flag: `--permission` (string) — project role/permission
- Build `AddProjectToGroupByIdRequest` with permission if provided
- Call `AddProject(ctx, groupID, projectID, request)`
- Output: `SuccessfulUpdate` message

#### `rp group project remove <group-id> <project-id>`
- Two positional args
- Call `RemoveProject(ctx, groupID, projectID)`
- Output: success message

### 2. Register in group.go

Add to `newGroupCmd()`:
```go
cmd.AddCommand(newGroupUserCmd())
cmd.AddCommand(newGroupProjectCmd())
```

## Test requirements

### Unit tests (`internal/cli/group_associations_test.go`)

- ID parsing: non-numeric group/user/project IDs rejected
- Positional arg counts enforced (1 for list, 2 for get/add/remove)
- `--permission` flag optional on project add
- Command wiring: verify group has user and project subcommands with correct verbs

## Edge cases

- **No `--project` flag:** Group commands are instance-scoped
- **Two positional args:** get/add/remove need both parent ID and child ID
- **`RemoveUser`/`RemoveProject` return error, not response:** Print a generic success message on nil error
- **Pagination:** Use the existing `WithGroupUserPageSize`/`WithGroupUserOffset` etc. options

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [ ] `internal/cli/group_associations.go` with user and project sub-resource commands
- [ ] Commands registered in `newGroupCmd()`
- [ ] Unit tests cover ID parsing, arg counts, command wiring
- [ ] Existing group CRUD unchanged
- [ ] `make lint` and `make test` pass
- [ ] This file moved to `prompts/completed/`
