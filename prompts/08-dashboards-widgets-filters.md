# Prompt 08: Dashboards, Widgets, and Filters Scopes (Milestone 8)

## Goal

Implement CRUD scopes in `pkg/client` for dashboards, widgets, and user filters (saved filters). These are the building blocks for "reporting as code" workflows. CLI exposure is deferred to a follow-up prompt — this slice is client-only.

## Context

Dashboards, widgets, and filters are the three pillars of Report Portal's reporting UI. They're tightly interrelated:

- **User filters** define saved search criteria (e.g., "failed tests in the last 7 days")
- **Widgets** render visualizations based on filter results (charts, tables, statistics)
- **Dashboards** are collections of positioned widgets

Automating these enables teams to version-control their reporting configuration, provision dashboards programmatically, and replicate setups across projects.

## Generated API endpoints available

### Dashboard controller

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetAllDashboardsUsingGETWithResponse(ctx, projectName, GetAllDashboardsUsingGETParams)` | List dashboards with pagination | -> `IterableDashboardResource` |
| `GetDashboardUsingGETWithResponse(ctx, projectName, dashboardId int64)` | Get a single dashboard by ID | -> `DashboardResource` |
| `CreateDashboardUsingPOSTWithResponse(ctx, projectName, CreateDashboardRQ)` | Create a new dashboard | `CreateDashboardRQ` -> `EntryCreatedRS` |
| `UpdateDashboardUsingPUTWithResponse(ctx, projectName, dashboardId int64, UpdateDashboardRQ)` | Update a dashboard | `UpdateDashboardRQ` -> `OperationCompletionRS` |
| `DeleteDashboardUsingDELETEWithResponse(ctx, projectName, dashboardId int64)` | Delete a dashboard | -> `OperationCompletionRS` |
| `AddWidgetUsingPUTWithResponse(ctx, projectName, dashboardId int64, AddWidgetRq)` | Add a widget to a dashboard | `AddWidgetRq` -> `OperationCompletionRS` |
| `RemoveWidgetUsingDELETEWithResponse(ctx, projectName, dashboardId int64, widgetId int64)` | Remove a widget from a dashboard | -> `OperationCompletionRS` |

### Widget controller

| Generated method | What it does | Request/Response |
|---|---|---|
| `CreateWidgetUsingPOSTWithResponse(ctx, projectName, WidgetRQ)` | Create a new widget | `WidgetRQ` -> `EntryCreatedRS` |
| `GetWidgetUsingGETWithResponse(ctx, projectName, widgetId int64, GetWidgetUsingGETParams)` | Get a widget by ID (with content) | -> `WidgetResource` |
| `GetWidgetUsingGET1WithResponse(ctx, projectName, widgetId int64)` | Get a widget by ID (without content/params) | -> `WidgetResource` |
| `UpdateWidgetUsingPUTWithResponse(ctx, projectName, widgetId int64, WidgetRQ)` | Update a widget | `WidgetRQ` -> `OperationCompletionRS` |
| `GetWidgetNamesUsingGETWithResponse(ctx, projectName, GetWidgetNamesUsingGETParams)` | Get widget names (for autocomplete) | -> string list |

Note: There is no `DeleteWidget` endpoint in the generated API. Widgets are removed from dashboards via `RemoveWidgetUsingDELETE` on the dashboard controller.

### User filter controller

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetAllFiltersUsingGETWithResponse(ctx, projectName, GetAllFiltersUsingGETParams)` | List all filters (shared/own) with pagination | -> `IterableUserFilterResource` |
| `GetUserFiltersUsingGETWithResponse(ctx, projectName, GetUserFiltersUsingGETParams)` | List own filters with pagination | -> `IterableUserFilterResource` |
| `GetFilterUsingGETWithResponse(ctx, projectName, filterId int64)` | Get a single filter by ID | -> `UserFilterResource` |
| `CreateFilterUsingPOSTWithResponse(ctx, projectName, UpdateUserFilterRQ)` | Create a new filter | `UpdateUserFilterRQ` -> `EntryCreatedRS` |
| `UpdateUserFilterUsingPUTWithResponse(ctx, projectName, filterId int64, UpdateUserFilterRQ)` | Update a filter | `UpdateUserFilterRQ` -> `OperationCompletionRS` |
| `DeleteFilterUsingDELETEWithResponse(ctx, projectName, filterId int64)` | Delete a filter | -> `OperationCompletionRS` |
| `GetAllFiltersNamesUsingGETWithResponse(ctx, projectName, GetAllFiltersNamesUsingGETParams)` | Get filter names (for autocomplete) | -> string list or similar |

### Key types (from `pkg/types/types.gen.go`)

