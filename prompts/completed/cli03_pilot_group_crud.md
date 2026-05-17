# PR 3: Pilot `group` Full CRUD

## Goal

Implement the second pilot resource — `rp group {list,get,create,update,delete}` — as an instance-scoped CRUD family. This PR establishes the patterns for instance-scoped commands (no `--project` required) and for translating an offset-based API pagination model into the CLI's page-based user-facing flags.

## Context

**Prerequisites:** PR 1 (noun-first refactor) and PR 2 (filter pilot) have landed. The noun-first command registration pattern and `--from-file` mutation flow are established. Groups differ from filters in two key ways:

1. **Instance-scoped** — no `--project` flag. The scope is `client.Groups()`, not `client.Project(name).Filters()`.
2. **Offset-based pagination** — the API uses `offset`/`limit` (not page number/page size). The CLI should present the same `--page`/`--page-size` flags as filter for user-facing consistency, converting to offset/limit internally.
3. **Simple payloads** — create and update take only `name` + optional `slug`, so inline flags are appropriate (no `--from-file` needed).

### API surface

The groups scope is accessed via `client.Groups()` and provides:

| Method | Signature |
|--------|-----------|
| `List` | `List(ctx, options ...ListGroupsOption) (*types.GroupPage, error)` |
| `Get` | `Get(ctx, groupID int64) (*types.GroupInfo, error)` |
| `Create` | `Create(ctx, request types.CreateGroupRequest) (*types.GroupInfo, error)` |
| `Update` | `Update(ctx, groupID int64, request types.UpdateGroupRequest) (*types.SuccessfulUpdate, error)` |
| `Delete` | `Delete(ctx, groupID int64) error` |

List options: `WithGroupPageSize(int32)`, `WithGroupOffset(int32)`, `WithGroupSort(string)`, `WithGroupOrder(string)`.

Note: association methods (`ListProjects`, `AddProject`, `ListUsers`, `AddUser`, etc.) exist on `GroupsScope` but are **out of scope** for this PR. They will be implemented in Phase 7 (PR 19).

### Key types

```go
type CreateGroupRequest struct {
    Name string  `json:"name"`
    Slug *string `json:"slug,omitempty"`
}

type UpdateGroupRequest struct {
    Name string  `json:"name"`
    Slug *string `json:"slug,omitempty"`
}

type GroupInfo struct {
    CreatedAt *time.Time          `json:"created_at,omitempty"`
    CreatedBy *int64              `json:"created_by,omitempty"`
    Id        *int64              `json:"id,omitempty"`
    Name      *string             `json:"name,omitempty"`
    Slug      *string             `json:"slug,omitempty"`
    Stats     *GroupInfoStats     `json:"stats,omitempty"`
    UpdatedAt *time.Time          `json:"updated_at,omitempty"`
    Uuid      *openapi_types.UUID `json:"uuid,omitempty"`
}

type GroupInfoStats struct {
    ProjectsCount *int32 `json:"projects_count,omitempty"`
    UsersCount    *int32 `json:"users_count,omitempty"`
}

type GroupPage struct {
    Items      *[]GroupInfo    `json:"items,omitempty"`
    Limit      *int32          `json:"limit,omitempty"`
    Offset     *int32          `json:"offset,omitempty"`
    Order      *GroupPageOrder `json:"order,omitempty"`  // "ASC" or "DESC"
    Sort       *string         `json:"sort,omitempty"`
    TotalCount *int32          `json:"total_count,omitempty"`
}

type SuccessfulUpdate struct {
    Message *string `json:"message,omitempty"`
}
```

**Important:** All fields on `GroupInfo` are pointers. Table formatting must handle nil for every field.

### Files to create

- `internal/cli/group.go` — all group commands and helpers
- `internal/cli/group_test.go` — unit tests

### Files to modify

- `internal/cli/root.go` — add `cmd.AddCommand(newGroupCmd())` to register the group parent command

## Detailed requirements

### 1. Parent command scaffolding

