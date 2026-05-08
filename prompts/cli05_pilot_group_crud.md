# PR 5: Pilot `group` Full CRUD

## Goal

Implement the complete `group` command family — list, get-by-ID, create, update, and delete — as the instance-scoped CRUD pilot. This PR establishes the baseline patterns for instance-scoped resources that do not require project resolution, and demonstrates a different pagination model (offset-based instead of page-number-based).

## Context

The `group` resource maps to `GroupsScope` on the client (`client.Groups()`). It is instance-scoped, meaning commands do **not** require `--project` / `-p`. This contrasts with the `filter` pilot (PR 4) and teaches the codebase how to cleanly separate the two scoping styles.

**Prerequisite:** PRs 1–3 must land first (parent command split, verb scaffolding, and centralized registration). PR 4 (filter pilot) should land first so this PR can reuse any patterns established there, though it is not a hard code dependency.

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `List` | `ctx`, `options ...ListGroupsOption` | `(*types.GroupPage, error)` |
| `Get` | `ctx`, `groupID int64` | `(*types.GroupInfo, error)` |
| `Create` | `ctx`, `request types.CreateGroupRequest` | `(*types.GroupInfo, error)` |
| `Update` | `ctx`, `groupID int64`, `request types.UpdateGroupRequest` | `(*types.SuccessfulUpdate, error)` |
| `Delete` | `ctx`, `groupID int64` | `error` (no response body) |

List options: `WithGroupPageSize(int32)`, `WithGroupOffset(int32)`, `WithGroupSort(string)`, `WithGroupOrder(string)`.

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
    Order      *GroupPageOrder `json:"order,omitempty"`
    Sort       *string         `json:"sort,omitempty"`
    TotalCount *int32          `json:"total_count,omitempty"`
}

type SuccessfulUpdate struct {
    Message *string `json:"message,omitempty"`
}
```

`GroupPageOrder` is a string enum: `"ASC"`, `"DESC"`.

## Scope

Create `internal/cli/group.go` and `internal/cli/group_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get group` | List groups (paginated, offset-based) |
| `rp get group <id>` | Get a single group by numeric ID |
| `rp create group --name <name> [--slug <slug>]` | Create a group |
| `rp update group <id> --name <name> [--slug <slug>]` | Update a group |
| `rp delete group <id>` | Delete a group by numeric ID |

## Implementation

### 1. Create `internal/cli/group.go`

This file contains all five command constructors and their run functions.

#### `newGetGroupCmd()`

```go
func newGetGroupCmd() *cobra.Command
```

- `Use`: `"group [id]"`
- `Short`: `"List groups or get a specific group"`
- `Long`: Include examples showing list, get-by-ID, pagination, and output format. Mention this is instance-scoped and does not require `--project`.
- `Args`: `cobra.MaximumNArgs(1)`
- `RunE`: `withPrintedErrors(runGetGroup)`
- Flags:
  - `--limit` (int32, default 10): maximum results per page
  - `--offset` (int32, default 0): pagination offset
  - `--sort` (string): field to sort by (e.g., `"name"`)
  - `--order` (string): sort direction (`"ASC"` or `"DESC"`)

**Do NOT** register `--project` — groups are instance-scoped.

**Do NOT** use `registerListFlags` — the group API uses offset/limit pagination, not page-number pagination. Define group-specific pagination flags.

The run function:
1. Resolve connection and client (no project resolution)
2. If `len(args) == 1`, parse arg as int64, call `scope.Get(ctx, id)`, print as single row
3. Otherwise, resolve pagination flags (with validation — see below), call `scope.List(ctx, options...)`, unwrap the `Items` slice from the `GroupPage` response, and pass the **slice** (not the page wrapper) to `printer.Print`

**Output contract:** In list mode, the value passed to `printer.Print` is the `[]types.GroupInfo` slice, not the `GroupPage` wrapper. This matches how existing commands (`get user`, `get launch`, etc.) work — they unwrap the pagination envelope and print only the resource slice. JSON mode outputs a JSON array of group objects.

#### `newCreateGroupCmd()`

```go
func newCreateGroupCmd() *cobra.Command
```

- `Use`: `"group"`
- `Short`: `"Create a group"`
- `Long`: Include examples showing inline flag usage. Note this is instance-scoped.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateGroup)`
- Flags:
  - `--name` (string, required): display name for the group
  - `--slug` (string, optional): URL-friendly identifier

