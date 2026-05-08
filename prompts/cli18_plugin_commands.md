# PR 18: `plugin` Commands

## Goal

Implement the `plugin` command family — list, upload (create), enable/disable (update), delete, and execute. Plugins use `IntegrationTypeResource` as their resource type and mix standard lifecycle commands with a specialized `exec` path that sends arbitrary JSON to a named plugin command.

## Context

The `plugin` resource maps to `PluginScope` on the client (`client.Plugins()`). The standard lifecycle commands (`get`, `create`, `update`, `delete`) are **instance-scoped** and do **not** require `--project`. The specialized `exec plugin` command uses standard project resolution unless `--public` is set. Most plugin operations are admin-only; the exception is `ListPublic`, which requires no auth on some deployments.

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `readRequestFromFile`, `mustGetStringFlag`, `optionalBoolFlag`, `messageResultHeaders`/`operationResultRows`)
- PR 17 (frozen UX: §1 plugin commands)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `List` | `ctx` | `([]types.IntegrationTypeResource, error)` |
| `ListPublic` | `ctx` | `([]types.IntegrationTypeResource, error)` |
| `Upload` | `ctx`, `filename string`, `data io.Reader` | `(*types.EntryCreatedRS, error)` |
| `UpdateState` | `ctx`, `pluginID int64`, `stateUpdate types.UpdatePluginStateRQ` | `(*types.OperationCompletionRS, error)` |
| `Delete` | `ctx`, `pluginID int64` | `(*types.OperationCompletionRS, error)` |
| `ExecuteCommand` | `ctx`, `projectName string`, `pluginName string`, `command string`, `body map[string]any` | `(map[string]any, error)` |
| `ExecutePublicCommand` | `ctx`, `pluginName string`, `command string`, `body map[string]any` | `(map[string]any, error)` |

**Notable:** `List` and `ListPublic` have no pagination — they return all results at once. `Upload` takes a filename and an `io.Reader` for binary data (plugin JAR file). `ExecuteCommand` requires a project name, while `ExecutePublicCommand` skips project resolution entirely. Both exec methods accept an arbitrary JSON body and treat `nil` as an empty object.

### Key types

```go
type IntegrationTypeResource struct {
    AuthFlow     *IntegrationTypeResourceAuthFlow `json:"authFlow,omitempty"`
    CreationDate *time.Time                       `json:"creationDate,omitempty"`
    Details      *map[string]interface{}          `json:"details,omitempty"`
    Enabled      *bool                            `json:"enabled,omitempty"`
    GroupType    *string                          `json:"groupType,omitempty"`
    Name         *string                          `json:"name,omitempty"`
    PluginType   *string                          `json:"pluginType,omitempty"`
    Type         *int64                           `json:"type,omitempty"`
}

type UpdatePluginStateRQ struct {
    Enabled *bool `json:"enabled,omitempty"`
}

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}
```

## Scope

Create `internal/cli/plugin.go` and `internal/cli/plugin_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get plugin` | List all installed plugins |
| `rp get plugin --public` | List public plugins only |
| `rp create plugin --file <path>` | Upload a plugin JAR file |
| `rp update plugin <id> --enabled` / `--enabled=false` | Enable or disable a plugin |
| `rp delete plugin <id>` | Delete a plugin by numeric ID |
| `rp exec plugin <plugin-name> <command> [--from-file <body.json>]` | Execute a command against a plugin |
| `rp exec plugin <plugin-name> <command> --public [--from-file <body.json>]` | Execute a command against a plugin via the public endpoint |

## Implementation

### 1. Create `internal/cli/plugin.go`

This file contains all six leaf command constructors and their run functions.

#### `newGetPluginCmd()`

```go
func newGetPluginCmd() *cobra.Command
```

