# PR 8: `integration` CRUD

## Goal

Implement `rp integration {list,get,create,update,delete}` with unified scope — the same command handles both global (instance-level) and project-scoped integrations. The `--project` flag determines scope: omit it for global (admin-only), include it for project-scoped.

## Context

This PR runs in the Phase 4 parallel fan-out. PRs 1–5 have landed.

Integrations are architecturally unique in the CLI because:
1. **Dual scope under one command** — the same data types (`IntegrationResource`, `IntegrationRQ`) are used for both global and project-scoped operations, but the client methods are on different scope objects.
2. **Not paginated** — both global and project list endpoints return the full slice.
3. **`--plugin` filter on list** — list can optionally filter by plugin name via `ListGlobalByPlugin` / `ListByPlugin`.
4. **`--plugin` required on create** — both `CreateGlobal` and project `Create` require a `pluginName` parameter separate from the request body.
5. **`--from-file` for create/update** — `IntegrationRQ.IntegrationParameters` is a `map[string]interface{}`, making it unsuitable for flags.

### Prerequisites

- PRs 1–5 have landed

### Client API surface

**Global** — `IntegrationsScope` (obtained via `client.Integrations()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `ListGlobal` | `ListGlobal(ctx)` | `[]types.IntegrationResource` |
| `ListGlobalByPlugin` | `ListGlobalByPlugin(ctx, pluginName string)` | `[]types.IntegrationResource` |
| `GetGlobal` | `GetGlobal(ctx, id int64)` | `*types.IntegrationResource` |
| `CreateGlobal` | `CreateGlobal(ctx, pluginName string, types.IntegrationRQ)` | `*types.EntryCreatedRS` |
| `UpdateGlobal` | `UpdateGlobal(ctx, id int64, types.IntegrationRQ)` | `*types.OperationCompletionRS` |
| `DeleteGlobal` | `DeleteGlobal(ctx, id int64)` | `*types.OperationCompletionRS` |

**Project-scoped** — `ProjectIntegrationsScope` (obtained via `client.Project(projectName).Integrations()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `List` | `List(ctx)` | `[]types.IntegrationResource` |
| `ListByPlugin` | `ListByPlugin(ctx, pluginName string)` | `[]types.IntegrationResource` |
| `Get` | `Get(ctx, id int64)` | `*types.IntegrationResource` |
| `Create` | `Create(ctx, pluginName string, types.IntegrationRQ)` | `*types.EntryCreatedRS` |
| `Update` | `Update(ctx, id int64, types.IntegrationRQ)` | `*types.OperationCompletionRS` |
| `Delete` | `Delete(ctx, id int64)` | `*types.OperationCompletionRS` |

### Relevant types

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

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

## Implementation plan

### 1. Create `internal/cli/integration.go`

This file defines the `rp integration` parent command and all verb subcommands.

#### Parent command

```go
func newIntegrationCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "integration",
        Short: "Manage integrations",
    }
    // Project flag is optional — determines global vs project scope
    cmd.PersistentFlags().StringP("project", "p", "", "project name (omit for global scope)")
    cmd.AddCommand(newIntegrationListCmd())
    cmd.AddCommand(newIntegrationGetCmd())
    cmd.AddCommand(newIntegrationCreateCmd())
    cmd.AddCommand(newIntegrationUpdateCmd())
    cmd.AddCommand(newIntegrationDeleteCmd())
    return cmd
}
```

Register in `root.go`: `cmd.AddCommand(newIntegrationCmd())`.

#### Scope resolution helper

The key architectural decision: use `--project` to determine scope. Create a helper that returns the appropriate integration operations interface:

```go
// integrationOps abstracts over global and project-scoped integration operations.
type integrationOps interface {
    list(ctx context.Context, plugin string) ([]types.IntegrationResource, error)
    get(ctx context.Context, id int64) (*types.IntegrationResource, error)
    create(ctx context.Context, plugin string, request types.IntegrationRQ) (*types.EntryCreatedRS, error)
    update(ctx context.Context, id int64, request types.IntegrationRQ) (*types.OperationCompletionRS, error)
    delete(ctx context.Context, id int64) (*types.OperationCompletionRS, error)
}
```

Implement two concrete types wrapping `IntegrationsScope` and `ProjectIntegrationsScope`. The handler resolves scope once and operates through the interface.

Alternatively, if an interface feels heavy, the handler can branch on `cmd.Flags().Changed("project")` and dispatch directly:

```go
func resolveIntegrationScope(cmd *cobra.Command, ctx *cmdContext) (isProject bool, project string, err error) {
    if !cmd.Flags().Changed("project") {
        return false, "", nil
    }

    project, err = ctx.resolveProject(cmd)
    if err != nil {
        return false, "", err
    }

    return true, project, nil
}
```

Use whichever approach keeps the code cleanest. The interface approach avoids duplicated dispatch logic in each verb handler.

#### `rp integration list`

- No positional arguments
- Optional flag: `--plugin` (string) — filter integrations by plugin name
- **No pagination** — returns all integrations
- Scope dispatch:
  - Without `--project`: call `ListGlobal` or `ListGlobalByPlugin(pluginName)` depending on `--plugin`
  - With `--project`: call `List` or `ListByPlugin(pluginName)` depending on `--plugin`
- Table output columns: `ID`, `NAME`, `PLUGIN`, `ENABLED`, `CREATOR`
  - `PLUGIN` is derived from `IntegrationType.Name` (nil-safe)
  - `ENABLED` is a boolean column
- JSON output: the raw `[]IntegrationResource` slice

#### `rp integration get <id>`

- Exactly one positional argument: numeric integration ID
- Parse using `parseIDArg(args[0], "integration")`
- Scope dispatch:
  - Without `--project`: call `GetGlobal(ctx, id)`
  - With `--project`: call `Get(ctx, id)`
- Table output: same columns as list, single row
- JSON output: the raw `*IntegrationResource`

#### `rp integration create --plugin <name> --from-file <path>`

- No positional arguments
- Required flags:
  - `--plugin` (string) — the plugin name (e.g., `"rally"`, `"jira"`, `"email"`)
  - `--from-file <path>` — path to a JSON file containing an `IntegrationRQ` payload
- Read file with `readRequestFromFile[types.IntegrationRQ](path)`
- Scope dispatch:
  - Without `--project`: call `CreateGlobal(ctx, pluginName, request)`
  - With `--project`: call `Create(ctx, pluginName, request)`
- Table output: `ID` column (from `EntryCreatedRS.Id`)
- JSON output: the raw `*EntryCreatedRS`

#### `rp integration update <id> --from-file <path>`

- Exactly one positional argument: numeric integration ID
- Required flag: `--from-file <path>`
- Read file with `readRequestFromFile[types.IntegrationRQ](path)`
- Scope dispatch:
  - Without `--project`: call `UpdateGlobal(ctx, id, request)`
  - With `--project`: call `Update(ctx, id, request)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

#### `rp integration delete <id>`

- Exactly one positional argument: numeric integration ID
- No additional flags
- Scope dispatch:
  - Without `--project`: call `DeleteGlobal(ctx, id)`
  - With `--project`: call `Delete(ctx, id)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

### 2. Table formatting

```go
func integrationHeaders() []string {
    return []string{"ID", "NAME", "PLUGIN", "ENABLED", "CREATOR"}
}

func integrationRow(r *types.IntegrationResource) ([]string, []string) {
    plugin := ""
    if r.IntegrationType != nil {
        plugin = formatStringPtr(r.IntegrationType.Name)
    }

    return integrationHeaders(), []string{
        formatInt64Ptr(r.Id),
        formatStringPtr(r.Name),
        plugin,
        formatBoolPtr(r.Enabled),
        formatStringPtr(r.Creator),
    }
}
```

### 3. Register in root.go

Add `cmd.AddCommand(newIntegrationCmd())` alongside the other resource commands.

## Test requirements

### Unit tests (`internal/cli/integration_test.go`)

- **ID parsing:** Verify that `rp integration get abc` returns an error
- **Scope resolution:**
  - Without `--project`, scope resolves to global
  - With `--project`, scope resolves to project
  - `--project ""` (empty) errors
- **Create validation:**
  - Missing `--plugin` errors
  - Missing `--from-file` errors
- **`--from-file` validation:** reuse patterns from existing file-based tests
  - Non-existent file errors
  - Malformed JSON errors
- **Plugin filter on list:** `--plugin` is correctly resolved and applied
- **Table formatting:** Verify `integrationRow` handles nil fields (all fields are pointers), especially nested `IntegrationType.Name`
- **Command wiring:** Verify `newIntegrationCmd()` has the expected subcommands

## Edge cases and constraints

- **All `IntegrationResource` fields are pointers:** Every field on `IntegrationResource` is a pointer type. Table formatting must handle nil gracefully for every column.
- **`IntegrationRQ` fields are all optional pointers:** The request type has no required fields at the Go type level. The API may enforce requirements server-side. Do not add client-side validation beyond checking the file was parsed successfully.
- **Global scope requires admin privileges:** The help text should note that global integration operations typically require administrator access.
- **`--project` flag behavior differs from other resources:** For most resources, `--project` is resolved via the standard `resolveProject` chain (flag → env → profile). For integrations, **omitting `--project` entirely is a valid choice** that means "global scope." The command should NOT fall back to env or profile defaults for project — only use the flag if explicitly provided. Use `cmd.Flags().Changed("project")` to detect explicit use, not `resolveProject`.
- **Empty `--plugin` on list:** If `--plugin` is set to an empty string, treat it as absent (no filter). Use `cmd.Flags().Changed("plugin")` to detect explicit use.
- **Nil response from list:** Both `ListGlobal` and project `List` can return nil slices. The client already converts nil to empty slices, so this should be handled, but verify.
- **`--plugin` is separate from `--from-file` on create:** The plugin name is a path parameter in the API, not part of the request body. The `IntegrationRQ` JSON file should NOT include a `plugin` field — it's provided via the `--plugin` flag.

## Example commands and expected output

```bash
# List all global integrations
$ rp integration list
ID    NAME                PLUGIN    ENABLED  CREATOR
1     JIRA Integration    jira      true     superadmin
2     Email Server        email     true     superadmin

# List global integrations filtered by plugin
$ rp integration list --plugin jira
ID    NAME                PLUGIN    ENABLED  CREATOR
1     JIRA Integration    jira      true     superadmin

# List project-scoped integrations
$ rp integration list -p my_project
ID    NAME                PLUGIN    ENABLED  CREATOR
10    Project JIRA        jira      true     admin

# Get a global integration
$ rp integration get 1
ID    NAME                PLUGIN    ENABLED  CREATOR
1     JIRA Integration    jira      true     superadmin

# Get a project integration
$ rp integration get 10 -p my_project
ID    NAME                PLUGIN    ENABLED  CREATOR
10    Project JIRA        jira      true     admin

# Create a global integration
$ rp integration create --plugin email --from-file email_integration.json
ID
3

# Create a project integration
$ rp integration create --plugin jira --from-file jira_integration.json -p my_project
ID
11

# Update
$ rp integration update 1 --from-file updated_integration.json
MESSAGE
Integration with ID = '1' has been successfully updated.

# Delete
$ rp integration delete 1
MESSAGE
Integration with ID = '1' has been successfully deleted.
```

Example `email_integration.json`:
```json
{
  "name": "Email Notifications",
  "enabled": true,
  "integrationParameters": {
    "host": "smtp.example.com",
    "port": 587,
    "protocol": "smtp",
    "authEnabled": true,
    "from": "reportportal@example.com"
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

- [ ] `internal/cli/integration.go` implements all five subcommands (list, get, create, update, delete)
- [ ] `rp integration` registered in `root.go`
- [ ] Unified scope: `--project` determines global vs project-scoped operations
- [ ] `--project` omission means global scope (does NOT fall back to env/profile defaults)
- [ ] `--plugin` required on create, optional filter on list
- [ ] `--from-file` for create/update with `IntegrationRQ` payload
- [ ] Table output with uppercase headers: ID, NAME, PLUGIN, ENABLED, CREATOR
- [ ] JSON output passes through raw API types unchanged
- [ ] Unit tests cover scope resolution, flag validation, table formatting, and command wiring
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
