# PR 6: Pilot `widget` Commands

## Goal

Implement the `widget` command family — get-by-ID, get-data (as a subcommand), create, and update — to lock in the subcommand-over-flag pattern for resources that expose conceptually different read operations. This PR establishes how the CLI handles the case where a resource has no list operation and where computed data is accessed via a nested subcommand.

## Context

The `widget` resource maps to `WidgetScope` on the client (`client.Project(name).Widgets()`). It is project-scoped. The critical design pattern this pilot locks in:

- `rp get widget <id>` — returns the **widget resource definition** (configuration)
- `rp get widget data <id>` — returns the **computed/rendered widget data** (runtime output)

These are conceptually different objects from the same underlying resource, accessed via two different API calls (`GetByID` vs `GetData`). Rather than using a flag (e.g., `--data`), the CLI uses a nested subcommand to make the distinction explicit.

**Important:** `WidgetScope` has **no `List` or `Delete` method**. When a user runs `rp get widget` without an ID, the command should produce a friendly error explaining that listing is not supported and an ID is required.

**Prerequisite:** PRs 1–3 must land first. PRs 4 and 5 (filter and group pilots) should land first so this PR can reuse patterns, though it is not a hard code dependency.

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `GetByID` | `ctx`, `id int64` | `(*types.WidgetResource, error)` |
| `Create` | `ctx`, `request types.WidgetRQ` | `(*types.EntryCreatedRS, error)` |
| `Update` | `ctx`, `id int64`, `request types.WidgetRQ` | `(*types.OperationCompletionRS, error)` |
| `GetData` | `ctx`, `id int64`, `options ...GetWidgetDataOption` | `(*types.WidgetResource, error)` |

GetData options: `WithWidgetAttributes(...string)`.

### Key types

```go
type WidgetRQ struct {
    ContentParameters *ContentParameters `json:"contentParameters,omitempty"`
    Description       *string            `json:"description,omitempty"`
    FilterIds         *[]int64           `json:"filterIds,omitempty"`
    Name              string             `json:"name"`
    WidgetType        WidgetRQWidgetType `json:"widgetType"`
}

type ContentParameters struct {
    ContentFields *[]string               `json:"contentFields,omitempty"`
    ItemsCount    *int32                  `json:"itemsCount,omitempty"`
    WidgetOptions *map[string]interface{} `json:"widgetOptions,omitempty"`
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

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}
```

`WidgetRQWidgetType` is a string enum with many values (e.g., `"launchStatistics"`, `"overallStatistics"`, `"casesTrend"`, etc.).

## Scope

Create `internal/cli/widget.go` and `internal/cli/widget_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get widget <id>` | Get a widget resource definition by numeric ID |
| `rp get widget` (no args) | Friendly error: listing not supported, ID required |
| `rp get widget data <id>` | Get computed widget data by numeric ID |
| `rp create widget --from-file <path>` | Create a widget from a JSON file |
| `rp update widget <id> --from-file <path>` | Update a widget from a JSON file |

Note: There is no `rp delete widget` because the client has no `Delete` method for widgets.

## Implementation

### 1. Create `internal/cli/widget.go`

This file contains the command constructors and run functions.

#### `newGetWidgetCmd()`

```go
func newGetWidgetCmd() *cobra.Command
```

- `Use`: `"widget [id]"`
- `Short`: `"Get a widget definition"`
- `Long`: Include examples for get-by-ID. Mention that listing is not supported and note the `data` subcommand for computed output.
- `Args`: `cobra.MaximumNArgs(1)`
- `RunE`: `withPrintedErrors(runGetWidget)`
- Flags:
  - `--project` / `-p` (string): project name override
- Subcommands:
  - Register `newGetWidgetDataCmd()` via `cmd.AddCommand(newGetWidgetDataCmd())`

The run function:
1. If `len(args) == 0`, return a friendly error: `"listing widgets is not supported; provide a numeric widget ID (e.g., rp get widget 42)"`
2. Parse `args[0]` as int64
3. Resolve connection, project, and client
4. Call `scope.GetByID(ctx, id)`
5. Print the `WidgetResource` result

**Cobra interaction note:** When the `widget` command has both a `RunE` and a subcommand (`data`), Cobra will:
- Route `rp get widget data 42` to the `data` subcommand
- Route `rp get widget 42` to the parent's `RunE` (since `"42"` is not a known subcommand name)
- Route `rp get widget` to the parent's `RunE` with no args

This is valid Cobra behavior and works correctly.

#### `newGetWidgetDataCmd()`

```go
func newGetWidgetDataCmd() *cobra.Command
```