The run function:
1. Validate `--name` is provided (non-empty)
2. Resolve `--slug` using `optionalStringFlag(cmd, "slug")` — returns nil if the flag was not explicitly passed, or `&value` (including `&""`) if it was. This means `--slug ""` sends an explicit empty string to the API, which will likely be rejected by server-side validation. This is the correct behavior — do not add special-case handling for empty strings.
3. Build `types.CreateGroupRequest{Name: name, Slug: slugPtr}`
4. Resolve connection and client
5. Call `scope.Create(ctx, request)`
6. Print the returned `GroupInfo` as a single table row

**Why inline flags, not `--from-file`?** The `CreateGroupRequest` is trivial (2 fields). Inline flags are the right UX per the CLI plan's decision rules: use `--from-file` for complex payloads only.

#### `newUpdateGroupCmd()`

```go
func newUpdateGroupCmd() *cobra.Command
```

- `Use`: `"group <id>"`
- `Short`: `"Update a group"`
- `Long`: Include examples showing update with name and/or slug
- `Args`: `cobra.ExactArgs(1)` (the numeric group ID)
- `RunE`: `withPrintedErrors(runUpdateGroup)`
- Flags:
  - `--name` (string, required): new display name
  - `--slug` (string, optional): new slug

The run function:
1. Parse `args[0]` as int64
2. Validate `--name` is provided (required by the API — `UpdateGroupRequest.Name` is not a pointer)
3. Resolve `--slug` using `optionalStringFlag(cmd, "slug")` — nil if flag was not passed; `&value` (possibly empty) if it was
4. Build `types.UpdateGroupRequest{Name: name, Slug: slugPtr}`
5. Resolve connection and client
6. Call `scope.Update(ctx, id, request)`
7. Print the `SuccessfulUpdate` result. Use a helper similar to `operationResultHeaders`/`operationResultRows` since `SuccessfulUpdate` has the same shape as `OperationCompletionRS` (a single `Message *string` field).

#### `newDeleteGroupCmd()`

```go
func newDeleteGroupCmd() *cobra.Command
```

- `Use`: `"group <id>"`
- `Short`: `"Delete a group"`
- `Long`: Include example showing delete by ID
- `Args`: `cobra.ExactArgs(1)` (the numeric group ID)
- `RunE`: `withPrintedErrors(runDeleteGroup)`
- No additional flags beyond the global ones.

The run function:
1. Parse `args[0]` as int64
2. Resolve connection and client
3. Call `scope.Delete(ctx, id)` — note this returns `error` only (no response body)
4. On success, construct a local `groupDeleteResult` and pass it to `printer.Print`:

```go
type groupDeleteResult struct {
    Message string `json:"message"`
}
```

Build the result as `groupDeleteResult{Message: fmt.Sprintf("Group %d deleted successfully", id)}` and pass it to `printer.Print` with `operationResultHeaders()` and a single-row slice containing the message.

This is an explicit exception to the "pass API response to printer" pattern — `Delete` has no response body, so we synthesize a structured result. The `groupDeleteResult` type ensures JSON mode outputs `{"message": "..."}` (consistent with `OperationCompletionRS`), and table mode uses the same `MESSAGE` header as other mutation results.

#### Table formatting

```go
func groupHeaders() []string {
    return []string{"ID", "NAME", "SLUG", "PROJECTS", "USERS"}
}

func groupRow(group *types.GroupInfo) ([]string, []string) {
    projects := ""
    users := ""
    if group.Stats != nil {
        projects = formatInt32Ptr(group.Stats.ProjectsCount)
        users = formatInt32Ptr(group.Stats.UsersCount)
    }

    return groupHeaders(), []string{
        formatInt64Ptr(group.Id),
        formatStringPtr(group.Name),
        formatStringPtr(group.Slug),
        projects,
        users,
    }
}
```

