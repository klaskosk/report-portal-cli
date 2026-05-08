# PR 32: Project-Scoped `activity` Commands

## Goal

Implement the read-only `activity` commands for viewing item-specific project activity. These are project-scoped commands that retrieve activity logs for specific test items.

## Context

Activity is managed through `ActivityScope` on the client (`client.Project(name).Activities()`). It is **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `parseResourceID`, `formatStringPtr`, `registerListFlagsWithDefaults`)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `GetByID` | `ctx`, `activityID int64` | `(*types.ActivityResource, error)` |
| `ListForItem` | `ctx`, `itemID int64`, `options ...ListActivitiesOption` | `(*types.PageActivityEventResource, error)` |

List options: `WithActivityPageNumber(int32)`, `WithActivityPageSize(int32)`, `WithActivitySort(...string)`.

### Key types

```go
type ActivityResource struct {
    ActionType     string      `json:"actionType"`
    Details        interface{} `json:"details,omitempty"`
    Id             int64       `json:"id"`
    LastModified   time.Time   `json:"lastModified"`
    LoggedObjectId int64       `json:"loggedObjectId"`
    ObjectName     *string     `json:"objectName,omitempty"`
    ObjectType     string      `json:"objectType"`
    ProjectId      int64       `json:"projectId"`
    ProjectName    *string     `json:"projectName,omitempty"`
    User           string      `json:"user"`
}

type PageActivityEventResource struct {
    Content []ActivityEventResource `json:"content"`
    Page    PageMetadata            `json:"page"`
}

type ActivityEventResource struct {
    CreatedAt   time.Time   `json:"created_at"`
    Details     interface{} `json:"details,omitempty"`
    EventName   string      `json:"event_name"`
    Id          int64       `json:"id"`
    ObjectId    int64       `json:"object_id"`
    ObjectName  string      `json:"object_name"`
    ObjectType  string      `json:"object_type"`
    ProjectId   int64       `json:"project_id"`
    ProjectName string      `json:"project_name"`
    SubjectId   string      `json:"subject_id"`
    SubjectName string      `json:"subject_name"`
    SubjectType string      `json:"subject_type"`
}
```

**Important:** `GetByID` returns `ActivityResource` while `ListForItem` returns a page of `ActivityEventResource`. These are different types with different field names (e.g., `ActionType` vs `EventName`, `User` vs `SubjectName`). Each needs its own table formatter.

## Scope

Create `internal/cli/activity.go` and `internal/cli/activity_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get activity <id>` | Get a specific activity by ID |
| `rp get activity --item <item-id>` | List activities for a test item (paginated) |

## Implementation

### 1. Create `internal/cli/activity.go`

#### `newGetActivityCmd()`

```go
func newGetActivityCmd() *cobra.Command
```

- `Use`: `"activity [id]"`
- `Short`: `"Get activities for a project"`
- `Long`: Include examples showing get-by-ID and list-for-item. Note that `--item` is required for listing (there is no general list-all-activities endpoint).
- `Args`: `cobra.MaximumNArgs(1)`
- `RunE`: `withPrintedErrors(runGetActivity)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--item` (int64, optional): test item ID to list activities for
  - Standard list flags via `registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: nil})`

The run function:
1. **Validate mutual exclusivity first.** If `len(args) == 1` AND `cmd.Flags().Changed("item")`, return an error: `"provide either an activity ID or --item, not both"`. This prevents the positional arg from silently winning.
2. Resolve connection, project, and client
3. If `len(args) == 1`:
   - Parse `args[0]` as int64 using `parseResourceID(args[0], "activity")`
   - Call `scope.GetByID(ctx, activityID)`
   - Print the `ActivityResource` as a single row using `activityResourceRow`
4. Else if `--item` was provided:
   - Read `--item` as int64 via `cmd.Flags().GetInt64("item")`
   - Validate item ID is positive
   - Resolve list flags, build options (`WithActivityPageSize`, `WithActivityPageNumber`, `WithActivitySort`)
   - Call `scope.ListForItem(ctx, itemID, options...)`
   - Unwrap the `Content` slice from the paginated response
   - Pass the `[]ActivityEventResource` slice to `printer.Print` using `activityEventRow`
5. Else:
   - Return an error: `"provide an activity ID or use --item to list activities for a test item"`

**Two different formatters for two different types:**

```go
func activityResourceHeaders() []string {
    return []string{"ID", "ACTION", "OBJECT TYPE", "OBJECT NAME", "USER", "LAST MODIFIED"}
}

func activityResourceRow(a *types.ActivityResource) ([]string, []string) {
    return activityResourceHeaders(), []string{
        strconv.FormatInt(a.Id, 10),
        a.ActionType,
        a.ObjectType,
        formatStringPtr(a.ObjectName),
        a.User,
        a.LastModified.Format(time.RFC3339),
    }
}

func activityEventHeaders() []string {
    return []string{"ID", "EVENT", "OBJECT TYPE", "OBJECT NAME", "SUBJECT", "CREATED AT"}
}

func activityEventRow(e *types.ActivityEventResource) ([]string, []string) {
    return activityEventHeaders(), []string{
        strconv.FormatInt(e.Id, 10),
        e.EventName,
        e.ObjectType,
        e.ObjectName,
        e.SubjectName,
        e.CreatedAt.Format(time.RFC3339),
    }
}
```

### 2. Register commands in `commands.go`

```go
// get
get.AddCommand(newGetActivityCmd())
```

### 3. Create `internal/cli/activity_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetActivityCmd_Flags`** — Verify `--project` / `-p`, `--item`, and pagination flags. Verify `MaximumNArgs(1)`.

2. **`TestActivityResourceRow_Formatting`** — Verify row output for a sample `ActivityResource`.

3. **`TestActivityEventRow_Formatting`** — Verify row output for a sample `ActivityEventResource`.

4. **`TestActivityResourceRow_NilOptionalFields`** — Verify nil `ObjectName` and `ProjectName` handling.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Two different response types.** `GetByID` returns `ActivityResource` and `ListForItem` returns `PageActivityEventResource` (containing `[]ActivityEventResource`). These have different field names and layouts, so they need separate formatters.
- **`--item` is required for listing.** There is no general "list all activities" endpoint. The user must specify which item's activities to view.
- **No create/update/delete.** Activities are read-only audit records.
- **`Details` field omitted from table.** The `Details` field is `interface{}` and contains structured change history. Use `-o json` for full details.
- **Reuse shared helpers.** Use `parseResourceID`, `formatStringPtr`, `registerListFlagsWithDefaults`.

## What NOT to do

- **Do not confuse with `activity-event`** (PR 33). That is the instance-scoped cross-project audit search.
- **Do not add mutation commands.** Activities are read-only.
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` shows `activity`
- `rp get activity --help` shows `--item`, `--project`, and pagination flags

## Definition of done

- [ ] `internal/cli/activity.go` created
- [ ] `internal/cli/activity_test.go` created
- [ ] Command registered under `get` parent
- [ ] Uses `--project` / `-p`
- [ ] Single-get-by-ID with positional arg
- [ ] List-for-item with `--item` flag
- [ ] Two separate formatters for `ActivityResource` and `ActivityEventResource`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
