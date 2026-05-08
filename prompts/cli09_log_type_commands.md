# PR 9: `log-type` Commands

## Goal

Implement the complete `log-type` command family — list, create, update, and delete. Log types are project-scoped but have several characteristics that differ from the standard CRUD pattern: the list endpoint returns all results (no pagination), there is no single-get-by-ID method, and mutation payloads include a nested `Style` object.

## Context

The `log-type` resource maps to `LogTypesScope` on the client (`client.Project(name).LogTypes()`). It is project-scoped, meaning every command requires a resolved project via `--project`, `RP_PROJECT`, or the profile's `default-project`.

**Prerequisite:** PRs 1–7 must land first (parent command split, verb scaffolding, centralized registration, all three pilots, and shared helper extraction).

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `List` | `ctx` | `([]types.LogTypeResponse, error)` |
| `Create` | `ctx`, `request types.LogTypeRequest` | `(*types.LogTypeResponse, error)` |
| `Update` | `ctx`, `logTypeID int64`, `request types.LogTypeRequest` | `(*types.SuccessfulUpdate, error)` |
| `Delete` | `ctx`, `logTypeID int64` | `error` (no response body) |

**Notable:** `List` has no pagination options — the API returns all log types at once. `Create` returns the created resource (not `EntryCreatedRS`). `Delete` returns only `error` (no response body, same pattern as group delete).

### Key types

```go
type LogTypeRequest struct {
    IsFilterable bool          `json:"is_filterable"`
    Level        int32         `json:"level"`
    Name         string        `json:"name"`
    Style        *LogTypeStyle `json:"style,omitempty"`
}

type LogTypeResponse struct {
    Id           *int64        `json:"id,omitempty"`
    IsFilterable bool          `json:"is_filterable"`
    IsSystem     *bool         `json:"is_system,omitempty"`
    Level        int32         `json:"level"`
    Name         string        `json:"name"`
    Style        *LogTypeStyle `json:"style,omitempty"`
}

type LogTypeStyle struct {
    BackgroundColor *string               `json:"background_color,omitempty"`
    LabelColor      *string               `json:"label_color,omitempty"`
    TextColor       *string               `json:"text_color,omitempty"`
    TextStyle       *LogTypeStyleTextStyle `json:"text_style,omitempty"`
}
```

`LogTypeStyleTextStyle` is a string type with allowed values `"bold"` and `"normal"`.

**All three primary fields on `LogTypeRequest` are non-pointer types** (`bool`, `int32`, `string`). This means the API always expects explicit values for `IsFilterable`, `Level`, and `Name` — there is no "omit to keep current value" partial update. Both create and update send the same request shape.

## Scope

Create `internal/cli/log_type.go` and `internal/cli/log_type_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get log-type` | List all log types in the project (no pagination) |
| `rp create log-type --name <name> --level <n> [flags]` | Create a log type |
| `rp update log-type <id> --name <name> --level <n> [flags]` | Update a log type |
| `rp delete log-type <id>` | Delete a log type by numeric ID |

## Implementation

### 1. Create `internal/cli/log_type.go`

This file contains all four command constructors and their run functions.

#### `newGetLogTypeCmd()`

```go
func newGetLogTypeCmd() *cobra.Command
```

- `Use`: `"log-type"`
- `Short`: `"List log types in a project"`
- `Long`: Include examples showing list and output format. Explicitly state that single-get-by-ID is not supported and the command always lists all log types.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGetLogType)`
- Flags:
  - `--project` / `-p` (string): project name override

**Do NOT** register pagination flags (`--page-size`, `--page`, `--sort`). The API returns all log types in a single response with no pagination support.

The run function:
1. Resolve connection, project, and client
2. Call `scope.List(ctx)` — returns `[]types.LogTypeResponse` directly (not a page wrapper)
3. Build headers and rows from the slice
4. Pass the slice to `printer.Print`

**Output contract:** The value passed to `printer.Print` is the `[]types.LogTypeResponse` slice. JSON mode outputs a JSON array. This is already unwrapped since the client normalizes the response.

#### `newCreateLogTypeCmd()`

```go
func newCreateLogTypeCmd() *cobra.Command
```

- `Use`: `"log-type"`
- `Short`: `"Create a log type"`
- `Long`: Include examples showing inline flag usage with required and optional flags. Mention style flags.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateLogType)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--name` (string, required): log type name
  - `--level` (int32, required): numeric log level (must be unique within the project)
  - `--is-filterable` (bool, default `false`): whether the log level is filterable
  - `--background-color` (string, optional): hex background color for the log type style
  - `--label-color` (string, optional): hex label color
  - `--text-color` (string, optional): hex text color
  - `--text-style` (string, optional): text style, either `"bold"` or `"normal"`

