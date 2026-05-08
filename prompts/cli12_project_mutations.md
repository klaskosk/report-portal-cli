# PR 12: `project` Mutations

## Goal

Extend the existing `project` command family with create, update, and delete commands. The existing `rp get project` command (in `project.go`) is left unchanged. This PR adds three new commands to the existing file, each with a different identifier pattern: create uses a `--name` flag, update takes a project name as a positional arg, and delete takes a numeric project ID as a positional arg.

## Context

The `project` resource maps to `ProjectsScope` on the client (`client.Projects()`). It is **instance-scoped** — mutation commands do **not** require `--project` / `-p`. The existing `get project` command already follows this convention.

**Prerequisite:** PRs 1–7 must land first (parent command split, verb scaffolding, centralized registration, all three pilots, and shared helper extraction).

### Client surface (mutations only)

| Method | Parameters | Returns |
|--------|------------|---------|
| `Create` | `ctx`, `request types.CreateProjectRQ` | `(*types.EntryCreatedRS, error)` |
| `Update` | `ctx`, `projectName string`, `request types.UpdateProjectRQ` | `(*types.OperationCompletionRS, error)` |
| `Delete` | `ctx`, `projectID int64` | `(*types.DeleteBulkRS, error)` |

**Critical asymmetry:** `Update` identifies the project by **name** (string), while `Delete` identifies it by **numeric ID** (int64). This is a design decision by the Report Portal API, not the CLI.

### Key types

```go
type CreateProjectRQ struct {
    EntryType   CreateProjectRQEntryType `json:"entryType"`
    ProjectName string                   `json:"projectName"`
}

// Only valid value:
const CreateProjectRQEntryTypeINTERNAL CreateProjectRQEntryType = "INTERNAL"

type UpdateProjectRQ struct {
    Configuration *ProjectConfigurationUpdate `json:"configuration,omitempty"`
    Users         *map[string]string          `json:"users,omitempty"`
}

type ProjectConfigurationUpdate struct {
    Attributes map[string]string `json:"attributes"`
}

type DeleteBulkRS struct {
    Errors              *[]ErrorRS `json:"errors,omitempty"`
    NotFound            *[]int64   `json:"notFound,omitempty"`
    SuccessfullyDeleted *[]int64   `json:"successfullyDeleted,omitempty"`
}

type ErrorRS struct {
    ErrorCode  int     `json:"errorCode"`
    Message    string  `json:"message"`
    StackTrace *string `json:"stackTrace,omitempty"`
}

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}
```

## Scope

Extend `internal/cli/project.go` with three new command constructors and their run functions. Create `internal/cli/project_mutation_test.go` for tests covering the new commands. Register the new commands in `internal/cli/commands.go`.

**Important:** Use a separate test file (`project_mutation_test.go`) rather than adding tests to an existing `project_test.go` to minimize conflict with the existing `get project` test surface.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get project` | *(existing — unchanged)* |
| `rp get project <name>` | *(existing — unchanged)* |
| `rp create project --name <name>` | Create a new project |
| `rp update project <name> --from-file <path>` | Update project configuration |
| `rp delete project <id>` | Delete a project by numeric ID |

## Implementation

### 1. Add commands to `internal/cli/project.go`

Add three new command constructors and their run functions to the existing file. Do NOT modify any existing functions in `project.go`.

#### `newCreateProjectCmd()`

```go
func newCreateProjectCmd() *cobra.Command
```

- `Use`: `"project"`
- `Short`: `"Create a project"`
- `Long`: Include examples showing `--name` flag usage.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateProject)`
- Flags:
  - `--name` (string, required): project name

**Do NOT** register `--project` / `-p` — project creation is instance-scoped.

The run function:
1. Validate `--name` is non-empty
2. Build `types.CreateProjectRQ{EntryType: types.CreateProjectRQEntryTypeINTERNAL, ProjectName: name}`
3. Resolve connection and client (no project resolution)
4. Call `scope.Create(ctx, request)`
5. Print the `EntryCreatedRS` result using `entryCreatedHeaders()` and `entryCreatedRows(result)`

**`EntryType` is hardcoded to `INTERNAL`.** The API currently only defines one valid value (`CreateProjectRQEntryTypeINTERNAL`). Exposing a flag for a single-valued enum adds UX noise with no benefit. If the API adds more entry types in the future, a flag can be introduced at that time.

#### `newUpdateProjectCmd()`

```go
func newUpdateProjectCmd() *cobra.Command
```

