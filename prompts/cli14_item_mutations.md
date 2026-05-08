# PR 14: `item` Mutations

## Goal

Extend the existing `item` command family with update and delete commands. The existing `rp get item` command (in `item.go`) is left unchanged. Both commands take a numeric item ID as a positional argument and use the project-scoped `TestItemScope`.

## Context

The `item` resource maps to `TestItemScope` on the client (`client.Project(name).Items()`). It is **project-scoped**, meaning mutation commands require a resolved project via `--project`, `RP_PROJECT`, or the profile's `default-project`.

**Prerequisite:** PRs 1–7 must land first (parent command split, verb scaffolding, centralized registration, all three pilots, and shared helper extraction).

### Client surface (mutations only)

| Method | Parameters | Returns |
|--------|------------|---------|
| `Update` | `ctx`, `itemID int64`, `request types.UpdateTestItemRQ` | `(*types.OperationCompletionRS, error)` |
| `Delete` | `ctx`, `itemID int64` | `(*types.OperationCompletionRS, error)` |

### Key types

```go
type UpdateTestItemRQ struct {
    Attributes  *[]ItemAttributeResource `json:"attributes,omitempty"`
    Description *string                  `json:"description,omitempty"`
    Status      *string                  `json:"status,omitempty"`
}

type ItemAttributeResource struct {
    Key   *string `json:"key,omitempty"`
    Value string  `json:"value"`
}
```

All fields on `UpdateTestItemRQ` are optional pointers. The API performs a partial update — only the fields present in the request are modified. This is structurally identical to `UpdateLaunchRQ` (PR 13) and follows the same `--from-file` pattern.

## Scope

Extend `internal/cli/item.go` with two new command constructors and their run functions. Create `internal/cli/item_mutation_test.go` for tests covering the new commands. Register the new commands in `internal/cli/commands.go`.

**Important:** Use a separate test file (`item_mutation_test.go`) rather than adding tests to `item_test.go` to minimize conflict with the existing `get item` test surface.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get item` | *(existing — unchanged)* |
| `rp get item <id-or-uuid>` | *(existing — unchanged)* |
| `rp update item <id> --from-file <path>` | Update a test item |
| `rp delete item <id>` | Delete a test item by numeric ID |

## Implementation

### 1. Add commands to `internal/cli/item.go`

Add two new command constructors and their run functions to the existing file. Do NOT modify any existing functions in `item.go`.

#### `newUpdateItemCmd()`

```go
func newUpdateItemCmd() *cobra.Command
```

- `Use`: `"item <id>"`
- `Short`: `"Update a test item"`
- `Long`: Include examples showing `--from-file` with a numeric item ID. Include sample JSON showing how to update just the description, just the status, or add attributes.
- `Args`: `cobra.ExactArgs(1)` (the numeric item ID)
- `RunE`: `withPrintedErrors(runUpdateItem)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--from-file` / `-f` (string, required): path to JSON file containing `UpdateTestItemRQ`

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "item")`
2. Validate `--from-file` is non-empty
3. Read and unmarshal the file using `readRequestFromFile[types.UpdateTestItemRQ](path)`
4. Resolve connection, project, and client
5. Call `scope.Update(ctx, id, request)`
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

**Why `--from-file`?** Same rationale as launch update (PR 13): `Attributes` is an array of structured objects. All fields are optional pointers, so partial updates work naturally — the user includes only the fields they want to change. Example JSON for a status-only update:

```json
{
  "status": "PASSED"
}
```

Example JSON with attributes:

```json
{
  "attributes": [
    {"key": "component", "value": "auth"},
    {"value": "flaky"}
  ],
  "description": "Updated after review"
}
```

#### `newDeleteItemCmd()`

```go
func newDeleteItemCmd() *cobra.Command
```

- `Use`: `"item <id>"`
- `Short`: `"Delete a test item"`
- `Long`: Include example showing delete by numeric ID
- `Args`: `cobra.ExactArgs(1)` (the numeric item ID)
- `RunE`: `withPrintedErrors(runDeleteItem)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "item")`
2. Resolve connection, project, and client
3. Call `scope.Delete(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// update
update.AddCommand(newUpdateItemCmd())

// delete
del.AddCommand(newDeleteItemCmd())
```

