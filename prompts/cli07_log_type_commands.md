# PR 7: `log-type` Commands

## Goal

Implement `rp log-type {list,create,update,delete}`. Log types are a project-scoped resource with **no single-get method** (read-only via `list`), **no pagination** (returns all entries), and a **void delete response**. This makes them structurally different from the standard CRUD pilots.

## Context

This PR runs in the Phase 4 parallel fan-out. PRs 1–5 have landed, establishing shared helpers and conventions.

Log types are unusual in several ways:
1. **No `get` subcommand** — the API has no single-get-by-ID method. Discovery is via `list` only.
2. **No pagination** — `List` returns all log types for the project, not a page.
3. **Void delete** — `Delete` returns only `error` (no response body), like group delete.
4. **Create returns the full resource** — `Create` returns `*types.LogTypeResponse`, not `EntryCreatedRS`.
5. **Update returns `*types.SuccessfulUpdate`** — not `OperationCompletionRS`.
6. **Create uses flags for required fields** — `LogTypeRequest` has 3 required scalar fields plus an optional nested `Style`.

### Prerequisites

- PRs 1–5 have landed

### Client API surface

`LogTypesScope` (obtained via `client.Project(projectName).LogTypes()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `List` | `List(ctx)` | `[]types.LogTypeResponse` |
| `Create` | `Create(ctx, types.LogTypeRequest)` | `*types.LogTypeResponse` |
| `Update` | `Update(ctx, logTypeID int64, types.LogTypeRequest)` | `*types.SuccessfulUpdate` |
| `Delete` | `Delete(ctx, logTypeID int64)` | `error` (void) |

**Note:** `List` takes no options. It returns all log types for the project.

### Relevant types

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

type SuccessfulUpdate struct {
    Message *string `json:"message,omitempty"`
}
```

`LogTypeStyleTextStyle` is a string type.

## Implementation plan

### 1. Create `internal/cli/log_type.go`

This file defines the `rp log-type` parent command and all verb subcommands.

#### Parent command

```go
func newLogTypeCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "log-type",
        Short: "Manage log types",
    }
    cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
    cmd.AddCommand(newLogTypeListCmd())
    cmd.AddCommand(newLogTypeCreateCmd())
    cmd.AddCommand(newLogTypeUpdateCmd())
    cmd.AddCommand(newLogTypeDeleteCmd())
    return cmd
}
```

Register in `root.go`: `cmd.AddCommand(newLogTypeCmd())`.

#### `rp log-type list`

- No positional arguments
- **No pagination flags** — `List` returns all log types for the project. Do NOT register `--page`, `--page-size`, or `--sort`.
- Table output columns: `ID`, `NAME`, `LEVEL`, `FILTERABLE`, `SYSTEM`
  - `FILTERABLE` and `SYSTEM` are boolean columns (`"true"` / `"false"`)
- JSON output: the raw `[]LogTypeResponse` slice

#### `rp log-type create`

- No positional arguments
- **Input: flags OR `--from-file`**
- Flags (when not using `--from-file`):
  - `--name` (string, required) — log type name
  - `--level` (int32, required) — numeric log level
  - `--filterable` (bool, default false) — whether the log type is filterable
- Optional: `--from-file <path>` — path to a JSON file containing a full `LogTypeRequest`
- When `--from-file` is provided, ignore individual flags and use the file contents. When `--from-file` is not provided, all three required flags must be set.
- **Style is only available via `--from-file`** — the nested `Style` struct is too complex for individual flags. Users who need custom styles must use `--from-file`. This is intentional and should be noted in help text.
- Validate: `Name` must be non-empty, `Level` must be > 0
- Call `scope.Create(ctx, request)`
- Table output: same columns as list (show the created log type)
- JSON output: the raw `*LogTypeResponse`

#### `rp log-type update <id> --from-file <path>`

- Exactly one positional argument: numeric log type ID
- Required flag: `--from-file <path>` — path to a JSON file containing a full `LogTypeRequest`
- **Update is full-replacement only via `--from-file`.** The API takes a complete `LogTypeRequest` (all fields required, not a partial patch). Since the user must provide the full state, `--from-file` is the appropriate input method. Individual flags would force re-specifying all fields every time, which is poor UX.
- Parse ID with `parseIDArg(args[0], "log-type")`
- Read file with `readRequestFromFile[types.LogTypeRequest](path)`
- Validate: same as create (Name non-empty, Level > 0)
- Call `scope.Update(ctx, id, request)`
- Table/JSON output: `SuccessfulUpdate` via `messageHeaders()`/`messageRows(formatMessage(result.Message))`

#### `rp log-type delete <id>`

- Exactly one positional argument: numeric log type ID
- No additional flags
- Call `scope.Delete(ctx, id)` — returns only `error` (void response)
- On success, synthesize a confirmation message:
  - Table output: `MESSAGE` column with value `"Log type deleted successfully"`
  - JSON output: `{"message": "Log type deleted successfully"}`
  - Use a local struct or `map[string]string` for the JSON case

### 2. Create/from-file mutual exclusivity

The create command supports two input modes:

1. **Flag mode:** `--name`, `--level`, `--filterable` are provided. `--from-file` is absent.
2. **File mode:** `--from-file` is provided. Individual flags are ignored.

Implementation approach:
- Register all flags (including `--from-file`) without marking any as required
- In the handler, check `cmd.Flags().Changed("from-file")`:
  - If true: read from file, validate, proceed
  - If false: check that `--name` is set and `--level` is set. Build `LogTypeRequest` from flags. `--filterable` defaults to `false` if not explicitly set.
- If neither `--from-file` nor the required flags are provided, return a clear error explaining both input modes

### 3. Table formatting

```go
func logTypeHeaders() []string {
    return []string{"ID", "NAME", "LEVEL", "FILTERABLE", "SYSTEM"}
}

func logTypeRow(lt *types.LogTypeResponse) ([]string, []string) {
    return logTypeHeaders(), []string{
        formatInt64Ptr(lt.Id),
        lt.Name,
        strconv.FormatInt(int64(lt.Level), 10),
        strconv.FormatBool(lt.IsFilterable),
        formatBoolPtr(lt.IsSystem),
    }
}
```

### 4. Register in root.go

Add `cmd.AddCommand(newLogTypeCmd())` alongside the other resource commands.

## Test requirements

### Unit tests (`internal/cli/log_type_test.go`)

- **ID parsing:** Verify that `rp log-type update abc --from-file f.json` returns an error
- **Create flag validation:**
  - Missing `--name` without `--from-file` errors
  - Missing `--level` without `--from-file` errors (level flag has a default of 0, which should fail validation since Level must be > 0)
  - Valid flags produce correct `LogTypeRequest`
  - `--filterable` defaults to false
- **Create from-file mode:**
  - Valid JSON produces correct `LogTypeRequest`
  - Missing name field errors
  - Zero level errors
  - Non-existent file errors
  - Malformed JSON errors
- **Create mode detection:** `--from-file` takes precedence over flags
- **Update validation:** Same file validation as create
- **Table formatting:** Verify `logTypeRow` produces expected columns, handles nil Id and IsSystem
- **Command wiring:** Verify `newLogTypeCmd()` has exactly `list`, `create`, `update`, `delete` subcommands (no `get`)

## Edge cases and constraints

- **No `get` subcommand:** The API has no single-get-by-ID method. Do NOT create a `get` subcommand.
- **No pagination:** `List` returns all log types. Do NOT register `--page`, `--page-size`, or `--sort` on the list command.
- **Delete returns void:** The client's `Delete` method returns only `error`. Synthesize a success message (same pattern as group delete).
- **`SuccessfulUpdate` vs `OperationCompletionRS`:** Both have a `Message *string` field with the same shape. The shared `formatMessage` helper works for both. Use `messageHeaders()`/`messageRows(formatMessage(result.Message))` for update output.
- **`LogTypeResponse.Id` is `*int64` (pointer):** Use `formatInt64Ptr` for table formatting.
- **`Level` validation:** Level must be > 0. A level of 0 is invalid. This applies to both flag mode and file mode.
- **`IsFilterable` is a non-pointer bool:** In `LogTypeRequest`, `IsFilterable` is `bool` (not `*bool`). When using flags, the default value of `false` is valid. The `--filterable` flag should be a bool flag (not `--filterable=true/false`); its presence sets the field to `true`.
- **Style in file mode:** When using `--from-file`, the JSON may include a `style` object. Pass it through unchanged. Do not validate style field values — the API handles that.
- **`List` returns `[]LogTypeResponse` directly:** Unlike paginated resources, there's no page wrapper. The returned slice may be nil if the API returns no log types; treat nil as empty.
- **File name `log_type.go`:** Use underscore to match Go naming conventions for multi-word names, consistent with `log_type` naming in `pkg/client/log_types.go`.

## Example commands and expected output

```bash
# List all log types
$ rp log-type list -p my_project
ID    NAME     LEVEL   FILTERABLE  SYSTEM
1     ERROR    40000   true        true
2     WARN     30000   true        true
3     INFO     20000   true        true
4     DEBUG    10000   true        true
5     TRACE    5000    false       false

# Create with flags
$ rp log-type create --name "TRACE" --level 5000 --filterable -p my_project
ID    NAME     LEVEL   FILTERABLE  SYSTEM
5     TRACE    5000    true        false

# Create from file (with style)
$ rp log-type create --from-file log_type.json -p my_project
ID    NAME     LEVEL   FILTERABLE  SYSTEM
6     CUSTOM   15000   true        false

# Update from file (full replacement)
$ rp log-type update 5 --from-file updated_log_type.json -p my_project
MESSAGE
Log type successfully updated

# Delete
$ rp log-type delete 5 -p my_project
MESSAGE
Log type deleted successfully
```

Example `log_type.json` (with style):
```json
{
  "name": "CUSTOM",
  "level": 15000,
  "is_filterable": true,
  "style": {
    "background_color": "#FFE0B2",
    "text_color": "#E65100",
    "label_color": "#FF6D00",
    "text_style": "bold"
  }
}
```

## Validation

Run before finalizing:

```
make lint
make test
```

## Definition of done

- [ ] `internal/cli/log_type.go` implements list, create, update, and delete subcommands
- [ ] No `get` subcommand (API has no single-get method)
- [ ] `rp log-type` registered in `root.go`
- [ ] Create supports flags (`--name`, `--level`, `--filterable`) OR `--from-file`
- [ ] Update requires `--from-file` (full replacement)
- [ ] Delete synthesizes a success message (void API response)
- [ ] Table output with uppercase headers: ID, NAME, LEVEL, FILTERABLE, SYSTEM
- [ ] JSON output passes through raw API types unchanged
- [ ] Unit tests cover both input modes, validation, table formatting, and command wiring
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