- `Use`: `"project <name>"`
- `Short`: `"Update a project"`
- `Long`: Include examples showing `--from-file` with a project name argument. Mention that the project is identified by name (not ID).
- `Args`: `cobra.ExactArgs(1)` (the project name)
- `RunE`: `withPrintedErrors(runUpdateProject)`
- Flags:
  - `--from-file` / `-f` (string, required): path to JSON file containing `UpdateProjectRQ`

**Do NOT** register `--project` / `-p` — the target project is specified via the positional arg.

The run function:
1. Read `args[0]` as the project name (string — no parsing needed)
2. Validate `--from-file` is non-empty
3. Read and unmarshal the file using `readRequestFromFile[types.UpdateProjectRQ](path)`
4. Resolve connection and client
5. Call `scope.Update(ctx, projectName, request)`
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

**Why `--from-file`?** `UpdateProjectRQ` contains `Configuration *ProjectConfigurationUpdate` (which wraps a `map[string]string`) and `Users *map[string]string`. Both are map types that are unwieldy as inline flags. Example JSON:

```json
{
  "configuration": {
    "attributes": {
      "analyzer.autoAnalyzerEnabled": "false"
    }
  }
}
```

#### `newDeleteProjectCmd()`

```go
func newDeleteProjectCmd() *cobra.Command
```

- `Use`: `"project <id>"`
- `Short`: `"Delete a project"`
- `Long`: Include example showing delete by numeric ID. **Explicitly note** that the project is identified by its numeric ID, not its name. Users can find a project's ID via `rp get project <name>` or `rp get project <name> -o json`.
- `Args`: `cobra.ExactArgs(1)` (the numeric project ID)
- `RunE`: `withPrintedErrors(runDeleteProject)`

**Do NOT** register `--project` / `-p` — the target project is specified via the positional arg.

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "project")`
2. Resolve connection and client
3. Call `scope.Delete(ctx, id)` — returns `*types.DeleteBulkRS`
4. Handle the response:
   - For JSON output: pass the full `DeleteBulkRS` through to `printer.PrintJSON`
   - For table output: check the response and build a user-friendly message:
     - If `result.SuccessfullyDeleted` contains the ID: message is `"Project <id> deleted successfully"`
     - If `result.NotFound` contains the ID: return an error `"project <id> not found"`
     - If `result.Errors` is non-empty: format the first error's message and return it as an error
   - Print the table using `messageResultHeaders()` with a single-row slice

```go
func formatProjectDeleteResult(id int64, result *types.DeleteBulkRS, printer *Printer) error {
    if printer.isJSON() {
        return printer.PrintJSON(result)
    }

    if result.Errors != nil && len(*result.Errors) > 0 {
        err := fmt.Errorf("deleting project %d: %s", id, (*result.Errors)[0].Message)
        printer.PrintError(err)

        return err
    }

    if result.NotFound != nil && len(*result.NotFound) > 0 {
        err := fmt.Errorf("project %d not found", id)
        printer.PrintError(err)

        return err
    }

    if result.SuccessfullyDeleted != nil && containsInt64(*result.SuccessfullyDeleted, id) {
        message := fmt.Sprintf("Project %d deleted successfully", id)

        return printer.Print(
            result,
            messageResultHeaders(),
            [][]string{{message}},
        )
    }

    err := fmt.Errorf("unexpected response: project %d was not confirmed as deleted", id)
    printer.PrintError(err)

    return err
}

func containsInt64(slice []int64, target int64) bool {
    for _, v := range slice {
        if v == target {
            return true
        }
    }

    return false
}
```

This function handles the complexity of `DeleteBulkRS` while providing clear, actionable output in both JSON and table modes. In JSON mode, the full response (including any errors or not-found IDs) is passed through unmodified. In table mode, the response is checked for explicit membership in `SuccessfullyDeleted` before claiming success — an empty or unexpected response produces a descriptive error rather than a false positive.

### 2. Register commands in `commands.go`

Add these lines to `buildCommandTree` in `internal/cli/commands.go`:

```go
// create
create.AddCommand(newCreateProjectCmd())

// update
update.AddCommand(newUpdateProjectCmd())

// delete
del.AddCommand(newDeleteProjectCmd())
```

### 3. Create `internal/cli/project_mutation_test.go`

Use a separate test file to avoid conflicts with the existing get project test surface:

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewCreateProjectCmd_Flags`** — Verify create has `--name` flag. Verify no `--project` flag and no `--entry-type` flag.

