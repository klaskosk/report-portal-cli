# PR 31: Issue Sub-Type Settings Commands

## Goal

Implement list, create, update, and delete commands for issue sub-types. Issue sub-types are a sub-resource of project settings, exposed as a flat resource name under standard verbs.

## Context

Issue sub-types are managed through `ProjectSettingsScope` on the client (`client.Project(name).Settings()`). They are **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `parseResourceID`, `formatInt64Ptr`, `formatStringPtr`, `messageResultHeaders`, `operationResultRows`)
- PR 17 (frozen UX: issue sub-type inline flags with `--locator` for update, per §6b)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `Get` | `ctx` | `(*types.ProjectSettingsResource, error)` |
| `CreateIssueSubType` | `ctx`, `request types.CreateIssueSubTypeRQ` | `(*types.IssueSubTypeCreatedRS, error)` |
| `UpdateIssueSubType` | `ctx`, `request types.UpdateOneIssueSubTypeRQ` | `(*types.OperationCompletionRS, error)` |
| `DeleteIssueSubType` | `ctx`, `id int64` | `(*types.OperationCompletionRS, error)` |

**Notable:** `UpdateIssueSubType` takes the full `UpdateOneIssueSubTypeRQ` — the client wraps it in `UpdateIssueSubTypeRQ{Ids: []UpdateOneIssueSubTypeRQ{request}}` for the API call. The `Locator` field in the update request identifies the sub-type.

### Key types

```go
type ProjectSettingsResource struct {
    SubTypes map[string][]IssueSubTypeResource `json:"subTypes,omitempty"`
}

type IssueSubTypeResource struct {
    Color     string `json:"color"`
    Id        int64  `json:"id"`
    Locator   string `json:"locator"`
    LongName  string `json:"longName"`
    ShortName string `json:"shortName"`
    TypeRef   string `json:"typeRef"`
}

type CreateIssueSubTypeRQ struct {
    Color     string `json:"color"`
    LongName  string `json:"longName"`
    ShortName string `json:"shortName"`
    TypeRef   string `json:"typeRef"`
}

type UpdateOneIssueSubTypeRQ struct {
    Color     string `json:"color"`
    Locator   string `json:"locator"`
    LongName  string `json:"longName"`
    ShortName string `json:"shortName"`
    TypeRef   string `json:"typeRef"`
}

type IssueSubTypeCreatedRS struct {
    Id      *int64  `json:"id,omitempty"`
    Locator *string `json:"locator,omitempty"`
}
```

## Scope

Create `internal/cli/issue_sub_type.go` and `internal/cli/issue_sub_type_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get issue-sub-type` | List issue sub-types for the project |
| `rp create issue-sub-type --color <hex> --long-name <name> --short-name <name> --type-ref <ref>` | Create an issue sub-type |
| `rp update issue-sub-type --locator <loc> --color <hex> --long-name <name> --short-name <name> --type-ref <ref>` | Update an issue sub-type (full replacement) |
| `rp delete issue-sub-type <id>` | Delete an issue sub-type by numeric ID |

**UX asymmetry:** Updates identify by `--locator` (string), deletes identify by positional numeric ID. This matches the API exactly — `UpdateIssueSubType` uses `Locator`, `DeleteIssueSubType` uses `int64`. Both identifiers are visible in `rp get issue-sub-type` output. The `Long` help text for both update and delete must explain which identifier each expects and why they differ.

## Implementation

### 1. Create `internal/cli/issue_sub_type.go`

#### `newGetIssueSubTypeCmd()`

```go
func newGetIssueSubTypeCmd() *cobra.Command
```

- `Use`: `"issue-sub-type"`
- `Short`: `"List issue sub-types for a project"`
- `Long`: Include example. Note this fetches the full project settings and flattens the sub-types map.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGetIssueSubType)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Resolve connection, project, and client
2. Call `scope.Get(ctx)` — returns `*types.ProjectSettingsResource`
3. Flatten the `SubTypes` map into a `[]types.IssueSubTypeResource` slice. Iterate the map keys in sorted order for deterministic output:
   ```go
   var subTypes []types.IssueSubTypeResource

   categories := make([]string, 0, len(settings.SubTypes))
   for category := range settings.SubTypes {
       categories = append(categories, category)
   }

   sort.Strings(categories)

   for _, category := range categories {
       subTypes = append(subTypes, settings.SubTypes[category]...)
   }
   ```
