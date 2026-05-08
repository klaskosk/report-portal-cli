# PR 4: Pilot `widget` Commands

## Goal

Implement the third pilot resource — `rp widget {get,create,update}` plus the `rp widget data get` sub-resource command. This PR establishes the sub-resource pattern (a nested noun under a parent noun) and confirms that the `--from-file` mutation flow (proven in PR 2) generalizes to a second resource. Widgets have no list or delete in the API, so this is intentionally not a full CRUD family.

## Context

**Prerequisites:** PR 1 (noun-first refactor), PR 2 (filter pilot), and PR 3 (group pilot) have landed. The `--from-file` pattern from the filter pilot should be reused (duplicated for now — Phase 3 will extract shared helpers).

### API surface

The widget scope is accessed via `client.Project(projectName).Widgets()` and provides:

| Method | Signature |
|--------|-----------|
| `GetByID` | `GetByID(ctx, id int64) (*types.WidgetResource, error)` |
| `Create` | `Create(ctx, request types.WidgetRQ) (*types.EntryCreatedRS, error)` |
| `Update` | `Update(ctx, id int64, request types.WidgetRQ) (*types.OperationCompletionRS, error)` |
| `GetData` | `GetData(ctx, id int64, options ...GetWidgetDataOption) (*types.WidgetResource, error)` |

GetData options: `WithWidgetAttributes(attributes ...string)`.

**Not available:** There is no `List` (returning widget resources) and no `Delete` in the widget scope. Do not invent them.

### Key types

```go
type WidgetRQ struct {
    ContentParameters *ContentParameters `json:"contentParameters,omitempty"`
    Description       *string            `json:"description,omitempty"`
    FilterIds         *[]int64           `json:"filterIds,omitempty"`
    Name              string             `json:"name"`
    WidgetType        WidgetRQWidgetType `json:"widgetType"`
}

type WidgetRQWidgetType string  // "launchStatistics", "overallStatistics", etc. (24 constants)

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

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

**Key difference between `GetByID` and `GetData`:** Both return `*types.WidgetResource`. `GetByID` returns the widget definition (metadata, filter references, content parameters). `GetData` returns the same structure but with the `Content` field populated with computed/rendered data. In table mode, `Content` is a `map[string]interface{}` that can contain anything — it is not suitable for tabular display.

### Files to create

- `internal/cli/widget.go` — all widget commands and helpers
- `internal/cli/widget_test.go` — unit tests

### Files to modify

- `internal/cli/root.go` — add `cmd.AddCommand(newWidgetCmd())` to register the widget parent command

## Detailed requirements

### 1. Command tree structure

The widget command tree has a nested `data` sub-resource:

```
rp widget
├── get <id>           # widget definition
├── create             # create widget (--from-file)
├── update <id>        # update widget (--from-file)
└── data
    └── get <id>       # computed widget data
```

Implement this as:

```go
func newWidgetCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "widget",
        Short: "Manage widgets",
    }
    cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
    cmd.AddCommand(newWidgetGetCmd())
    cmd.AddCommand(newWidgetCreateCmd())
    cmd.AddCommand(newWidgetUpdateCmd())
    cmd.AddCommand(newWidgetDataCmd())
    return cmd
}

func newWidgetDataCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "data",
        Short: "Access computed widget data",
    }
    cmd.AddCommand(newWidgetDataGetCmd())
    return cmd
}
```

The `data` command is a grouping noun — it has no `RunE` of its own, only child commands. This establishes the sub-resource pattern that Phase 7 will reuse (e.g., `rp dashboard widget {add,remove}`, `rp group user {list,...}`).

Register `-p`/`--project` as a **persistent flag** on the `widget` parent so all subcommands (including nested `data get`) inherit it.

### 2. `rp widget get <id>`

**Command shape:** `rp widget get <id>`

- `Use: "get <id>"`, `Args: cobra.ExactArgs(1)`
- Parse the positional argument as `int64` (widgets use numeric IDs only). On failure: `invalid widget ID "<value>": must be a numeric ID`
- Call `scope.GetByID(ctx, id)`
- Print using `widgetRow` formatting

**Table columns:**

| Header | Source | Notes |
|--------|--------|-------|
| ID | `Id` | `strconv.FormatInt` |
| NAME | `Name` | Direct string |
| TYPE | `WidgetType` | Direct string (cast from `WidgetResourceWidgetType`) |
| OWNER | `Owner` | `formatStringPtr` |
| DESCRIPTION | `Description` | `formatStringPtr`, truncate to 50 chars with `...` suffix |
| FILTERS | `len(AppliedFilters)` | Count of applied filters. Nil-safe: treat nil as 0 |

Define `widgetHeaders() []string` and `widgetRow(*types.WidgetResource) ([]string, []string)`.

### 3. `rp widget data get <id>`

**Command shape:** `rp widget data get <id> [--attribute key ...]`

- `Use: "get <id>"`, `Args: cobra.ExactArgs(1)`
- Parse ID as `int64`
- `--attribute` (`StringArray`, optional, repeatable) — attribute keys to pass to the API

**Options mapping:**

If `--attribute` flags are provided, map them to `client.WithWidgetAttributes(attrs...)`.

**Flow:**

1. Parse widget ID
2. Read `--attribute` flags
3. Build options slice (only include `WithWidgetAttributes` if attributes are non-empty)
4. Call `scope.GetData(ctx, id, options...)`
5. Print result

**Table output for data get:**

The `Content` field is a `map[string]interface{}` with arbitrary structure — it cannot be meaningfully rendered as table rows. For table mode, display the widget metadata (same columns as `widget get`) with an additional column:

| Header | Source | Notes |
|--------|--------|-------|
| ID | `Id` | Same as widget get |
| NAME | `Name` | Same as widget get |
| TYPE | `WidgetType` | Same as widget get |
| CONTENT KEYS | `Content` | Sorted, comma-joined top-level keys of the content map. Nil content → `(none)`. This gives the user a summary without attempting to flatten arbitrary nested data. |

Define `widgetDataHeaders() []string` and `widgetDataRow(*types.WidgetResource) ([]string, []string)`.

In the command's `Long` help text, note that `-o json` (or `--json`) is recommended for inspecting the full computed content.

**JSON output:** Returns the full `WidgetResource` including the populated `Content` field — unchanged from the API response.

### 4. `rp widget create --from-file <path>`

**Command shape:** `rp widget create --from-file <path> [--name override] [--description override]`

- `Use: "create"`, `Args: cobra.NoArgs`
- `--from-file` (`string`, required) — path to JSON file containing a `WidgetRQ` payload. `-` reads from stdin.
- `--name` (`string`, optional) — overrides the `name` field from the file
- `--description` (`string`, optional) — overrides the `description` field from the file
- Validation: `--from-file` is required. Error: `--from-file is required for widget creation (widget definitions contain complex content parameters)`

**Flow:**

1. Read and parse JSON file into `types.WidgetRQ` (see section 6)
2. Apply flag overrides: if `--name` is changed, overwrite `request.Name`; if `--description` is changed, overwrite `request.Description`
3. Validate: `request.Name` must be non-empty after overrides. Error: `widget name is required (provide via --from-file or --name flag)`
4. Call `scope.Create(ctx, request)`
5. Print `EntryCreatedRS`: table with `ID` header showing created widget ID (same pattern as filter create in PR 2)

### 5. `rp widget update <id> --from-file <path>`

**Command shape:** `rp widget update <id> --from-file <path> [--name override] [--description override]`

- `Use: "update <id>"`, `Args: cobra.ExactArgs(1)`
- Parse ID from positional argument
- `--from-file` (`string`, required) — same as create
- `--name` and `--description` — same override behavior as create
- Validation: `--from-file` is required. Error: `--from-file is required for widget update (the API expects the complete widget definition)`

**Flow:**

1. Parse widget ID
2. Read and parse JSON file
3. Apply flag overrides
4. Call `scope.Update(ctx, id, request)`
5. Print `OperationCompletionRS` using the operation result helpers from `user.go`

### 6. `--from-file` implementation

Implement a file-reading helper within `widget.go`, duplicating the pattern from the filter pilot (PR 2). Phase 3 will extract a shared generic helper.

```go
func readWidgetRequestFromFile(path string) (types.WidgetRQ, error)
```

**Behavior (identical pattern to filter's `readFilterRequestFromFile`):**

- If `path == "-"`, read from `os.Stdin`
- Otherwise, read from `os.ReadFile(path)`
- Unmarshal JSON into `types.WidgetRQ` using `json.Unmarshal`
- **Do NOT** use `json.Decoder.DisallowUnknownFields()` — unknown fields are silently ignored. This enables round-trip: `rp widget get 42 -o json > w.json && rp widget create --from-file w.json`
- Return clear errors for: file not found, read errors, JSON parse errors

### 7. Help text and examples

Each subcommand should have a `Long` description with CLI examples:

```
rp widget get 42                           Get widget definition
rp widget get 42 -o json                   Get widget as JSON
rp widget data get 42                      Get computed widget data
rp widget data get 42 --json               Full computed data as JSON (recommended)
rp widget data get 42 --attribute key1     Get data with attribute filter
rp widget create --from-file widget.json   Create a widget from file
rp widget update 42 --from-file w.json     Update a widget from file
```

The `data get` help text should note that `-o json` / `--json` is recommended for inspecting the full computed content, since the table view shows only a summary.

### 8. Standard boilerplate

Follow the established patterns:

- **Printer creation:** `outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")` then `newPrinter(outputFmt)`
- **Connection resolution:** `resolveConnection(cmd)` → `resolveProject(cmd, conn)` → `newClient(conn, logger)` (project-scoped)
- **Error handling:** call `printer.PrintError(err)` before returning, wrap with `withPrintedErrors`
- **Logger:** `loggerFromCmd(cmd)` at top of each RunE

