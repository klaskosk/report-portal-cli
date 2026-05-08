# PR 19: Lifecycle Commands

## Goal

Implement the `start` and `finish` lifecycle commands for launches and items. These are execution lifecycle operations, not resource CRUD — they use UUIDs for identification (not numeric IDs) and involve complex request payloads with timestamps, attributes, and status values.

## Context

Lifecycle commands span two scopes:
- `LaunchScope` (`client.Project(name).Launches()`): start, finish, force-finish
- `TestItemScope` (`client.Project(name).Items()`): start (root/child), finish

All lifecycle commands are **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 2 (verb group scaffolding: `start` and `finish` parent commands)
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `readRequestFromFile`, `mustGetStringFlag`, `optionalBoolFlag`, `entryCreatedHeaders`/`entryCreatedRows`)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `LaunchScope.Start` | `ctx`, `request types.StartLaunchRQ` | `(*types.StartLaunchRS, error)` |
| `LaunchScope.StartAsync` | `ctx`, `request types.StartLaunchRQ` | `(*types.StartLaunchRS, error)` |
| `LaunchScope.Finish` | `ctx`, `launchUUID string`, `request types.FinishExecutionRQ` | `(*types.FinishLaunchRS, error)` |
| `LaunchScope.FinishAsync` | `ctx`, `launchUUID string`, `request types.FinishExecutionRQ` | `(*types.FinishLaunchRS, error)` |
| `LaunchScope.ForceFinish` | `ctx`, `launchID int64`, `request types.FinishExecutionRQ` | `(*types.OperationCompletionRS, error)` |
| `TestItemScope.StartRoot` | `ctx`, `request types.StartTestItemRQ` | `(*types.EntryCreatedAsyncRS, error)` |
| `TestItemScope.StartChild` | `ctx`, `parentItemUUID string`, `request types.StartTestItemRQ` | `(*types.EntryCreatedAsyncRS, error)` |
| `TestItemScope.StartRootAsync` | `ctx`, `request types.StartTestItemRQ` | `(*types.EntryCreatedAsyncRS, error)` |
| `TestItemScope.StartChildAsync` | `ctx`, `parentItemUUID string`, `request types.StartTestItemRQ` | `(*types.EntryCreatedAsyncRS, error)` |
| `TestItemScope.Finish` | `ctx`, `itemUUID string`, `request types.FinishTestItemRQ` | `(*types.OperationCompletionRS, error)` |
| `TestItemScope.FinishAsync` | `ctx`, `itemUUID string`, `request types.FinishTestItemRQ` | `(*types.OperationCompletionRS, error)` |

### Key types

```go
type StartLaunchRQ struct {
    Attributes  *[]ItemAttributesRQ `json:"attributes,omitempty"`
    Description *string             `json:"description,omitempty"`
    Mode        *StartLaunchRQMode  `json:"mode,omitempty"`
    Name        string              `json:"name"`
    Rerun       *bool               `json:"rerun,omitempty"`
    RerunOf     *string             `json:"rerunOf,omitempty"`
    StartTime   time.Time           `json:"startTime"`
    Uuid        string              `json:"uuid"`
}

type StartLaunchRS struct {
    Id     *string `json:"id,omitempty"`
    Number *int64  `json:"number,omitempty"`
}

type FinishExecutionRQ struct {
    Attributes  *[]ItemAttributesRQ      `json:"attributes,omitempty"`
    Description *string                  `json:"description,omitempty"`
    EndTime     time.Time                `json:"endTime"`
    Status      *FinishExecutionRQStatus `json:"status,omitempty"`
}

type FinishLaunchRS struct {
    Id     *string `json:"id,omitempty"`
    Link   *string `json:"link,omitempty"`
    Number *int64  `json:"number,omitempty"`
}

type StartTestItemRQ struct {
    Attributes  *[]ItemAttributesRQ `json:"attributes,omitempty"`
    CodeRef     *string             `json:"codeRef,omitempty"`
    Description *string             `json:"description,omitempty"`
    HasStats    *bool               `json:"hasStats,omitempty"`
    LaunchUuid  string              `json:"launchUuid"`
    Name        string              `json:"name"`
    Parameters  *[]ParameterResource `json:"parameters,omitempty"`
    Retry       *bool                `json:"retry,omitempty"`
    RetryOf     *string              `json:"retryOf,omitempty"`
    StartTime   time.Time            `json:"startTime"`
    TestCaseId  *string              `json:"testCaseId,omitempty"`
    Type        StartTestItemRQType  `json:"type"`
    UniqueId    *string              `json:"uniqueId,omitempty"`
    Uuid        string               `json:"uuid"`
}

type FinishTestItemRQ struct {
    Attributes  *[]ItemAttributesRQ     `json:"attributes,omitempty"`
    Description *string                 `json:"description,omitempty"`
    EndTime     time.Time               `json:"endTime"`
    Issue       *Issue                  `json:"issue,omitempty"`
    LaunchUuid  string                  `json:"launchUuid"`
    Retry       *bool                   `json:"retry,omitempty"`
    RetryOf     *string                 `json:"retryOf,omitempty"`
    Status      *FinishTestItemRQStatus `json:"status,omitempty"`
    TestCaseId  *string                 `json:"testCaseId,omitempty"`
}

type EntryCreatedAsyncRS struct {
    Id *string `json:"id,omitempty"`
}
```