4. Pass the `[]types.IssueSubTypeResource` slice to `printer.Print` using `issueSubTypeRow`

**Table formatting for `IssueSubTypeResource`:**

```go
func issueSubTypeHeaders() []string {
    return []string{"ID", "LOCATOR", "TYPE REF", "SHORT NAME", "LONG NAME", "COLOR"}
}

func issueSubTypeRow(s *types.IssueSubTypeResource) ([]string, []string) {
    return issueSubTypeHeaders(), []string{
        strconv.FormatInt(s.Id, 10),
        s.Locator,
        s.TypeRef,
        s.ShortName,
        s.LongName,
        s.Color,
    }
}
```

Both ID and LOCATOR are shown so users can find the right identifier for update (locator) vs delete (ID).

#### `newCreateIssueSubTypeCmd()`

```go
func newCreateIssueSubTypeCmd() *cobra.Command
```

- `Use`: `"issue-sub-type"`
- `Short`: `"Create an issue sub-type"`
- `Long`: Include examples showing all required flags. Document valid `--type-ref` values (e.g., `"TO_INVESTIGATE"`, `"PRODUCT_BUG"`, `"AUTOMATION_BUG"`, `"SYSTEM_ISSUE"`, `"NO_DEFECT"`).
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateIssueSubType)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--color` (string, required): hex color code (e.g., `"#ff0000"`)
  - `--long-name` (string, required): long display name
  - `--short-name` (string, required): short display name
  - `--type-ref` (string, required): parent issue type reference

**Why inline flags?** `CreateIssueSubTypeRQ` has 4 simple string fields. This is within the inline flag comfort zone.

The run function:
1. Validate all four flags are non-empty
2. Build `types.CreateIssueSubTypeRQ{Color: color, LongName: longName, ShortName: shortName, TypeRef: typeRef}`
3. Resolve connection, project, and client
4. Call `scope.CreateIssueSubType(ctx, request)`
5. Print the `IssueSubTypeCreatedRS` result

**Result formatting for `IssueSubTypeCreatedRS`:**

```go
func issueSubTypeCreatedHeaders() []string {
    return []string{"ID", "LOCATOR"}
}

func issueSubTypeCreatedRows(result *types.IssueSubTypeCreatedRS) [][]string {
    return [][]string{{formatInt64Ptr(result.Id), formatStringPtr(result.Locator)}}
}
```

#### `newUpdateIssueSubTypeCmd()`

```go
func newUpdateIssueSubTypeCmd() *cobra.Command
```

- `Use`: `"issue-sub-type"`
- `Short`: `"Update an issue sub-type"`
- `Long`: Include examples showing all required flags. Note that this is a **full replacement** — all fields must be provided. Include an example showing the complete workflow: `rp get issue-sub-type` → find locator → `rp update issue-sub-type --locator <loc> ...`. Explain that `--locator` identifies the target (not a positional arg or numeric ID) because the API uses the locator string, not the numeric ID, for updates.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runUpdateIssueSubType)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--locator` (string, required): locator of the sub-type to update
  - `--type-ref` (string, required): parent issue type reference
  - `--short-name` (string, required): short display name
  - `--long-name` (string, required): long display name
  - `--color` (string, required): hex color code

**Why inline flags (not `--from-file`)?** `UpdateOneIssueSubTypeRQ` has 5 simple string fields. Per the settled UX in PR 17 §6b, this is manageable with inline flags. Using flags also makes the `--locator` identifier explicit at the command line, reducing the chance of editing the wrong sub-type.

