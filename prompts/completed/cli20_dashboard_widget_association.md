# PR 20: Dashboard Widget Association

## Goal

Add `rp dashboard widget {add,remove}` as a sub-resource of the existing dashboard command.

## Context

Phase 7. The dashboard command already has list/get/create/update/delete/config/preconfigured. The `DashboardScope` client already has `AddWidget` and `RemoveWidget` methods.

### Client API surface

**DashboardScope** (via `client.Project(name).Dashboards()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `AddWidget` | `AddWidget(ctx, dashboardID int64, types.AddWidgetRq)` | `*types.OperationCompletionRS` |
| `RemoveWidget` | `RemoveWidget(ctx, dashboardID int64, widgetID int64)` | `*types.OperationCompletionRS` |

### Relevant types

```go
type AddWidgetRq struct {
    WidgetId     int64              `json:"widgetId"`
    WidgetName   string             `json:"widgetName"`
    WidgetType   string             `json:"widgetType"`
    WidgetSize   *WidgetSize        `json:"widgetSize,omitempty"`
    WidgetPosition *WidgetPosition  `json:"widgetPosition,omitempty"`
}

type WidgetSize struct {
    Height int32 `json:"height"`
    Width  int32 `json:"width"`
}

type WidgetPosition struct {
    PositionX int32 `json:"positionX"`
    PositionY int32 `json:"positionY"`
}
```

Look up the exact types in `pkg/types/` as they may differ slightly.

## Implementation plan

### 1. Create `internal/cli/dashboard_widget.go`

#### `rp dashboard widget add <dashboard-id>`

- One positional arg: numeric dashboard ID
- Required flag: `--widget-id` (int64) — the widget to add
- Required flag: `--widget-name` (string)
- Required flag: `--widget-type` (string)
- Optional flags:
  - `--width` (int32) and `--height` (int32) — widget size
  - `--position-x` (int32) and `--position-y` (int32) — widget position
  - `--from-file` (string) — reads full `AddWidgetRq` from JSON, overrides flags
- Build `types.AddWidgetRq` from flags
- Call `scope.AddWidget(ctx, dashboardID, request)`
- Table/JSON output: `OperationCompletionRS` message

#### `rp dashboard widget remove <dashboard-id> <widget-id>`

- Two positional args: dashboard ID and widget ID (both numeric)
- No additional flags
- Call `scope.RemoveWidget(ctx, dashboardID, widgetID)`
- Table/JSON output: `OperationCompletionRS` message

### 2. Register in dashboard.go

Add to `newDashboardCmd()`:
```go
cmd.AddCommand(newDashboardWidgetCmd())
```

Where `newDashboardWidgetCmd()` is the parent for add/remove.

## Test requirements

### Unit tests (`internal/cli/dashboard_widget_test.go`)

- **Add:** dashboard ID positional arg required, `--widget-id`, `--widget-name`, `--widget-type` required
- **Add --from-file:** valid JSON loads correctly
- **Remove:** exactly 2 positional args required, both numeric
- **Command wiring:** verify dashboard has a `widget` subcommand with `add` and `remove`

## Edge cases

- **Project-scoped:** Dashboard widget operations inherit `-p`/`--project` from the dashboard parent
- **`--from-file` overrides flags:** When file is provided, flag values are ignored
- **Size/position optional:** Widget can be added without explicit size/position (API accepts nil)
- **Numeric IDs only:** Both dashboard and widget IDs are numeric, validated with `parseIDArg`

## Example commands

```bash
# Add a widget to a dashboard
$ rp dashboard widget add 5 --widget-id 10 --widget-name "Pass Rate" --widget-type "statisticTrend" -p my_project
MESSAGE
Widget has been added to dashboard successfully.

# Add with size and position
$ rp dashboard widget add 5 --widget-id 10 --widget-name "Pass Rate" --widget-type "statisticTrend" --width 6 --height 4 --position-x 0 --position-y 0 -p my_project
MESSAGE
Widget has been added to dashboard successfully.

# Remove a widget from a dashboard
$ rp dashboard widget remove 5 10 -p my_project
MESSAGE
Widget has been removed from dashboard successfully.
```

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [ ] `internal/cli/dashboard_widget.go` with add and remove commands
- [ ] Registered in `newDashboardCmd()`
- [ ] `--from-file` support on add
- [ ] Unit tests cover arg/flag validation and command wiring
- [ ] Existing dashboard subcommands unchanged
- [ ] `make lint` and `make test` pass
- [ ] This file moved to `prompts/completed/`