```go
type DashboardResource struct {
    Description *string              `json:"description,omitempty"`
    Id          int64                `json:"id"`
    Name        string               `json:"name"`
    Owner       *string              `json:"owner,omitempty"`
    Widgets     *[]WidgetObjectModel `json:"widgets,omitempty"`
}

type CreateDashboardRQ struct {
    Description *string `json:"description,omitempty"`
    Name        string  `json:"name"`
}

type UpdateDashboardRQ struct {
    Description *string              `json:"description,omitempty"`
    Name        *string              `json:"name,omitempty"`
    Widgets     *[]WidgetObjectModel `json:"widgets,omitempty"`
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

type WidgetRQ struct {
    ContentParameters *ContentParameters `json:"contentParameters,omitempty"`
    Description       *string            `json:"description,omitempty"`
    FilterIds         *[]int64           `json:"filterIds,omitempty"`
    Name              string             `json:"name"`
    WidgetType        WidgetRQWidgetType `json:"widgetType"`
}

type UserFilterResource struct {
    Conditions  []UserFilterCondition `json:"conditions"`
    Description *string               `json:"description,omitempty"`
    Id          int64                 `json:"id"`
    Name        string                `json:"name"`
    Orders      []Order               `json:"orders"`
    Owner       string                `json:"owner"`
    Type        string                `json:"type"`
}

type UpdateUserFilterRQ struct {
    Conditions  *[]UserFilterCondition `json:"conditions,omitempty"`
    Description *string                `json:"description,omitempty"`
    Name        *string                `json:"name,omitempty"`
    Orders      *[]Order               `json:"orders,omitempty"`
    Share       *bool                  `json:"share,omitempty"`
    Type        *string                `json:"type,omitempty"`
}

type UserFilterCondition struct {
    Condition      *string `json:"condition,omitempty"`
    FilteringField *string `json:"filteringField,omitempty"`
    Value          *string `json:"value,omitempty"`
}

type AddWidgetRq struct {
    AddWidget *WidgetObjectModel `json:"addWidget,omitempty"`
}

type WidgetObjectModel struct {
    WidgetId       *int64                  `json:"widgetId,omitempty"`
    WidgetName     *string                 `json:"widgetName,omitempty"`
    WidgetOptions  *map[string]interface{} `json:"widgetOptions,omitempty"`
    WidgetPosition *Position               `json:"widgetPosition,omitempty"`
    WidgetSize     *Size                   `json:"widgetSize,omitempty"`
    WidgetType     *string                 `json:"widgetType,omitempty"`
}
```

## Client requirements

### DashboardScope

Create `pkg/client/dashboard.go` with a `DashboardScope` struct:

- **Project-scoped**: accessible via `client.Project(name).Dashboards()`
- Add a `Dashboards()` method to `ProjectScope` in `pkg/client/project.go`
- Struct holds `project *ProjectScope`

Methods to implement:

1. **`List(ctx, ...ListDashboardsOption) (*types.IterableDashboardResource, error)`** — list dashboards with pagination
2. **`GetByID(ctx, id int64) (*types.DashboardResource, error)`** — get a single dashboard
3. **`Create(ctx, request CreateDashboardRQ) (*types.EntryCreatedRS, error)`** — create a dashboard
4. **`Update(ctx, id int64, request UpdateDashboardRQ) (*types.OperationCompletionRS, error)`** — update a dashboard
5. **`Delete(ctx, id int64) (*types.OperationCompletionRS, error)`** — delete a dashboard
6. **`AddWidget(ctx, dashboardID int64, request AddWidgetRq) (*types.OperationCompletionRS, error)`** — add a widget to a dashboard
7. **`RemoveWidget(ctx, dashboardID int64, widgetID int64) (*types.OperationCompletionRS, error)`** — remove a widget from a dashboard

Functional options for `List`:

- `WithDashboardPageNumber(n int32)`
- `WithDashboardPageSize(size int32)`
- `WithDashboardSort(fields ...string)`

### WidgetScope

Create `pkg/client/widget.go` with a `WidgetScope` struct:

- **Project-scoped**: accessible via `client.Project(name).Widgets()`
- Add a `Widgets()` method to `ProjectScope` in `pkg/client/project.go`

Methods to implement:

1. **`GetByID(ctx, id int64) (*types.WidgetResource, error)`** — get a widget by ID (use the `GetWidgetUsingGET1` endpoint which returns the widget without extra params)
2. **`Create(ctx, request WidgetRQ) (*types.EntryCreatedRS, error)`** — create a widget
3. **`Update(ctx, id int64, request WidgetRQ) (*types.OperationCompletionRS, error)`** — update a widget
4. **`GetNames(ctx, filter string) ([]string, error)`** — list widget names (autocomplete)

Note: There is no delete-widget endpoint. Widgets are removed from dashboards via `DashboardScope.RemoveWidget`.

### UserFilterScope

Create `pkg/client/filter.go` with a `UserFilterScope` struct:

