# PR 4: Pilot `filter` Full CRUD

## Goal

Implement the complete `filter` command family — list, get-by-ID, create, update, and delete — as the first project-scoped CRUD pilot. This PR establishes the baseline patterns for project-scoped resources with pagination, single-get-by-ID, and `--from-file` mutation flow.

## Context

The `filter` resource maps to `UserFilterScope` on the client (`client.Project(name).Filters()`). It is project-scoped, meaning every command requires a resolved project via `--project`, `RP_PROJECT`, or the profile's `default-project`.

**Prerequisite:** PRs 1–3 must land first (parent command split, verb scaffolding, and centralized registration).

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `List` | `ctx`, `options ...ListFiltersOption` | `(*types.PageUserFilterResource, error)` |
| `GetByID` | `ctx`, `id int64` | `(*types.UserFilterResource, error)` |
| `Create` | `ctx`, `request types.UpdateUserFilterRQ` | `(*types.EntryCreatedRS, error)` |
| `Update` | `ctx`, `id int64`, `request types.UpdateUserFilterRQ` | `(*types.OperationCompletionRS, error)` |
| `Delete` | `ctx`, `id int64` | `(*types.OperationCompletionRS, error)` |

List options: `WithFilterPageNumber(int32)`, `WithFilterPageSize(int32)`, `WithFilterSort(...string)`.

### Key types

```go
type UpdateUserFilterRQ struct {
    Conditions  []UserFilterCondition  `json:"conditions"`
    Description *string                `json:"description,omitempty"`
    Name        string                 `json:"name"`
    Orders      []Order                `json:"orders"`
    Type        UpdateUserFilterRQType `json:"type"`
}

type UserFilterCondition struct {
    Condition      string `json:"condition"`
    FilteringField string `json:"filteringField"`
    Value          string `json:"value"`
}

type Order struct {
    IsAsc         bool   `json:"isAsc"`
    SortingColumn string `json:"sortingColumn"`
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

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}
```

`UpdateUserFilterRQType` is a string enum: `"launch"`, `"log"`, `"testitem"`.

## Scope

Create `internal/cli/filter.go` and `internal/cli/filter_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get filter` | List filters in the project (paginated) |
| `rp get filter <id>` | Get a single filter by numeric ID |
| `rp create filter --from-file <path>` | Create a filter from a JSON file |
| `rp update filter <id> --from-file <path>` | Update a filter from a JSON file |
| `rp delete filter <id>` | Delete a filter by numeric ID |

## Implementation

### 1. Create `internal/cli/filter.go`

This file contains all five command constructors and their run functions.

#### `newGetFilterCmd()`

```go
func newGetFilterCmd() *cobra.Command
```

- `Use`: `"filter [id]"`
- `Short`: `"List filters or get a specific filter"`
- `Long`: Include examples showing list, get-by-ID, project override, and output format
- `Args`: `cobra.MaximumNArgs(1)`
- `RunE`: `withPrintedErrors(runGetFilter)`
- Flags:
  - `--project` / `-p` (string): project name override
  - Standard list flags via `registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: nil})` (provides `--page-size`, `--page`, `--sort` with no default sort)

**Do NOT** use the bare `registerListFlags(cmd)` helper — it hardcodes a default sort of `startTime,desc` which is launch-specific. Filters have no `startTime` field. Use `registerListFlagsWithDefaults` with a nil sort default so `--sort` is empty unless the user explicitly provides one.

The run function follows the same dispatch pattern as `runGetLaunch`:
1. Resolve connection, project, and client
2. If `len(args) == 1`, parse arg as int64 and call `GetByID`; print the single `UserFilterResource` as one row
3. Otherwise, resolve list flags, call `List`, unwrap the `Content` slice from the paginated response, and pass the **slice** (not the page wrapper) to `printer.Print`

**Output contract:** In list mode, the value passed to `printer.Print` is the `[]types.UserFilterResource` slice, not the `PageUserFilterResource` wrapper. This matches how existing commands (`get user`, `get launch`, `get item`, `get log`) work — they all unwrap the pagination envelope and print only the resource slice. JSON mode therefore outputs a JSON array of filter objects, not the page metadata.

