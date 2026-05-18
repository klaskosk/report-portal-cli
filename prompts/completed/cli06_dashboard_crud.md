# PR 6: `dashboard` CRUD

## Goal

Implement the complete `rp dashboard {list,get,create,update,delete}` command family. Dashboards are a standard project-scoped CRUD resource with paginated listing, simple flag-based mutations, and the typical `OperationCompletionRS` response for updates and deletes.

## Context

This PR runs in the Phase 4 parallel fan-out. PRs 1–5 have landed, establishing:

- Noun-first command structure (`rp <resource> <verb>`)
- Shared helpers: `resolveCmd`, `cmdContext.resolveProject`, `parseIDArg`, `readRequestFromFile[T]`, `messageHeaders`/`messageRows`, format helpers
- Pilot patterns from filter (project-scoped CRUD with `--from-file`), group (instance-scoped CRUD with flags), and widget (sub-resource pattern)

Dashboard CRUD follows the filter pilot pattern most closely — project-scoped, paginated list, simple mutations — but with flag-based create/update instead of `--from-file` because `CreateDashboardRQ` and `UpdateDashboardRQ` have few fields.

### Prerequisites

- PRs 1–5 have landed

### Client API surface

`DashboardScope` (obtained via `client.Project(projectName).Dashboards()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `List` | `List(ctx, ...ListDashboardsOption)` | `*types.PageDashboardResource` |
| `GetByID` | `GetByID(ctx, id int64)` | `*types.DashboardResource` |
| `Create` | `Create(ctx, types.CreateDashboardRQ)` | `*types.EntryCreatedRS` |
| `Update` | `Update(ctx, id int64, types.UpdateDashboardRQ)` | `*types.OperationCompletionRS` |
| `Delete` | `Delete(ctx, id int64)` | `*types.OperationCompletionRS` |

List options: `WithDashboardPageNumber(int32)`, `WithDashboardPageSize(int32)`, `WithDashboardSort(...string)`.

**Note:** `AddWidget`, `RemoveWidget`, `GetConfig`, and `CreatePreconfigured` are not part of this PR. They belong to Phase 7 (PR 20/21).

### Relevant types

```go
type CreateDashboardRQ struct {
    Description *string `json:"description,omitempty"`
    Name        string  `json:"name"`
}

type UpdateDashboardRQ struct {
    Description   *string              `json:"description,omitempty"`
    Name          string               `json:"name"`
    UpdateWidgets *[]WidgetObjectModel `json:"updateWidgets,omitempty"`
}

type DashboardResource struct {
    Description *string              `json:"description,omitempty"`
    Id          int64                `json:"id"`
    Name        string               `json:"name"`
    Owner       *string              `json:"owner,omitempty"`
    Widgets     *[]WidgetObjectModel `json:"widgets,omitempty"`
}

type PageDashboardResource struct {
    Content *[]DashboardResource `json:"content,omitempty"`
    Page    *PageMetadata        `json:"page,omitempty"`
}

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

## Implementation plan

### 1. Create `internal/cli/dashboard.go`

This file defines the `rp dashboard` parent command and all verb subcommands.

#### Parent command

```go
func newDashboardCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "dashboard",
        Short: "Manage dashboards",
    }
    cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
    cmd.AddCommand(newDashboardListCmd())
    cmd.AddCommand(newDashboardGetCmd())
    cmd.AddCommand(newDashboardCreateCmd())
    cmd.AddCommand(newDashboardUpdateCmd())
    cmd.AddCommand(newDashboardDeleteCmd())
    return cmd
}
```

Register in `root.go`: `cmd.AddCommand(newDashboardCmd())`.

#### `rp dashboard list`

- No positional arguments
- Pagination flags: `--page`, `--page-size`, `--sort` via `registerListFlagsWithDefaults` with `sort: nil` (dashboards have no natural time-based default sort)
- Table output columns: `ID`, `NAME`, `OWNER`, `DESCRIPTION`
  - Truncate `DESCRIPTION` to 60 characters in table mode
- JSON output: the raw `[]DashboardResource` slice (unwrapped from `PageDashboardResource.Content`)
  - Pass the slice to `printer.Print`, not the page wrapper, consistent with how `listUsers` handles it
- Use `resolveCmd` and `cmdContext.resolveProject` for the preamble

#### `rp dashboard get <id>`

- Exactly one positional argument: numeric dashboard ID
- Parse using `parseIDArg(args[0], "dashboard")`
- Table output: same columns as list, single row
- JSON output: the raw `*DashboardResource`

#### `rp dashboard create`

- No positional arguments
- Flags:
  - `--name` (string, required) — dashboard name
  - `--description` (string, optional) — dashboard description
- Mark `--name` as required via `cmd.MarkFlagRequired("name")`
- Build `types.CreateDashboardRQ` from flags
- Call `scope.Create(ctx, request)`
- Table output: `ID` column only (from `EntryCreatedRS.Id`)
- JSON output: the raw `*EntryCreatedRS`

#### `rp dashboard update <id>`

- Exactly one positional argument: numeric dashboard ID
- Flags:
  - `--name` (string, optional) — updated name
  - `--description` (string, optional) — updated description
- At least one flag must be provided (validate like `editUserFlags.hasChanges`)
- **Fetch-and-merge for `Name`:** Since `UpdateDashboardRQ.Name` is a required (non-pointer) field, if `--name` is not provided, fetch the current dashboard via `scope.GetByID` and use its existing name. This follows the same pattern as `user update` merging `FullName`.
- Build `types.UpdateDashboardRQ` from flags (do NOT set `UpdateWidgets` — that field is deferred to Phase 7)
- Call `scope.Update(ctx, id, request)`
- Table/JSON output: `OperationCompletionRS` via `messageHeaders()`/`messageRows(formatMessage(result.Message))`

#### `rp dashboard delete <id>`

- Exactly one positional argument: numeric dashboard ID
- No additional flags
- Call `scope.Delete(ctx, id)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

### 2. Table formatting

```go
func dashboardHeaders() []string {
    return []string{"ID", "NAME", "OWNER", "DESCRIPTION"}
}

func dashboardRow(d *types.DashboardResource) ([]string, []string) {
    description := formatStringPtr(d.Description)
    if len(description) > 60 {
        description = description[:60] + "..."
    }

    return dashboardHeaders(), []string{
        strconv.FormatInt(d.Id, 10),
        d.Name,
        formatStringPtr(d.Owner),
        description,
    }
}
```

### 3. Register in root.go

Add `cmd.AddCommand(newDashboardCmd())` alongside the other resource commands.

## Test requirements

### Unit tests (`internal/cli/dashboard_test.go`)

- **ID parsing:** Verify that `rp dashboard get abc` returns an error (non-numeric ID)
- **List flag defaults:** Verify that dashboard list uses no default sort
- **Create validation:** Missing `--name` errors
- **Update validation:**
  - At least one flag must be provided (no-op update errors)
  - `--name` and `--description` are correctly applied when provided
- **Table formatting:** Verify `dashboardRow` produces expected columns, handles nil Description/Owner
- **Description truncation:** Verify descriptions longer than 60 characters are truncated with `...` suffix
- **Command wiring:** Verify `newDashboardCmd()` has the expected subcommands (list, get, create, update, delete)

Use the same white-box test style as existing tests (same package `cli`, `//nolint:testpackage`, testify `require`/`assert`, `t.Parallel()`).

## Edge cases and constraints

- **`UpdateWidgets` is not set by this PR.** The update command only modifies `Name` and `Description`. The `UpdateWidgets` field in `UpdateDashboardRQ` will be used by Phase 7's `rp dashboard widget` commands. Do NOT expose `UpdateWidgets` through flags or `--from-file` in this PR.
- **`DashboardResource.Id` is a value type (`int64`):** Not a pointer, so no nil handling needed for the ID column.
- **`DashboardResource.Widgets` in JSON output:** JSON output includes the full `*[]WidgetObjectModel` from the API response. Do not filter or strip it.
- **Empty page response:** If `PageDashboardResource.Content` is nil, treat it as an empty slice (no rows, no error).
- **Fetch-and-merge on update:** When the user omits `--name`, the command must fetch the current dashboard first. If the fetch fails (e.g., dashboard not found), surface that error clearly before attempting the update.
- **No `--from-file` for this resource:** The create/update payloads are simple enough for flags. Do not add `--from-file` support for dashboard CRUD.

## Example commands and expected output

```bash
# List dashboards
$ rp dashboard list -p my_project
ID    NAME                  OWNER    DESCRIPTION
1     Test Overview          admin    Shows overall test results for...
2     Sprint Dashboard       admin    Sprint metrics and trends

# Get single dashboard
$ rp dashboard get 1 -p my_project
ID    NAME                  OWNER    DESCRIPTION
1     Test Overview          admin    Shows overall test results for the current release

# Create
$ rp dashboard create --name "New Dashboard" -p my_project
ID
42

# Create with description
$ rp dashboard create --name "New Dashboard" --description "My custom dashboard" -p my_project
ID
42

# Update name only
$ rp dashboard update 42 --name "Renamed Dashboard" -p my_project
MESSAGE
Dashboard with ID = '42' has been successfully updated.

# Update description only (fetches current name automatically)
$ rp dashboard update 42 --description "Updated description" -p my_project
MESSAGE
Dashboard with ID = '42' has been successfully updated.

# Delete
$ rp dashboard delete 42 -p my_project
MESSAGE
Dashboard with ID = '42' has been successfully deleted.
```

## Validation

Run before finalizing:

```
make lint
make test
```

## Definition of done

- [ ] `internal/cli/dashboard.go` implements all five subcommands (list, get, create, update, delete)
- [ ] `rp dashboard` registered in `root.go`
- [ ] Flag-based create (`--name`, `--description`) and update with fetch-and-merge for name
- [ ] `UpdateWidgets` is NOT exposed — deferred to Phase 7
- [ ] Table output with uppercase headers: ID, NAME, OWNER, DESCRIPTION
- [ ] JSON output passes through raw API types unchanged
- [ ] Unit tests cover ID parsing, flag validation, table formatting, and command wiring
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
