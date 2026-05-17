# PR 2: Pilot `filter` Full CRUD

## Goal

Implement the first pilot resource — `rp filter {list,get,create,update,delete}` — as a project-scoped CRUD family. This PR establishes the baseline patterns for project-scoped resource commands, pagination, `--from-file` mutation flow, and table formatting that subsequent resource families will follow.

## Context

**Prerequisite:** PR 1 (noun-first refactor) has landed. The CLI now uses noun-first command registration (`rp <noun> <verb>`), with each resource defining a parent command and registering verb subcommands. The `--json` global shortcut is available.

### API surface

The filter scope is accessed via `client.Project(projectName).Filters()` and provides:

| Method | Signature |
|--------|-----------|
| `List` | `List(ctx, options ...ListFiltersOption) (*types.PageUserFilterResource, error)` |
| `GetByID` | `GetByID(ctx, id int64) (*types.UserFilterResource, error)` |
| `Create` | `Create(ctx, request types.UpdateUserFilterRQ) (*types.EntryCreatedRS, error)` |
| `Update` | `Update(ctx, id int64, request types.UpdateUserFilterRQ) (*types.OperationCompletionRS, error)` |
| `Delete` | `Delete(ctx, id int64) (*types.OperationCompletionRS, error)` |

List options: `WithFilterPageNumber(int32)`, `WithFilterPageSize(int32)`, `WithFilterSort(...string)`.

### Key types