`FinishExecutionRQStatus` values: `PASSED`, `FAILED`, `STOPPED`, `SKIPPED`, `INTERRUPTED`, `CANCELLED`, `INFO`, `WARN`.
`StartTestItemRQType` values: `SUITE`, `STORY`, `TEST`, `SCENARIO`, `STEP`, `BEFORE_CLASS`, `BEFORE_GROUPS,BEFORE_METHOD`, `BEFORE_SUITE`, `BEFORE_TEST`, `AFTER_CLASS`, `AFTER_GROUPS`, `AFTER_METHOD`, `AFTER_SUITE`, `AFTER_TEST`.

**Note:** `BEFORE_GROUPS,BEFORE_METHOD` is a **single** enum value (the string literal includes the comma). In the generated Go types it is the constant `BEFOREGROUPSBEFOREMETHOD`. It is NOT two separate values. When a user specifies `"type": "BEFORE_GROUPS,BEFORE_METHOD"` in JSON, it must be treated as one atomic string.

## Scope

Create `internal/cli/lifecycle.go` and `internal/cli/lifecycle_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp start launch --file <path> [--async]` | Start a new launch |
| `rp finish launch <uuid> --file <path> [--async]` | Finish a launch by UUID |
| `rp finish launch <id> --file <path> --force` | Force-finish a launch by numeric ID |
| `rp start item --file <path> [--parent <uuid>] [--async]` | Start a test item (root or child) |
| `rp finish item <uuid> --file <path> [--async]` | Finish a test item by UUID |

## Implementation

### 1. Create `internal/cli/lifecycle.go`

This file contains all command constructors and run functions for lifecycle operations.

#### `newStartLaunchCmd()`

```go
func newStartLaunchCmd() *cobra.Command
```

- `Use`: `"launch"`
- `Short`: `"Start a new launch"`
- `Long`: Include examples showing `--file` and `--async`. Include sample JSON with required fields (`name`, `startTime`, `uuid`) and optional fields (`description`, `mode`, `attributes`). Note that `startTime` must be RFC 3339 format and `uuid` must be a valid UUID.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runStartLaunch)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `StartLaunchRQ`
  - `--async` (bool, default `false`): use the asynchronous start endpoint

The run function:
1. Validate `--file` is non-empty
2. Read and unmarshal using `readRequestFromFile[types.StartLaunchRQ](path)`
3. Resolve connection, project, and client
4. If `--async`, call `scope.StartAsync(ctx, request)`. Otherwise call `scope.Start(ctx, request)`.
5. Print via `printer.Print(result, startLaunchHeaders(), startLaunchRows(result))`

**Result formatting for `StartLaunchRS`:**