#### `newCreateFilterCmd()`

```go
func newCreateFilterCmd() *cobra.Command
```

- `Use`: `"filter"`
- `Short`: `"Create a filter"`
- `Long`: Include examples showing `--from-file` usage
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateFilter)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--from-file` / `-f` (string, required): path to JSON file containing `UpdateUserFilterRQ`

The run function:
1. Resolve the `--from-file` path, read the file, unmarshal into `types.UpdateUserFilterRQ`
2. Resolve connection, project, and client
3. Call `scope.Create(ctx, request)`
4. Print the `EntryCreatedRS` result (table shows ID column)

#### `newUpdateFilterCmd()`

```go
func newUpdateFilterCmd() *cobra.Command
```

- `Use`: `"filter <id>"`
- `Short`: `"Update a filter"`
- `Long`: Include examples showing `--from-file` with an ID argument
- `Args`: `cobra.ExactArgs(1)` (the numeric filter ID)
- `RunE`: `withPrintedErrors(runUpdateFilter)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--from-file` / `-f` (string, required): path to JSON file containing `UpdateUserFilterRQ`

The run function:
1. Parse `args[0]` as int64
2. Resolve the `--from-file` path, read the file, unmarshal into `types.UpdateUserFilterRQ`
3. Resolve connection, project, and client
4. Call `scope.Update(ctx, id, request)`
5. Print the `OperationCompletionRS` result using `operationResultHeaders()` and `operationResultRows()`

#### `newDeleteFilterCmd()`

```go
func newDeleteFilterCmd() *cobra.Command
```

- `Use`: `"filter <id>"`
- `Short`: `"Delete a filter"`
- `Long`: Include example showing delete by ID
- `Args`: `cobra.ExactArgs(1)` (the numeric filter ID)
- `RunE`: `withPrintedErrors(runDeleteFilter)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64
2. Resolve connection, project, and client
3. Call `scope.Delete(ctx, id)`
4. Print the `OperationCompletionRS` result using `operationResultHeaders()` and `operationResultRows()`

#### Table formatting

```go
func filterHeaders() []string {
    return []string{"ID", "NAME", "TYPE", "OWNER", "DESCRIPTION"}
}

func filterRow(filter *types.UserFilterResource) ([]string, []string) {
    return filterHeaders(), []string{
        strconv.FormatInt(filter.Id, 10),
        filter.Name,
        filter.Type,
        filter.Owner,
        formatStringPtr(filter.Description),
    }
}
```

#### `--from-file` helper

Implement a local helper to read and unmarshal the JSON file:

```go
func readFilterRequestFromFile(path string) (types.UpdateUserFilterRQ, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return types.UpdateUserFilterRQ{}, fmt.Errorf("reading file %s: %w", path, err)
    }

    var request types.UpdateUserFilterRQ
    if err := json.Unmarshal(data, &request); err != nil {
        return types.UpdateUserFilterRQ{}, fmt.Errorf("parsing JSON from %s: %w", path, err)
    }

    return request, nil
}
```

Mark `--from-file` as required on create and update commands using `cobra.MarkFlagRequired` or validate it manually in the run function with a helpful error message. Prefer manual validation with a clear error over `cobra.MarkFlagRequired` since it allows the error to be printed through the printer infrastructure.

#### ID parsing helper

Implement a local helper to parse the numeric ID from args:

```go
func parseFilterID(idStr string) (int64, error) {
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid filter ID %q: must be a numeric ID", idStr)
    }

    return id, nil
}
```

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// get
get.AddCommand(newGetFilterCmd())

// create
create.AddCommand(newCreateFilterCmd())

// update
update.AddCommand(newUpdateFilterCmd())

// delete
del.AddCommand(newDeleteFilterCmd())
```

### 3. Create `internal/cli/filter_test.go`

Use the same testing pattern as `user_test.go` and `launch_test.go`:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetFilterCmd_DefaultListOptions`** — Verify the default list flags are registered correctly (page-size=10, page=1, sort=empty/nil).

