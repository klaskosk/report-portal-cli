# PR 30: Pattern-Template Settings Commands

## Goal

Implement create, update, and delete commands for pattern templates. Pattern templates are a sub-resource of project settings, exposed as a flat resource name under standard verbs. There is no dedicated list endpoint and `ProjectSettingsResource` does not include pattern templates (it only contains `SubTypes` for issue sub-types), so there is no CLI-accessible discovery path for existing template IDs.

## Context

Pattern templates are managed through `ProjectSettingsScope` on the client (`client.Project(name).Settings()`). They are **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `parseResourceID`, `mustGetStringFlag`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`)
- PR 17 (frozen UX: §6b pattern templates as flat resource under standard verbs)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `CreatePatternTemplate` | `ctx`, `request types.CreatePatternTemplateRQ` | `(*types.EntryCreatedRS, error)` |
| `UpdatePatternTemplate` | `ctx`, `id int64`, `request types.UpdatePatternTemplateRQ` | `(*types.OperationCompletionRS, error)` |
| `DeletePatternTemplate` | `ctx`, `id int64` | `(*types.OperationCompletionRS, error)` |

**Notable:** There is no `GetPatternTemplates` or `ListPatternTemplates` method, and `ProjectSettingsResource` only contains `Project` (int64) and `SubTypes` (issue sub-types) — it does **not** include pattern templates. There is no CLI-accessible way to list existing pattern templates. The only way to know a template's ID is from the `EntryCreatedRS` returned by the create command. Help text should document this limitation.

### Key types

```go
type CreatePatternTemplateRQ struct {
    Enabled bool   `json:"enabled"`
    Name    string `json:"name"`
    Type    string `json:"type"`
    Value   string `json:"value"`
}

type UpdatePatternTemplateRQ struct {
    Enabled bool   `json:"enabled"`
    Name    string `json:"name"`
}
```

**All fields are non-pointer types.** Both create and update require all their fields to be explicitly set. `UpdatePatternTemplateRQ` has only 2 fields — the API does not support partial updates for pattern templates.

## Scope

Create `internal/cli/pattern_template.go` and `internal/cli/pattern_template_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp create pattern-template --name <name> --type <type> --value <value> [--enabled=true|false]` | Create a pattern template (enabled by default) |
| `rp update pattern-template <id> --name <name> --enabled=true|false` | Update a pattern template |
| `rp delete pattern-template <id>` | Delete a pattern template by numeric ID |

## Implementation

### 1. Create `internal/cli/pattern_template.go`

#### `newCreatePatternTemplateCmd()`

```go
func newCreatePatternTemplateCmd() *cobra.Command
```

- `Use`: `"pattern-template"`
- `Short`: `"Create a pattern template"`
- `Long`: Include examples. Document that `--type` specifies the analysis pattern type (e.g., `"string"`, `"regex"`) and `--value` is the pattern expression.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreatePatternTemplate)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--name` (string, required): template name
  - `--type` (string, required): pattern type
  - `--value` (string, required): pattern value/expression
  - `--enabled` (bool, default `true`): whether the template is enabled. Defaults to `true` because templates are typically created to be used immediately. Pass `--enabled=false` to create a disabled template.

**Why inline flags?** `CreatePatternTemplateRQ` has 4 simple scalar fields. This is within the inline flag comfort zone.

The run function:
1. Validate `--name` is non-empty
2. Validate `--type` is non-empty
3. Validate `--value` is non-empty
4. Read `--enabled` via `cmd.Flags().GetBool("enabled")`
5. Build `types.CreatePatternTemplateRQ{Name: name, Type: patternType, Value: value, Enabled: enabled}`
6. Resolve connection, project, and client
7. Call `scope.CreatePatternTemplate(ctx, request)`
8. Print the `EntryCreatedRS` result using `entryCreatedHeaders()` and `entryCreatedRows(result)`

#### `newUpdatePatternTemplateCmd()`

```go
func newUpdatePatternTemplateCmd() *cobra.Command
```