Define `newGroupCmd()` returning a `*cobra.Command`:

```go
func newGroupCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "group",
        Short: "Manage groups",
    }
    cmd.AddCommand(newGroupListCmd())
    cmd.AddCommand(newGroupGetCmd())
    cmd.AddCommand(newGroupCreateCmd())
    cmd.AddCommand(newGroupUpdateCmd())
    cmd.AddCommand(newGroupDeleteCmd())
    return cmd
}
```

**No `-p`/`--project` flag.** Groups are instance-scoped. The connection is resolved via `resolveConnection(cmd)` only — no `resolveProject` call.

### 2. `rp group list`

**Command shape:** `rp group list [--page N] [--page-size N] [--sort field,dir]`

- `Use: "list"`, `Args: cobra.NoArgs`
- Register standard list flags using `registerListFlagsWithDefaults` with a default sort of `name,ASC`

**Pagination translation (page → offset):**

The user-facing flags use 1-based page numbers and page size (same as filter list). The groups API expects `offset` and `limit`. Convert as follows:

```
offset = (page - 1) * pageSize     // page is already 0-based after resolveListFlags
limit  = pageSize
```

Since `resolveListFlags` converts the user's 1-based page to 0-based, the offset calculation is: `offset = flags.page * flags.pageSize`.

**Sort translation:**

The existing `--sort` flag accepts values like `"name,ASC"`. The groups API separates sort field and order direction into two parameters. Parse the sort value:

- If the sort string contains a comma, split into `field` and `direction`
- If no comma, treat the entire string as the field name, defaulting direction to `ASC`
- Pass `field` to `WithGroupSort` and `direction` (uppercased) to `WithGroupOrder`
- If multiple `--sort` values are provided, use only the first one (the groups API accepts a single sort field). Log a debug warning if extras are ignored.

Define a helper:

```go
func parseGroupSort(sort []string) (field string, order string)
```

**List flow:**

1. Resolve list flags
2. Convert to offset/limit
3. Parse sort field and order
4. Build options: `WithGroupPageSize(limit)`, `WithGroupOffset(offset)`, `WithGroupSort(field)`, `WithGroupOrder(order)`
5. Call `scope.List(ctx, options...)`
6. Extract `[]GroupInfo` from `result.Items` (nil-safe)
7. Print

**Table columns:**

| Header | Source | Notes |
|--------|--------|-------|
| ID | `Id` | `formatInt64Ptr` |
| NAME | `Name` | `formatStringPtr` |
| SLUG | `Slug` | `formatStringPtr` |
| USERS | `Stats.UsersCount` | `formatInt32Ptr` (see below) |
| PROJECTS | `Stats.ProjectsCount` | `formatInt32Ptr` |
| CREATED | `CreatedAt` | Format with `time.DateTime` in UTC, or empty if nil |

Define `groupHeaders() []string` and `groupRow(*types.GroupInfo) ([]string, []string)`.

Since `Stats` itself is a pointer, check for nil before accessing its fields. If `Stats` is nil, show empty strings for USERS and PROJECTS.

**New formatting helper:** `formatInt32Ptr(*int32) string` — returns formatted int or empty string for nil. This is analogous to the existing `formatInt64Ptr` for int64. Define it in `group.go`; Phase 3 may relocate it if other pilots need it.

### 3. `rp group get <id>`

**Command shape:** `rp group get <id>`

- `Use: "get <id>"`, `Args: cobra.ExactArgs(1)`
- Parse the positional argument as `int64`. On failure: `invalid group ID "<value>": must be a numeric ID`
- Call `scope.Get(ctx, groupID)`
- Print using the same `groupRow` formatting as list

### 4. `rp group create`

**Command shape:** `rp group create --name <name> [--slug <slug>]`

- `Use: "create"`, `Args: cobra.NoArgs`
- `--name` (`string`, required) — display name for the group
- `--slug` (`string`, optional) — URL-safe identifier. Must contain only lowercase letters, numbers, and hyphens; must not start or end with a hyphen.
- Validation: `--name` is required. If empty, error: `--name is required`
- **Do not validate slug format in the CLI** — let the API enforce its own slug constraints and relay the error. The CLI help text should describe the format requirement so users know what to provide.

