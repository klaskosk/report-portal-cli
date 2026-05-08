# PR 15: `log` Mutations

## Goal

Extend the existing `log` command family with create and delete commands. The existing `rp get log` command (in `log.go`) is left unchanged. The create command uses `--from-file` because `SaveLogRQ` contains structured fields including a `time.Time` timestamp. An `--async` flag selects between the synchronous and asynchronous create endpoints.

## Context

The `log` resource maps to `LogScope` on the client (`client.Project(name).Logs()`). It is **project-scoped**, meaning mutation commands require a resolved project via `--project`, `RP_PROJECT`, or the profile's `default-project`.

**Prerequisite:** PRs 1–7 must land first (parent command split, verb scaffolding, centralized registration, all three pilots, and shared helper extraction).

### Client surface (mutations only)

| Method | Parameters | Returns |
|--------|------------|---------|
| `Create` | `ctx`, `request types.SaveLogRQ` | `(*types.EntryCreatedAsyncRS, error)` |
| `CreateAsync` | `ctx`, `request types.SaveLogRQ` | `(*types.EntryCreatedAsyncRS, error)` |
| `Delete` | `ctx`, `logID int64` | `(*types.OperationCompletionRS, error)` |

**Notable:** Both `Create` and `CreateAsync` accept the same request and return the same response type. The difference is which API endpoint they hit (synchronous v1 vs asynchronous v2). `EntryCreatedAsyncRS` has `Id *string` (a string, not int64) — this differs from `EntryCreatedRS` used by other resources.

### Key types

```go
type SaveLogRQ struct {
    File       *File     `json:"file,omitempty"`
    ItemUuid   *string   `json:"itemUuid,omitempty"`
    LaunchUuid string    `json:"launchUuid"`
    Level      *string   `json:"level,omitempty"`
    Message    *string   `json:"message,omitempty"`
    Time       time.Time `json:"time"`
    Uuid       *string   `json:"uuid,omitempty"`
}

type File struct {
    Name *string `json:"name,omitempty"`
}

type EntryCreatedAsyncRS struct {
    Id *string `json:"id,omitempty"`
}
```

**`Time` is `time.Time`.** Go's `json.Unmarshal` parses this from RFC 3339 format (e.g., `"2024-01-15T10:30:00Z"`). The JSON file must use this format for the `time` field.

**`LaunchUuid` is required** (non-pointer `string`). The log entry must be associated with a launch.

## Scope

Extend `internal/cli/log.go` with two new command constructors and their run functions. Create `internal/cli/log_mutation_test.go` for tests covering the new commands. Register the new commands in `internal/cli/commands.go`.

**Important:** Use a separate test file (`log_mutation_test.go`) rather than adding tests to `log_test.go` to minimize conflict with the existing `get log` test surface.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get log` | *(existing — unchanged)* |
| `rp get log <id-or-uuid>` | *(existing — unchanged)* |
| `rp create log --from-file <path> [--async]` | Create a log entry |
| `rp delete log <id>` | Delete a log entry by numeric ID |

## Implementation

### 1. Add commands to `internal/cli/log.go`

Add two new command constructors and their run functions to the existing file. Do NOT modify any existing functions in `log.go`.

#### `newCreateLogCmd()`

```go
func newCreateLogCmd() *cobra.Command
```

- `Use`: `"log"`
- `Short`: `"Create a log entry"`
- `Long`: Include examples showing `--from-file` usage and the `--async` flag. Include a sample JSON file demonstrating the required fields (`launchUuid`, `time`) and optional fields (`itemUuid`, `level`, `message`). Note that `time` must be in RFC 3339 format.

Example help text JSON:

```
  Example JSON file:
  {
    "launchUuid": "550e8400-e29b-41d4-a716-446655440000",
    "time": "2024-01-15T10:30:00Z",
    "level": "ERROR",
    "message": "Test assertion failed: expected 200, got 500",
    "itemUuid": "660e8400-e29b-41d4-a716-446655440001"
  }