The run function:
1. Validate `--name` is non-empty. Return a descriptive error if missing.
2. Validate `--level` was explicitly provided. Since `int32` defaults to 0, use `cmd.Flags().Changed("level")` to distinguish "user provided 0" from "user didn't provide a value". If not changed, return an error: `"--level is required"`.
3. Read `--is-filterable` via `cmd.Flags().GetBool("is-filterable")`
4. Build `*types.LogTypeStyle` from the four style flags — only allocate the struct if at least one style flag was explicitly provided (`cmd.Flags().Changed`). If no style flags were set, leave `Style` as nil.
5. Build `types.LogTypeRequest{Name: name, Level: level, IsFilterable: isFilterable, Style: style}`
6. Resolve connection, project, and client
7. Call `scope.Create(ctx, request)` — returns `*types.LogTypeResponse`
8. Print the created `LogTypeResponse` as a single table row using `logTypeRow`

**Note:** Create returns the full `LogTypeResponse` (not `EntryCreatedRS`). Use the same `logTypeRow` formatter as the list command.

#### `newUpdateLogTypeCmd()`

```go
func newUpdateLogTypeCmd() *cobra.Command
```

- `Use`: `"log-type <id>"`
- `Short`: `"Update a log type"`
- `Long`: Include examples. Explicitly note that `--name` and `--level` are required because the API replaces the entire log type definition (not a partial update).
- `Args`: `cobra.ExactArgs(1)` (the numeric log type ID)
- `RunE`: `withPrintedErrors(runUpdateLogType)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--name` (string, required): log type name
  - `--level` (int32, required): numeric log level
  - `--is-filterable` (bool, default `false` — see below)
  - `--background-color` (string, optional): hex background color
  - `--label-color` (string, optional): hex label color
  - `--text-color` (string, optional): hex text color
  - `--text-style` (string, optional): text style (`"bold"` or `"normal"`)

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "log-type")`
2. Validate `--name` is non-empty
3. Validate `--level` was explicitly provided via `cmd.Flags().Changed("level")` — this is required because `Level int32` is non-pointer: omitting it sends 0, which silently changes the log level
4. Read `--is-filterable` via `cmd.Flags().GetBool("is-filterable")`. For update, this flag does not need `Changed` validation — the zero value (`false`) is a valid intentional choice. The help text should note that omitting `--is-filterable` sets it to `false`, so users should explicitly pass `--is-filterable` if the log type should remain filterable.
5. Build `*types.LogTypeStyle` from the four style flags (same logic as create — only allocate if at least one was provided; nil omits the field from the JSON request due to `omitempty`)
6. Build `types.LogTypeRequest{Name: name, Level: level, IsFilterable: isFilterable, Style: style}`
7. Resolve connection, project, and client
8. Call `scope.Update(ctx, id, request)` — returns `*types.SuccessfulUpdate`
9. Print the result using `messageResultHeaders()` and `successfulUpdateRows(result)`

#### `newDeleteLogTypeCmd()`

```go
func newDeleteLogTypeCmd() *cobra.Command
```

- `Use`: `"log-type <id>"`
- `Short`: `"Delete a log type"`
- `Long`: Include example showing delete by ID. Note that system log types cannot be deleted.
- `Args`: `cobra.ExactArgs(1)` (the numeric log type ID)
- `RunE`: `withPrintedErrors(runDeleteLogType)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "log-type")`
2. Resolve connection, project, and client
3. Call `scope.Delete(ctx, id)` — returns `error` only (no response body)
4. On success, synthesize a structured result and print it:

```go
type logTypeDeleteResult struct {
    Message string `json:"message"`
}
```

Build the result as `logTypeDeleteResult{Message: fmt.Sprintf("Log type %d deleted successfully", id)}` and pass it to `printer.Print` with `messageResultHeaders()` and a single-row slice. This follows the same pattern as group delete (PR 5) for delete operations with no response body.

#### Table formatting

```go
func logTypeHeaders() []string {
    return []string{"ID", "NAME", "LEVEL", "FILTERABLE", "SYSTEM"}
}