```go
func startLaunchHeaders() []string {
    return []string{"ID", "NUMBER"}
}

func startLaunchRows(result *types.StartLaunchRS) [][]string {
    id := ""
    if result.Id != nil {
        id = *result.Id
    }

    number := ""
    if result.Number != nil {
        number = strconv.FormatInt(*result.Number, 10)
    }

    return [][]string{{id, number}}
}
```

#### `newFinishLaunchCmd()`

```go
func newFinishLaunchCmd() *cobra.Command
```

- `Use`: `"launch <identifier>"`
- `Short`: `"Finish or force-finish a launch"`
- `Long`: Include examples showing normal finish (by UUID), async finish, and force-finish (by numeric ID). Explain that `--force` changes the expected identifier from UUID to numeric ID.
- `Args`: `cobra.ExactArgs(1)` (UUID for normal finish, numeric ID for force finish)
- `RunE`: `withPrintedErrors(runFinishLaunch)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `FinishExecutionRQ`
  - `--async` (bool, default `false`): use the asynchronous finish endpoint (incompatible with `--force`)
  - `--force` (bool, default `false`): force-finish by numeric launch ID (incompatible with `--async`)

The run function:
1. Validate `--file` is non-empty
2. Read and unmarshal using `readRequestFromFile[types.FinishExecutionRQ](path)`
3. Validate `--force` and `--async` are not both set:
   ```go
   if force && async {
       return fmt.Errorf("--force and --async are mutually exclusive")
   }
   ```
4. Resolve connection, project, and client
5. Branch on `--force`:
   - **Force finish:** Parse `args[0]` as int64 using `parseResourceID(args[0], "launch")`. Call `scope.ForceFinish(ctx, id, request)`. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`.
   - **Normal finish:** Use `args[0]` as the UUID string. If `--async`, call `scope.FinishAsync(ctx, uuid, request)`. Otherwise call `scope.Finish(ctx, uuid, request)`. Print via `printer.Print(result, finishLaunchHeaders(), finishLaunchRows(result))`.

**Result formatting for `FinishLaunchRS`:**

```go
func finishLaunchHeaders() []string {
    return []string{"ID", "LINK", "NUMBER"}
}

func finishLaunchRows(result *types.FinishLaunchRS) [][]string {
    id := ""
    if result.Id != nil {
        id = *result.Id
    }

    link := ""
    if result.Link != nil {
        link = *result.Link
    }

    number := ""
    if result.Number != nil {
        number = strconv.FormatInt(*result.Number, 10)
    }

    return [][]string{{id, link, number}}
}
```

#### `newStartItemCmd()`

```go
func newStartItemCmd() *cobra.Command
```

- `Use`: `"item"`
- `Short`: `"Start a test item"`
- `Long`: Include examples showing root item start, child item start (with `--parent`), and async mode. Include sample JSON with required fields (`name`, `startTime`, `uuid`, `launchUuid`, `type`) and optional fields. Note that `type` must be one of the valid `StartTestItemRQType` values.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runStartItem)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `StartTestItemRQ`
  - `--parent` (string, optional): parent item UUID (if set, starts a child item; if omitted, starts a root item)
  - `--async` (bool, default `false`): use the asynchronous start endpoint

The run function:
1. Validate `--file` is non-empty
2. Read and unmarshal using `readRequestFromFile[types.StartTestItemRQ](path)`
3. Resolve connection, project, and client
4. Read `--parent` flag
5. Branch on parent and async:
   - If `--parent` is set and `--async`: call `scope.StartChildAsync(ctx, parentUUID, request)`
   - If `--parent` is set and not `--async`: call `scope.StartChild(ctx, parentUUID, request)`
   - If `--parent` is not set and `--async`: call `scope.StartRootAsync(ctx, request)`
   - If `--parent` is not set and not `--async`: call `scope.StartRoot(ctx, request)`
6. Print via `printer.Print(result, asyncEntryCreatedHeaders(), asyncEntryCreatedRows(result))`