- `Use`: `"plugin"`
- `Short`: `"List installed plugins"`
- `Long`: Include examples showing list all and list public. Mention this is instance-scoped (admin-only) and does not require `--project`.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGetPlugin)`
- Flags:
  - `--public` (bool, default `false`): list only public plugins

**Do NOT** register `--project` — plugins are instance-scoped.

**Do NOT** register pagination flags — the API returns all results at once.

**No single-get-by-ID.** The plugin API does not expose a get-by-ID method. The `get` command always lists.

The run function:
1. Resolve connection and client (no project resolution)
2. Obtain the plugin scope: `scope := clientInstance.Plugins()`
3. Read `--public` flag
4. If `--public` is true, call `scope.ListPublic(ctx)`. Otherwise call `scope.List(ctx)`.
5. Build table rows from the result slice using `pluginHeaders()` and `pluginRow` per entry
6. Print via `printer.Print(plugins, pluginHeaders(), rows)`

#### `newCreatePluginCmd()`

```go
func newCreatePluginCmd() *cobra.Command
```

- `Use`: `"plugin"`
- `Short`: `"Upload a plugin"`
- `Long`: Include example showing `--file <path>` usage. Note that the file must be a JAR file.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreatePlugin)`
- Flags:
  - `--file` (string, required): path to the plugin JAR file to upload

**Do NOT** use `--from-file` / `-f` here. Per PR 17, `--file` is for commands where the file is the primary or only input mode. `create plugin` always uploads the plugin archive from disk, so `--file` is the correct flag. The fact that this command streams raw bytes rather than unmarshaling JSON is a consequence of the API shape, not the primary reason for the flag name.

The run function:
1. Validate `--file` is non-empty
2. Open the file for reading: `os.Open(filePath)`. Defer close.
3. Extract the base filename: `filepath.Base(filePath)`
4. Resolve connection and client
5. Obtain the plugin scope: `scope := clientInstance.Plugins()`
6. Call `scope.Upload(ctx, filename, file)` — passes the `*os.File` as `io.Reader`
7. Print via `printer.Print(result, entryCreatedHeaders(), entryCreatedRows(result))`

Handle the file-not-found error with a descriptive message: `fmt.Errorf("opening plugin file %s: %w", filePath, err)`.

#### `newUpdatePluginCmd()`

```go
func newUpdatePluginCmd() *cobra.Command
```

- `Use`: `"plugin <id>"`
- `Short`: `"Enable or disable a plugin"`
- `Long`: Include examples showing `--enabled` and `--enabled=false`. Explain that the flag must be explicitly set.
- `Args`: `cobra.ExactArgs(1)` (the numeric plugin ID, which is the `Type` field from `IntegrationTypeResource`)
- `RunE`: `withPrintedErrors(runUpdatePlugin)`
- Flags:
  - `--enabled` (bool, default `false`): enable or disable the plugin (`--enabled` or `--enabled=false`)

**This uses the existing `optionalBoolFlag` pattern** (same as `--active` on `update user`). Register as a regular bool flag; check `Changed` to distinguish "not provided" from "explicitly false."

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "plugin")`
2. Read the `--enabled` flag using `optionalBoolFlag(cmd, "enabled")`
3. Validate that `--enabled` was explicitly provided:
   ```go
   enabled := optionalBoolFlag(cmd, "enabled")
   if enabled == nil {
       return fmt.Errorf("--enabled is required (use --enabled or --enabled=false)")
   }
   ```
4. Build `types.UpdatePluginStateRQ{Enabled: enabled}`
5. Resolve connection and client
6. Obtain scope: `scope := clientInstance.Plugins()`
7. Call `scope.UpdateState(ctx, id, stateUpdate)`
8. Print via `printer.Print(result, messageResultHeaders(), operationResultRows(result))`

#### `newDeletePluginCmd()`

```go
func newDeletePluginCmd() *cobra.Command
```

- `Use`: `"plugin <id>"`
- `Short`: `"Delete a plugin"`
- `Long`: Include example showing delete by numeric ID
- `Args`: `cobra.ExactArgs(1)` (the numeric plugin ID)
- `RunE`: `withPrintedErrors(runDeletePlugin)`
- No additional flags beyond the global ones.

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "plugin")`
2. Resolve connection and client
3. Obtain scope: `scope := clientInstance.Plugins()`
4. Call `scope.Delete(ctx, id)`
5. Print via `printer.Print(result, messageResultHeaders(), operationResultRows(result))`

#### `newExecPluginCmd()`

```go
func newExecPluginCmd() *cobra.Command
```

