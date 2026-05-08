# PR 8: `dashboard` CRUD

## Goal

Implement the complete `dashboard` command family — list, get-by-ID, create, update, and delete — as a standard project-scoped CRUD resource. Dashboards use page-number pagination (like `filter`) and inline flags for mutations (like `group`), since both `CreateDashboardRQ` and `UpdateDashboardRQ` have simple scalar fields.

## Context

The `dashboard` resource maps to `DashboardScope` on the client (`client.Project(name).Dashboards()`). It is project-scoped, meaning every command requires a resolved project via `--project`, `RP_PROJECT`, or the profile's `default-project`.

**Prerequisite:** PRs 1–7 must land first (parent command split, verb scaffolding, centralized registration, all three pilots, and shared helper extraction).

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `List` | `ctx`, `options ...ListDashboardsOption` | `(*types.PageDashboardResource, error)` |
| `GetByID` | `ctx`, `id int64` | `(*types.DashboardResource, error)` |
| `Create` | `ctx`, `request types.CreateDashboardRQ` | `(*types.EntryCreatedRS, error)` |
| `Update` | `ctx`, `id int64`, `request types.UpdateDashboardRQ` | `(*types.OperationCompletionRS, error)` |
| `Delete` | `ctx`, `id int64` | `(*types.OperationCompletionRS, error)` |

List options: `WithDashboardPageNumber(int32)`, `WithDashboardPageSize(int32)`, `WithDashboardSort(...string)`.

### Key types

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
```

## Scope

Create `internal/cli/dashboard.go` and `internal/cli/dashboard_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get dashboard` | List dashboards in the project (paginated) |
| `rp get dashboard <id>` | Get a single dashboard by numeric ID |
| `rp create dashboard --name <name> [--description <desc>]` | Create a dashboard |
| `rp update dashboard <id> --name <name> [--description <desc>]` | Update a dashboard |
| `rp delete dashboard <id>` | Delete a dashboard by numeric ID |

## Implementation

### 1. Create `internal/cli/dashboard.go`

This file contains all five command constructors and their run functions.

#### `newGetDashboardCmd()`

```go
func newGetDashboardCmd() *cobra.Command
```

- `Use`: `"dashboard [id]"`
- `Short`: `"List dashboards or get a specific dashboard"`
- `Long`: Include examples showing list, get-by-ID, project override, and output format
- `Args`: `cobra.MaximumNArgs(1)`
- `RunE`: `withPrintedErrors(runGetDashboard)`
- Flags:
  - `--project` / `-p` (string): project name override
  - Standard list flags via `registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: nil})` (provides `--page-size`, `--page`, `--sort` with no default sort)

**Do NOT** use the bare `registerListFlags(cmd)` — it hardcodes a default sort of `startTime,desc` which is launch-specific. Dashboards have no `startTime` field. Use `registerListFlagsWithDefaults` with a nil sort default so `--sort` is empty unless the user explicitly provides one.

The run function follows the same dispatch pattern as `runGetLaunch`:
1. Resolve connection, project, and client
2. If `len(args) == 1`, parse arg as int64 using `parseResourceID(args[0], "dashboard")` and call `GetByID`; print the single `DashboardResource` as one row
3. Otherwise, resolve list flags, build options (`WithDashboardPageSize`, `WithDashboardPageNumber`, `WithDashboardSort`), call `List`, unwrap the `Content` slice from the paginated response, and pass the **slice** (not the page wrapper) to `printer.Print`

**Output contract:** In list mode, the value passed to `printer.Print` is the `[]types.DashboardResource` slice, not the `PageDashboardResource` wrapper. This matches how all existing list commands work. JSON mode outputs a JSON array of dashboard objects.

#### `newCreateDashboardCmd()`

```go
func newCreateDashboardCmd() *cobra.Command
```

- `Use`: `"dashboard"`
- `Short`: `"Create a dashboard"`
- `Long`: Include examples showing inline flag usage
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateDashboard)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--name` (string, required): dashboard name
  - `--description` (string, optional): dashboard description

The run function:
1. Validate `--name` is provided (non-empty). Return a descriptive error if missing.
2. Resolve `--description` using `optionalStringFlag(cmd, "description")` — returns nil if the flag was not explicitly passed, or `&value` if it was
3. Build `types.CreateDashboardRQ{Name: name, Description: descPtr}`
4. Resolve connection, project, and client
5. Call `scope.Create(ctx, request)`
6. Print the `EntryCreatedRS` result using `entryCreatedHeaders()` and `entryCreatedRows(result)`

**Why inline flags, not `--from-file`?** `CreateDashboardRQ` has only 2 fields (Name string, Description *string). Inline flags are the right UX per the CLI plan's decision rules: use `--from-file` only for complex payloads.

#### `newUpdateDashboardCmd()`

```go
func newUpdateDashboardCmd() *cobra.Command
```

- `Use`: `"dashboard <id>"`
- `Short`: `"Update a dashboard"`
- `Long`: Include examples showing update with name and optional description
- `Args`: `cobra.ExactArgs(1)` (the numeric dashboard ID)
- `RunE`: `withPrintedErrors(runUpdateDashboard)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--name` (string, required): new dashboard name
  - `--description` (string, optional): new dashboard description

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "dashboard")`
2. Validate `--name` is provided (required by the API — `UpdateDashboardRQ.Name` is not a pointer)
3. Resolve `--description` using `optionalStringFlag(cmd, "description")` — nil if flag was not passed; `&value` (possibly empty) if it was
4. Build `types.UpdateDashboardRQ{Name: name, Description: descPtr}` — `UpdateWidgets` is intentionally left nil (see design notes)
5. Resolve connection, project, and client
6. Call `scope.Update(ctx, id, request)`
7. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newDeleteDashboardCmd()`