- `Use`: `"data <id>"`
- `Short`: `"Get computed widget data"`
- `Long`: Include examples showing basic usage and the `--attribute` flag. Explain this returns runtime-computed data, not the widget definition.
- `Args`: `cobra.ExactArgs(1)` (the numeric widget ID)
- `RunE`: `withPrintedErrors(runGetWidgetData)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--attribute` (string array, repeatable): attributes to pass to the widget data computation

The run function:
1. Parse `args[0]` as int64
2. Resolve connection, project, and client
3. Build options: if `--attribute` flags are provided, use `client.WithWidgetAttributes(attrs...)`
4. Call `scope.GetData(ctx, id, options...)`
5. Print the result

**Output note for `data`:** Both `GetByID` and `GetData` return a `*types.WidgetResource`, so reuse the same `widgetRow` formatter for both. The `Content` field (the computed data payload) is only meaningfully visible in JSON output (`-o json`) — the table shows the stable resource metadata (ID, NAME, WIDGET TYPE, OWNER, DESCRIPTION). This is by design; computed data inspection is the primary use case for JSON output.

#### `newCreateWidgetCmd()`

```go
func newCreateWidgetCmd() *cobra.Command
```

- `Use`: `"widget"`
- `Short`: `"Create a widget"`
- `Long`: Include examples showing `--from-file` usage. Note widget definitions are complex JSON.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateWidget)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--from-file` / `-f` (string, required): path to JSON file containing `WidgetRQ`

The run function:
1. Resolve `--from-file`, read the file, unmarshal into `types.WidgetRQ`
2. Resolve connection, project, and client
3. Call `scope.Create(ctx, request)`
4. Print the `EntryCreatedRS` result (table shows ID column)

#### `newUpdateWidgetCmd()`

```go
func newUpdateWidgetCmd() *cobra.Command
```

- `Use`: `"widget <id>"`
- `Short`: `"Update a widget"`
- `Long`: Include examples showing `--from-file` with an ID argument
- `Args`: `cobra.ExactArgs(1)` (the numeric widget ID)
- `RunE`: `withPrintedErrors(runUpdateWidget)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--from-file` / `-f` (string, required): path to JSON file containing `WidgetRQ`

The run function:
1. Parse `args[0]` as int64
2. Resolve `--from-file`, read the file, unmarshal into `types.WidgetRQ`
3. Resolve connection, project, and client
4. Call `scope.Update(ctx, id, request)`
5. Print the `OperationCompletionRS` result using `operationResultHeaders()` and `operationResultRows()`

#### Table formatting

For `GetByID` (the widget definition):

```go
func widgetHeaders() []string {
    return []string{"ID", "NAME", "WIDGET TYPE", "OWNER", "DESCRIPTION"}
}

func widgetRow(widget *types.WidgetResource) ([]string, []string) {
    return widgetHeaders(), []string{
        strconv.FormatInt(widget.Id, 10),
        widget.Name,
        string(widget.WidgetType),
        formatStringPtr(widget.Owner),
        formatStringPtr(widget.Description),
    }
}
```

For `GetData` (computed data), reuse `widgetRow` — both `GetByID` and `GetData` return `*types.WidgetResource`. The table columns are the stable metadata; the `Content` field (the actual computed data) only appears in JSON output. Users who need the computed payload should use `-o json`.

#### `--from-file` helper

```go
func readWidgetRequestFromFile(path string) (types.WidgetRQ, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return types.WidgetRQ{}, fmt.Errorf("reading file %s: %w", path, err)
    }

    var request types.WidgetRQ
    if err := json.Unmarshal(data, &request); err != nil {
        return types.WidgetRQ{}, fmt.Errorf("parsing JSON from %s: %w", path, err)
    }

    return request, nil
}
```

#### ID parsing helper

```go
func parseWidgetID(idStr string) (int64, error) {
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid widget ID %q: must be a numeric ID", idStr)
    }

    return id, nil
}
```

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// get
get.AddCommand(newGetWidgetCmd())

// create
create.AddCommand(newCreateWidgetCmd())

// update
update.AddCommand(newUpdateWidgetCmd())
```

Note: No `delete` registration because there is no delete operation for widgets.

### 3. Create `internal/cli/widget_test.go`

