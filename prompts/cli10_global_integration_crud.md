# PR 10: Global `integration` CRUD

## Goal

Implement the global integration command family — list, get-by-ID, create, update, and delete — for instance-scoped admin integration management. Global integrations use `--from-file` for mutations because `IntegrationRQ` contains an arbitrary-shape `IntegrationParameters` map. Create additionally requires a `--plugin` flag to identify which plugin type the integration belongs to.

## Context

The `integration` resource maps to `IntegrationsScope` on the client (`client.Integrations()`). It is **instance-scoped** — commands do **not** require `--project`. This is an admin-only resource.

**Prerequisite:** PRs 1–7 must land first (parent command split, verb scaffolding, centralized registration, all three pilots, and shared helper extraction).

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `ListGlobal` | `ctx` | `([]types.IntegrationResource, error)` |
| `GetGlobal` | `ctx`, `id int64` | `(*types.IntegrationResource, error)` |
| `CreateGlobal` | `ctx`, `pluginName string`, `request types.IntegrationRQ` | `(*types.EntryCreatedRS, error)` |
| `UpdateGlobal` | `ctx`, `id int64`, `request types.IntegrationRQ` | `(*types.OperationCompletionRS, error)` |
| `DeleteGlobal` | `ctx`, `id int64` | `(*types.OperationCompletionRS, error)` |

**Notable:** `ListGlobal` has no pagination — the API returns all global integrations at once. `CreateGlobal` takes a `pluginName` parameter separate from the request body. There is also `ListGlobalByPlugin(ctx, pluginName)` which filters by plugin name.

### Key types

```go
type IntegrationRQ struct {
    Enabled               *bool                   `json:"enabled,omitempty"`
    IntegrationParameters *map[string]interface{} `json:"integrationParameters,omitempty"`
    Name                  *string                 `json:"name,omitempty"`
}

type IntegrationResource struct {
    CreationDate          *time.Time               `json:"creationDate,omitempty"`
    Creator               *string                  `json:"creator,omitempty"`
    Enabled               *bool                    `json:"enabled,omitempty"`
    Id                    *int64                   `json:"id,omitempty"`
    IntegrationParameters *map[string]interface{}  `json:"integrationParameters,omitempty"`
    IntegrationType       *IntegrationTypeResource `json:"integrationType,omitempty"`
    Name                  *string                  `json:"name,omitempty"`
    ProjectId             *int64                   `json:"projectId,omitempty"`
}

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

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}
```

## Scope

Create `internal/cli/integration.go` and `internal/cli/integration_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get integration` | List all global integrations |
| `rp get integration [--plugin <name>]` | List global integrations filtered by plugin type |
| `rp get integration <id>` | Get a single global integration by numeric ID |
| `rp create integration --plugin <name> --from-file <path>` | Create a global integration |
| `rp update integration <id> --from-file <path>` | Update a global integration |
| `rp delete integration <id>` | Delete a global integration by numeric ID |

## Implementation

### 1. Create `internal/cli/integration.go`

This file contains all five command constructors and their run functions.

#### `newGetIntegrationCmd()`

```go
func newGetIntegrationCmd() *cobra.Command
```

- `Use`: `"integration [id]"`
- `Short`: `"List global integrations or get a specific integration"`
- `Long`: Include examples showing list, list filtered by plugin, get-by-ID, and output format. Mention this is instance-scoped (admin-only) and does not require `--project`.
- `Args`: `cobra.MaximumNArgs(1)`
- `RunE`: `withPrintedErrors(runGetIntegration)`
- Flags:
  - `--plugin` (string, optional): filter list results by plugin name (only meaningful in list mode)

**Do NOT** register `--project` — global integrations are instance-scoped.

**Do NOT** register pagination flags — the API returns all results at once.