2. **`TestNewCreateProjectCmd_NoArgs`** — Verify create accepts no positional arguments (`cobra.NoArgs`).

3. **`TestNewUpdateProjectCmd_Flags`** — Verify update has `--from-file` / `-f` flag. Verify no `--project` flag. Verify it accepts exactly 1 arg.

4. **`TestNewDeleteProjectCmd_Args`** — Verify delete accepts exactly 1 arg. Verify no `--project` flag.

5. **`TestFormatProjectDeleteResult_Success`** — Build a `DeleteBulkRS` with `SuccessfullyDeleted: &[]int64{42}` and empty errors/not-found. Verify the formatted message contains "deleted successfully".

6. **`TestFormatProjectDeleteResult_NotFound`** — Build a `DeleteBulkRS` with `NotFound: &[]int64{42}` and empty errors. Verify an error is returned containing "not found".

7. **`TestFormatProjectDeleteResult_WithErrors`** — Build a `DeleteBulkRS` with `Errors: &[]types.ErrorRS{{ErrorCode: 4004, Message: "Access denied"}}`. Verify an error is returned containing the error message.

8. **`TestFormatProjectDeleteResult_EmptyResponse`** — Build a `DeleteBulkRS` with all nil fields (`Errors`, `NotFound`, and `SuccessfullyDeleted` all nil). Verify an error is returned containing "not confirmed as deleted" rather than a false success.

9. **`TestFormatProjectDeleteResult_MissingFromSuccessful`** — Build a `DeleteBulkRS` with `SuccessfullyDeleted: &[]int64{99}` (a different ID than the one requested). Verify an error is returned rather than a false success.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Instance-scoped — no `--project` flag.** Project mutations operate on the instance level. The target project is identified via `--name` (create), positional arg name (update), or positional arg ID (delete).
- **Asymmetric identifiers.** The API uses project name for `Update` and project ID for `Delete`. This is documented clearly in the help text. The help text for `delete` explicitly tells users how to find their project's ID.
- **`EntryType` is hardcoded to `INTERNAL`.** The API only defines one valid value today. A flag would be speculative surface area.
- **`--from-file` for update.** `UpdateProjectRQ` contains map-typed fields that are not amenable to inline flags.
- **`DeleteBulkRS` handling.** The API returns a bulk delete response even for single-project deletes. The CLI normalizes this to a clear message for table output while preserving the full response for JSON output.
- **Separate test file.** Using `project_mutation_test.go` avoids merge conflicts with the existing `project.go` test surface (if any exists or is added for `get project`).
- **Do not modify existing `get project` code.** The existing functions (`newGetProjectCmd`, `runGetProject`, `getSingleProject`, `listProjects`, `projectRow`) must remain untouched.
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `readRequestFromFile[types.UpdateProjectRQ]`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`.

## What NOT to do

- **Do not modify `newGetProjectCmd`, `runGetProject`, `getSingleProject`, `listProjects`, or `projectRow`.** Those are existing functions.
- **Do not add `--project` / `-p` flags** to any project mutation command.
- **Do not attempt to resolve a project name to an ID for delete.** The user provides the numeric ID. If the user needs the ID, they use `rp get project <name>`.
- **Do not implement project export.** That is PR 27.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp create --help` now shows `project` as a subcommand
- `rp update --help` now shows `project` as a subcommand
- `rp delete --help` now shows `project` as a subcommand
- `rp create project --help` shows `--name` flag (no `--project`, no `--entry-type`)
- `rp update project --help` shows `--from-file` flag and accepts a project name arg
- `rp delete project --help` shows that a numeric ID is required
- `rp get project` and `rp get project <name>` still work identically to before
- No existing commands are affected

## Definition of done

- [ ] `newCreateProjectCmd`, `newUpdateProjectCmd`, `newDeleteProjectCmd` added to `internal/cli/project.go`
- [ ] `internal/cli/project_mutation_test.go` created with unit tests for new commands
- [ ] Commands registered in `commands.go`
- [ ] No `--project` flag on any project mutation command
- [ ] Create uses `--name` (required) and hardcodes `EntryType` to `INTERNAL`
- [ ] Update takes project name as positional arg and uses `--from-file`
- [ ] Delete takes project numeric ID as positional arg
- [ ] `DeleteBulkRS` handled with clear table messages and full JSON passthrough
- [ ] Existing `get project` code is completely untouched
- [ ] Shared helpers used: `parseResourceID`, `readRequestFromFile`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