#### `formatInt32Ptr` helper

The existing codebase has `formatInt64Ptr` (in `item.go`) but no `formatInt32Ptr`. Add a local helper:

```go
func formatInt32Ptr(value *int32) string {
    if value == nil {
        return ""
    }

    return strconv.FormatInt(int64(*value), 10)
}
```

#### Group pagination flags

```go
type groupListFlags struct {
    limit  int32
    offset int32
    sort   string
    order  string
}

func registerGroupListFlags(cmd *cobra.Command) {
    cmd.Flags().Int32("limit", 10, "maximum number of results")
    cmd.Flags().Int32("offset", 0, "pagination offset")
    cmd.Flags().String("sort", "", "field to sort by")
    cmd.Flags().String("order", "", "sort direction (ASC or DESC)")
}

func resolveGroupListFlags(cmd *cobra.Command) (groupListFlags, error) {
    limit, _ := cmd.Flags().GetInt32("limit")
    offset, _ := cmd.Flags().GetInt32("offset")
    sort, _ := cmd.Flags().GetString("sort")
    order, _ := cmd.Flags().GetString("order")

    if limit < 1 {
        return groupListFlags{}, fmt.Errorf("--limit must be at least 1, got %d", limit)
    }

    if offset < 0 {
        return groupListFlags{}, fmt.Errorf("--offset must be non-negative, got %d", offset)
    }

    if order != "" && order != "ASC" && order != "DESC" {
        return groupListFlags{}, fmt.Errorf("--order must be ASC or DESC, got %q", order)
    }

    return groupListFlags{limit: limit, offset: offset, sort: sort, order: order}, nil
}
```

This validation catches invalid values locally rather than leaking them to opaque API errors. It mirrors how `resolveListFlags` in `listflags.go` validates page >= 1.

Convert to client options:

```go
func (f groupListFlags) clientOptions() []client.ListGroupsOption {
    options := []client.ListGroupsOption{
        client.WithGroupPageSize(f.limit),
        client.WithGroupOffset(f.offset),
    }

    if f.sort != "" {
        options = append(options, client.WithGroupSort(f.sort))
    }

    if f.order != "" {
        options = append(options, client.WithGroupOrder(f.order))
    }

    return options
}
```

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// get
get.AddCommand(newGetGroupCmd())

// create
create.AddCommand(newCreateGroupCmd())

// update
update.AddCommand(newUpdateGroupCmd())

// delete
del.AddCommand(newDeleteGroupCmd())
```

### 3. Create `internal/cli/group_test.go`

Use the same testing pattern as `user_test.go` and `launch_test.go`:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetGroupCmd_DefaultListFlags`** — Verify default flag values: limit=10, offset=0, sort="", order="".

2. **`TestResolveGroupListFlags_WithValues`** — Set flags via `cmd.ParseFlags`, verify the struct is populated correctly.

3. **`TestResolveGroupListFlags_InvalidLimit`** — Verify `--limit 0` and `--limit -1` produce descriptive errors.

4. **`TestResolveGroupListFlags_InvalidOffset`** — Verify `--offset -1` produces a descriptive error.

5. **`TestResolveGroupListFlags_InvalidOrder`** — Verify an invalid `--order` value (e.g., `"RANDOM"`) produces a descriptive error mentioning ASC/DESC.

6. **`TestGroupListFlags_ClientOptions`** — Verify `clientOptions()` produces the right number of options and handles empty sort/order correctly (only includes sort/order options when non-empty).

7. **`TestParseGroupID_Valid`** — Verify valid numeric string parsing.

8. **`TestParseGroupID_Invalid`** — Verify non-numeric strings produce a descriptive error.

9. **`TestGroupRow_Formatting`** — Verify `groupRow` handles a fully-populated `GroupInfo` and one with nil Stats.

