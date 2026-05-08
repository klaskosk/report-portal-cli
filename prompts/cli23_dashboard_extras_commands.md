# PR 23: Dashboard Extras Commands

## Goal

Implement the non-CRUD dashboard operations: retrieving the dashboard widget configuration and creating a fully preconfigured dashboard. These commands extend the existing dashboard surface with operations that return or accept conceptually different views of the dashboard resource.

## Context

Dashboard extras map to `DashboardScope` on the client (`client.Project(name).Dashboards()`). Both operations are **project-scoped**, requiring project resolution via `--project`, `RP_PROJECT`, or the profile's `default-project`.

These commands were settled in PR 17:
- `rp get dashboard config <id>` — returns the widget layout definitions (a different view from the dashboard metadata returned by `rp get dashboard <id>`)
- `rp create dashboard --preconfigured --from-file <path>` — creates a dashboard from a full template with widgets included

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `readRequestFromFile`, `parseResourceID`, `entryCreatedHeaders`/`entryCreatedRows`)
- PR 8 (dashboard CRUD — `create dashboard` must exist before `--preconfigured` mode extends it)
- PR 17 (frozen UX: §3 dashboard config and preconfigured-dashboard)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `GetConfig` | `ctx`, `dashboardID int64` | `(*types.DashboardConfigResource, error)` |
| `CreatePreconfigured` | `ctx`, `request types.DashboardPreconfiguredRq` | `(*types.EntryCreatedRS, error)` |

### Key types

```go
type DashboardConfigResource struct {
    Widgets *[]WidgetConfigResource `json:"widgets,omitempty"`
}

type WidgetConfigResource struct {
    WidgetObject   WidgetObjectModel `json:"widgetObject"`
    WidgetResource WidgetResource    `json:"widgetResource"`
}

type WidgetObjectModel struct {
    WidgetId       int64                   `json:"widgetId"`
    WidgetName     *string                 `json:"widgetName,omitempty"`
    WidgetOptions  *map[string]interface{} `json:"widgetOptions,omitempty"`
    WidgetPosition *Position               `json:"widgetPosition,omitempty"`
    WidgetSize     *Size                   `json:"widgetSize,omitempty"`
    WidgetType     *string                 `json:"widgetType,omitempty"`
}

type WidgetResource struct {
    AppliedFilters    *[]UserFilterResource    `json:"appliedFilters,omitempty"`
    Content           *map[string]interface{}  `json:"content,omitempty"`
    ContentParameters ContentParameters        `json:"contentParameters"`
    Description       *string                  `json:"description,omitempty"`
    Id                int64                    `json:"id"`
    Name              string                   `json:"name"`
    Owner             *string                  `json:"owner,omitempty"`
    WidgetType        WidgetResourceWidgetType `json:"widgetType"`
}

type DashboardPreconfiguredRq struct {
    Config      DashboardConfigResource `json:"config"`
    Description *string                 `json:"description,omitempty"`
    Name        string                  `json:"name"`
}

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}
```

## Scope

Extend `internal/cli/dashboard.go` with one new command constructor (`newGetDashboardConfigCmd`) and modify the existing `newCreateDashboardCmd` to support `--preconfigured` mode. Create `internal/cli/dashboard_extras_test.go` for the new functionality. Update `internal/cli/commands.go` if needed.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get dashboard <id>` | *(existing — unchanged)* |
| `rp get dashboard config <id>` | Get the dashboard widget configuration |
| `rp create dashboard --name <name> [--description <desc>]` | *(existing — unchanged)* |
| `rp create dashboard --preconfigured --from-file <path>` | Create a fully configured dashboard with widgets |

## Implementation

### 1. Add `config` subcommand to `internal/cli/dashboard.go`

#### `newGetDashboardConfigCmd()`

```go
func newGetDashboardConfigCmd() *cobra.Command
```

- `Use`: `"config <dashboard-id>"`
- `Short`: `"Get dashboard widget configuration"`
- `Long`: Include examples showing basic usage. Explain that this returns the full widget layout definitions (position, size, widget resources) for the dashboard. Recommend `-o json` for inspecting the full nested structure.
- `Args`: `cobra.ExactArgs(1)` (the numeric dashboard ID)
- `RunE`: `withPrintedErrors(runGetDashboardConfig)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "dashboard")`
2. Resolve connection, project, and client
3. Call `scope.GetConfig(ctx, dashboardID)`
4. Print the `DashboardConfigResource` result

**Cobra routing:** The `config` subcommand is registered on the existing `get dashboard` command via `cmd.AddCommand(newGetDashboardConfigCmd())` inside `newGetDashboardCmd()`. This follows the same pattern as `rp get widget data <id>` (PR 6) where a subcommand is registered on a command that also has its own `RunE`.

Routing behavior:
- `rp get dashboard config 42` → routes to `config` subcommand (Cobra matches "config" as a subcommand name)
- `rp get dashboard 42` → routes to parent's `RunE` (Cobra sees "42" is not a subcommand name)
- `rp get dashboard` → routes to parent's `RunE` with no args (list mode)

#### Table formatting for `DashboardConfigResource`

The `DashboardConfigResource` contains a `Widgets` slice of `WidgetConfigResource`, each of which contains both layout info (`WidgetObjectModel`) and the full widget definition (`WidgetResource`). This is deeply nested and best consumed as JSON.

For table mode, show a summary of widgets in the config:

```go
func dashboardConfigHeaders() []string {
    return []string{"WIDGET ID", "WIDGET NAME", "TYPE", "POSITION", "SIZE"}
}