Use the same testing pattern as the other pilots:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetWidgetCmd_HasDataSubcommand`** — Verify that `newGetWidgetCmd()` has a subcommand named `"data"`.

2. **`TestRunGetWidget_NoArgs_ReturnsError`** — Create the command, execute with no args, verify a friendly error about listing not being supported is returned.

3. **`TestParseWidgetID_Valid`** — Verify valid numeric string parsing.

4. **`TestParseWidgetID_Invalid`** — Verify non-numeric strings produce a descriptive error.

5. **`TestWidgetRow_Formatting`** — Verify `widgetRow` produces expected headers and values for a sample `WidgetResource`.

6. **`TestWidgetRow_NilFields`** — Verify `widgetRow` handles nil Owner and Description gracefully.

7. **`TestReadWidgetRequestFromFile_ValidJSON`** — Write a temp JSON file with a valid `WidgetRQ`, verify deserialization.

8. **`TestReadWidgetRequestFromFile_MissingFile`** — Call with nonexistent path, verify error.

9. **`TestReadWidgetRequestFromFile_InvalidJSON`** — Write invalid JSON, verify parsing error.

10. **`TestNewGetWidgetDataCmd_Flags`** — Verify the `data` subcommand has `--project` and `--attribute` flags registered.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Subcommand for data, not a flag.** Per the CLI plan's decision rule: "If an operation returns a conceptually different object or computed view from the base resource, prefer a subcommand." Widget definition vs. widget data are conceptually different: one is configuration, the other is runtime output.
- **No list operation.** The API does not support listing widgets independently — they are accessed via dashboards. The CLI explicitly communicates this limitation via a friendly error message.
- **No delete operation.** The client has no `Delete` method for widgets. Do not add a `rp delete widget` command.
- **`--from-file` for create/update.** Widget definitions are complex nested JSON (`ContentParameters`, `FilterIds`, `WidgetType`). This matches the `filter` pilot pattern.
- **`--attribute` flag on `data` subcommand.** This is the only runtime parameter that affects widget data computation. It maps to `client.WithWidgetAttributes(...)`.
- **Cobra command-with-subcommand pattern.** A Cobra command can have both `RunE` and child commands. When `rp get widget 42` is invoked, Cobra checks if "42" is a registered subcommand name. Since it isn't, it dispatches to the parent's `RunE`. When `rp get widget data 42` is invoked, "data" matches and Cobra dispatches to the child.

### 4. Update `CLI_IMPLEMENTATION_PLAN.md`

This PR settles the previously-open UX decision for widget data access. Update **all** occurrences of the stale `widget-data` form across the plan:

1. **Top-level conventions section** (the "Subcommands over mode flags" bullet): Change the example from `rp get widget-data <id>` to `rp get widget data <id>`.

2. **Decision rules section** (rule 1, "Different object → subcommand"): Change the example from `rp get widget-data <id>` to `rp get widget data <id>`.

3. **Widget subsection** ("Widget — resource vs. computed data split"):
   - Change the status from "Needs UX decision" to "Settled (PR 6)"
   - Replace both `rp get widget-data <id>` references with `rp get widget data <id>`
   - Note that `data` is a subcommand of the `widget` command under `get`, not a hyphenated sibling resource name

All three locations currently use the placeholder `widget-data` form. Updating only the widget subsection would leave the conventions and decision rules contradicting the implementation.

This prevents future prompts from drifting back to the `widget-data` form that the plan previously used as a placeholder.

## What NOT to do

- **Do not implement `GetNames`.** That is a utility for autocomplete, not a user-facing command in this pilot.
- **Do not add a `rp delete widget` command.** The client has no `Delete` method.
- **Do not add a `rp get widget` list mode** that attempts to work around the missing List API. Return a clear error instead.
- **Do not extract shared `--from-file` helpers.** Both this PR and the filter PR (PR 4) define local helpers. Extraction into a shared utility happens in PR 7.
- **Do not implement dashboard-widget association commands.** Those are PR 22.
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` now shows `widget` as a subcommand
- `rp create --help` now shows `widget` as a subcommand
- `rp update --help` now shows `widget` as a subcommand
- `rp delete --help` does NOT show `widget`
- `rp get widget --help` shows the `data` subcommand and mentions listing is not supported
- `rp get widget data --help` shows `--attribute` and `--project` flags
- `rp create widget --help` shows `--from-file` flag

## Definition of done

- [ ] `internal/cli/widget.go` created with all command constructors and run functions
- [ ] `internal/cli/widget_test.go` created with unit tests
- [ ] Commands registered in `commands.go` (get, create, update — no delete)
- [ ] `rp get widget <id>` returns the widget resource definition
- [ ] `rp get widget data <id>` returns computed widget data via a nested subcommand
- [ ] `rp get widget` (no args) returns a friendly error about listing not being supported
- [ ] Create and update use `--from-file` / `-f` for input
- [ ] `--attribute` flag available on the `data` subcommand
- [ ] Table output shows ID, NAME, WIDGET TYPE, OWNER, DESCRIPTION with uppercase headers (same for both `get widget` and `get widget data`)
- [ ] JSON output prints the full `WidgetResource` object (includes `Content` for data mode)
- [ ] `CLI_IMPLEMENTATION_PLAN.md` updated to mark the widget UX decision as settled and use the `rp get widget data <id>` form
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
