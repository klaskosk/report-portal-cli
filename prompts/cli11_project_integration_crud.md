# PR 11: `project-integration` CRUD

## Goal

Implement the project-integration command family — list, get-by-ID, create, update, and delete — for project-scoped integration management. Project integrations mirror the global integration pattern (PR 10) but are scoped to a specific project and accessed through `client.Project(name).Integrations()`.

## Context

The `project-integration` resource maps to `ProjectIntegrationsScope` on the client (`client.Project(name).Integrations()`). It is **project-scoped**, meaning every command requires a resolved project via `--project`, `RP_PROJECT`, or the profile's `default-project`.

**Important:** The method name on `ProjectScope` is `.Integrations()`, which returns a `*ProjectIntegrationsScope` — not to be confused with the instance-level `client.Integrations()` which returns `*IntegrationsScope` for global integrations (PR 10). These are completely separate scopes.

**Prerequisite:** PRs 1–7 must land first. PR 10 (global integrations) should land first so the table formatting and `--from-file` patterns are consistent, though it is not a hard code dependency.

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `List` | `ctx` | `([]types.IntegrationResource, error)` |
| `Get` | `ctx`, `id int64` | `(*types.IntegrationResource, error)` |
| `Create` | `ctx`, `pluginName string`, `request types.IntegrationRQ` | `(*types.EntryCreatedRS, error)` |
| `Update` | `ctx`, `id int64`, `request types.IntegrationRQ` | `(*types.OperationCompletionRS, error)` |
| `Delete` | `ctx`, `id int64` | `(*types.OperationCompletionRS, error)` |

**Notable:** `List` has no pagination — the API returns all integrations for the project at once. `Create` takes a `pluginName` parameter separate from the request body (same pattern as global integrations). There is also `ListByPlugin(ctx, pluginName)` for filtered listing.

### Key types

The request and response types are identical to global integrations:

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
```

## Scope

Create `internal/cli/project_integration.go` and `internal/cli/project_integration_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get project-integration` | List all integrations in a project |
| `rp get project-integration [--plugin <name>]` | List integrations filtered by plugin type |
| `rp get project-integration <id>` | Get a single project integration by numeric ID |
| `rp create project-integration --plugin <name> --from-file <path>` | Create a project integration |
| `rp update project-integration <id> --from-file <path>` | Update a project integration |
| `rp delete project-integration <id>` | Delete a project integration by numeric ID |

## Implementation

### 1. Create `internal/cli/project_integration.go`

This file contains all five command constructors and their run functions. The structure closely mirrors `integration.go` (PR 10) but with project scoping.

#### `newGetProjectIntegrationCmd()`

```go
func newGetProjectIntegrationCmd() *cobra.Command
```

- `Use`: `"project-integration [id]"`
- `Short`: `"List project integrations or get a specific integration"`
- `Long`: Include examples showing list, list filtered by plugin, get-by-ID, project override, and output format.
- `Args`: `cobra.MaximumNArgs(1)`
- `RunE`: `withPrintedErrors(runGetProjectIntegration)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--plugin` (string, optional): filter list results by plugin name (only meaningful in list mode)

**Do NOT** register pagination flags — the API returns all results at once.

The run function:
1. Resolve connection, project, and client
2. If `len(args) == 1`, parse arg as int64 using `parseResourceID(args[0], "project-integration")`, call `scope.Get(ctx, id)`, print as single row. **Ignore `--plugin` in this path** — an explicit ID is unambiguous and `--plugin` is only meaningful for list filtering.
3. Otherwise:
   - If `--plugin` is non-empty, call `scope.ListByPlugin(ctx, pluginName)`
   - Otherwise, call `scope.List(ctx)`
   - Pass the resulting `[]types.IntegrationResource` slice to `printer.Print`

#### `newCreateProjectIntegrationCmd()`

```go
func newCreateProjectIntegrationCmd() *cobra.Command
```

- `Use`: `"project-integration"`
- `Short`: `"Create a project integration"`
- `Long`: Include examples showing `--plugin` and `--from-file` usage
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateProjectIntegration)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--plugin` (string, required): the plugin type name for the integration
  - `--from-file` / `-f` (string, required): path to JSON file containing `IntegrationRQ`

The run function:
1. Validate `--plugin` is non-empty
2. Validate `--from-file` is non-empty
3. Read and unmarshal the file using `readRequestFromFile[types.IntegrationRQ](path)`
4. Resolve connection, project, and client
5. Call `scope.Create(ctx, pluginName, request)`
6. Print the `EntryCreatedRS` result using `entryCreatedHeaders()` and `entryCreatedRows(result)`

#### `newUpdateProjectIntegrationCmd()`

```go
func newUpdateProjectIntegrationCmd() *cobra.Command
```

- `Use`: `"project-integration <id>"`
- `Short`: `"Update a project integration"`
- `Long`: Include examples showing `--from-file` with an ID argument
- `Args`: `cobra.ExactArgs(1)` (the numeric integration ID)
- `RunE`: `withPrintedErrors(runUpdateProjectIntegration)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--from-file` / `-f` (string, required): path to JSON file containing `IntegrationRQ`

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "project-integration")`
2. Validate `--from-file` is non-empty
3. Read and unmarshal the file using `readRequestFromFile[types.IntegrationRQ](path)`
4. Resolve connection, project, and client
5. Call `scope.Update(ctx, id, request)`
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newDeleteProjectIntegrationCmd()`

```go
func newDeleteProjectIntegrationCmd() *cobra.Command
```

