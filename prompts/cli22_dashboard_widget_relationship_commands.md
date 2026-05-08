# PR 22: Dashboard-Widget Relationship Commands

## Goal

Implement the project-scoped `assign widget` and `unassign widget` commands that manage the dashboard↔widget relationship. Adding a widget to a dashboard requires a complex payload (widget position, size, options); removing is a simple ID-to-ID operation.

## Context

Dashboard-widget relationship commands map to `DashboardScope` on the client (`client.Project(name).Dashboards()`). Both operations are **project-scoped**, requiring project resolution via `--project`, `RP_PROJECT`, or the profile's `default-project`.

These are association commands, not base-resource CRUD. They mutate the relationship between an existing dashboard and an existing widget. The widget must already exist (created via `rp create widget`) before it can be assigned to a dashboard.

**Prerequisite PRs and dependencies:**
- PR 2 (verb group scaffolding: `assign` and `unassign` parent commands)
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 6 (widget CRUD — widgets must exist before they can be assigned to dashboards)
- PR 7 (shared helpers: `readRequestFromFile`, `parseResourceID`, `messageResultHeaders`/`operationResultRows`)
- PR 8 (dashboard CRUD — dashboards must exist before widgets can be assigned to them)
- PR 17 (frozen UX: §2 dashboard↔widget relationship commands)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `AddWidget` | `ctx`, `dashboardID int64`, `request types.AddWidgetRq` | `(*types.OperationCompletionRS, error)` |
| `RemoveWidget` | `ctx`, `dashboardID int64`, `widgetID int64` | `(*types.OperationCompletionRS, error)` |

### Key types

```go
type AddWidgetRq struct {
    AddWidget WidgetObjectModel `json:"addWidget"`
}

type WidgetObjectModel struct {
    WidgetId       int64                   `json:"widgetId"`
    WidgetName     *string                 `json:"widgetName,omitempty"`
    WidgetOptions  *map[string]interface{} `json:"widgetOptions,omitempty"`
    WidgetPosition *Position               `json:"widgetPosition,omitempty"`
    WidgetSize     *Size                   `json:"widgetSize,omitempty"`
    WidgetType     *string                 `json:"widgetType,omitempty"`
}

type Position struct {
    PositionX *int32 `json:"positionX,omitempty"`
    PositionY *int32 `json:"positionY,omitempty"`
}

type Size struct {
    Height *int32 `json:"height,omitempty"`
    Width  *int32 `json:"width,omitempty"`
}

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

## Scope

Create `internal/cli/dashboard_widget.go` and `internal/cli/dashboard_widget_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp assign widget --dashboard <dashboard-id> --file <path>` | Add a widget to a dashboard |
| `rp unassign widget <widget-id> --dashboard <dashboard-id>` | Remove a widget from a dashboard |

Both commands are project-scoped (`--project` / `-p`).

## Implementation

### 1. Create `internal/cli/dashboard_widget.go`

#### `newAssignWidgetCmd()`

```go
func newAssignWidgetCmd() *cobra.Command
```

- `Use`: `"widget"`
- `Short`: `"Add a widget to a dashboard"`
- `Long`: Include examples showing `--dashboard` and `--file` usage. Show a sample JSON file with widget position and size. Explain that the widget must already exist (created via `rp create widget`).
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runAssignWidget)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--dashboard` (int64, required): the dashboard ID to add the widget to
  - `--file` / `-f` (string, required): path to JSON file containing `AddWidgetRq`

The run function:
1. Validate `--dashboard` is non-zero
2. Validate `--file` is non-empty
3. Read and unmarshal using `readRequestFromFile[types.AddWidgetRq](path)`
4. Resolve connection, project, and client
5. Call `scope.AddWidget(ctx, dashboardID, request)`
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

**Sample JSON file for help text:**

```json
{
  "addWidget": {
    "widgetId": 42,
    "widgetName": "Launch Statistics",
    "widgetPosition": {
      "positionX": 0,
      "positionY": 0
    },
    "widgetSize": {
      "width": 6,
      "height": 4
    }
  }
}
```

#### `newUnassignWidgetCmd()`

```go
func newUnassignWidgetCmd() *cobra.Command
```

- `Use`: `"widget <widget-id>"`
- `Short`: `"Remove a widget from a dashboard"`
- `Long`: Include examples showing the positional widget ID and `--dashboard` flag.
- `Args`: `cobra.ExactArgs(1)` (the numeric widget ID)
- `RunE`: `withPrintedErrors(runUnassignWidget)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--dashboard` (int64, required): the dashboard ID to remove the widget from

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "widget")`
2. Validate `--dashboard` is non-zero
3. Resolve connection, project, and client
4. Call `scope.RemoveWidget(ctx, dashboardID, widgetID)`
5. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// assign
assign.AddCommand(newAssignWidgetCmd())

// unassign
unassign.AddCommand(newUnassignWidgetCmd())
```