```

- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateLog)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--from-file` / `-f` (string, required): path to JSON file containing `SaveLogRQ`
  - `--async` (bool, default `false`): use the asynchronous create endpoint

The run function:
1. Validate `--from-file` is non-empty
2. Read and unmarshal the file using `readRequestFromFile[types.SaveLogRQ](path)`
3. Resolve connection, project, and client
4. Read `--async` flag
5. If `--async` is true, call `scope.CreateAsync(ctx, request)`. Otherwise call `scope.Create(ctx, request)`.
6. Print the `EntryCreatedAsyncRS` result

**Required field validation is deferred to the API.** `SaveLogRQ` has two required fields (`LaunchUuid string` and `Time time.Time`), but the CLI does not validate them locally. This is consistent with the general CLI philosophy — the CLI passes through user input and lets the API reject invalid payloads with a descriptive error. Go's zero values (empty string for `LaunchUuid`, zero time for `Time`) will serialize to JSON and the API will return an error if they are unacceptable. The help text and example JSON should make the required fields clear so users get it right on the first try.

**Result formatting for `EntryCreatedAsyncRS`:** This type has `Id *string` (not `*int64`), so it cannot use the shared `entryCreatedHeaders`/`entryCreatedRows` helpers. Define a local formatter:

```go
func logEntryCreatedHeaders() []string {
    return []string{"ID"}
}

func logEntryCreatedRows(result *types.EntryCreatedAsyncRS) [][]string {
    id := ""
    if result.Id != nil {
        id = *result.Id
    }

    return [][]string{{id}}
}
```

Print using: `printer.Print(result, logEntryCreatedHeaders(), logEntryCreatedRows(result))`

#### `newDeleteLogCmd()`

```go
func newDeleteLogCmd() *cobra.Command
```

- `Use`: `"log <id>"`
- `Short`: `"Delete a log entry"`
- `Long`: Include example showing delete by numeric ID
- `Args`: `cobra.ExactArgs(1)` (the numeric log ID)
- `RunE`: `withPrintedErrors(runDeleteLog)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "log")`
2. Resolve connection, project, and client
3. Call `scope.Delete(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// create
create.AddCommand(newCreateLogCmd())

// delete
del.AddCommand(newDeleteLogCmd())
```

### 3. Create `internal/cli/log_mutation_test.go`

