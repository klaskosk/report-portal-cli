# PR 13: `plugin` Base Commands

## Goal

Implement `rp plugin {list,delete,update}` for the standard plugin operations. The `exec` and `create` (file upload) commands are deferred to Phase 7 (PR 16).

## Context

This PR runs in the Phase 4 parallel fan-out. PRs 1–5 have landed.

Plugins are instance-scoped (no `--project` flag) and admin-only. They have an unusual shape:

1. **`List` returns `[]IntegrationTypeResource`** — plugins are integration *types*, not integration instances. The resource type is shared with the integration type metadata.
2. **`UpdateState` is a toggle, not a general update** — it enables or disables a plugin. The request body is just `{enabled: bool}`.
3. **No `get` method** — the API has no single-get-by-ID for plugins. Discovery is via `list`.
4. **`exec` and `create --file` are deferred** — these are specialized operations for Phase 7.

### Prerequisites

- PRs 1–5 have landed

### Client API surface

`PluginScope` (obtained via `client.Plugins()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `List` | `List(ctx)` | `[]types.IntegrationTypeResource` |
| `UpdateState` | `UpdateState(ctx, pluginID int64, types.UpdatePluginStateRQ)` | `*types.OperationCompletionRS` |
| `Delete` | `Delete(ctx, pluginID int64)` | `*types.OperationCompletionRS` |

### Relevant types

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

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
}
```

`IntegrationTypeResource.Type` is the numeric plugin ID (`*int64`). The field name is `Type` in the Go struct, corresponding to `"type"` in JSON. This is the value used as `pluginID` in `UpdateState` and `Delete`.

## Implementation plan

### 1. Create `internal/cli/plugin.go`

This file defines the `rp plugin` parent command and all verb subcommands.

#### Parent command

```go
func newPluginCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "plugin",
        Short: "Manage plugins (admin)",
    }
    cmd.AddCommand(newPluginListCmd())
    cmd.AddCommand(newPluginUpdateCmd())
    cmd.AddCommand(newPluginDeleteCmd())
    return cmd
}
```

Register in `root.go`: `cmd.AddCommand(newPluginCmd())`.

**No `-p`/`--project` flag** — plugins are instance-scoped.

#### `rp plugin list`

- No positional arguments
- **No pagination flags** — `List` returns all plugins. Do NOT register `--page`, `--page-size`, or `--sort`.
- Table output columns: `ID`, `NAME`, `GROUP`, `ENABLED`, `PLUGIN TYPE`
  - `ID` is from `IntegrationTypeResource.Type` (the numeric ID field)
  - `NAME` from `Name`
  - `GROUP` from `GroupType`
  - `ENABLED` is boolean
  - `PLUGIN TYPE` from `PluginType`
- JSON output: the raw `[]IntegrationTypeResource` slice

#### `rp plugin update <id>`

- Exactly one positional argument: numeric plugin ID
- Parse using `parseIDArg(args[0], "plugin")`
- Required flag: `--enabled` (bool) — set the plugin's enabled state
- The `--enabled` flag must be explicitly provided. Unlike a typical boolean flag that defaults to `false`, this flag controls a toggle and should require explicit intent. Use `cmd.MarkFlagRequired("enabled")` so the user must write `--enabled=true` or `--enabled=false`.
- Build `types.UpdatePluginStateRQ{Enabled: &enabled}`
- Call `scope.UpdateState(ctx, pluginID, request)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

#### `rp plugin delete <id>`

- Exactly one positional argument: numeric plugin ID
- Parse using `parseIDArg(args[0], "plugin")`
- No additional flags
- Call `scope.Delete(ctx, pluginID)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

### 2. Table formatting

```go
func pluginHeaders() []string {
    return []string{"ID", "NAME", "GROUP", "ENABLED", "PLUGIN TYPE"}
}

func pluginRow(p *types.IntegrationTypeResource) ([]string, []string) {
    return pluginHeaders(), []string{
        formatInt64Ptr(p.Type),
        formatStringPtr(p.Name),
        formatStringPtr(p.GroupType),
        formatBoolPtr(p.Enabled),
        formatStringPtr(p.PluginType),
    }
}
```

### 3. Register in root.go

Add `cmd.AddCommand(newPluginCmd())` alongside the other resource commands.

## Test requirements

### Unit tests (`internal/cli/plugin_test.go`)

- **ID parsing:** Verify that `rp plugin update abc --enabled=true` returns an error
- **Update validation:**
  - Missing `--enabled` flag errors (flag is required)
  - `--enabled=true` produces `UpdatePluginStateRQ{Enabled: ptr(true)}`
  - `--enabled=false` produces `UpdatePluginStateRQ{Enabled: ptr(false)}`
- **Table formatting:** Verify `pluginRow` produces expected columns, handles nil for all pointer fields (every field on `IntegrationTypeResource` is a pointer)
- **Command wiring:** Verify `newPluginCmd()` has exactly `list`, `update`, `delete` subcommands (no `get`, no `create`, no `exec`)
- **No project flag:** Verify that no `-p`/`--project` flag is registered on the plugin command tree

## Edge cases and constraints

- **No `get` subcommand:** The API has no single-get-by-ID for plugins. Do NOT create a `get` subcommand.
- **No `create` subcommand in this PR:** Plugin creation (file upload) is deferred to Phase 7 (PR 16). Do NOT create a `create` subcommand.
- **No `exec` subcommand in this PR:** Plugin command execution is deferred to Phase 7 (PR 16). Do NOT create an `exec` subcommand.
- **All `IntegrationTypeResource` fields are pointers:** Every field is a pointer type. Table formatting must handle nil gracefully for every column.
- **`IntegrationTypeResource.Type` is the plugin ID:** The field is confusingly named `Type` in Go (from `"type"` in JSON), but it's the numeric ID used in `UpdateState` and `Delete`. Make sure the table `ID` column displays this field, not the `PluginType` string.
- **`--enabled` is required, not optional:** Unlike most boolean flags that default to `false` and are optional, the `--enabled` flag is required because the update operation is specifically about changing the enabled state. Without it, there's nothing to update. Use `MarkFlagRequired`.
- **Admin-only operations:** All plugin operations require admin privileges. Note this in the parent command's help text (Short/Long description).
- **`List` returns a direct slice:** No pagination, no page wrapper. The client handles nil-to-empty conversion internally, but verify the CLI handles an empty list gracefully.
- **Plugin `Details` field:** `IntegrationTypeResource.Details` is a `*map[string]interface{}`. It's included in JSON output but not in table output (too complex for tabular display).

## Example commands and expected output

```bash
# List all plugins
$ rp plugin list
ID    NAME              GROUP        ENABLED  PLUGIN TYPE
1     Rally             bts          true     rally
2     JIRA              bts          true     jira
3     Email Server      notification true     email
4     LDAP              auth         false    ldap

# Enable a plugin
$ rp plugin update 4 --enabled=true
MESSAGE
Plugin state has been updated successfully.

# Disable a plugin
$ rp plugin update 4 --enabled=false
MESSAGE
Plugin state has been updated successfully.

# Delete a plugin
$ rp plugin delete 4
MESSAGE
Plugin has been deleted successfully.
```

## Validation

Run before finalizing:

```
make lint
make test
```

## Definition of done

- [ ] `internal/cli/plugin.go` implements list, update (state toggle), and delete subcommands
- [ ] No `get`, `create`, or `exec` subcommands (deferred to Phase 7)
- [ ] `rp plugin` registered in `root.go`
- [ ] `--enabled` flag is required on update (explicit toggle)
- [ ] Table output with uppercase headers: ID, NAME, GROUP, ENABLED, PLUGIN TYPE
- [ ] JSON output passes through raw API types unchanged
- [ ] Admin-only noted in help text
- [ ] Unit tests cover ID parsing, update validation, table formatting, and command wiring
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