```go
func newDeleteDashboardCmd() *cobra.Command
```

- `Use`: `"dashboard <id>"`
- `Short`: `"Delete a dashboard"`
- `Long`: Include example showing delete by ID
- `Args`: `cobra.ExactArgs(1)` (the numeric dashboard ID)
- `RunE`: `withPrintedErrors(runDeleteDashboard)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "dashboard")`
2. Resolve connection, project, and client
3. Call `scope.Delete(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### Table formatting

```go
func dashboardHeaders() []string {
    return []string{"ID", "NAME", "OWNER", "DESCRIPTION"}
}

func dashboardRow(dashboard *types.DashboardResource) ([]string, []string) {
    return dashboardHeaders(), []string{
        strconv.FormatInt(dashboard.Id, 10),
        dashboard.Name,
        formatStringPtr(dashboard.Owner),
        formatStringPtr(dashboard.Description),
    }
}
```

The `Widgets` field is intentionally omitted from the table. It is a complex nested structure (`[]WidgetObjectModel`) that is not meaningful in a summary table. Users who need widget details should use `-o json`.

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// get
get.AddCommand(newGetDashboardCmd())

// create
create.AddCommand(newCreateDashboardCmd())

// update
update.AddCommand(newUpdateDashboardCmd())

// delete
del.AddCommand(newDeleteDashboardCmd())
```

### 3. Create `internal/cli/dashboard_test.go`

Use the same testing pattern as the pilot test files:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetDashboardCmd_DefaultListOptions`** — Verify the default list flags are registered correctly (page-size=10, page=1, sort=empty/nil).

2. **`TestNewGetDashboardCmd_HasProjectFlag`** — Verify the `--project` / `-p` flag is registered.

3. **`TestDashboardRow_Formatting`** — Verify `dashboardRow` produces the expected headers and values for a sample `DashboardResource`, including a nil description.

4. **`TestDashboardRow_NilOptionalFields`** — Verify `dashboardRow` handles a `DashboardResource` with nil Owner and nil Description (produces empty strings).

5. **`TestNewCreateDashboardCmd_Flags`** — Verify the create command has `--name`, `--description`, and `--project` flags registered.

6. **`TestNewUpdateDashboardCmd_Flags`** — Verify the update command has `--name`, `--description`, and `--project` flags registered and accepts exactly 1 arg.

7. **`TestNewDeleteDashboardCmd_Args`** — Verify the delete command accepts exactly 1 arg.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Inline flags for create and update, not `--from-file`.** Both `CreateDashboardRQ` (2 fields) and the used portion of `UpdateDashboardRQ` (2 fields) are simple scalar payloads. This follows the group pilot pattern.
- **`UpdateWidgets` is intentionally not exposed in this PR.** The `UpdateDashboardRQ.UpdateWidgets` field allows repositioning existing widgets on a dashboard. This is a dashboard↔widget relationship operation that belongs in PR 22 (dashboard-widget relationship commands), not the standard CRUD slice. Because the field uses `json:"updateWidgets,omitempty"`, a nil value is omitted from the serialized JSON — the server is never sent an `updateWidgets` key.
- **Page-number pagination (not offset-based).** Dashboard list options use `WithDashboardPageNumber`/`WithDashboardPageSize`, matching the standard `registerListFlagsWithDefaults` / `resolveListFlags` pattern from `listflags.go`. This is the same pagination model as filter, launch, item, and log.
- **Reuse all shared helpers from PR 7.** Use `parseResourceID`, `formatStringPtr`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`, `optionalStringFlag`. Do not redefine any of these locally.
- **`--description` uses `optionalStringFlag` semantics.** The flag is nil when not explicitly passed by the user. If the user passes `--description ""`, it sends an explicit empty string to the API. This is consistent with how `--slug` works in the group pilot.

## What NOT to do

- **Do not implement `AddWidget`, `RemoveWidget`, `UpdateWidgets`, `GetConfig`, or `CreatePreconfigured`.** Those are separate PRs (PR 22 for dashboard↔widget relationships including `AddWidget`, `RemoveWidget`, and `UpdateWidgets`; PR 23 for dashboard extras like `GetConfig` and `CreatePreconfigured`).
- **Do not expose `UpdateWidgets` in the update command.** Leave it nil. That is PR 22's scope.
- **Do not add resource-specific filter flags** (e.g., filter by owner). Dashboard list options only support pagination and sort.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` now shows `dashboard` as a subcommand
- `rp create --help` now shows `dashboard` as a subcommand
- `rp update --help` now shows `dashboard` as a subcommand
- `rp delete --help` now shows `dashboard` as a subcommand
- `rp get dashboard --help` shows pagination flags and `--project`
- `rp create dashboard --help` shows `--name`, `--description`, and `--project` flags
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/dashboard.go` created with all five command constructors and their run functions
- [ ] `internal/cli/dashboard_test.go` created with unit tests for flag registration and table formatting
- [ ] Commands registered in `commands.go`
- [ ] All commands use `--project` / `-p` for project resolution
- [ ] Create and update use inline `--name` / `--description` flags (not `--from-file`)
- [ ] Page-number pagination flags (`--page-size`, `--page`, `--sort`) with no default sort
- [ ] Table output shows ID, NAME, OWNER, DESCRIPTION with uppercase headers
- [ ] JSON output for single-get prints the `DashboardResource` object; for list mode prints the unwrapped `[]DashboardResource` slice (no page wrapper)
- [ ] `UpdateWidgets` is NOT exposed — update constructs `UpdateDashboardRQ` with nil `UpdateWidgets`
- [ ] Shared helpers used: `parseResourceID`, `formatStringPtr`, `optionalStringFlag`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