## Test requirements

Create `internal/cli/widget_test.go` in `package cli` (white-box tests).

### Required tests

1. **`widgetRow` formatting:** verify headers and row values for a `WidgetResource` with all fields populated, including nil `Description`, nil `Owner`, nil `AppliedFilters`

2. **`widgetDataRow` formatting:** verify the CONTENT KEYS column:
   - Non-nil content with keys `{"chart": ..., "summary": ...}` → `"chart, summary"` (sorted)
   - Nil content → `"(none)"`
   - Empty map → `"(none)"`

3. **`readWidgetRequestFromFile` happy path:** write a temporary JSON file with all `WidgetRQ` fields, verify deserialization

4. **`readWidgetRequestFromFile` ignores unknown fields:** write JSON with extra fields (`id`, `owner`, `content`), verify silently ignored

5. **`readWidgetRequestFromFile` invalid JSON:** verify clear error for malformed JSON

6. **`readWidgetRequestFromFile` missing file:** verify error for non-existent path

7. **Flag override logic:** verify `--name` and `--description` override file values, and unchanged flags preserve file values

8. **Widget ID parsing:** verify non-numeric IDs produce clear error

9. **`newWidgetCmd` command tree:** verify the parent has children `get`, `create`, `update`, `data`; verify `data` has child `get`

### Test patterns

- Use `testing.T` with `t.Parallel()` for all tests and subtests
- Use `testify/assert` and `testify/require`
- Use `t.TempDir()` for temporary file creation
- Table-driven tests for row formatting variants
- Follow naming: `TestWidgetRow_*`, `TestWidgetDataRow_*`, `TestReadWidgetRequestFromFile_*`, etc.

## Edge cases and constraints

- **No list, no delete:** The widget API does not have `List` (returning resources) or `Delete`. Do not create these commands. `GetNames` exists but returns only name strings — it is not suitable as a list command for the resource.
- **`GetByID` vs `GetData` return the same type:** Both return `*types.WidgetResource`. The difference is semantic — `GetData` populates the `Content` field with computed results. The CLI distinguishes them as separate commands (`widget get` vs `widget data get`) per the "sub-resources over mode flags" convention.
- **`Content` is opaque:** `Content` is `*map[string]interface{}`. In table mode, do not attempt to flatten or pretty-print it. Show only the top-level key names as a summary. In JSON mode, return the full structure unchanged.
- **Sub-resource command nesting:** The `data` noun is a `cobra.Command` with no `RunE` — it only groups its children. Running `rp widget data` with no subcommand should print the data subcommand's help text (Cobra's default behavior for group commands without a RunE).
- **`--from-file` duplication:** The file-reading helper in this PR duplicates the one from the filter pilot (PR 2). This is intentional. Phase 3 (PR 5) will extract a shared generic helper once the pattern is confirmed across two resources.
- **Round-trip with extra fields:** `rp widget get 42 -o json` returns `WidgetResource` which has fields not in `WidgetRQ` (e.g., `id`, `owner`, `content`, `appliedFilters`). Since `json.Unmarshal` ignores unknown fields by default, piping this JSON to `--from-file` for create/update works seamlessly.
- **Widget type validation:** Do not validate `widgetType` against the known constant list client-side. The API should enforce valid types. The 24 known types may change with API versions.
- **`--attribute` flag on `data get`:** The attributes are passed directly to `WithWidgetAttributes`. If no `--attribute` flags are provided, do not add the option (let the API use its defaults).
- **Project flag inheritance:** `-p`/`--project` is on the `widget` parent as a persistent flag. It must be accessible from `widget data get` (two levels deep). Cobra's persistent flag inheritance handles this, but verify in tests.

## Validation

Run before finalizing:

```
make lint
make vet
make test
```

## Definition of done

- [ ] `internal/cli/widget.go` implements `rp widget {get,create,update}` and `rp widget data get`
- [ ] `internal/cli/widget_test.go` covers row formatting, data row content keys, file reading, flag overrides, ID parsing, and command tree structure
- [ ] `internal/cli/root.go` registers `newWidgetCmd()`
- [ ] `-p`/`--project` is a persistent flag on the widget parent, inherited by all subcommands including nested `data get`
- [ ] `data` sub-resource establishes the nested-noun pattern for future sub-resources
- [ ] `--from-file` supports both file paths and `-` for stdin (same pattern as filter pilot)
- [ ] `--name` and `--description` override file values on create and update
- [ ] `--attribute` flag on `data get` passes through to the API
- [ ] Table output for `data get` shows content key summary, not raw content
- [ ] JSON output returns raw API types unchanged
- [ ] No `list` or `delete` commands (API doesn't support them)
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