func logTypeRow(logType *types.LogTypeResponse) ([]string, []string) {
    return logTypeHeaders(), []string{
        formatInt64Ptr(logType.Id),
        logType.Name,
        strconv.FormatInt(int64(logType.Level), 10),
        strconv.FormatBool(logType.IsFilterable),
        formatBoolPtr(logType.IsSystem),
    }
}
```

The `Style` field is intentionally omitted from the table. Users who need style details should use `-o json`.

#### Style flag builder helper

```go
func buildLogTypeStyle(cmd *cobra.Command) *types.LogTypeStyle {
    bgChanged := cmd.Flags().Changed("background-color")
    labelChanged := cmd.Flags().Changed("label-color")
    textChanged := cmd.Flags().Changed("text-color")
    styleChanged := cmd.Flags().Changed("text-style")

    if !bgChanged && !labelChanged && !textChanged && !styleChanged {
        return nil
    }

    style := &types.LogTypeStyle{}

    if bgChanged {
        style.BackgroundColor = optionalStringFlag(cmd, "background-color")
    }

    if labelChanged {
        style.LabelColor = optionalStringFlag(cmd, "label-color")
    }

    if textChanged {
        style.TextColor = optionalStringFlag(cmd, "text-color")
    }

    if styleChanged {
        raw := mustGetStringFlag(cmd, "text-style")
        textStyle := types.LogTypeStyleTextStyle(raw)
        style.TextStyle = &textStyle
    }

    return style
}
```

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// get
get.AddCommand(newGetLogTypeCmd())

// create
create.AddCommand(newCreateLogTypeCmd())

// update
update.AddCommand(newUpdateLogTypeCmd())

// delete
del.AddCommand(newDeleteLogTypeCmd())
```

### 3. Create `internal/cli/log_type_test.go`