2. **`TestParseFilterID_Valid`** — Verify a valid numeric string returns the expected int64.

3. **`TestParseFilterID_Invalid`** — Verify non-numeric strings produce a descriptive error.

4. **`TestFilterRow_Formatting`** — Verify `filterRow` produces the expected headers and values for a sample `UserFilterResource`, including a nil description.

5. **`TestReadFilterRequestFromFile_ValidJSON`** — Write a temp JSON file, call `readFilterRequestFromFile`, verify the struct is populated correctly.

6. **`TestReadFilterRequestFromFile_MissingFile`** — Call with a nonexistent path, verify error mentions the path.

7. **`TestReadFilterRequestFromFile_InvalidJSON`** — Write a temp file with invalid JSON, verify error mentions parsing.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **`--from-file` is the only mutation input for create and update.** Filter definitions are complex nested JSON (conditions, orders, type). Exposing individual flags for each field would be unwieldy. This is the pattern recommended by the CLI plan for complex payloads.
- **No `--name`, `--type`, etc. override flags on mutations.** Keep this pilot simple. If needed, flag overrides can be added in a later PR after the helper extraction phase.
- **Reuse existing helpers.** Use `formatStringPtr` (from `item.go`), `operationResultHeaders`/`operationResultRows` (from `user.go`), `resolveProject`, `resolveConnection`, `newClient`, `resolveListFlags`, `registerListFlagsWithDefaults`, and `withPrintedErrors` (all existing).
- **No filter-specific list filters in this pilot.** The `ListFiltersOption` interface only has page/size/sort — no domain-specific filters to add.
- **List mode prints the resource slice, not the page wrapper.** This is consistent with all existing list commands in the CLI. The `printer.Print` call receives `[]types.UserFilterResource` as its value argument. In JSON mode this produces a JSON array; in table mode it produces rows.

### 4. Clarify the JSON output rule in `CLI_IMPLEMENTATION_PLAN.md`

The plan's "Current CLI direction" section states: "**JSON output** is the raw API type, passed through unchanged." This is accurate for single-get commands, but all existing list commands already unwrap the pagination envelope and print only the resource slice. Update this bullet to clarify:

> - **JSON output** is the resource type passed through unchanged. For single-get commands this is the raw API response object. For list commands this is the unwrapped resource array (page metadata is not included).

This makes the plan consistent with how `get user`, `get launch`, `get item`, `get log`, and now `get filter` all behave in list mode.

## What NOT to do

- **Do not implement `GetByIds`, `GetNames`, or `BulkUpdate`.** Those are stretch goals not in the core CRUD slice.
- **Do not extract shared `--from-file` helpers** into a separate file yet. Keep the helper in `filter.go`. Extraction happens in PR 7 after the other pilots establish the pattern.
- **Do not move `formatStringPtr`, `formatInt64Ptr`, `operationResultHeaders`, or `operationResultRows`** to shared files. That is PR 7's job.
- **Do not add integration tests.** Unit tests for flag resolution, ID parsing, file reading, and table formatting are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` now shows `filter` as a subcommand
- `rp create --help` now shows `filter` as a subcommand
- `rp update --help` now shows `filter` as a subcommand
- `rp delete --help` now shows `filter` as a subcommand
- `rp get filter --help` shows the expected flags and examples
- `rp create filter --help` shows `--from-file` flag

## Definition of done

- [ ] `internal/cli/filter.go` created with all five command constructors and their run functions
- [ ] `internal/cli/filter_test.go` created with unit tests for ID parsing, table formatting, and file reading
- [ ] Commands registered in `commands.go`
- [ ] All commands use `--project` / `-p` for project resolution
- [ ] Create and update use `--from-file` / `-f` for input
- [ ] Table output shows ID, NAME, TYPE, OWNER, DESCRIPTION with uppercase headers
- [ ] JSON output for single-get prints the `UserFilterResource` object; for list mode prints the unwrapped `[]UserFilterResource` slice (no page wrapper)
- [ ] `CLI_IMPLEMENTATION_PLAN.md` JSON output rule clarified to distinguish single-get (raw object) from list (unwrapped array)
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
