# PR 12: `log` Mutations

## Goal

Extend the existing `rp log` command with `create` and `delete` subcommands while leaving the existing `list` and `get` intact. The create command supports four variants: single sync, single async, batch sync, and batch async.

## Context

This PR runs in the Phase 4 parallel fan-out. PRs 1–5 have landed. After PR 1, the log resource has been refactored to noun-first with `rp log list` and `rp log get <id-or-uuid>`.

Logs are project-scoped. The create command is more complex than typical CRUD because the API exposes four creation methods with different semantics:

| Variant | Client method | Input type | Response type |
|---------|--------------|------------|---------------|
| Single sync | `Create(ctx, SaveLogRQ)` | `types.SaveLogRQ` | `*types.EntryCreatedAsyncRS` |
| Single async | `CreateAsync(ctx, SaveLogRQ)` | `types.SaveLogRQ` | `*types.EntryCreatedAsyncRS` |
| Batch sync | `CreateBatch(ctx, ...BatchLogEntry)` | `client.BatchLogEntry` | `*types.BatchSaveOperatingRS` |
| Batch async | `CreateBatchAsync(ctx, ...BatchLogEntry)` | `client.BatchLogEntry` | `*types.BatchSaveOperatingRS` |

The `--async` and `--batch` flags select the variant.

### Prerequisites

- PRs 1–5 have landed
- `internal/cli/log.go` already defines `newLogCmd()` with `list` and `get` subcommands

### Client API surface