Use the same testing pattern as the pilot test files:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetLogTypeCmd_NoListFlags`** — Verify the get command does NOT have `--page-size`, `--page`, or `--sort` flags registered. Verify it has `--project`.

2. **`TestNewGetLogTypeCmd_NoArgs`** — Verify `Args` is `cobra.NoArgs` (no positional arguments accepted).

3. **`TestLogTypeRow_Formatting`** — Verify `logTypeRow` produces the expected headers and values for a fully-populated `LogTypeResponse`.

4. **`TestLogTypeRow_NilFields`** — Verify `logTypeRow` handles a `LogTypeResponse` with nil `Id` and nil `IsSystem` (produces empty strings for those columns, but `IsFilterable` and `Level` always have values since they are non-pointer).

5. **`TestBuildLogTypeStyle_NoFlags`** — Create a command, register flags, don't set any style flags. Verify `buildLogTypeStyle` returns nil.

6. **`TestBuildLogTypeStyle_SomeFlags`** — Set `--background-color` and `--text-style`, verify the returned `LogTypeStyle` has those fields set and others nil.

7. **`TestBuildLogTypeStyle_AllFlags`** — Set all four style flags, verify all are populated.

8. **`TestNewCreateLogTypeCmd_Flags`** — Verify create has `--name`, `--level`, `--is-filterable`, `--project`, and the four style flags registered.

9. **`TestNewUpdateLogTypeCmd_Flags`** — Verify update has the same flags as create plus accepts exactly 1 arg.

10. **`TestNewDeleteLogTypeCmd_Args`** — Verify delete accepts exactly 1 arg.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **No single-get-by-ID.** The API has no method to retrieve a single log type by ID. The `get` command always lists all log types. Unlike the widget pilot (which returns a friendly error for the unsupported list operation), log-type simply uses `cobra.NoArgs` and always lists.
- **No pagination.** The list endpoint returns all log types at once. Do not register pagination flags.
- **Inline flags for create and update.** `LogTypeRequest` has 4 top-level fields (3 scalars + 1 optional struct with 4 scalar sub-fields). This totals 7 flags, all of which are simple scalars. This is within the inline flag comfort zone — the "complex payload" rule applies to deeply nested structures and arrays, not flat scalar fields.
- **`--level` validation uses `Changed`.** Since `int32` defaults to 0 and 0 could be a valid log level, the only way to detect whether the user explicitly provided `--level` is via `cmd.Flags().Changed("level")`. If the user truly wants level 0, they must write `--level 0` explicitly.
- **`IsFilterable` on update defaults to `false`.** This is a known sharp edge: if a log type currently has `is_filterable=true` and the user omits `--is-filterable` during update, it will be set to `false`. The help text should warn about this. The API does not support partial updates, so there is no way to avoid this without a fundamentally different command design (e.g., `--from-file` with a get-modify-put workflow).
- **Style is nil when no style flags are provided.** Because the `Style` field uses `json:"style,omitempty"`, a nil value is omitted from the serialized JSON — it is **not** sent as an explicit `null`. Whether the server preserves or clears the existing style on omission is server-defined behavior. The help text should note that omitting all style flags does not explicitly send a style value, and users who need to guarantee clearing should set all style fields to empty strings.
- **Delete returns no body — use a local result struct.** Same pattern as group delete (PR 5). The `logTypeDeleteResult` struct ensures JSON mode outputs `{"message": "..."}` consistently.
- **Create returns the full resource.** Unlike most other create operations (which return `EntryCreatedRS`), log-type create returns `*types.LogTypeResponse`. Use `logTypeRow` to format it, not `entryCreatedHeaders`/`entryCreatedRows`.
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `formatInt64Ptr`, `formatBoolPtr`, `optionalStringFlag`, `mustGetStringFlag`, `messageResultHeaders`, `successfulUpdateRows`.

## What NOT to do

- **Do not add a single-get-by-ID mode** to the get command. The API does not support it.
- **Do not add pagination flags** to the get command. The API returns all results at once.
- **Do not use `--from-file` for mutations.** The payload is simple enough for inline flags. Consistency with the group pilot pattern (simple payload = inline flags) is more important than uniformity across all resources.
- **Do not validate style values** (e.g., hex color format). Let the API validate — the CLI should pass through user input.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` now shows `log-type` as a subcommand
- `rp create --help` now shows `log-type` as a subcommand
- `rp update --help` now shows `log-type` as a subcommand
- `rp delete --help` now shows `log-type` as a subcommand
- `rp get log-type --help` shows `--project` but NOT pagination flags
- `rp create log-type --help` shows `--name`, `--level`, `--is-filterable`, `--project`, and style flags
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/log_type.go` created with all four command constructors and their run functions
- [ ] `internal/cli/log_type_test.go` created with unit tests for flag registration, table formatting, and style builder
- [ ] Commands registered in `commands.go`
- [ ] All commands use `--project` / `-p` for project resolution
- [ ] `get` uses `cobra.NoArgs` — always lists all log types, no single-get
- [ ] No pagination flags on the get command
- [ ] Create and update use inline flags for all fields
- [ ] `--level` validated via `cmd.Flags().Changed` to distinguish intentional 0 from omission
- [ ] Style flags only allocate `LogTypeStyle` when at least one is provided
- [ ] Table output shows ID, NAME, LEVEL, FILTERABLE, SYSTEM with uppercase headers
- [ ] Create command prints the returned `LogTypeResponse` (not `EntryCreatedRS`)
- [ ] Delete uses a local `logTypeDeleteResult` struct for consistent JSON/table output
- [ ] Shared helpers used: `parseResourceID`, `formatInt64Ptr`, `formatBoolPtr`, `optionalStringFlag`, `mustGetStringFlag`, `messageResultHeaders`, `successfulUpdateRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