### 3. Create `internal/cli/item_mutation_test.go`

Use a separate test file to avoid conflicts with the existing `item_test.go`:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewUpdateItemCmd_Flags`** — Verify update has `--from-file` / `-f` and `--project` / `-p` flags. Verify it accepts exactly 1 arg.

2. **`TestNewUpdateItemCmd_NoGetFlags`** — Verify update does NOT have `--page-size`, `--page`, `--sort`, or any of the item-specific filter flags (`--launch`, `--parent`, `--type`, `--suites`, `--name`, `--status`, `--has-children`).

3. **`TestNewDeleteItemCmd_Flags`** — Verify delete has `--project` / `-p` flag. Verify it accepts exactly 1 arg.

4. **`TestNewDeleteItemCmd_NoFromFile`** — Verify delete does NOT have a `--from-file` flag.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Extends existing file.** Both commands are added to `item.go`. The existing `newGetItemCmd`, `runGetItem`, `getSingleItem`, `listItems`, `itemRow`, `itemHeaders`, `formatStringPtr`, `formatInt64Ptr`, and all filter/option functions remain untouched. **Note:** After PR 7 lands, `formatStringPtr` and `formatInt64Ptr` will have been moved to `helpers.go` — the definitions will no longer be in `item.go`. This does not affect this PR since the functions remain accessible within the `cli` package.
- **Numeric ID only for mutations.** Unlike `get item` which accepts both numeric IDs and UUIDs, `update` and `delete` only accept numeric item IDs. This is because `TestItemScope.Update` and `TestItemScope.Delete` take `int64`, not string. The `Use` field and help text should make this clear.
- **`--from-file` for update.** Same rationale as launch update (PR 13): `Attributes` is a structured array. The pattern is identical.
- **Separate test file.** Using `item_mutation_test.go` avoids merge conflicts with the existing `item_test.go`, which tests the `get item` command surface (filter flags, parsing helpers, etc.).
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `readRequestFromFile[types.UpdateTestItemRQ]`, `messageResultHeaders`/`operationResultRows`.
- **Bulk delete is out of scope.** `TestItemScope.DeleteBulk` is a stretch goal not in the core PR.
- **Lifecycle commands (start/finish) are out of scope.** Those are PR 19.
- **Issue link/unlink commands are out of scope.** Those are PR 24.

## What NOT to do

- **Do not modify any existing functions** in `item.go`. The `get item` surface (command, run function, filter flags, list options, `parseOptionalInt64Flag`) must remain identical.
- **Do not implement `DeleteBulk`.** That is a stretch goal.
- **Do not implement `StartRoot`, `StartChild`, `Finish`, or any lifecycle commands.** Those belong to PR 19.
- **Do not implement `LinkExternalIssues` or `UnlinkExternalIssues`.** Those belong to PR 24.
- **Do not accept UUIDs for update or delete.** The API methods take `int64` only.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp update --help` now shows `item` as a subcommand
- `rp delete --help` now shows `item` as a subcommand
- `rp update item --help` shows `--from-file` and `--project` flags
- `rp delete item --help` shows `--project` flag and expects a numeric ID arg
- `rp get item` and all existing item list/get/filter behavior is unchanged
- No existing commands are affected

## Definition of done

- [ ] `newUpdateItemCmd` and `newDeleteItemCmd` added to `internal/cli/item.go`
- [ ] `internal/cli/item_mutation_test.go` created with unit tests for new commands
- [ ] Commands registered in `commands.go`
- [ ] Both commands use `--project` / `-p` for project resolution
- [ ] Update uses `--from-file` / `-f` for input
- [ ] Both commands take a numeric item ID as positional arg (no UUID support)
- [ ] Existing `get item` code is completely untouched
- [ ] Shared helpers used: `parseResourceID`, `readRequestFromFile`, `messageResultHeaders`/`operationResultRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