**Result formatting for `EntryCreatedAsyncRS`:** Define a local formatter in `lifecycle.go`. Note that PR 7's shared `entryCreatedHeaders`/`entryCreatedRows` are for `EntryCreatedRS` (which has `Id *int64`), **not** `EntryCreatedAsyncRS` (which has `Id *string`). These are different types — do not reuse the shared helpers for this case.

```go
func asyncEntryCreatedHeaders() []string {
    return []string{"ID"}
}

func asyncEntryCreatedRows(result *types.EntryCreatedAsyncRS) [][]string {
    id := ""
    if result.Id != nil {
        id = *result.Id
    }

    return [][]string{{id}}
}
```

If PR 16 has explicitly extracted a shared version for `EntryCreatedAsyncRS`, use that. Otherwise keep this local.

#### `newFinishItemCmd()`

```go
func newFinishItemCmd() *cobra.Command
```

- `Use`: `"item <uuid>"`
- `Short`: `"Finish a test item"`
- `Long`: Include examples showing `--file` with a UUID argument. Include sample JSON with `endTime`, `launchUuid`, and optional `status`, `issue`, `attributes`.
- `Args`: `cobra.ExactArgs(1)` (the item UUID)
- `RunE`: `withPrintedErrors(runFinishItem)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `FinishTestItemRQ`
  - `--async` (bool, default `false`): use the asynchronous finish endpoint

The run function:
1. Use `args[0]` as the item UUID string (no parsing — UUIDs are strings)
2. Validate `--file` is non-empty
3. Read and unmarshal using `readRequestFromFile[types.FinishTestItemRQ](path)`
4. Resolve connection, project, and client
5. If `--async`, call `scope.FinishAsync(ctx, uuid, request)`. Otherwise call `scope.Finish(ctx, uuid, request)`.
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// start
start.AddCommand(newStartLaunchCmd())
start.AddCommand(newStartItemCmd())

// finish
finish.AddCommand(newFinishLaunchCmd())
finish.AddCommand(newFinishItemCmd())
```

### 3. Create `internal/cli/lifecycle_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewStartLaunchCmd_Flags`** — Verify start launch has `--file` / `-f`, `--async`, and `--project` / `-p` flags.

2. **`TestNewStartLaunchCmd_NoArgs`** — Verify `Args` is `cobra.NoArgs`.

3. **`TestNewFinishLaunchCmd_Flags`** — Verify finish launch has `--file` / `-f`, `--async`, `--force`, and `--project` / `-p` flags. Verify it accepts exactly 1 arg.

4. **`TestNewStartItemCmd_Flags`** — Verify start item has `--file` / `-f`, `--async`, `--parent`, and `--project` / `-p` flags.

5. **`TestNewStartItemCmd_NoArgs`** — Verify `Args` is `cobra.NoArgs`.

6. **`TestNewFinishItemCmd_Flags`** — Verify finish item has `--file` / `-f`, `--async`, and `--project` / `-p` flags. Verify it accepts exactly 1 arg. Verify it does NOT have `--force`.

7. **`TestFinishLaunchCmd_ForceAsyncMutualExclusion`** — Create the command, set both `--force` and `--async`, verify the run function returns a descriptive error about mutual exclusivity.

8. **`TestStartLaunchRows_WithValues`** — Verify `startLaunchRows` produces the expected row for a fully-populated `StartLaunchRS`.

9. **`TestStartLaunchRows_NilFields`** — Verify `startLaunchRows` handles nil fields.

10. **`TestFinishLaunchRows_WithValues`** — Verify `finishLaunchRows` produces the expected row for a fully-populated `FinishLaunchRS`.

11. **`TestFinishLaunchRows_NilFields`** — Verify `finishLaunchRows` handles nil fields.

12. **`TestAsyncEntryCreatedRows_WithID`** — Verify `asyncEntryCreatedRows` produces the expected row for a non-nil string ID.