- `Use`: `"pattern-template <id>"`
- `Short`: `"Update a pattern template"`
- `Long`: Include examples. Note that both `--name` and `--enabled` must be provided. Explain that the API requires both fields and there is no read endpoint that would let the CLI preserve `Enabled` implicitly.
- `Args`: `cobra.ExactArgs(1)` (the numeric pattern template ID)
- `RunE`: `withPrintedErrors(runUpdatePatternTemplate)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--name` (string, required): new template name
  - `--enabled` (bool, default `false`): whether the template is enabled (`--enabled` or `--enabled=false`)

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "pattern-template")`
2. Validate `--name` is non-empty
3. Validate that `--enabled` was explicitly provided via `cmd.Flags().Changed("enabled")`; if not, return `fmt.Errorf("--enabled is required (use --enabled or --enabled=false)")`
4. Read `--enabled` via `cmd.Flags().GetBool("enabled")`
5. Build `types.UpdatePatternTemplateRQ{Name: name, Enabled: enabled}`
6. Resolve connection, project, and client
7. Call `scope.UpdatePatternTemplate(ctx, id, request)`
8. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

**Important:** `Enabled` is a required API field on update. Even though the Go type is `bool`, the CLI should require the user to set `--enabled` explicitly so it does not silently disable a template by omission.

#### `newDeletePatternTemplateCmd()`

```go
func newDeletePatternTemplateCmd() *cobra.Command
```

- `Use`: `"pattern-template <id>"`
- `Short`: `"Delete a pattern template"`
- `Long`: Include example.
- `Args`: `cobra.ExactArgs(1)` (the numeric ID)
- `RunE`: `withPrintedErrors(runDeletePatternTemplate)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "pattern-template")`
2. Resolve connection, project, and client
3. Call `scope.DeletePatternTemplate(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

### 2. Register commands in `commands.go`

```go
// create
create.AddCommand(newCreatePatternTemplateCmd())

// update
update.AddCommand(newUpdatePatternTemplateCmd())

// delete
del.AddCommand(newDeletePatternTemplateCmd())
```

### 3. Create `internal/cli/pattern_template_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewCreatePatternTemplateCmd_Flags`** — Verify `--name`, `--type`, `--value`, `--enabled`, and `--project` / `-p` flags. Verify `--enabled` defaults to `true`.

2. **`TestNewUpdatePatternTemplateCmd_Flags`** — Verify `--name`, `--enabled`, and `--project` / `-p`. Verify 1 arg. Verify no `--type` or `--value` (not updatable).

3. **`TestUpdatePatternTemplateCmd_RequiresEnabled`** — Verify update returns a descriptive error when `--enabled` is omitted.

4. **`TestNewDeletePatternTemplateCmd_Flags`** — Verify `--project` / `-p`. Verify 1 arg.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **No list command and no discovery path.** There is no dedicated list endpoint, and `ProjectSettingsResource` does not include pattern templates. The only way to know a template's ID is from the `EntryCreatedRS` returned at creation time. Help text must document this.
- **Inline flags for both create and update.** Both request types are simple scalars (3–4 fields).
- **Create defaults `--enabled` to true, but update requires explicit `--enabled`.** Per PR 17, create may omit the flag and default to enabled. Update must force the user to choose `--enabled` or `--enabled=false` so the CLI does not silently disable the template.
- **`Type` and `Value` are not updatable.** `UpdatePatternTemplateRQ` only has `Name` and `Enabled`. If a user needs to change the pattern type or value, they must delete and re-create.
- **Reuse shared helpers.** Use `parseResourceID`, `mustGetStringFlag`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`.

## What NOT to do

- **Do not add a `get` command** for pattern templates. There is no list endpoint, and `ProjectSettingsResource` does not include them.
- **Do not validate `--type` values locally.** Valid types may vary.
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp create --help` shows `pattern-template`
- `rp update --help` shows `pattern-template`
- `rp delete --help` shows `pattern-template`
- `rp create pattern-template --help` shows all flags

## Definition of done

- [ ] `internal/cli/pattern_template.go` created
- [ ] `internal/cli/pattern_template_test.go` created
- [ ] Commands registered in `commands.go` (create, update, delete — no get)
- [ ] All commands use `--project` / `-p`
- [ ] Inline flags for simple payloads
- [ ] Update requires explicit `--enabled`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