- `Use`: `"plugin <plugin-name> <command>"`
- `Short`: `"Execute a plugin command"`
- `Long`: Include examples for both the standard and `--public` flows. Explain that the request body is arbitrary JSON and defaults to `{}` when `--from-file` is omitted. Document that `--public` and `--project` are mutually exclusive.
- `Args`: `cobra.ExactArgs(2)` (plugin name and command name)
- `RunE`: `withPrintedErrors(runExecPlugin)`
- Flags:
  - `--project` / `-p` (string): project name override for the standard exec path
  - `--public` (bool, default `false`): call the public plugin execution endpoint instead of the project-scoped endpoint
  - `--from-file` / `-f` (string, optional): path to a JSON file containing the request body

**Why `--from-file` here?** Per PR 17, `exec plugin` uses `--from-file` because the file-backed JSON body is optional; the default behavior is to send an empty object.

The run function:
1. Read `args[0]` as `pluginName` and `args[1]` as `commandName`
2. If `--from-file` was provided, read and unmarshal it into `map[string]any` using `readRequestFromFile[map[string]any](path)`; otherwise start with `map[string]any{}`
3. Read `--public`
4. If `--public` is true:
   - Return an error if `--project` was also explicitly provided: `fmt.Errorf("--public and --project are mutually exclusive")`
   - Resolve connection and client (no project resolution)
   - Call `scope.ExecutePublicCommand(ctx, pluginName, commandName, body)`
5. If `--public` is false:
   - Resolve connection, project, and client using the standard project resolution chain
   - Call `scope.ExecuteCommand(ctx, project, pluginName, commandName, body)`
6. Print the arbitrary JSON result. JSON and Go-template modes should pass through the raw `map[string]any`; for table mode, either reuse an existing generic key/value formatter or flatten top-level keys into `KEY` / `VALUE` rows.

#### Table formatting

```go
func pluginHeaders() []string {
    return []string{"ID", "NAME", "GROUP TYPE", "ENABLED"}
}

func pluginRow(plugin *types.IntegrationTypeResource) ([]string, []string) {
    return pluginHeaders(), []string{
        formatInt64Ptr(plugin.Type),
        formatStringPtr(plugin.Name),
        formatStringPtr(plugin.GroupType),
        formatBoolPtr(plugin.Enabled),
    }
}
```

The ID column shows `plugin.Type` (the numeric identifier), not a dedicated `Id` field. This matches how `UpdateState` and `Delete` identify plugins by `pluginID int64`, which corresponds to the `Type` field. The `Details` map is omitted from the table; users should use `-o json` for full plugin details.

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// get
get.AddCommand(newGetPluginCmd())

// create
create.AddCommand(newCreatePluginCmd())

// update
update.AddCommand(newUpdatePluginCmd())

// delete
del.AddCommand(newDeletePluginCmd())

// exec
exec.AddCommand(newExecPluginCmd())
```

### 3. Create `internal/cli/plugin_test.go`

Use the same testing pattern as the other command test files:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetPluginCmd_HasPublicFlag`** — Verify the get command has a `--public` flag registered with default `false`.

2. **`TestNewGetPluginCmd_NoProjectFlag`** — Verify the get command does NOT have a `--project` flag.

3. **`TestNewGetPluginCmd_NoListFlags`** — Verify the get command does NOT have `--page-size`, `--page`, or `--sort` flags.

4. **`TestNewGetPluginCmd_NoArgs`** — Verify `Args` is `cobra.NoArgs`.

5. **`TestPluginRow_Formatting`** — Verify `pluginRow` produces the expected headers and values for a sample `IntegrationTypeResource`.

6. **`TestPluginRow_NilFields`** — Verify `pluginRow` handles an `IntegrationTypeResource` with all nil fields.

7. **`TestNewCreatePluginCmd_Flags`** — Verify create has `--file` flag but NOT `--from-file`, `--project`, or `--name`.

8. **`TestNewUpdatePluginCmd_Flags`** — Verify update has `--enabled` flag (no `--disabled` flag). Verify it accepts exactly 1 arg.

9. **`TestNewDeletePluginCmd_Args`** — Verify delete accepts exactly 1 arg and has no `--project` flag.

10. **`TestNewExecPluginCmd_Flags`** — Verify exec has `--project`, `--public`, and optional `--from-file` / `-f`. Verify it accepts exactly 2 args.

11. **`TestExecPluginValidation_PublicWithProject`** — Verify exec returns a descriptive error when `--public` and `--project` are used together.