```go
// The request type used for BOTH create and update.
type UpdateUserFilterRQ struct {
    Conditions  []UserFilterCondition  `json:"conditions"`
    Description *string                `json:"description,omitempty"`
    Name        string                 `json:"name"`
    Orders      []Order                `json:"orders"`
    Type        UpdateUserFilterRQType `json:"type"`  // "launch", "log", "testitem"
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

type PageUserFilterResource struct {
    Content *[]UserFilterResource `json:"content,omitempty"`
    Page    *PageMetadata         `json:"page,omitempty"`
}

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

### Files to create

- `internal/cli/filter.go` — all filter commands and helpers
- `internal/cli/filter_test.go` — unit tests

### Files to modify

- `internal/cli/root.go` — add `cmd.AddCommand(newFilterCmd())` to register the filter parent command

## Detailed requirements

### 1. Parent command scaffolding

Define `newFilterCmd()` returning a `*cobra.Command` that groups all filter verb subcommands:

```go
func newFilterCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "filter",
        Short: "Manage saved filters",
    }
    cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
    cmd.AddCommand(newFilterListCmd())
    cmd.AddCommand(newFilterGetCmd())
    cmd.AddCommand(newFilterCreateCmd())
    cmd.AddCommand(newFilterUpdateCmd())
    cmd.AddCommand(newFilterDeleteCmd())
    return cmd
}
```

Register `-p`/`--project` as a **persistent flag** on the parent so all subcommands inherit it.

### 2. `rp filter list`

**Command shape:** `rp filter list [--page N] [--page-size N] [--sort field,dir]`

- `Use: "list"`, `Args: cobra.NoArgs`
- Register standard list flags using `registerListFlagsWithDefaults` with a default sort of `name,ASC` (filters are most naturally listed alphabetically)
- Resolve pagination flags via `resolveListFlags`
- Map resolved flags to `ListFiltersOption` values: `WithFilterPageNumber`, `WithFilterPageSize`, `WithFilterSort`
- Call `scope.List(ctx, options...)`
- Extract `[]UserFilterResource` from `result.Content` (nil-safe: treat nil Content as empty slice)
- Print using `printer.Print(filters, headers, rows)`

**Table columns (all uppercase headers):**

| Header | Source | Notes |
|--------|--------|-------|
| ID | `Id` | `strconv.FormatInt` |
| NAME | `Name` | Direct string |
| TYPE | `Type` | Direct string |
| OWNER | `Owner` | Direct string |
| DESCRIPTION | `Description` | `formatStringPtr`, truncate to 50 chars with `...` suffix |
| CONDITIONS | `len(Conditions)` | Integer count, formatted as string |

Define `filterHeaders() []string` and `filterRow(*types.UserFilterResource) ([]string, []string)` following the existing pattern (return both headers and row values).

### 3. `rp filter get <id>`

**Command shape:** `rp filter get <id>`

- `Use: "get <id>"`, `Args: cobra.ExactArgs(1)`
- Parse the positional argument as `int64` (filters are always identified by numeric ID — there is no UUID variant)
- If parsing fails, return a clear error: `invalid filter ID "<value>": must be a numeric ID`
- Call `scope.GetByID(ctx, id)`
- Print using the same `filterRow` formatting as list

### 4. `rp filter create --from-file <path>`

**Command shape:** `rp filter create --from-file <path> [--name override] [--description override]`

- `Use: "create"`, `Args: cobra.NoArgs`
- `--from-file` (`string`, required) — path to a JSON file containing an `UpdateUserFilterRQ` payload. The special value `-` reads from stdin.
- `--name` (`string`, optional) — overrides the `name` field from the file
- `--description` (`string`, optional) — overrides the `description` field from the file
- Validation: `--from-file` is required. If not provided, return an error: `--from-file is required for filter creation (filter definitions contain structured conditions and orders)`

**Flow:**

1. Read and parse the JSON file into `types.UpdateUserFilterRQ` (see section 7 for `--from-file` implementation)
2. Apply flag overrides: if `--name` is changed, overwrite `request.Name`; if `--description` is changed, overwrite `request.Description`
3. Validate: `request.Name` must be non-empty after overrides. If empty, error: `filter name is required (provide via --from-file or --name flag)`
4. Call `scope.Create(ctx, request)`
5. Print the result (`EntryCreatedRS`): table with `ID` header showing the created filter's ID. For JSON mode, print the raw `EntryCreatedRS`.

**Table for create result:**

| Header | Source |
|--------|--------|
| ID | `formatInt64Ptr(result.Id)` |

### 5. `rp filter update <id> --from-file <path>`

**Command shape:** `rp filter update <id> --from-file <path> [--name override] [--description override]`

- `Use: "update <id>"`, `Args: cobra.ExactArgs(1)`
- Parse the positional argument as `int64` (same validation as get)
- `--from-file` (`string`, required) — same as create
- `--name` and `--description` — same override behavior as create
- Validation: `--from-file` is required. Error message: `--from-file is required for filter update (the API expects the complete filter definition)`

**Flow:**

1. Parse ID from positional argument
2. Read and parse JSON file
3. Apply flag overrides
4. Call `scope.Update(ctx, id, request)`
5. Print the result (`OperationCompletionRS`): use `operationResultHeaders()` / `operationResultRows()` from `user.go`

### 6. `rp filter delete <id>`

**Command shape:** `rp filter delete <id>`

- `Use: "delete <id>"`, `Args: cobra.ExactArgs(1)`
- Parse the positional argument as `int64`
- Call `scope.Delete(ctx, id)`
- Print the result (`OperationCompletionRS`) using the same operation result formatting

### 7. `--from-file` implementation

Implement a file-reading helper within `filter.go`. This helper will be duplicated in the widget pilot (PR 4); Phase 3 (PR 5) will extract a shared version.

```go
func readFilterRequestFromFile(path string) (types.UpdateUserFilterRQ, error)
```

**Behavior:**

- If `path == "-"`, read from `os.Stdin`
- Otherwise, read from `os.Open(path)` (use `os.ReadFile` or `io.ReadAll` after opening)
- Unmarshal JSON into `types.UpdateUserFilterRQ` using `json.Unmarshal`
- **Do NOT** use `json.Decoder.DisallowUnknownFields()` — unknown fields should be silently ignored. This enables the round-trip workflow: `rp filter get 42 -o json > f.json && rp filter create --from-file f.json` (the `id` and `owner` fields from the get output will be harmlessly ignored)
- Return clear errors for: file not found, read errors, JSON parse errors

### 8. Help text and examples

Each subcommand should have a `Long` description with CLI examples, following the existing pattern. For example:

```
rp filter list                             List saved filters
rp filter list --page-size 25              List with custom page size
rp filter get 42                           Get filter by ID
rp filter get 42 -o json                   Get filter as JSON
rp filter create --from-file filter.json   Create a filter from file
rp filter update 42 --from-file f.json     Update a filter from file
rp filter delete 42                        Delete a filter
```

### 9. Standard boilerplate

Follow the established patterns from existing resource commands:

- **Printer creation:** `outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")` then `newPrinter(outputFmt)`
- **Connection resolution:** `resolveConnection(cmd)` → `resolveProject(cmd, conn)` → `newClient(conn, logger)`
- **Error handling:** call `printer.PrintError(err)` before returning the error, and wrap with `withPrintedErrors`
- **Logger:** `loggerFromCmd(cmd)` at the top of each RunE function