13. **`TestAsyncEntryCreatedRows_NilID`** — Verify `asyncEntryCreatedRows` produces an empty string for nil.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **All lifecycle payloads use `--file`.** Per PR 17's frozen UX, `--file` is used when the file is the command's primary request-body input mode, while `--from-file` is reserved for cases where a file is just one alternative among multiple construction modes. These lifecycle commands still take positional UUIDs to identify the target launch or item, but the request body itself always comes from disk, so `--file` is the correct user-facing flag name. Keep the `-f` short form.
- **`--async` selects the async endpoint.** Both sync and async variants have identical signatures and return types. A boolean flag is the simplest approach.
- **`--force` on finish launch changes the identifier type.** Normal finish uses a UUID string; force-finish uses a numeric ID. The `--force` flag makes this explicit and the help text documents the difference. `--force` and `--async` are mutually exclusive because `ForceFinish` has no async variant.
- **`--parent` on start item selects root vs child.** If `--parent <uuid>` is provided, the command uses `StartChild` / `StartChildAsync`. If omitted, it uses `StartRoot` / `StartRootAsync`. This is more ergonomic than separate `rp start item-root` and `rp start item-child` subcommands.
- **UUID args are not parsed.** UUIDs are passed through as strings. The API will validate UUID format server-side.
- **Response types differ by operation.** `StartLaunchRS`, `FinishLaunchRS`, `EntryCreatedAsyncRS`, and `OperationCompletionRS` each need their own formatters. Shared formatters from PR 7 cover `OperationCompletionRS` (`messageResultHeaders`/`operationResultRows`) and `EntryCreatedRS` (`entryCreatedHeaders`/`entryCreatedRows`). However, `EntryCreatedAsyncRS` (string ID) is **not** the same as `EntryCreatedRS` (int64 ID) — define `asyncEntryCreatedHeaders`/`asyncEntryCreatedRows` locally. Local formatters also cover `StartLaunchRS` and `FinishLaunchRS`.
- **Route all output through `printer.Print`.** Use `printer.Print(result, headers, rows)` for single results and slices alike. This ensures `-o json`, `-o go-template`, and table output all work correctly. Errors use `printer.PrintError(err)` before returning.
- **`BulkForceFinish` is out of scope.** It is explicitly a stretch goal in the implementation plan.
- **Reuse shared helpers from PR 7.** Use `parseResourceID` (for force-finish ID), `readRequestFromFile`, `messageResultHeaders`/`operationResultRows`, `withPrintedErrors`.
- **Command registration follows PR 3's centralized command-tree layout.** All subcommands are registered in `buildCommandTree` in `commands.go`.

## What NOT to do

- **Do not implement `BulkForceFinish`.** That is a stretch goal.
- **Do not modify existing launch or item commands** (`get launch`, `get item`, `update launch`, `delete launch`, etc.).
- **Do not validate `StartTestItemRQType` values locally.** Let the API validate — the enum values may change between versions.
- **Do not validate UUID format locally.** Pass through to the API.
- **Do not validate `time.Time` fields locally.** `json.Unmarshal` handles RFC 3339 parsing and will produce a clear error for invalid formats.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp start --help` shows `launch` and `item` as subcommands
- `rp finish --help` shows `launch` and `item` as subcommands
- `rp start launch --help` shows `--file`, `--async`, `--project` flags
- `rp finish launch --help` shows `--file`, `--async`, `--force`, `--project` flags
- `rp start item --help` shows `--file`, `--async`, `--parent`, `--project` flags
- `rp finish item --help` shows `--file`, `--async`, `--project` flags (no `--force`)
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/lifecycle.go` created with all command constructors and run functions
- [ ] `internal/cli/lifecycle_test.go` created with unit tests
- [ ] Commands registered under `start` and `finish` parent commands in `commands.go`
- [ ] All commands use `--project` / `-p` for project resolution
- [ ] All commands use `--file` / `-f` for input
- [ ] `--async` flag selects async endpoint variants
- [ ] `--force` on finish launch switches from UUID to numeric ID (mutually exclusive with `--async`)
- [ ] Mutual exclusivity of `--force` and `--async` is tested
- [ ] `--parent` on start item switches from root to child item
- [ ] Result formatters handle all response types: `StartLaunchRS`, `FinishLaunchRS`, `EntryCreatedAsyncRS`, `OperationCompletionRS`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