**Note:** These add to the same `assign` and `unassign` parents that PR 21 also populates. If PRs 21 and 22 are being developed in parallel worktrees, expect a trivial rebase on `commands.go` when merging.

### 3. Create `internal/cli/dashboard_widget_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewAssignWidgetCmd_Flags`** — Verify `--dashboard`, `--file` / `-f`, and `--project` / `-p` flags are registered.

2. **`TestNewAssignWidgetCmd_NoArgs`** — Verify the command accepts no positional arguments.

3. **`TestNewUnassignWidgetCmd_Flags`** — Verify `--dashboard` and `--project` / `-p` flags are registered.

4. **`TestNewUnassignWidgetCmd_Args`** — Verify the command accepts exactly 1 positional argument.

5. **`TestReadAddWidgetRq_ValidJSON`** — Write a temp JSON file with a valid `AddWidgetRq`, verify deserialization. Check that `WidgetId`, `WidgetPosition`, and `WidgetSize` are populated correctly.

6. **`TestReadAddWidgetRq_MinimalJSON`** — Write a JSON file with only `{"addWidget": {"widgetId": 1}}`, verify it deserializes without error (other fields nil).

7. **`TestReadAddWidgetRq_InvalidJSON`** — Write invalid JSON, verify parsing error.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **`assign` / `unassign` verbs, not `add` / `remove`.** Per PR 17 decision: all relationship commands in the CLI use `assign` / `unassign` for consistency (group↔project, group↔user, project↔user, dashboard↔widget).
- **`--file` for assign.** Per PR 17's frozen UX, `assign widget` uses `--file` because the JSON file is the command's primary input mode. The entire `AddWidgetRq` structure lives in the file, including the top-level `"addWidget"` key.
- **Positional widget ID for unassign.** Removing a widget from a dashboard is a simple two-ID operation. The widget ID is the "thing being removed" (positional arg). The dashboard ID identifies "where it's being removed from" (flag). This parallels `unassign project <project-id> --group <group-id>` from PR 21.
- **`--dashboard` flag (not `-d`).** Single-letter shorthands are reserved for commonly-used global flags. `-d` could be confused with `--debug` or other flags. The full `--dashboard` name is explicit.
- **Project-scoped.** Both commands need project resolution because dashboards and widgets exist within projects. This is different from PR 21's instance-scoped assignments.
- **Reuse shared helpers.** Use `parseResourceID`, `readRequestFromFile`, `messageResultHeaders`/`operationResultRows`, `withPrintedErrors`.
- **Route all output through `printer.Print`.** Use `printer.Print(result, messageResultHeaders(), operationResultRows(result))`. This ensures JSON, Go-template, and table modes work correctly.
- **Use client error predicates.** Handle API errors with predicate functions (`IsNotFound`, `IsUnauthorized`, etc.) rather than type assertions.
- **Command registration follows PR 3's centralized command-tree layout.** Shared mutation/result helpers assume PR 7's naming conventions.

## What NOT to do

- **Do not implement `UpdateWidgets` (widget repositioning within a dashboard).** That is a separate concern from adding/removing. The `UpdateDashboardRQ.UpdateWidgets` field is explicitly deferred from the dashboard CRUD PR (PR 8). If repositioning is needed later, it would be a separate command or integrated into `rp update dashboard`.
- **Do not add widget listing within a dashboard.** Users can see dashboard widgets via `rp get dashboard <id> -o json` (the `Widgets` field in `DashboardResource`). A dedicated list command is not needed.
- **Do not validate widget existence before assigning.** The API will return an error if the widget ID is invalid. Client-side pre-validation adds complexity without value.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp assign --help` shows `widget` as a subcommand (alongside `project` and `user` from PR 21)
- `rp unassign --help` shows `widget` as a subcommand
- `rp assign widget --help` shows `--dashboard`, `--file`, and `--project` flags
- `rp unassign widget --help` shows `--dashboard` and `--project` flags, expects a widget ID arg
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/dashboard_widget.go` created with both command constructors and run functions
- [ ] `internal/cli/dashboard_widget_test.go` created with unit tests
- [ ] Commands registered in `commands.go` (assign widget, unassign widget)
- [ ] Both commands use `--project` / `-p` for project resolution
- [ ] `assign widget` uses `--file` / `-f` for the `AddWidgetRq` payload
- [ ] `assign widget` uses `--dashboard` (int64) to identify the target dashboard
- [ ] `unassign widget` takes widget ID as positional arg and `--dashboard` as flag
- [ ] Shared helpers used: `parseResourceID`, `readRequestFromFile`, `messageResultHeaders`/`operationResultRows`, `withPrintedErrors`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