The run function:
1. Validate all five flags are non-empty
2. Build `types.UpdateOneIssueSubTypeRQ{Locator: locator, TypeRef: typeRef, ShortName: shortName, LongName: longName, Color: color}`
3. Resolve connection, project, and client
4. Call `scope.UpdateIssueSubType(ctx, request)`
5. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newDeleteIssueSubTypeCmd()`

```go
func newDeleteIssueSubTypeCmd() *cobra.Command
```

- `Use`: `"issue-sub-type <id>"`
- `Short`: `"Delete an issue sub-type"`
- `Long`: Include example. Note that built-in/system sub-types cannot be deleted. Explain that delete uses the **numeric ID** (not the locator string) — find the ID via `rp get issue-sub-type`.
- `Args`: `cobra.ExactArgs(1)` (the numeric ID)
- `RunE`: `withPrintedErrors(runDeleteIssueSubType)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "issue-sub-type")`
2. Resolve connection, project, and client
3. Call `scope.DeleteIssueSubType(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

### 2. Register commands in `commands.go`

```go
// get
get.AddCommand(newGetIssueSubTypeCmd())

// create
create.AddCommand(newCreateIssueSubTypeCmd())

// update
update.AddCommand(newUpdateIssueSubTypeCmd())

// delete
del.AddCommand(newDeleteIssueSubTypeCmd())
```

### 3. Create `internal/cli/issue_sub_type_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetIssueSubTypeCmd_Flags`** — Verify `--project` / `-p`. Verify `cobra.NoArgs`.

2. **`TestNewCreateIssueSubTypeCmd_Flags`** — Verify `--color`, `--long-name`, `--short-name`, `--type-ref`, and `--project` / `-p`. Verify `cobra.NoArgs`.

3. **`TestNewUpdateIssueSubTypeCmd_Flags`** — Verify `--locator`, `--type-ref`, `--short-name`, `--long-name`, `--color`, and `--project` / `-p`. Verify `cobra.NoArgs`. Verify no `--from-file`.

4. **`TestNewDeleteIssueSubTypeCmd_Flags`** — Verify `--project` / `-p`. Verify 1 arg.

5. **`TestIssueSubTypeRow_Formatting`** — Verify row output for a sample `IssueSubTypeResource`. Verify both ID and LOCATOR appear in the output.

6. **`TestIssueSubTypeCreatedRows_WithValues`** — Verify row output for a fully-populated `IssueSubTypeCreatedRS`.

7. **`TestIssueSubTypeCreatedRows_NilFields`** — Verify nil handling.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **List command calls `Settings().Get()`.** There is no dedicated `ListIssueSubTypes` method. The list command fetches the full project settings and flattens the `SubTypes` map. This is the same endpoint as `rp config get-project` (PR 28) but presents only the sub-types with a focused table format.
- **Both create and update use inline flags.** Create has 4 fields, update has 5 — all simple strings. Per the settled UX in PR 17 §6b, this is within the inline flag comfort zone.
- **Update uses `--locator`, not a positional arg or `--from-file`.** The `Locator` field is a string identifier, not a numeric ID. Making it a flag (`--locator`) is clearer than a positional arg because it avoids confusion with the numeric ID used by delete. This matches the frozen CLI shape from PR 17.
- **UX asymmetry between update and delete is intentional.** Updates identify by locator (string), deletes identify by numeric ID. Both identifiers are shown in `rp get issue-sub-type` output. Help text must explain this clearly.
- **`IssueSubTypeCreatedRS` has both `Id` and `Locator`.** Both are shown in the create result table.
- **Do not validate color format or type-ref values locally.** Let the API validate.
- **Reuse shared helpers.** Use `parseResourceID`, `formatInt64Ptr`, `formatStringPtr`, `messageResultHeaders`/`operationResultRows`.

## What NOT to do

- **Do not use `--from-file` for update.** The settled UX uses inline flags.
- **Do not validate color format or type-ref values locally.**
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` shows `issue-sub-type`
- `rp create --help` shows `issue-sub-type`
- `rp update --help` shows `issue-sub-type`
- `rp delete --help` shows `issue-sub-type`
- `rp update issue-sub-type --help` shows `--locator`, NOT `--from-file`

## Definition of done

- [ ] `internal/cli/issue_sub_type.go` created
- [ ] `internal/cli/issue_sub_type_test.go` created
- [ ] Commands registered in `commands.go` (get, create, update, delete)
- [ ] All commands use `--project` / `-p`
- [ ] Get command flattens `SubTypes` map with sorted categories
- [ ] Table shows both ID and LOCATOR columns
- [ ] Create and update use inline flags (no `--from-file`)
- [ ] Update uses `--locator` to identify the target
- [ ] Delete uses positional numeric ID
- [ ] Help text explains the locator-vs-ID asymmetry
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
