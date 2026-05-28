# PR 17: Lifecycle Commands

## Goal

Add `rp launch start`, `rp launch finish`, `rp item start`, and `rp item finish` subcommands in separate sub-files (`launch_lifecycle.go`, `item_lifecycle.go`) to avoid conflicts with other Phase 7 PRs that extend the same resources.

## Context

This PR runs in Phase 7. Phases 1–6 and the Phase 6 shape freeze (PR 15) have landed. The launch and item parent commands already exist with list/get/update/delete subcommands.

### Prerequisites

- PRs 1–15 have landed
- `internal/cli/launch.go` defines `newLaunchCmd()` with list, get, update, delete, report
- `internal/cli/item.go` defines `newItemCmd()` with list, get, update, delete

### Client API surface

**LaunchScope** (via `client.Project(name).Launches()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `Start` | `Start(ctx, types.StartLaunchRQ)` | `*types.StartLaunchRS` |
| `Finish` | `Finish(ctx, launchUUID string, types.FinishExecutionRQ)` | `*types.FinishLaunchRS` |

**TestItemScope** (via `client.Project(name).TestItems()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `StartRoot` | `StartRoot(ctx, types.StartTestItemRQ)` | `*types.EntryCreatedAsyncRS` |
| `StartChild` | `StartChild(ctx, parentUUID string, types.StartTestItemRQ)` | `*types.EntryCreatedAsyncRS` |
| `Finish` | `Finish(ctx, itemUUID string, types.FinishTestItemRQ)` | `*types.OperationCompletionRS` |

### Relevant types

Look these up in `pkg/types/` — key fields:

- `StartLaunchRQ`: Name (required), StartTime, Description, Mode, Attributes, Uuid
- `FinishExecutionRQ`: EndTime, Status, Description, Attributes
- `StartLaunchRS`: Id, Number (the launch ID and number returned)
- `FinishLaunchRS`: Id, Number, Link
- `StartTestItemRQ`: Name (required), LaunchUuid, Type (SUITE/STORY/TEST/STEP/etc), StartTime, Description, Attributes, HasStats, Parameters
- `FinishTestItemRQ`: EndTime, Status, Description, Issue, Attributes
- `EntryCreatedAsyncRS`: Id
- `OperationCompletionRS`: Message

## Implementation plan

### 1. Create `internal/cli/launch_lifecycle.go`

This file defines `rp launch start` and `rp launch finish`.

#### `rp launch start`

- No positional arguments
- Required flag: `--name` (string) — launch name
- Optional flags:
  - `--description` (string) — launch description
  - `--mode` (string) — "DEFAULT" or "DEBUG" (case-insensitive, validate)
  - `--attribute` (string slice, repeatable) — `key:value` format
  - `--uuid` (string) — optional user-supplied UUID
  - `--from-file` (string) — reads full `StartLaunchRQ` from JSON; when provided, flags are ignored
- Build `types.StartLaunchRQ` from flags (StartTime set to `time.Now()` if not in file)
- Call `scope.Start(ctx, request)`
- Table output: ID, NUMBER columns from `StartLaunchRS`
- JSON output: raw `StartLaunchRS`

#### `rp launch finish <uuid>`

- Exactly one positional argument: launch UUID (string, not numeric ID)
- Optional flags:
  - `--status` (string) — final status
  - `--description` (string) — final description
  - `--attribute` (string slice) — `key:value` format
  - `--from-file` (string) — reads full `FinishExecutionRQ` from JSON
- Build `types.FinishExecutionRQ` (EndTime set to `time.Now()` if not in file)
- Call `scope.Finish(ctx, uuid, request)`
- Table output: ID, NUMBER, LINK columns from `FinishLaunchRS`
- JSON output: raw `FinishLaunchRS`

### 2. Create `internal/cli/item_lifecycle.go`

This file defines `rp item start` and `rp item finish`.

#### `rp item start`

- No positional arguments
- Required flag: `--name` (string) — item name
- Required flag: `--launch-uuid` (string) — UUID of the parent launch
- Required flag: `--type` (string) — item type (SUITE, STORY, TEST, STEP, BEFORE_CLASS, etc.)
- Optional flags:
  - `--parent-uuid` (string) — if provided, creates a child item under this parent; otherwise creates root item
  - `--description` (string)
  - `--attribute` (string slice) — `key:value` format
  - `--has-stats` (bool, default true)
  - `--from-file` (string) — reads full `StartTestItemRQ` from JSON
- If `--parent-uuid` is set: call `scope.StartChild(ctx, parentUUID, request)`
- Else: call `scope.StartRoot(ctx, request)`
- Table output: ID column from `EntryCreatedAsyncRS`
- JSON output: raw `EntryCreatedAsyncRS`

#### `rp item finish <uuid>`

- Exactly one positional argument: item UUID
- Optional flags:
  - `--status` (string) — final status (PASSED, FAILED, SKIPPED, etc.)
  - `--description` (string)
  - `--attribute` (string slice) — `key:value` format
  - `--from-file` (string) — reads full `FinishTestItemRQ` from JSON
- Build `types.FinishTestItemRQ` (EndTime set to `time.Now()` if not in file)
- Call `scope.Finish(ctx, uuid, request)`
- Table output/JSON output: `OperationCompletionRS` message

### 3. Register commands in parents

In `launch.go`, add to `newLaunchCmd()`:
```go
cmd.AddCommand(newLaunchStartCmd())
cmd.AddCommand(newLaunchFinishCmd())
```

In `item.go`, add to `newItemCmd()`:
```go
cmd.AddCommand(newItemStartCmd())
cmd.AddCommand(newItemFinishCmd())
```

### 4. Attribute building

Reuse the existing `parseAttribute` and `buildAttributeResources` helpers that are already in `launch.go`. For items, create a similar helper or import from a shared location if one exists.

## Test requirements

### Unit tests

- **`internal/cli/launch_lifecycle_test.go`:**
  - `launch start` requires `--name` (missing name → error)
  - `launch start --mode` validation (DEBUG/DEFAULT accepted, invalid rejected)
  - `launch start --from-file` loads valid JSON
  - `launch finish` requires exactly 1 positional arg (UUID)
  - `launch finish --from-file` loads valid JSON
  - Command wiring: verify `newLaunchCmd()` includes start and finish

- **`internal/cli/item_lifecycle_test.go`:**
  - `item start` requires `--name`, `--launch-uuid`, `--type`
  - `item start --type` validation (known types accepted)
  - `item start --parent-uuid` routes to StartChild
  - `item start` without `--parent-uuid` routes to StartRoot
  - `item finish` requires exactly 1 positional arg (UUID)
  - Command wiring: verify `newItemCmd()` includes start and finish

## Edge cases and constraints

- **Sub-files only:** Do NOT modify existing functions in `launch.go` or `item.go` beyond adding `cmd.AddCommand(...)` lines.
- **UUID, not numeric ID:** Both `launch finish` and `item finish` take a UUID string (returned by start), not a numeric ID.
- **Time defaults:** When not using `--from-file`, set `StartTime`/`EndTime` to current time automatically.
- **Reuse existing helpers:** Use `parseAttribute`, `buildAttributeResources`, `readRequestFromFile`, `parseIDArg` where applicable.
- **Mode validation:** For launch start, reuse `resolveUpdateLaunchMode` pattern.
- **`--from-file` takes precedence:** When both file and flags are provided, file wins silently.

## Example commands

```bash
# Start a launch
$ rp launch start --name "Nightly Run" --mode DEFAULT -p my_project
ID    NUMBER
123   5

# Finish a launch
$ rp launch finish abc-uuid-123 -p my_project
ID    NUMBER  LINK
123   5       http://rp.example.com/ui/#my_project/launches/all/123

# Start a root test item
$ rp item start --name "Login Suite" --launch-uuid abc-uuid --type SUITE -p my_project
ID
456

# Start a child test item
$ rp item start --name "Login Test" --launch-uuid abc-uuid --type TEST --parent-uuid def-uuid -p my_project
ID
789

# Finish a test item
$ rp item finish item-uuid-789 --status PASSED -p my_project
MESSAGE
Test item finished successfully.
```

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [ ] `internal/cli/launch_lifecycle.go` with `rp launch start` and `rp launch finish`
- [ ] `internal/cli/item_lifecycle.go` with `rp item start` and `rp item finish`
- [ ] Commands registered in respective parent commands
- [ ] Unit tests cover flag validation, command wiring, and from-file loading
- [ ] Existing launch/item subcommands unchanged
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