The run function:
1. Resolve connection and client (no project resolution)
2. If `len(args) == 1`, parse arg as int64 using `parseResourceID(args[0], "integration")`, call `scope.GetGlobal(ctx, id)`, print as single row. **Ignore `--plugin` in this path** — an explicit ID is unambiguous and `--plugin` is only meaningful for list filtering.
3. Otherwise:
   - If `--plugin` is non-empty, call `scope.ListGlobalByPlugin(ctx, pluginName)`
   - Otherwise, call `scope.ListGlobal(ctx)`
   - Pass the resulting `[]types.IntegrationResource` slice to `printer.Print`

#### `newCreateIntegrationCmd()`

```go
func newCreateIntegrationCmd() *cobra.Command
```

- `Use`: `"integration"`
- `Short`: `"Create a global integration"`
- `Long`: Include examples showing `--plugin` and `--from-file` usage. Note this is admin-only.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateIntegration)`
- Flags:
  - `--plugin` (string, required): the plugin type name for the integration
  - `--from-file` / `-f` (string, required): path to JSON file containing `IntegrationRQ`

The run function:
1. Validate `--plugin` is non-empty
2. Validate `--from-file` is non-empty
3. Read and unmarshal the file using `readRequestFromFile[types.IntegrationRQ](path)`
4. Resolve connection and client
5. Call `scope.CreateGlobal(ctx, pluginName, request)`
6. Print the `EntryCreatedRS` result using `entryCreatedHeaders()` and `entryCreatedRows(result)`

#### `newUpdateIntegrationCmd()`

```go
func newUpdateIntegrationCmd() *cobra.Command
```

- `Use`: `"integration <id>"`
- `Short`: `"Update a global integration"`
- `Long`: Include examples showing `--from-file` with an ID argument
- `Args`: `cobra.ExactArgs(1)` (the numeric integration ID)
- `RunE`: `withPrintedErrors(runUpdateIntegration)`
- Flags:
  - `--from-file` / `-f` (string, required): path to JSON file containing `IntegrationRQ`

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "integration")`
2. Validate `--from-file` is non-empty
3. Read and unmarshal the file using `readRequestFromFile[types.IntegrationRQ](path)`
4. Resolve connection and client
5. Call `scope.UpdateGlobal(ctx, id, request)`
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newDeleteIntegrationCmd()`

```go
func newDeleteIntegrationCmd() *cobra.Command
```

- `Use`: `"integration <id>"`
- `Short`: `"Delete a global integration"`
- `Long`: Include example showing delete by ID
- `Args`: `cobra.ExactArgs(1)` (the numeric integration ID)
- `RunE`: `withPrintedErrors(runDeleteIntegration)`
- No additional flags beyond the global ones.

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "integration")`
2. Resolve connection and client
3. Call `scope.DeleteGlobal(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### Table formatting

```go
func integrationHeaders() []string {
    return []string{"ID", "NAME", "TYPE", "ENABLED", "CREATOR"}
}

func integrationRow(integration *types.IntegrationResource) ([]string, []string) {
    typeName := ""
    if integration.IntegrationType != nil {
        typeName = formatStringPtr(integration.IntegrationType.Name)
    }

    return integrationHeaders(), []string{
        formatInt64Ptr(integration.Id),
        formatStringPtr(integration.Name),
        typeName,
        formatBoolPtr(integration.Enabled),
        formatStringPtr(integration.Creator),
    }
}
```

The `IntegrationParameters` map is intentionally omitted from the table — it is an arbitrary key-value map whose contents vary by plugin type. Users who need parameter details should use `-o json`.

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// get
get.AddCommand(newGetIntegrationCmd())

// create
create.AddCommand(newCreateIntegrationCmd())

// update
update.AddCommand(newUpdateIntegrationCmd())

// delete
del.AddCommand(newDeleteIntegrationCmd())
```

### 3. Create `internal/cli/integration_test.go`