## Test requirements

Create `internal/cli/filter_test.go` in `package cli` (white-box tests, matching existing test files).

### Required tests

1. **`filterRow` formatting:** verify headers and row values for a `UserFilterResource` with all fields populated, including nil `Description` and non-nil `Description` (truncation at 50 chars)

2. **`readFilterRequestFromFile` happy path:** write a temporary JSON file, read it, verify all fields are deserialized correctly

3. **`readFilterRequestFromFile` ignores unknown fields:** write JSON with extra fields (`id`, `owner`), verify they're silently ignored and the known fields parse correctly

4. **`readFilterRequestFromFile` invalid JSON:** verify a clear error is returned for malformed JSON

5. **`readFilterRequestFromFile` missing file:** verify error for non-existent path

6. **Flag override logic for create/update:** verify that `--name` and `--description` flags override values from the parsed file, and that unchanged flags preserve file values

7. **Filter ID parsing:** verify that non-numeric IDs produce a clear error message

8. **`newFilterCmd` subcommands:** verify the parent command has the expected child command names (list, get, create, update, delete)

### Test patterns

- Use `testing.T` with `t.Parallel()` for all tests and subtests
- Use `testify/assert` and `testify/require`
- Use `t.TempDir()` for temporary file creation
- Use table-driven tests where multiple inputs map to the same assertion shape
- Follow naming: `TestFilterRow_*`, `TestReadFilterRequestFromFile_*`, etc.

## Edge cases and constraints

- **ID type:** Filters use only numeric IDs. Do not implement UUID lookup — the API does not support it.
- **Create type reuse:** The API uses `UpdateUserFilterRQ` for both create and update. This is not a mistake — the type name is from the OpenAPI spec. Do not create a wrapper type.
- **Empty conditions:** A filter with `"conditions": []` is technically valid JSON and should be accepted. The API will decide whether to reject it.
- **Empty file content:** If `--from-file` points to an empty file, `json.Unmarshal` will fail. The error from JSON parsing is sufficient — no special handling needed.
- **Large description truncation:** In table mode, truncate `Description` at 50 characters with `...` suffix. In JSON mode, the full description is always included (it's the raw API response).
- **`--from-file` is required for create and update.** Do not support creating or updating filters via inline flags alone. The `conditions` and `orders` fields are structured arrays that cannot be ergonomically expressed as CLI flags. This is a deliberate design choice aligned with the project's `--from-file` policy for complex payloads.
- **Nil content in list response:** `PageUserFilterResource.Content` may be nil (empty page). Treat as empty slice.
- **Phase 3 extraction:** The `--from-file` reading logic will be duplicated in the widget pilot (PR 4). This is intentional — Phase 3 will extract a shared helper after both pilots prove the pattern.

## Validation

Run before finalizing:

```
make lint
make vet
make test
```

## Definition of done

- [ ] `internal/cli/filter.go` implements `rp filter {list,get,create,update,delete}`
- [ ] `internal/cli/filter_test.go` covers row formatting, file reading, flag overrides, ID parsing, and subcommand structure
- [ ] `internal/cli/root.go` registers `newFilterCmd()`
- [ ] `-p`/`--project` is a persistent flag on the filter parent command, inherited by all subcommands
- [ ] `--from-file` supports both file paths and `-` for stdin
- [ ] `--name` and `--description` flags override values from `--from-file` on create and update
- [ ] List uses standard pagination flags (`--page`, `--page-size`, `--sort`) with `name,ASC` default sort
- [ ] Table output uses uppercase headers and consistent formatting
- [ ] JSON output returns raw API types unchanged
- [ ] All error paths call `printer.PrintError` before returning
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
