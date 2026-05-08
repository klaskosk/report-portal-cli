# PR 13: `launch` Mutations

## Goal

Extend the existing `launch` command family with update and delete commands. The existing `rp get launch` command (in `launch.go`) is left unchanged. Both commands take a numeric launch ID as a positional argument and use the project-scoped `LaunchScope`.

## Context

The `launch` resource maps to `LaunchScope` on the client (`client.Project(name).Launches()`). It is **project-scoped**, meaning mutation commands require a resolved project via `--project`, `RP_PROJECT`, or the profile's `default-project`.

**Prerequisite:** PRs 1–7 must land first (parent command split, verb scaffolding, centralized registration, all three pilots, and shared helper extraction).

### Client surface (mutations only)

| Method | Parameters | Returns |
|--------|------------|---------|
| `Update` | `ctx`, `launchID int64`, `request types.UpdateLaunchRQ` | `(*types.OperationCompletionRS, error)` |
| `Delete` | `ctx`, `launchID int64` | `(*types.OperationCompletionRS, error)` |

### Key types

```go
type UpdateLaunchRQ struct {
    Attributes  *[]ItemAttributeResource `json:"attributes,omitempty"`
    Description *string                  `json:"description,omitempty"`
    Mode        *UpdateLaunchRQMode      `json:"mode,omitempty"`
}

type ItemAttributeResource struct {
    Key   *string `json:"key,omitempty"`
    Value string  `json:"value"`
}

// Valid values for UpdateLaunchRQMode:
const (
    UpdateLaunchRQModeDEBUG   UpdateLaunchRQMode = "DEBUG"
    UpdateLaunchRQModeDEFAULT UpdateLaunchRQMode = "DEFAULT"
)
```

All fields on `UpdateLaunchRQ` are optional pointers. The API performs a partial update — only the fields present in the request are modified. This makes `--from-file` a natural fit since the user can provide a JSON file with only the fields they want to change.

## Scope

Extend `internal/cli/launch.go` with two new command constructors and their run functions. Create `internal/cli/launch_mutation_test.go` for tests covering the new commands. Register the new commands in `internal/cli/commands.go`.

**Important:** Use a separate test file (`launch_mutation_test.go`) rather than adding tests to `launch_test.go` to minimize conflict with the existing `get launch` test surface.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get launch` | *(existing — unchanged)* |
| `rp get launch <id-or-uuid>` | *(existing — unchanged)* |
| `rp update launch <id> --from-file <path>` | Update a launch |
| `rp delete launch <id>` | Delete a launch by numeric ID |

## Implementation

### 1. Add commands to `internal/cli/launch.go`

Add two new command constructors and their run functions to the existing file. Do NOT modify any existing functions in `launch.go`.

#### `newUpdateLaunchCmd()`

```go
func newUpdateLaunchCmd() *cobra.Command
```

- `Use`: `"launch <id>"`
- `Short`: `"Update a launch"`
- `Long`: Include examples showing `--from-file` with a numeric launch ID. Include a sample JSON showing how to update just the description, just the mode, or add attributes. Mention the valid mode values (DEBUG, DEFAULT).
- `Args`: `cobra.ExactArgs(1)` (the numeric launch ID)
- `RunE`: `withPrintedErrors(runUpdateLaunch)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--from-file` / `-f` (string, required): path to JSON file containing `UpdateLaunchRQ`

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "launch")`
2. Validate `--from-file` is non-empty
3. Read and unmarshal the file using `readRequestFromFile[types.UpdateLaunchRQ](path)`
4. Resolve connection, project, and client
5. Call `scope.Update(ctx, id, request)`
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

**Why `--from-file`?** `UpdateLaunchRQ` contains `Attributes *[]ItemAttributeResource`, which is an array of structured objects. Per the CLI plan's decision rules, complex/structured payloads use `--from-file`. Example JSON for a description-only update:

```json
{
  "description": "Updated description"
}
```

Example JSON with attributes and mode:

```json
{
  "attributes": [
    {"key": "env", "value": "production"},
    {"value": "regression"}
  ],
  "mode": "DEFAULT"
}
```

#### `newDeleteLaunchCmd()`

```go
func newDeleteLaunchCmd() *cobra.Command
```