10. **`TestGroupRow_NilFields`** — Verify `groupRow` handles a `GroupInfo` with all-nil fields (produces empty strings).

11. **`TestFormatInt32Ptr`** — Verify nil returns `""` and a non-nil value returns the expected string.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Instance-scoped means no `--project` flag.** This is the key difference from the filter pilot. The `Long` description should explicitly state this.
- **Offset/limit pagination differs from page-number pagination.** The group API uses `limit`/`offset`, not `page`/`page-size`. Do not reuse `registerListFlags` / `resolveListFlags` from `listflags.go`. Define group-specific pagination helpers in `group.go`.
- **Local validation of pagination flags.** Catch invalid values (negative limit/offset, invalid order) before they reach the API. This establishes the pattern that offset-based pagination commands validate locally, just as `resolveListFlags` validates page >= 1 for page-number pagination.
- **Inline flags for create/update, not `--from-file`.** The group request bodies are trivial (1–2 fields). This contrasts with the `filter` pilot which uses `--from-file` for complex payloads. Together they demonstrate both mutation input patterns.
- **`--slug` uses `optionalStringFlag` semantics.** The flag is nil when not explicitly passed by the user; if the user explicitly passes `--slug ""`, `optionalStringFlag` returns `&""` and the empty string is sent to the API (which will likely reject it server-side). Do not add CLI-side special-casing for empty values — let the API validate. This is consistent with how `user.go` uses `optionalStringFlag` (line 768): flag-was-changed means send the value, regardless of content.
- **Delete returns no body — use a local result struct.** Unlike `OperationCompletionRS`-returning deletes, `GroupsScope.Delete` returns only `error`. Define a `groupDeleteResult` struct with a `Message string` field and a `json:"message"` tag so that JSON output remains structured and consistent with other mutation responses.
- **List mode prints the resource slice, not the page wrapper.** Consistent with all existing list commands.
- **Group association commands (add/remove project, add/remove user) are NOT in scope.** Those are association commands implemented in a later phase.

## What NOT to do

- **Do not implement group-project or group-user association commands** (`ListProjects`, `AddProject`, `RemoveProject`, `ListUsers`, `AddUser`, `RemoveUser`). Those belong to the association phase (PR 21).
- **Do not add `--project` / `-p` flags** to any group command. Groups are instance-scoped.
- **Do not reuse `registerListFlags`** from `listflags.go`. The group API has a different pagination model.
- **Do not extract shared helpers** for offset-based pagination into a separate file yet. Keep helpers in `group.go`. Extraction happens in PR 7.
- **Do not add integration tests.** Unit tests for flag resolution, ID parsing, and table formatting are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` now shows `group` as a subcommand
- `rp create --help` now shows `group` as a subcommand
- `rp update --help` now shows `group` as a subcommand
- `rp delete --help` now shows `group` as a subcommand
- `rp get group --help` shows `--limit`, `--offset`, `--sort`, `--order` flags (no `--project`)
- `rp create group --help` shows `--name` and `--slug` flags
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/group.go` created with all five command constructors and their run functions
- [ ] `internal/cli/group_test.go` created with unit tests for flag resolution, ID parsing, and table formatting
- [ ] Commands registered in `commands.go`
- [ ] No `--project` flag on any group command
- [ ] Create and update use inline `--name` / `--slug` flags (not `--from-file`)
- [ ] Offset-based pagination flags (`--limit`, `--offset`, `--sort`, `--order`)
- [ ] Table output shows ID, NAME, SLUG, PROJECTS, USERS with uppercase headers
- [ ] JSON output for single-get prints the `GroupInfo` object; for list mode prints the unwrapped `[]GroupInfo` slice (no page wrapper)
- [ ] Delete uses a local `groupDeleteResult` struct so JSON mode outputs `{"message": "..."}` consistently with other mutations
- [ ] Pagination flags are validated locally (limit >= 1, offset >= 0, order in {ASC, DESC, ""})
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