func dashboardConfigRows(config *types.DashboardConfigResource) [][]string {
    if config.Widgets == nil {
        return nil
    }

    rows := make([][]string, 0, len(*config.Widgets))
    for _, wc := range *config.Widgets {
        rows = append(rows, []string{
            strconv.FormatInt(wc.WidgetObject.WidgetId, 10),
            formatStringPtr(wc.WidgetObject.WidgetName),
            formatStringPtr(wc.WidgetObject.WidgetType),
            formatPosition(wc.WidgetObject.WidgetPosition),
            formatSize(wc.WidgetObject.WidgetSize),
        })
    }

    return rows
}

func formatPosition(pos *types.Position) string {
    if pos == nil {
        return ""
    }
    return fmt.Sprintf("(%s,%s)", formatInt32Ptr(pos.PositionX), formatInt32Ptr(pos.PositionY))
}

func formatSize(size *types.Size) string {
    if size == nil {
        return ""
    }
    return fmt.Sprintf("%sx%s", formatInt32Ptr(size.Width), formatInt32Ptr(size.Height))
}
```

### 2. Modify existing `newCreateDashboardCmd()` for `--preconfigured` mode

Add a `--preconfigured` flag and `--from-file` / `-f` flag to the existing create dashboard command. The run function branches based on whether `--preconfigured` is set.

**Add flags to `newCreateDashboardCmd()`:**

```go
cmd.Flags().Bool("preconfigured", false, "Create from a preconfigured template (requires --from-file)")
cmd.Flags().StringP("from-file", "f", "", "Path to JSON file (required with --preconfigured)")
```

**Modify the run function (`runCreateDashboard`)** to dispatch based on mode:

```go
func runCreateDashboard(cmd *cobra.Command, args []string) error {
    preconfigured, _ := cmd.Flags().GetBool("preconfigured")

    if preconfigured {
        return runCreatePreconfiguredDashboard(cmd, args)
    }

    return runCreateStandardDashboard(cmd, args)
}
```

Extract the existing create logic into `runCreateStandardDashboard` (no behavioral changes to existing flow).

#### `runCreatePreconfiguredDashboard(cmd, args)`

1. Validate `--from-file` is non-empty; return error: `"--from-file is required when using --preconfigured"`
2. Validate `--name` and `--description` are NOT explicitly set (neither flag may be `Changed`). If either is set alongside `--preconfigured`, return a descriptive error explaining that standard creation uses inline `--name` / `--description` flags while preconfigured creation uses `--from-file`.
3. Read and unmarshal using `readRequestFromFile[types.DashboardPreconfiguredRq](path)`
4. Resolve connection, project, and client
5. Call `scope.CreatePreconfigured(ctx, request)`
6. Print the `EntryCreatedRS` result using `entryCreatedHeaders()` and `entryCreatedRows(result)`

**Mutual exclusivity rules:**
- `--preconfigured` requires `--from-file`
- `--preconfigured` is mutually exclusive with `--name` and `--description`
- Without `--preconfigured`, behavior is unchanged (existing inline flag flow)

#### `runCreateStandardDashboard(cmd, args)`

This is the existing create behavior extracted to a helper function. If `--from-file` is passed without `--preconfigured`, return a descriptive error: `"--from-file is only valid with --preconfigured; for standard creation use --name and --description"`.

### 3. Register the `config` subcommand

Inside `newGetDashboardCmd()` in `dashboard.go`, add:

```go
cmd.AddCommand(newGetDashboardConfigCmd())
```

This must be added **before** the `return cmd` statement. The `config` subcommand registration happens inside the parent command constructor, not in `commands.go`, because it's a nested subcommand (child of `get dashboard`, not child of `get`).

**No change needed in `commands.go`** for the `config` subcommand (it's registered on the parent, not directly on a verb).

The `--preconfigured` and `--from-file` flags on `create dashboard` also require no `commands.go` change — they're flags on an already-registered command.

### 4. Create `internal/cli/dashboard_extras_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetDashboardCmd_HasConfigSubcommand`** — Verify that `newGetDashboardCmd()` has a subcommand named `"config"`.

2. **`TestNewGetDashboardConfigCmd_Args`** — Verify it accepts exactly 1 arg.

3. **`TestNewGetDashboardConfigCmd_HasProjectFlag`** — Verify `--project` / `-p` is registered.

4. **`TestDashboardConfigRows_WithWidgets`** — Create a sample `DashboardConfigResource` with 2 widgets, verify `dashboardConfigRows` produces 2 rows with expected values.

5. **`TestDashboardConfigRows_NilWidgets`** — Verify nil `Widgets` produces nil/empty rows.

6. **`TestDashboardConfigRows_EmptyWidgets`** — Verify empty `Widgets` slice produces empty rows.

7. **`TestFormatPosition_Nil`** — Verify nil returns empty string.

8. **`TestFormatPosition_Values`** — Verify non-nil position formats as `(x,y)`.

9. **`TestFormatSize_Nil`** — Verify nil returns empty string.

10. **`TestFormatSize_Values`** — Verify non-nil size formats as `WxH`.

11. **`TestNewCreateDashboardCmd_HasPreconfiguredFlag`** — Verify `--preconfigured` flag is registered as bool.

12. **`TestNewCreateDashboardCmd_HasFromFileFlag`** — Verify `--from-file` / `-f` flag is registered.

13. **`TestReadDashboardPreconfiguredRq_ValidJSON`** — Write a temp JSON file with a valid `DashboardPreconfiguredRq`, verify deserialization including nested `Config.Widgets`.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **`config` as a subcommand of `get dashboard`.** This follows the widget pilot's pattern (PR 6). The dashboard config is a conceptually different view from the dashboard metadata — it contains the full widget layout definitions. Per decision rule 1: "Different object → subcommand."
- **`--preconfigured` as a mode flag on `create dashboard`.** Per PR 17 decision: a separate `preconfigured-dashboard` resource would only exist under `create` with no `get`/`update`/`delete` counterpart. This breaks the CLI's symmetric verb-resource taxonomy. The mode flag keeps the dashboard resource unified.
- **Mutual exclusivity enforcement.** The two creation modes have incompatible input mechanisms: standard mode uses `--name`/`--description` flags; preconfigured mode uses `--from-file`. Allowing both would be confusing. Clear error messages guide the user.
- **Table mode for config shows widget summary.** The full `DashboardConfigResource` is deeply nested (each widget contains its full resource definition). Table mode extracts the practical summary (ID, name, type, position, size). Users needing the full detail should use `-o json`.
- **`formatPosition` and `formatSize` are local helpers.** They format nested position/size objects as compact strings for table display. If other commands need them in the future, they can be extracted to shared helpers in a normalization pass.
- **Modifying existing `create dashboard`.** This PR modifies `newCreateDashboardCmd` from PR 8 to add the `--preconfigured` flag path. The existing behavior (inline `--name`/`--description`) remains identical when `--preconfigured` is not set. The refactor extracts the existing logic into `runCreateStandardDashboard` without changing its behavior.
- **Reuse shared helpers.** Use `parseResourceID`, `readRequestFromFile`, `formatStringPtr`, `formatInt32Ptr`, `entryCreatedHeaders`/`entryCreatedRows`, `withPrintedErrors`.
- **Route all output through `printer.Print`.** Use `printer.Print(result, headers, rows)` for all results. This ensures JSON, Go-template, and table modes work correctly.
- **Use client error predicates.** Handle API errors with predicate functions (`IsNotFound`, `IsUnauthorized`, etc.) rather than type assertions.
- **Command registration follows PR 3's centralized command-tree layout.** Shared mutation/result helpers assume PR 7's naming conventions.

## What NOT to do

- **Do not modify the existing `get dashboard` list/single-get behavior.** Only add the `config` subcommand to the command.
- **Do not change table formatting for `rp get dashboard <id>`.** The existing dashboard table (ID, NAME, OWNER, DESCRIPTION) stays the same.
- **Do not add `UpdateWidgets` (widget repositioning).** That is a separate concern.
- **Do not create a separate `preconfigured-dashboard` resource name.** Use `--preconfigured` flag on existing `create dashboard`.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get dashboard --help` shows `config` as a subcommand
- `rp get dashboard config --help` shows `--project` flag and expects a dashboard ID arg
- `rp get dashboard 42` still works (routes to parent RunE, not config subcommand)
- `rp get dashboard` still works (list mode)
- `rp create dashboard --help` shows `--preconfigured` and `--from-file` alongside existing `--name`/`--description`
- `rp create dashboard --name test` still works (standard mode unchanged)
- No existing commands are affected

## Definition of done

- [ ] `newGetDashboardConfigCmd` added to `internal/cli/dashboard.go`
- [ ] `config` subcommand registered on the `get dashboard` command
- [ ] `--preconfigured` and `--from-file` flags added to `create dashboard`
- [ ] `runCreateDashboard` dispatches to standard vs. preconfigured based on `--preconfigured` flag
- [ ] Mutual exclusivity enforced: `--preconfigured` requires `--from-file` and rejects either inline `--name` or `--description`
- [ ] `internal/cli/dashboard_extras_test.go` created with unit tests
- [ ] `get dashboard config <id>` returns widget layout definitions with table summary
- [ ] Table output for config shows WIDGET ID, WIDGET NAME, TYPE, POSITION, SIZE
- [ ] `formatPosition` and `formatSize` helpers produce compact display strings
- [ ] Existing `get dashboard` and `create dashboard` behavior is unchanged when new flags are not used
- [ ] Shared helpers used: `parseResourceID`, `readRequestFromFile`, `formatStringPtr`, `formatInt32Ptr`, `entryCreatedHeaders`/`entryCreatedRows`, `withPrintedErrors`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