Use a separate test file to avoid conflicts with the existing `log_test.go`:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewCreateLogCmd_Flags`** — Verify create has `--from-file` / `-f`, `--async`, and `--project` / `-p` flags. Verify `--async` defaults to `false`.

2. **`TestNewCreateLogCmd_NoArgs`** — Verify create accepts no positional arguments (`cobra.NoArgs`).

3. **`TestNewCreateLogCmd_NoGetFlags`** — Verify create does NOT have `--page-size`, `--page`, `--sort`, or any of the log-specific filter flags (`--item`, `--launch`, `--level`).

4. **`TestLogEntryCreatedRows_WithID`** — Verify `logEntryCreatedRows` produces the expected row for a non-nil string ID.

5. **`TestLogEntryCreatedRows_NilID`** — Verify `logEntryCreatedRows` produces an empty string for a nil ID.

6. **`TestNewDeleteLogCmd_Flags`** — Verify delete has `--project` / `-p` flag. Verify it accepts exactly 1 arg.

7. **`TestNewDeleteLogCmd_NoFromFile`** — Verify delete does NOT have a `--from-file` flag.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Extends existing file.** Both commands are added to `log.go`. The existing `newGetLogCmd`, `runGetLog`, `getSingleLog`, `listLogs`, `logRow`, `logHeaders`, `logFilterFlags`, and all filter/option functions remain untouched.
- **`--from-file` for create.** `SaveLogRQ` contains `time.Time` (which requires RFC 3339 formatting) and structured fields. This is not amenable to inline flags — the time format is particularly error-prone from the command line. The CLI plan's decision rules recommend `--from-file` for complex payloads.
- **`--async` flag.** Both `Create` and `CreateAsync` have identical signatures. The `--async` flag is a simple boolean that selects the endpoint. This is the lightest-weight approach — no separate subcommand needed.
- **`EntryCreatedAsyncRS` vs `EntryCreatedRS`.** The create methods return `EntryCreatedAsyncRS` which has `Id *string`, not `Id *int64`. This means the shared `entryCreatedHeaders`/`entryCreatedRows` helpers (designed for `EntryCreatedRS`) cannot be used. A local formatter pair is needed.
- **Numeric ID only for delete.** `LogScope.Delete` takes `int64`. The `Use` field and help text should make clear that a numeric ID is required (not a UUID).
- **Batch create is out of scope.** `LogScope.CreateBatch` and `CreateBatchAsync` use multipart form encoding for binary attachments (`BatchLogEntry` with `AttachmentData []byte`). This is a fundamentally different input model that requires separate CLI treatment (likely a dedicated `rp create log-batch` command or `--batch` flag with a different file format). It is explicitly a stretch goal.
- **File attachment via `SaveLogRQ.File` is informational only.** The `File` field in `SaveLogRQ` only has a `Name *string` — it does not carry binary data. Actual file attachment requires the batch create API. The `File` field can be included in the JSON if needed, but users should be aware that it only sets metadata. This is documented in the help text if relevant.
- **Separate test file.** Using `log_mutation_test.go` avoids merge conflicts with the existing `log_test.go`, which tests the `get log` command surface (filter flags, log level validation, etc.).
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `readRequestFromFile[types.SaveLogRQ]`, `messageResultHeaders`/`operationResultRows`.

## What NOT to do

- **Do not modify any existing functions** in `log.go`. The `get log` surface (command, run function, filter flags, `parseOptionalInt64Flag`) must remain identical.
- **Do not implement `CreateBatch` or `CreateBatchAsync`.** Those are stretch goals requiring multipart form encoding for binary attachments.
- **Do not accept UUIDs for delete.** The API method takes `int64` only.
- **Do not use `entryCreatedHeaders`/`entryCreatedRows`** for the create result. `EntryCreatedAsyncRS.Id` is `*string`, not `*int64`. Use a local formatter.
- **Do not validate log levels on create.** The `get log` command validates `--level` for filtering, but `SaveLogRQ.Level` is a free-form string. Let the API validate — different Report Portal versions may support different level names.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp create --help` now shows `log` as a subcommand
- `rp delete --help` now shows `log` as a subcommand
- `rp create log --help` shows `--from-file`, `--async`, and `--project` flags
- `rp delete log --help` shows `--project` flag and expects a numeric ID arg
- `rp get log` and all existing log list/get/filter behavior is unchanged
- No existing commands are affected

## Definition of done

- [ ] `newCreateLogCmd` and `newDeleteLogCmd` added to `internal/cli/log.go`
- [ ] `internal/cli/log_mutation_test.go` created with unit tests for new commands
- [ ] Commands registered in `commands.go`
- [ ] Both commands use `--project` / `-p` for project resolution
- [ ] Create uses `--from-file` / `-f` for input
- [ ] Create supports `--async` flag to select asynchronous endpoint
- [ ] Create result formatted with local `logEntryCreatedHeaders`/`logEntryCreatedRows` (not shared `entryCreatedHeaders`)
- [ ] Delete takes a numeric log ID as positional arg
- [ ] Existing `get log` code is completely untouched
- [ ] Batch create (`CreateBatch`, `CreateBatchAsync`) is NOT implemented
- [ ] Shared helpers used: `parseResourceID`, `readRequestFromFile`, `messageResultHeaders`/`operationResultRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
