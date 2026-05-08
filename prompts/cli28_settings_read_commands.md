# PR 28: Settings Commands Under `config`

## Goal

Implement the `config` command family settled in PR 17:

- `rp config get-server`
- `rp config get-project`
- `rp config set-server`
- `rp config set-analytics`

This PR owns both the read and write settings commands. The `config` verb intentionally mixes instance-scoped and project-scoped subcommands.

## Context

Settings operations span two scopes:
- `SettingsScope` (`client.Settings()`): server-level reads and mutations — **instance-scoped**, admin-only
- `ProjectSettingsScope` (`client.Project(name).Settings()`): project-level settings read — **project-scoped**

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `formatStringPtr`, `formatInt64Ptr`, `messageResultHeaders`, `operationResultRows`)
- PR 17 (frozen UX: §6a settings grouping and §7 settings mutations)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `SettingsScope.GetServerSettings` | `ctx` | `(map[string]string, error)` |
| `ProjectSettingsScope.Get` | `ctx` | `(*types.ProjectSettingsResource, error)` |
| `SettingsScope.UpdateServerSettings` | `ctx`, `types.UpdateSettingsRq` | `(*types.OperationCompletionRS, error)` |
| `SettingsScope.UpdateAnalytics` | `ctx`, `types.AnalyticsResource` | `(*types.OperationCompletionRS, error)` |

### Key types

```go
type UpdateSettingsRq struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type AnalyticsResource struct {
    Enabled *bool  `json:"enabled,omitempty"`
    Type    string `json:"type"`
}

type ProjectSettingsResource struct {
    Project  int64                             `json:"project"`
    SubTypes map[string][]IssueSubTypeResource `json:"subTypes"`
}
```

## Scope

Create `internal/cli/settings.go` and `internal/cli/settings_test.go`. Register the `config` subcommands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp config get-server` | Get server settings |
| `rp config get-project` | Get project settings |
| `rp config set-server --key <key> --value <value>` | Update a single server setting |
| `rp config set-analytics --enabled [--type <type>]` | Update analytics configuration |

## Implementation

### 1. Create `internal/cli/settings.go`

Define a lightweight `config` parent if PR 2 only scaffolded the verb placeholder, then attach the four subcommands below.

#### `newConfigGetServerCmd()`

```go
func newConfigGetServerCmd() *cobra.Command
```

- `Use`: `"get-server"`
- `Short`: `"Get server settings (instance-scoped)"`
- `Long`: Include example. Note this is instance-scoped and admin-only.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runConfigGetServer)`

**Do NOT** register `--project` — server settings are instance-scoped.

The run function:
1. Resolve connection and client
2. Call `clientInstance.Settings().GetServerSettings(ctx)`
3. Print the result

**Table formatting for `map[string]string`:** Flatten the map into sorted rows:

```go
func serverSettingsHeaders() []string {
    return []string{"KEY", "VALUE"}
}

func serverSettingsRows(settings map[string]string) [][]string {
    keys := make([]string, 0, len(settings))
    for k := range settings {
        keys = append(keys, k)
    }

    sort.Strings(keys)

    rows := make([][]string, 0, len(keys))
    for _, k := range keys {
        rows = append(rows, []string{k, settings[k]})
    }

    return rows
}
```

For JSON output, pass the map through unchanged.

#### `newConfigGetProjectCmd()`

```go
func newConfigGetProjectCmd() *cobra.Command
```

- `Use`: `"get-project"`
- `Short`: `"Get project settings (requires --project)"`
- `Long`: Include example. Note this shows the project's issue sub-type configuration.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runConfigGetProject)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Resolve connection, project, and client
2. Call `clientInstance.Project(project).Settings().Get(ctx)`
3. Print the result

**Table formatting for `ProjectSettingsResource`:** Flatten the `SubTypes` map into sorted rows:

```go
func projectSettingsHeaders() []string {
    return []string{"CATEGORY", "ID", "LOCATOR", "SHORT NAME", "LONG NAME", "COLOR"}
}

func projectSettingsRows(settings *types.ProjectSettingsResource) [][]string {
    categories := make([]string, 0, len(settings.SubTypes))
    for k := range settings.SubTypes {
        categories = append(categories, k)
    }

    sort.Strings(categories)

    var rows [][]string
    for _, category := range categories {
        for _, st := range settings.SubTypes[category] {
            rows = append(rows, []string{
                category,
                formatInt64Ptr(st.Id),
                formatStringPtr(st.Locator),
                formatStringPtr(st.ShortName),
                formatStringPtr(st.LongName),
                formatStringPtr(st.Color),
            })
        }
    }

    return rows
}
```

For JSON output, pass the full `ProjectSettingsResource` through unchanged.

#### `newConfigSetServerCmd()`

```go
func newConfigSetServerCmd() *cobra.Command
```