- `Use`: `"launch <id>"`
- `Short`: `"Delete a launch"`
- `Long`: Include example showing delete by numeric ID
- `Args`: `cobra.ExactArgs(1)` (the numeric launch ID)
- `RunE`: `withPrintedErrors(runDeleteLaunch)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "launch")`
2. Resolve connection, project, and client
3. Call `scope.Delete(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// update
update.AddCommand(newUpdateLaunchCmd())

// delete
del.AddCommand(newDeleteLaunchCmd())
```

### 3. Create `internal/cli/launch_mutation_test.go`

Use a separate test file to avoid conflicts with the existing `launch_test.go`:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewUpdateLaunchCmd_Flags`** — Verify update has `--from-file` / `-f` and `--project` / `-p` flags. Verify it accepts exactly 1 arg.

2. **`TestNewUpdateLaunchCmd_NoGetFlags`** — Verify update does NOT have `--page-size`, `--page`, `--sort`, or any of the launch-specific filter flags (`--user`, `--mode`, `--latest`, `--debug`, `--attribute`, `--description`, `--has-retries`).

3. **`TestNewDeleteLaunchCmd_Flags`** — Verify delete has `--project` / `-p` flag. Verify it accepts exactly 1 arg.

4. **`TestNewDeleteLaunchCmd_NoFromFile`** — Verify delete does NOT have a `--from-file` flag.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Extends existing file.** Both commands are added to `launch.go`. The existing `newGetLaunchCmd`, `runGetLaunch`, `getSingleLaunch`, `listLaunches`, `launchRow`, `launchHeaders`, and all filter/option functions remain untouched.
- **Numeric ID only for mutations.** Unlike `get launch` which accepts both numeric IDs and UUIDs, `update` and `delete` only accept numeric launch IDs. This is because `LaunchScope.Update` and `LaunchScope.Delete` take `int64`, not string. The `Use` field and help text should make this clear.
- **`--from-file` for update.** `Attributes` is an array of `ItemAttributeResource` objects, which is a structured payload. Even though `Description` and `Mode` are simple scalars, the `Attributes` field pushes the request into `--from-file` territory. All fields on `UpdateLaunchRQ` are optional pointers, so partial updates work naturally — the user includes only the fields they want to change.
- **Separate test file.** Using `launch_mutation_test.go` avoids merge conflicts with the existing `launch_test.go`, which tests the `get launch` command surface (filter flags, list options, attribute parsing, etc.).
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `readRequestFromFile[types.UpdateLaunchRQ]`, `messageResultHeaders`/`operationResultRows`.
- **Bulk delete is out of scope.** `LaunchScope.DeleteBulk` is a stretch goal not in the core PR.
- **Lifecycle commands (start/finish) are out of scope.** Those are PR 19.

## What NOT to do

- **Do not modify any existing functions** in `launch.go`. The `get launch` surface (command, run function, filter flags, list options, attribute parsing) must remain identical.
- **Do not implement `DeleteBulk`.** That is a stretch goal.
- **Do not implement `Start`, `Finish`, `ForceFinish`, `Analyze`, `Compare`, or `Merge`.** Those belong to later phase PRs.
- **Do not accept UUIDs for update or delete.** The API methods take `int64` only.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp update --help` now shows `launch` as a subcommand
- `rp delete --help` now shows `launch` as a subcommand
- `rp update launch --help` shows `--from-file` and `--project` flags
- `rp delete launch --help` shows `--project` flag and expects a numeric ID arg
- `rp get launch` and all existing launch list/get/filter behavior is unchanged
- No existing commands are affected

## Definition of done

- [ ] `newUpdateLaunchCmd` and `newDeleteLaunchCmd` added to `internal/cli/launch.go`
- [ ] `internal/cli/launch_mutation_test.go` created with unit tests for new commands
- [ ] Commands registered in `commands.go`
- [ ] Both commands use `--project` / `-p` for project resolution
- [ ] Update uses `--from-file` / `-f` for input
- [ ] Both commands take a numeric launch ID as positional arg (no UUID support)
- [ ] Existing `get launch` code is completely untouched
- [ ] Shared helpers used: `parseResourceID`, `readRequestFromFile`, `messageResultHeaders`/`operationResultRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