12. **`TestUpdatePluginValidation_FlagOmitted`** — Create the command without setting `--enabled`, verify the validation produces a descriptive error requiring the flag.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **The base plugin lifecycle commands are instance-scoped.** `get`, `create`, `update`, and `delete` do not use `--project`. The `--public` variant of `get plugin` may not require auth on some deployments, but it is still instance-scoped.
- **No pagination.** Both list endpoints return all plugins at once.
- **No single-get-by-ID.** The API does not expose a single plugin retrieval method.
- **`--file` not `--from-file`.** Per PR 17, `--file` is used when the file is the command's primary or only input mode. `create plugin` always takes the plugin archive from disk, so `--file` is the right UX. The upload path happens to stream raw bytes rather than deserialize JSON, but that follows from the API shape rather than defining the naming rule.
- **`--enabled` uses the `optionalBoolFlag` pattern.** The API's `UpdatePluginStateRQ` has a single `Enabled *bool` field. A single `--enabled` flag maps directly: `--enabled` sets true, `--enabled=false` sets false, omission is an error. This matches the `--active` flag on `update user` and avoids a second flag that would need mutual-exclusion wiring.
- **Plugin ID is `IntegrationTypeResource.Type`.** The `Type *int64` field serves as the plugin's numeric identifier. This is what `UpdateState` and `Delete` accept as `pluginID`. The table shows it as `ID` for clarity.
- **`exec plugin` is in scope; `Import` and `GetPublicFile` remain deferred.** PR 17 froze the standard and public exec shapes in this PR. Only the multipart import and raw public file download stay out of scope.
- **Standard exec uses normal project resolution; public exec does not.** `rp exec plugin ...` follows the usual `--project` / env / profile chain. `rp exec plugin ... --public` skips project resolution entirely, and must reject `--project`.
- **`--from-file` is optional for exec.** Unlike upload, the file is one alternative among multiple request-construction modes. Omitting it sends an empty JSON object, so `--from-file` remains the correct flag name.
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`, `withPrintedErrors`.
- **Route all output through `printer.Print`.** This ensures JSON, Go-template, and table modes all work correctly. Errors use `printer.PrintError(err)` before returning.
- **Use client error predicates.** Handle API errors with predicate functions (`IsNotFound`, `IsUnauthorized`, etc.) rather than type assertions.
- **Command registration follows PR 3's centralized command-tree layout.** All subcommands are registered in `buildCommandTree` in `commands.go`.

## What NOT to do

- **Do not implement `Import` or `GetPublicFile`.** Those remain follow-on work.
- **Do not add `--project` / `-p` to get/create/update/delete.** Only the non-public `exec plugin` path uses project resolution.
- **Do not add pagination flags.**
- **Do not use `--from-file`** for the upload command. Use `--file` for binary uploads.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` now shows `plugin` as a subcommand
- `rp create --help` now shows `plugin` as a subcommand
- `rp update --help` now shows `plugin` as a subcommand
- `rp delete --help` now shows `plugin` as a subcommand
- `rp exec --help` now shows `plugin` as a subcommand
- `rp get plugin --help` shows `--public` flag but NO `--project` or pagination flags
- `rp create plugin --help` shows `--file` flag (not `--from-file`)
- `rp update plugin --help` shows `--enabled` flag (no `--disabled` flag)
- `rp exec plugin --help` shows `--project`, `--public`, and optional `--from-file`
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/plugin.go` created with all six command constructors and their run functions
- [ ] `internal/cli/plugin_test.go` created with unit tests
- [ ] Commands registered in `commands.go`
- [ ] Get/create/update/delete remain instance-scoped with no `--project`
- [ ] Get supports `--public` for public-only listing
- [ ] Create uses `--file` for binary upload (not `--from-file`)
- [ ] Update uses `optionalBoolFlag` pattern: `--enabled` / `--enabled=false`, error if omitted
- [ ] Exec supports both standard and `--public` paths, with optional `--from-file`
- [ ] Exec rejects `--public` combined with `--project`
- [ ] No pagination flags
- [ ] Table output shows ID (from `Type` field), NAME, GROUP TYPE, ENABLED with uppercase headers
- [ ] Shared helpers used: `parseResourceID`, `formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