- `Use`: `"project-integration <id>"`
- `Short`: `"Delete a project integration"`
- `Long`: Include example showing delete by ID
- `Args`: `cobra.ExactArgs(1)` (the numeric integration ID)
- `RunE`: `withPrintedErrors(runDeleteProjectIntegration)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "project-integration")`
2. Resolve connection, project, and client
3. Call `scope.Delete(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### Table formatting

Reuse the same column layout as global integrations (PR 10):

```go
func projectIntegrationHeaders() []string {
    return []string{"ID", "NAME", "TYPE", "ENABLED", "CREATOR"}
}

func projectIntegrationRow(integration *types.IntegrationResource) ([]string, []string) {
    typeName := ""
    if integration.IntegrationType != nil {
        typeName = formatStringPtr(integration.IntegrationType.Name)
    }

    return projectIntegrationHeaders(), []string{
        formatInt64Ptr(integration.Id),
        formatStringPtr(integration.Name),
        typeName,
        formatBoolPtr(integration.Enabled),
        formatStringPtr(integration.Creator),
    }
}
```

**Do NOT** share the formatter functions between `integration.go` and `project_integration.go`. Although the table layout is identical, keeping the formatters separate avoids coupling two files that should be independently maintainable. If the columns diverge later (e.g., project integrations show `ProjectId`), no refactoring is needed.

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// get
get.AddCommand(newGetProjectIntegrationCmd())

// create
create.AddCommand(newCreateProjectIntegrationCmd())

// update
update.AddCommand(newUpdateProjectIntegrationCmd())

// delete
del.AddCommand(newDeleteProjectIntegrationCmd())
```

### 3. Create `internal/cli/project_integration_test.go`

Use the same testing pattern as the pilot test files:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetProjectIntegrationCmd_HasProjectFlag`** — Verify the get command has a `--project` / `-p` flag registered.

2. **`TestNewGetProjectIntegrationCmd_HasPluginFlag`** — Verify the get command has a `--plugin` flag registered.

3. **`TestNewGetProjectIntegrationCmd_NoListFlags`** — Verify the get command does NOT have `--page-size`, `--page`, or `--sort` flags.

4. **`TestProjectIntegrationRow_Formatting`** — Verify `projectIntegrationRow` produces the expected headers and values for a sample `IntegrationResource`.

5. **`TestProjectIntegrationRow_NilIntegrationType`** — Verify `projectIntegrationRow` handles nil `IntegrationType`.

6. **`TestProjectIntegrationRow_AllNilFields`** — Verify `projectIntegrationRow` handles an `IntegrationResource` with all nil fields (produces empty strings for all columns).

7. **`TestNewCreateProjectIntegrationCmd_Flags`** — Verify create has `--plugin`, `--from-file` / `-f`, and `--project` flags.

8. **`TestNewUpdateProjectIntegrationCmd_Flags`** — Verify update has `--from-file` / `-f` and `--project` flags and accepts exactly 1 arg.

9. **`TestNewDeleteProjectIntegrationCmd_Args`** — Verify delete accepts exactly 1 arg and has `--project` flag.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Project-scoped means `--project` is required.** This is the key difference from global integrations (PR 10). All commands register `--project` / `-p`.
- **Same request/response types as global integrations.** `IntegrationRQ` and `IntegrationResource` are shared types. The difference is which scope methods are called and whether project resolution is needed.
- **Same table format as global integrations.** This ensures a consistent user experience. The columns are identical; only the resource name in `Use` differs.
- **Separate formatter functions.** Although the formatters are structurally identical to `integration.go`, keeping them separate is intentional. These two files are in separate worktrees during Phase 4 and should not share private functions. If duplication becomes a maintenance concern, it can be consolidated in PR 16 (Phase 5 fan-in).
- **`--from-file` for create and update.** Same rationale as global integrations — `IntegrationParameters` is an arbitrary map.
- **`--plugin` on create and get.** Same pattern as global integrations.
- **No pagination.** List endpoints return all integrations for the project.
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr`, `readRequestFromFile[types.IntegrationRQ]`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`.

## What NOT to do

- **Do not implement `ExecuteCommand`, `TestConnection`, or `DeleteAllByType`.** Those are stretch goals or specialized operations for later phases.
- **Do not share formatter functions with `integration.go`.** Keep them separate for independent maintainability.
- **Do not add pagination flags** to the get command.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` now shows `project-integration` as a subcommand
- `rp create --help` now shows `project-integration` as a subcommand
- `rp update --help` now shows `project-integration` as a subcommand
- `rp delete --help` now shows `project-integration` as a subcommand
- `rp get project-integration --help` shows `--project` and `--plugin` flags but NO pagination flags
- `rp create project-integration --help` shows `--plugin`, `--from-file`, and `--project` flags
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/project_integration.go` created with all five command constructors and their run functions
- [ ] `internal/cli/project_integration_test.go` created with unit tests for flag registration and table formatting
- [ ] Commands registered in `commands.go`
- [ ] All commands use `--project` / `-p` for project resolution
- [ ] Get command supports `--plugin` for filtered listing
- [ ] Create uses `--plugin` and `--from-file`; update uses `--from-file` only
- [ ] No pagination flags
- [ ] Table output shows ID, NAME, TYPE, ENABLED, CREATOR with uppercase headers
- [ ] JSON output for single-get prints the `IntegrationResource` object; for list mode prints the `[]IntegrationResource` slice
- [ ] Formatter functions are local to `project_integration.go` (not shared with `integration.go`)
- [ ] Shared helpers used: `parseResourceID`, `formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr`, `readRequestFromFile[types.IntegrationRQ]`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