- **Project-scoped**: accessible via `client.Project(name).Filters()`
- Add a `Filters()` method to `ProjectScope` in `pkg/client/project.go`

Methods to implement:

1. **`List(ctx, ...ListFiltersOption) (*types.IterableUserFilterResource, error)`** — list filters with pagination
2. **`GetByID(ctx, id int64) (*types.UserFilterResource, error)`** — get a single filter
3. **`Create(ctx, request UpdateUserFilterRQ) (*types.EntryCreatedRS, error)`** — create a filter (note: the API uses `UpdateUserFilterRQ` as the create request type)
4. **`Update(ctx, id int64, request UpdateUserFilterRQ) (*types.OperationCompletionRS, error)`** — update a filter
5. **`Delete(ctx, id int64) (*types.OperationCompletionRS, error)`** — delete a filter
6. **`GetNames(ctx, filter string) ([]string, error)`** — list filter names (autocomplete)

Functional options for `List`:

- `WithFilterPageNumber(n int32)`
- `WithFilterPageSize(size int32)`
- `WithFilterSort(fields ...string)`

## Client tests

### `pkg/client/dashboard_test.go` (new, `package client_test`)

- **List**: success (200 with pagination), unauthorized (401)
- **GetByID**: success (200), not found (404)
- **Create**: success (201), bad request (400 — missing name)
- **Update**: success (200), not found (404)
- **Delete**: success (200), not found (404)
- **AddWidget**: success (200), not found (404 — dashboard or widget not found)
- **RemoveWidget**: success (200), not found (404)

### `pkg/client/widget_test.go` (new, `package client_test`)

- **GetByID**: success (200), not found (404)
- **Create**: success (201), bad request (400)
- **Update**: success (200), not found (404)
- **GetNames**: success (200), empty results

### `pkg/client/filter_test.go` (new, `package client_test`)

- **List**: success (200 with pagination), unauthorized (401)
- **GetByID**: success (200), not found (404)
- **Create**: success (201), bad request (400)
- **Update**: success (200), not found (404)
- **Delete**: success (200), not found (404)
- **GetNames**: success (200), empty results

### `export_test.go`

Add test-only constructors for `DashboardScope`, `WidgetScope`, and `UserFilterScope` in `pkg/client/export_test.go`.

## Integration tests

### `test/integration/client_dashboard_test.go` (new)

- Use `Label("client", "dashboards")`
- **CRUD lifecycle**: create a dashboard -> get by ID -> update name/description -> verify change -> delete -> verify `IsNotFound`
- **List**: verify the created dashboard appears in the list
- **Widget management**: create a widget, add it to a dashboard, verify it appears, remove it

### `test/integration/client_widget_test.go` (new)

- Use `Label("client", "widgets")`
- **Create + Get**: create a widget (needs a filter first), get by ID, verify fields
- **Update**: update widget name, verify change
- **GetNames**: verify the created widget name appears

### `test/integration/client_filter_test.go` (new)

- Use `Label("client", "filters")`
- **CRUD lifecycle**: create a filter -> get by ID -> update -> verify -> delete -> verify `IsNotFound`
- **List**: verify the created filter appears in the list

### Seeding notes

- Filters need to be created first since widgets reference filter IDs
- Use simple filter conditions (e.g., filter launches by name)
- Widgets need a widget type — use a simple type like `statisticTrend` or `launchStatistics`
- Create a dedicated project or use the existing test project
- Clean up created resources in `AfterAll` blocks

## Design considerations

- **No CLI in this PR**: CLI commands (`rp get dashboard`, `rp get widget`, `rp get filter`) are a follow-up. The roadmap notes that create/update may need `--from-file` workflows, which is a design decision for the CLI prompt.
- **Widget lifecycle**: Widgets don't have a standalone delete endpoint. They exist as long as they're referenced by at least one dashboard. This is a Report Portal design choice.
- **Filter creation uses UpdateUserFilterRQ**: The API reuses the update request type for creation. This is reflected in the generated client. The wrapper method should still be named `Create` for clarity.
- **ContentParameters**: Widget creation requires `ContentParameters` which defines what data the widget displays. For integration tests, use simple configurations. The `ContentParameters` type has `ContentFields`, `ItemsCount`, and `WidgetOptions`.

## Definition of done

- [ ] `pkg/client/dashboard.go` with `DashboardScope` (List, GetByID, Create, Update, Delete, AddWidget, RemoveWidget)
- [ ] `pkg/client/widget.go` with `WidgetScope` (GetByID, Create, Update, GetNames)
- [ ] `pkg/client/filter.go` with `UserFilterScope` (List, GetByID, Create, Update, Delete, GetNames)
- [ ] `ProjectScope.Dashboards()`, `.Widgets()`, `.Filters()` methods added
- [ ] Unit tests for all new scopes
- [ ] Integration tests for CRUD lifecycles
- [ ] `code-verifier` subagent says your changes are good to go