**Flow:**

1. Read flags
2. Build `types.CreateGroupRequest{Name: name, Slug: slugPtr}`
3. Call `scope.Create(ctx, request)`
4. Print the returned `*types.GroupInfo` using `groupRow` formatting (the API returns the full created group)

### 5. `rp group update <id>`

**Command shape:** `rp group update <id> --name <name> [--slug <slug>]`

- `Use: "update <id>"`, `Args: cobra.ExactArgs(1)`
- Parse ID from positional argument
- `--name` (`string`, required) — the API requires `name` in every update (it's not a pointer in `UpdateGroupRequest`)
- `--slug` (`string`, optional) — if provided, updates the slug
- Validation: `--name` is required. If empty and `--name` was not explicitly set, the command should fetch the current group to preserve the existing name, similar to the user update pattern.

**Flow:**

1. Parse group ID
2. If `--name` was not explicitly provided via flag, fetch the current group with `scope.Get(ctx, groupID)` and use its `Name` field
3. Build `types.UpdateGroupRequest{Name: name, Slug: slugPtr}`
4. Call `scope.Update(ctx, groupID, request)`
5. Print the returned `*types.SuccessfulUpdate`: table with `MESSAGE` header and the message value (same visual pattern as `operationResultHeaders()` / `operationResultRows()` in `user.go`)

**Note on `SuccessfulUpdate` vs `OperationCompletionRS`:** Both types have a single `Message *string` field. However, they are distinct generated types — `operationResultRows()` accepts `*types.OperationCompletionRS` and cannot be called with `*types.SuccessfulUpdate`. Write a local helper (e.g., `successfulUpdateRows(*types.SuccessfulUpdate) [][]string`) that formats the message field. The table rendering is identical (single MESSAGE column).

### 6. `rp group delete <id>`

**Command shape:** `rp group delete <id>`

- `Use: "delete <id>"`, `Args: cobra.ExactArgs(1)`
- Parse ID from positional argument
- Call `scope.Delete(ctx, groupID)` — returns only `error`, no response body
- On success, synthesize output:
  - **Table mode:** print a table with `MESSAGE` header and value `Group <id> deleted`
  - **JSON mode:** print `{"message": "Group <id> deleted"}`
- Use a consistent approach: create a small struct or map for the success output and pass it through `printer.Print`

### 7. Help text and examples

Each subcommand should have a `Long` description with CLI examples:

```
rp group list                              List groups
rp group list --page-size 25               List with custom page size
rp group get 42                            Get group by ID
rp group get 42 -o json                    Get group as JSON
rp group create --name "My Group"          Create a group
rp group create --name "My Group" --slug my-group  Create with explicit slug
rp group update 42 --name "New Name"       Rename a group
rp group update 42 --slug new-slug         Update slug (preserves current name)
rp group delete 42                         Delete a group
```

### 8. Standard boilerplate

Follow the same patterns established in the filter pilot and existing resources:

- **Printer creation:** `outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")` then `newPrinter(outputFmt)`
- **Connection resolution:** `resolveConnection(cmd)` → `newClient(conn, logger)` (no `resolveProject` — instance-scoped)
- **Error handling:** call `printer.PrintError(err)` before returning the error, and wrap with `withPrintedErrors`
- **Logger:** `loggerFromCmd(cmd)` at the top of each RunE function

## Test requirements

Create `internal/cli/group_test.go` in `package cli` (white-box tests).

### Required tests

1. **`groupRow` formatting:** verify headers and row values for a `GroupInfo` with all fields populated

2. **`groupRow` nil handling:** verify formatting when `Stats` is nil, when `Name` is nil, when `CreatedAt` is nil — all should produce empty strings, not panics

3. **`parseGroupSort` parsing:**
   - `"name,ASC"` → field `"name"`, order `"ASC"`
   - `"name,desc"` → field `"name"`, order `"DESC"` (uppercased)
   - `"name"` → field `"name"`, order `"ASC"` (default)
   - `""` → default behavior (use default sort)

4. **Pagination offset calculation:** verify that page=0 (0-based, from resolveListFlags) + pageSize=10 → offset=0, limit=10; page=2 + pageSize=25 → offset=50, limit=25

5. **`formatInt32Ptr` helper:** nil returns empty string, non-nil returns formatted integer

6. **`newGroupCmd` subcommands:** verify the parent command has the expected child command names (list, get, create, update, delete)

7. **Create flag validation:** verify that missing `--name` produces a clear error

8. **Group ID parsing:** verify that non-numeric IDs produce a clear error message

### Test patterns

- Use `testing.T` with `t.Parallel()` for all tests and subtests
- Use `testify/assert` and `testify/require`
- Use table-driven tests for sort parsing and pagination calculation
- Follow naming: `TestGroupRow_*`, `TestParseGroupSort_*`, etc.

## Edge cases and constraints

- **All-pointer `GroupInfo`:** Every field on `GroupInfo` is a pointer, unlike most other resource types. The table formatting helpers must nil-check each field individually. This includes nested access through `Stats` (which is itself a pointer).
- **Delete returns no body:** `GroupsScope.Delete` returns only `error`. The CLI must synthesize a success message rather than printing a server response. This is different from filter/user delete which return `OperationCompletionRS`.
- **Update requires `name`:** `UpdateGroupRequest.Name` is a non-pointer `string` — the API requires it on every update. The CLI should fetch the current group to preserve the name when `--name` is not explicitly provided, matching the user update pattern (which fetches the current `fullName`). Since `GroupInfo.Name` is `*string`, handle the nil case: if the fetched group's name is nil, error with `current group has no name; specify --name explicitly` (same pattern as the user update's fullName resolution).
- **Single sort field:** The groups API accepts only one sort field (not a slice). If the user passes multiple `--sort` flags, use the first and log a debug-level warning. Do not error.
- **Offset-based pagination:** The groups API uses `offset`/`limit`, but the CLI presents `--page`/`--page-size` for consistency with other resources. The translation is `offset = page * pageSize` where `page` is 0-based (already converted by `resolveListFlags`).
- **Sort direction as separate parameter:** The groups API separates sort field from order direction. Parse from the combined `"field,dir"` format that `--sort` uses. Default direction is `ASC` if omitted.
- **Slug validation:** Do not validate slug format client-side. The API has its own validation rules (lowercase alphanumeric + hyphens, no leading/trailing hyphens). Relay API errors directly.
- **Association methods are out of scope:** `ListProjects`, `AddProject`, `ListUsers`, `AddUser`, etc. are part of Phase 7 (PR 19). Do not implement them here.
- **`SuccessfulUpdate` vs `OperationCompletionRS`:** These are structurally identical (both have `Message *string`) but are different generated types. For table formatting, you can write a small local helper or format the message field directly. Do not create a type assertion or conversion between them.

## Validation

Run before finalizing:

```
make lint
make vet
make test
```

## Definition of done

- [ ] `internal/cli/group.go` implements `rp group {list,get,create,update,delete}`
- [ ] `internal/cli/group_test.go` covers row formatting, nil handling, sort parsing, offset calculation, flag validation, and subcommand structure
- [ ] `internal/cli/root.go` registers `newGroupCmd()`
- [ ] No `-p`/`--project` flag on any group subcommand (instance-scoped)
- [ ] `--page`/`--page-size` flags translate to offset/limit for the groups API
- [ ] `--sort "field,dir"` parses into separate sort and order parameters
- [ ] Create and update use inline flags (`--name`, `--slug`) — no `--from-file`
- [ ] Update fetches the current group to preserve `name` when `--name` is not provided
- [ ] Delete synthesizes a success message (the API returns no body)
- [ ] Table output handles all-pointer `GroupInfo` fields without panics
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