`LogScope` (obtained via `client.Project(projectName).Logs()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `Create` | `Create(ctx, types.SaveLogRQ)` | `*types.EntryCreatedAsyncRS` |
| `CreateAsync` | `CreateAsync(ctx, types.SaveLogRQ)` | `*types.EntryCreatedAsyncRS` |
| `CreateBatch` | `CreateBatch(ctx, ...client.BatchLogEntry)` | `*types.BatchSaveOperatingRS` |
| `CreateBatchAsync` | `CreateBatchAsync(ctx, ...client.BatchLogEntry)` | `*types.BatchSaveOperatingRS` |
| `Delete` | `Delete(ctx, logID int64)` | `*types.OperationCompletionRS` |

### Relevant types

```go
// API type for single log creation
type SaveLogRQ struct {
    File       *File     `json:"file,omitempty"`
    ItemUuid   *string   `json:"itemUuid,omitempty"`
    LaunchUuid string    `json:"launchUuid"`
    Level      *string   `json:"level,omitempty"`
    Message    *string   `json:"message,omitempty"`
    Time       time.Time `json:"time"`
    Uuid       *string   `json:"uuid,omitempty"`
}

// Client-side type for batch creation (not a generated type)
type BatchLogEntry struct {
    LaunchUUID     string
    ItemUUID       string
    Level          string
    Message        string
    Time           time.Time
    AttachmentName string
    AttachmentData []byte
}

// Single create response
type EntryCreatedAsyncRS struct {
    Id *string `json:"id,omitempty"`
}

// Batch create response
type BatchSaveOperatingRS struct {
    Responses *[]BatchElementCreatedRS `json:"responses,omitempty"`
}

type BatchElementCreatedRS struct {
    Id         *string `json:"id,omitempty"`
    Message    *string `json:"message,omitempty"`
    StackTrace *string `json:"stackTrace,omitempty"`
}

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

## Implementation plan

### 1. Extend `internal/cli/log.go`

Add two new command constructors and register them in the existing `newLogCmd()` parent. **Do not modify the existing `list` and `get` subcommands or any existing helpers.**

Add to the existing `newLogCmd()`:
```go
cmd.AddCommand(newLogCreateCmd())
cmd.AddCommand(newLogDeleteCmd())
```

#### `rp log create`

- No positional arguments
- **Input: `--from-file` required for all variants**
- Flags:
  - `--from-file <path>` (string, required) — path to a JSON file
  - `--async` (bool, default false) — use the async API variant
  - `--batch` (bool, default false) — interpret the file as a batch of log entries
- Mark `--from-file` as required via `cmd.MarkFlagRequired("from-file")`
- Variant selection based on flag combination:
  - Neither `--batch` nor `--async`: single sync → `scope.Create`
  - `--async` only: single async → `scope.CreateAsync`
  - `--batch` only: batch sync → `scope.CreateBatch`
  - `--batch --async`: batch async → `scope.CreateBatchAsync`

**Single mode (`--batch` absent):**
- Read file with `readRequestFromFile[types.SaveLogRQ](path)`
- Validate: `LaunchUuid` must be non-empty, `Time` must not be zero
- Call appropriate single-create method
- Table output: `ID` column (from `EntryCreatedAsyncRS.Id`)
- JSON output: the raw `*EntryCreatedAsyncRS`

**Batch mode (`--batch` present):**
- Read file as a JSON array of batch entry objects (see format below)
- Parse into `[]client.BatchLogEntry` — see batch entry JSON format section
- Validate: at least one entry, each entry must have non-empty `LaunchUUID` and non-zero `Time`
- Call appropriate batch-create method with entries as variadic args
- Table output: `ID`, `MESSAGE` columns (one row per `BatchElementCreatedRS` in the response)
- JSON output: the raw `*BatchSaveOperatingRS`

#### `rp log delete <id>`

- Exactly one positional argument: numeric log ID
- Parse using `parseIDArg(args[0], "log")`
- No additional flags
- Call `scope.Delete(ctx, logID)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

### 2. Batch entry JSON format

Since `BatchLogEntry` is a client-side struct (not a generated API type), define a JSON-compatible representation for the `--from-file` input:

```go
type batchLogEntryJSON struct {
    LaunchUUID string    `json:"launchUuid"`
    ItemUUID   string    `json:"itemUuid,omitempty"`
    Level      string    `json:"level,omitempty"`
    Message    string    `json:"message,omitempty"`
    Time       time.Time `json:"time"`
}
```

The `--from-file` content for batch mode is a JSON array of these objects:

```json
[
  {
    "launchUuid": "abc-123",
    "level": "INFO",
    "message": "Test started",
    "time": "2024-01-15T10:30:00Z"
  },
  {
    "launchUuid": "abc-123",
    "itemUuid": "def-456",
    "level": "ERROR",
    "message": "Assertion failed",
    "time": "2024-01-15T10:31:00Z"
  }
]
```

Convert each `batchLogEntryJSON` to `client.BatchLogEntry`:

```go
func toBatchLogEntry(entry batchLogEntryJSON) client.BatchLogEntry {
    return client.BatchLogEntry{
        LaunchUUID: entry.LaunchUUID,
        ItemUUID:   entry.ItemUUID,
        Level:      entry.Level,
        Message:    entry.Message,
        Time:       entry.Time,
    }
}
```

**Attachments are not supported via CLI.** The `AttachmentName` and `AttachmentData` fields of `BatchLogEntry` are left empty. File attachments through the CLI would require a different UX (multiple file paths, MIME types, etc.) and are deferred as a stretch goal.

### 3. Table formatting for batch response

```go
func batchLogHeaders() []string {
    return []string{"ID", "MESSAGE"}
}

func batchLogRows(result *types.BatchSaveOperatingRS) [][]string {
    if result.Responses == nil {
        return nil
    }

    rows := make([][]string, 0, len(*result.Responses))
    for _, r := range *result.Responses {
        rows = append(rows, []string{
            formatStringPtr(r.Id),
            formatStringPtr(r.Message),
        })
    }

    return rows
}
```

### 4. No changes to root.go

The `newLogCmd()` function is already registered in `root.go` from PR 1.

## Test requirements

### Unit tests (add to `internal/cli/log_test.go` or a new section)

- **Delete ID parsing:** Verify that `rp log delete abc` returns an error
- **Create flag validation:**
  - Missing `--from-file` errors
  - Default (no `--batch`, no `--async`) selects single sync mode
  - `--async` alone selects single async mode
  - `--batch` alone selects batch sync mode
  - `--batch --async` selects batch async mode
- **Single mode validation:**
  - Valid `SaveLogRQ` JSON parses correctly
  - Missing `launchUuid` errors
  - Zero time errors
  - Malformed JSON errors
- **Batch mode validation:**
  - Valid batch array JSON parses correctly into `[]batchLogEntryJSON`
  - Empty array errors (at least one entry required)
  - Entry with missing `launchUuid` errors
  - Entry with zero time errors
  - Non-array JSON in batch mode errors
- **Batch conversion:** Verify `toBatchLogEntry` correctly maps all fields
- **Batch table formatting:** Verify `batchLogRows` handles nil `Responses`, single entry, and multiple entries
- **Command wiring:** Verify `newLogCmd()` now includes create and delete alongside list and get

## Edge cases and constraints

- **Extend, do not rewrite:** Do NOT modify existing functions (`listLogs`, `getSingleLog`, `logRow`, log filter flags, etc.).
- **`--from-file` is always required for create.** Log creation always requires structured data (timestamps, UUIDs). Inline flag alternatives would be cumbersome and error-prone. All four variants use the same `--from-file` flag.
- **`EntryCreatedAsyncRS.Id` is `*string`, not `*int64`:** Unlike most create responses, the log entry ID is a string (UUID). Table formatting should use `formatStringPtr`, not `formatInt64Ptr`.
- **`BatchLogEntry` is a client type, not generated:** It lives in `pkg/client/log.go`, not `pkg/types/`. Import it as `client.BatchLogEntry`.
- **Time parsing:** JSON `time.Time` fields are expected in RFC 3339 format (e.g., `"2024-01-15T10:30:00Z"`). Go's `json.Unmarshal` handles this natively. A zero time (`time.Time{}`) indicates the field was missing or empty.
- **Attachments are not supported:** The batch entry JSON format intentionally omits `AttachmentName` and `AttachmentData`. Document this in the command's help text.
- **Batch entries are variadic:** `CreateBatch` and `CreateBatchAsync` accept `...BatchLogEntry`. Convert the slice with `entries...`.
- **`SaveLogRQ.File` field:** This field exists in the type but is not useful for CLI JSON input. The `File` type only has a `Name *string` field. Users can include it in their JSON if needed, but it will be passed through unchanged.
- **Project flag is already on the parent:** Inherited from `newLogCmd()`.
- **`SaveLogRQ.Uuid` field:** This is an optional client-generated UUID for deduplication. Users may include it in their JSON file. Do not auto-generate it.

## Example commands and expected output

```bash
# Create a single log entry (sync)
$ rp log create --from-file log_entry.json -p my_project
ID
550e8400-e29b-41d4-a716-446655440000

# Create a single log entry (async)
$ rp log create --from-file log_entry.json --async -p my_project
ID
550e8400-e29b-41d4-a716-446655440000

# Create batch log entries (sync)
$ rp log create --batch --from-file batch_entries.json -p my_project
ID                                      MESSAGE
550e8400-e29b-41d4-a716-446655440001
550e8400-e29b-41d4-a716-446655440002

# Create batch log entries (async)
$ rp log create --batch --async --from-file batch_entries.json -p my_project
ID                                      MESSAGE
550e8400-e29b-41d4-a716-446655440001
550e8400-e29b-41d4-a716-446655440002

# Delete
$ rp log delete 42 -p my_project
MESSAGE
Log entry with ID = '42' has been successfully deleted.
```

Example `log_entry.json` (single):
```json
{
  "launchUuid": "550e8400-e29b-41d4-a716-446655440000",
  "itemUuid": "661f9500-f30c-52e5-b827-557766551111",
  "level": "ERROR",
  "message": "java.lang.NullPointerException at com.example.Test.run(Test.java:42)",
  "time": "2024-01-15T10:30:00Z"
}
```

Example `batch_entries.json` (batch):
```json
[
  {
    "launchUuid": "550e8400-e29b-41d4-a716-446655440000",
    "level": "INFO",
    "message": "Test suite started",
    "time": "2024-01-15T10:30:00Z"
  },
  {
    "launchUuid": "550e8400-e29b-41d4-a716-446655440000",
    "itemUuid": "661f9500-f30c-52e5-b827-557766551111",
    "level": "ERROR",
    "message": "Assertion failed: expected 200, got 500",
    "time": "2024-01-15T10:31:15Z"
  }
]
```

## Validation

Run before finalizing:

```
make lint
make test
```

## Definition of done

- [ ] `rp log create --from-file` with `--async` and `--batch` flags for all four variants
- [ ] `rp log delete <id>` implemented
- [ ] Existing `list` and `get` subcommands unchanged
- [ ] Single mode: reads `SaveLogRQ` JSON, validates `launchUuid` and `time`
- [ ] Batch mode: reads JSON array, converts to `[]client.BatchLogEntry`, validates entries
- [ ] Batch response table with ID and MESSAGE columns
- [ ] Attachments explicitly not supported (documented in help text)
- [ ] Unit tests cover variant selection, single/batch validation, batch conversion, and command wiring
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