Use the same testing pattern as the pilot test files:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetIntegrationCmd_NoProjectFlag`** — Verify the get command does NOT have a `--project` flag registered.

2. **`TestNewGetIntegrationCmd_HasPluginFlag`** — Verify the get command has a `--plugin` flag registered.

3. **`TestNewGetIntegrationCmd_NoListFlags`** — Verify the get command does NOT have `--page-size`, `--page`, or `--sort` flags.

4. **`TestIntegrationRow_Formatting`** — Verify `integrationRow` produces the expected headers and values for a sample `IntegrationResource` with a populated `IntegrationType`.

5. **`TestIntegrationRow_NilIntegrationType`** — Verify `integrationRow` handles nil `IntegrationType` (produces empty string for TYPE column).

6. **`TestIntegrationRow_AllNilFields`** — Verify `integrationRow` handles an `IntegrationResource` with all nil fields.

7. **`TestNewCreateIntegrationCmd_Flags`** — Verify create has `--plugin` and `--from-file` / `-f` flags but NOT `--project`.

8. **`TestNewUpdateIntegrationCmd_Flags`** — Verify update has `--from-file` / `-f` flag and accepts exactly 1 arg.

9. **`TestNewDeleteIntegrationCmd_Args`** — Verify delete accepts exactly 1 arg and has no `--project` flag.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Instance-scoped means no `--project` flag.** Global integrations are admin-level, instance-wide resources.
- **No pagination.** The list endpoints return all integrations at once.
- **`--plugin` flag on get for filtering.** The API provides `ListGlobalByPlugin` as a separate method. Exposing it as a `--plugin` filter on the get command is natural — it is analogous to `--launch` on `get item`.
- **`--plugin` on create is a method parameter, not part of `IntegrationRQ`.** The `CreateGlobal` method takes `pluginName` as a separate string parameter. This is why `--plugin` is a flag rather than being included in the `--from-file` JSON.
- **`--from-file` for create and update.** `IntegrationRQ` contains `IntegrationParameters *map[string]interface{}` — an arbitrary key-value map whose structure depends on the plugin type (e.g., LDAP, JIRA, Rally). This is clearly a complex/structured payload per the CLI plan's decision rules.
- **`IntegrationType.Name` for the TYPE column.** The `IntegrationTypeResource` has multiple fields, but `Name` is the most meaningful for display (e.g., "jira", "rally", "ldap").
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr`, `readRequestFromFile[types.IntegrationRQ]`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`.

## What NOT to do

- **Do not implement `TestGlobalConnection` or `DeleteAllByType`.** Those are stretch goals.
- **Do not add `--project` / `-p` flags** to any integration command. Global integrations are instance-scoped.
- **Do not add pagination flags** to the get command.
- **Do not confuse global integrations with project integrations.** Project integrations are a separate resource family (PR 11) accessed via `client.Project(name).Integrations()`.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` now shows `integration` as a subcommand
- `rp create --help` now shows `integration` as a subcommand
- `rp update --help` now shows `integration` as a subcommand
- `rp delete --help` now shows `integration` as a subcommand
- `rp get integration --help` shows `--plugin` flag but NO `--project` or pagination flags
- `rp create integration --help` shows `--plugin` and `--from-file` flags
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/integration.go` created with all five command constructors and their run functions
- [ ] `internal/cli/integration_test.go` created with unit tests for flag registration and table formatting
- [ ] Commands registered in `commands.go`
- [ ] No `--project` flag on any integration command
- [ ] Get command supports `--plugin` for filtered listing
- [ ] Create uses `--plugin` and `--from-file`; update uses `--from-file` only
- [ ] No pagination flags
- [ ] Table output shows ID, NAME, TYPE, ENABLED, CREATOR with uppercase headers; TYPE is derived from `IntegrationType.Name`
- [ ] JSON output for single-get prints the `IntegrationResource` object; for list mode prints the `[]IntegrationResource` slice
- [ ] Shared helpers used: `parseResourceID`, `formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr`, `readRequestFromFile`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