- `Use`: `"set-server"`
- `Short`: `"Update a server setting (instance-scoped)"`
- `Long`: Include examples. Note this is admin-only and instance-scoped.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runConfigSetServer)`
- Flags:
  - `--key` (string, required): server setting key
  - `--value` (string, required): server setting value

The run function:
1. Validate `--key` and `--value`
2. Build `types.UpdateSettingsRq{Key: key, Value: value}`
3. Resolve connection and client
4. Call `clientInstance.Settings().UpdateServerSettings(ctx, request)`
5. Print via `printer.Print(result, messageResultHeaders(), operationResultRows(result))`

#### `newConfigSetAnalyticsCmd()`

```go
func newConfigSetAnalyticsCmd() *cobra.Command
```

- `Use`: `"set-analytics"`
- `Short`: `"Update analytics settings (instance-scoped)"`
- `Long`: Include examples for `--enabled` and `--enabled=false`. Note this is admin-only and instance-scoped. Explain that `--type` defaults to the standard analytics type when omitted on servers that support only one analytics backend.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runConfigSetAnalytics)`
- Flags:
  - `--enabled` (bool, default `false`): analytics enabled state (`--enabled` or `--enabled=false`)
  - `--type` (string, optional): analytics type

Use the `optionalBoolFlag` pattern so `--enabled` must be explicitly set. If omitted, return `fmt.Errorf("--enabled is required (use --enabled or --enabled=false)")`.

The run function:
1. Read `enabled := optionalBoolFlag(cmd, "enabled")` and validate it is non-nil
2. Read `--type`
3. If `--type` was omitted, default it to the standard analytics type expected by the current server deployment (per PR 17's frozen UX, the flag remains optional because some servers only support one type today)
4. Build `types.AnalyticsResource{Enabled: enabled, Type: analyticsType}`
5. Resolve connection and client
6. Call `clientInstance.Settings().UpdateAnalytics(ctx, request)`
7. Print via `printer.Print(result, messageResultHeaders(), operationResultRows(result))`

### 2. Register commands in `commands.go`

Register all four commands under the `config` verb:

```go
// config
config.AddCommand(newConfigGetServerCmd())
config.AddCommand(newConfigGetProjectCmd())
config.AddCommand(newConfigSetServerCmd())
config.AddCommand(newConfigSetAnalyticsCmd())
```

### 3. Create `internal/cli/settings_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewConfigGetServerCmd_NoProjectFlag`** — Verify no `--project` flag. Verify `cobra.NoArgs`.

2. **`TestNewConfigGetProjectCmd_HasProjectFlag`** — Verify `--project` / `-p` is registered.

3. **`TestNewConfigSetServerCmd_Flags`** — Verify `--key` and `--value`. No `--project`. Verify `cobra.NoArgs`.

4. **`TestNewConfigSetAnalyticsCmd_Flags`** — Verify `--enabled` and `--type`. No `--project`. Verify `cobra.NoArgs`.

5. **`TestConfigSetAnalyticsCmd_RequiresEnabled`** — Verify omitting `--enabled` returns a descriptive error.

6. **`TestServerSettingsRows_WithData`** — Build a `map[string]string` with 3 entries, verify rows are sorted by key.

7. **`TestServerSettingsRows_EmptyMap`** — Verify empty map produces empty rows.

8. **`TestProjectSettingsRows_WithData`** — Build a `ProjectSettingsResource` with two categories and sub-types, verify row output with sorted categories.

9. **`TestProjectSettingsRows_EmptySubTypes`** — Verify empty `SubTypes` map produces empty rows.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **These commands live under `config`, not `get`.** PR 17 explicitly settles the settings grouping around the dedicated `config` verb.
- **Mixed scoping under `config` is intentional.** `get-server`, `set-server`, and `set-analytics` are instance-scoped. `get-project` is project-scoped. The `Short` descriptions should make that visible.
- **`set-analytics` keeps `--type` optional.** Per PR 17, omit `--type` when the deployment only supports the standard analytics backend and default it internally. The flag remains exposed for forward compatibility with servers that support multiple analytics types.
- **Server settings are a flat key-value map.** Table output sorts keys alphabetically for deterministic output.
- **Project settings show issue sub-type configuration.** The `ProjectSettingsResource` table flattens the nested `SubTypes` map for readability.
- **Notifications, pattern templates, and issue sub-types do not move under `config`.** Those remain top-level CRUD resources in PRs 29–31.
- **Reuse shared helpers.** Use `formatStringPtr`, `formatInt64Ptr`, `messageResultHeaders`, `operationResultRows`, and `optionalBoolFlag`.

## What NOT to do

- **Do not register these under `get`.** The settled UX is `rp config ...`.
- **Do not add `--project` to `get-server`, `set-server`, or `set-analytics`.**
- **Do not move notifications, pattern templates, or issue sub-types under `config`.**
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp config --help` shows `get-server`, `get-project`, `set-server`, and `set-analytics`
- `rp config get-server --help` has no `--project`
- `rp config get-project --help` has `--project`
- `rp config set-server --help` shows `--key` and `--value`
- `rp config set-analytics --help` shows `--enabled` and `--type`

## Definition of done

- [ ] `internal/cli/settings.go` created
- [ ] `internal/cli/settings_test.go` created
- [ ] Commands registered under `config`
- [ ] `get-server`, `set-server`, and `set-analytics` are instance-scoped
- [ ] `get-project` is project-scoped
- [ ] Map/nested outputs sorted for deterministic table display
- [ ] `set-analytics` requires explicit `--enabled`
- [ ] `set-analytics` defaults `--type` to the standard analytics type when omitted
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
