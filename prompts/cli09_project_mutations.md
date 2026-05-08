# PR 9: `project` Mutations

## Goal

Extend the existing `rp project` command with `create`, `update`, and `delete` subcommands while leaving the existing `list` and `get` intact.

## Context

This PR runs in the Phase 4 parallel fan-out. PRs 1–5 have landed. After PR 1, the project resource has been refactored to noun-first with `rp project list` and `rp project get <name>`.

Projects are instance-scoped (no `--project` flag). This PR adds the three remaining mutation verbs.

Key API quirks for projects:
1. **`Update` is keyed by project name** (string), not numeric ID.
2. **`Delete` is keyed by numeric project ID** (int64), not name.
3. **`Delete` returns `*types.DeleteBulkRS`**, not `OperationCompletionRS`.
4. **`UpdateProjectRQ` has a nested `Configuration` map**, making `--from-file` the right input method.

### Prerequisites

- PRs 1–5 have landed
- `internal/cli/project.go` already defines `newProjectCmd()` with `list` and `get` subcommands

### Client API surface

`ProjectsScope` (obtained via `client.Projects()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `Create` | `Create(ctx, types.CreateProjectRQ)` | `*types.EntryCreatedRS` |
| `Update` | `Update(ctx, projectName string, types.UpdateProjectRQ)` | `*types.OperationCompletionRS` |
| `Delete` | `Delete(ctx, projectID int64)` | `*types.DeleteBulkRS` |
| `Get` | `Get(ctx, projectName string)` | `*types.ProjectResource` |

`Get` is listed because the delete command needs it to resolve a project name to a numeric ID.

### Relevant types

```go
type CreateProjectRQ struct {
    EntryType   CreateProjectRQEntryType `json:"entryType"`
    ProjectName string                   `json:"projectName"`
}
// CreateProjectRQEntryType = "INTERNAL"

type UpdateProjectRQ struct {
    Configuration *ProjectConfigurationUpdate `json:"configuration,omitempty"`
    Users         *map[string]string          `json:"users,omitempty"`
}

type ProjectConfigurationUpdate struct {
    Attributes map[string]string `json:"attributes"`
}

type EntryCreatedRS struct {
    Id *int64 `json:"id,omitempty"`
}

type OperationCompletionRS struct {
    Message *string `json:"message,omitempty"`
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

type ProjectResource struct {
    ProjectId    int64     `json:"projectId"`
    ProjectName  string    `json:"projectName"`
    EntryType    string    `json:"entryType"`
    Organization *string   `json:"organization,omitempty"`
    CreationDate time.Time `json:"creationDate"`
    // ... additional fields
}
```

## Implementation plan

### 1. Extend `internal/cli/project.go`

Add three new command constructors and register them in the existing `newProjectCmd()` parent. **Do not modify the existing `list` and `get` subcommands.**

Add to the existing `newProjectCmd()`:
```go
cmd.AddCommand(newProjectCreateCmd())
cmd.AddCommand(newProjectUpdateCmd())
cmd.AddCommand(newProjectDeleteCmd())
```

#### `rp project create`

- No positional arguments
- Flags:
  - `--name` (string, required) — project name
  - `--entry-type` (string, optional, default `"INTERNAL"`) — project entry type
- Mark `--name` as required
- Build `types.CreateProjectRQ{ProjectName: name, EntryType: types.CreateProjectRQEntryType(entryType)}`
- Call `scope.Create(ctx, request)`
- Table output: `ID` column (from `EntryCreatedRS.Id`)
- JSON output: the raw `*EntryCreatedRS`

#### `rp project update <name> --from-file <path>`

- Exactly one positional argument: project name (string, not numeric ID)
- Required flag: `--from-file <path>` — path to a JSON file containing an `UpdateProjectRQ` payload
- Read file with `readRequestFromFile[types.UpdateProjectRQ](path)`
- Call `scope.Update(ctx, projectName, request)`
- Table/JSON output: `OperationCompletionRS` via shared message helpers

The `UpdateProjectRQ` has two optional fields: `Configuration` (project attributes) and `Users` (user-to-role assignments). Both are maps. The `--from-file` approach is appropriate because these are complex structures.

#### `rp project delete <name>`

- Exactly one positional argument: project name (string)
- No additional flags
- **Name-to-ID resolution:** The API deletes by numeric ID, but the CLI takes a project name for consistency with `get` and `update`. Resolve the name to a numeric ID:
  1. Call `scope.Get(ctx, projectName)` to fetch the `ProjectResource`
  2. Extract `ProjectId` from the response
  3. Call `scope.Delete(ctx, projectId)`
- If the name resolution fails (project not found), surface the error clearly
- Table output: Since `DeleteBulkRS` has a different structure, format it as:
  - Headers: `STATUS`, `MESSAGE`
  - On success (non-empty `SuccessfullyDeleted`): `"Deleted"`, `"Project '<name>' deleted successfully"`
  - On errors (non-empty `Errors`): `"Error"`, first error message
  - On not-found (non-empty `NotFound`): `"Not Found"`, `"Project ID <id> not found"`
- JSON output: the raw `*DeleteBulkRS`

### 2. `DeleteBulkRS` table formatting

Since `DeleteBulkRS` is structurally different from the simple `OperationCompletionRS`, create a helper specifically for this response:

```go
func formatDeleteBulkResult(result *types.DeleteBulkRS, resourceName string) ([]string, [][]string) {
    headers := []string{"STATUS", "MESSAGE"}

    if result.SuccessfullyDeleted != nil && len(*result.SuccessfullyDeleted) > 0 {
        return headers, [][]string{{"Deleted", fmt.Sprintf("%s deleted successfully", resourceName)}}
    }

    if result.Errors != nil && len(*result.Errors) > 0 {
        return headers, [][]string{{"Error", (*result.Errors)[0].Message}}
    }

    if result.NotFound != nil && len(*result.NotFound) > 0 {
        return headers, [][]string{{"Not Found", fmt.Sprintf("ID %d not found", (*result.NotFound)[0])}}
    }

    return headers, [][]string{{"Unknown", "No status information in response"}}
}
```

### 3. No changes to root.go

The `newProjectCmd()` function is already registered in `root.go` from PR 1. This PR only adds subcommands to the existing parent.

## Test requirements

### Unit tests (add to `internal/cli/project_test.go` or a new file)

- **Create validation:**
  - Missing `--name` errors
  - Default `--entry-type` is `"INTERNAL"`
  - Custom `--entry-type` is passed through
- **Update validation:**
  - Missing `--from-file` errors
  - Non-existent file errors
  - Malformed JSON errors
  - Valid JSON produces correct `UpdateProjectRQ`
- **Delete argument:** Verify delete takes exactly 1 positional argument (project name)
- **`DeleteBulkRS` formatting:**
  - Success case: `SuccessfullyDeleted` non-empty → shows "Deleted"
  - Error case: `Errors` non-empty → shows error message
  - Not-found case: `NotFound` non-empty → shows "Not Found"
  - Empty response: all fields nil → shows "Unknown"
- **Command wiring:** Verify `newProjectCmd()` now has list, get, create, update, and delete subcommands

## Edge cases and constraints

- **Extend, do not rewrite:** This PR adds three subcommands to `newProjectCmd()`. Do NOT modify the existing `list`, `get`, `listProjects`, `getSingleProject`, or `projectRow` functions.
- **`Update` takes project name, `Delete` resolves name to ID:** These are different API behaviors. Update passes the name string directly. Delete must fetch the project first to get the numeric ID.
- **`DeleteBulkRS` is not `OperationCompletionRS`:** Do not use `messageHeaders`/`messageRows` for delete. The response structure is different and needs its own formatting.
- **`CreateProjectRQEntryType` is a string enum:** Currently only `"INTERNAL"` is defined. Do not validate the value client-side — pass it through and let the API reject unknown types.
- **Project name validation:** Do not validate project name format client-side. The API enforces naming rules (e.g., lowercase, no spaces in some deployments).
- **`UpdateProjectRQ` fields are all optional:** The JSON file can contain just `configuration`, just `users`, or both. Do not require specific fields to be present — the API handles partial updates.
- **Instance-scoped resource:** Do NOT register a `-p`/`--project` flag on any of the new subcommands. Projects are instance-scoped resources.
- **Name-to-ID resolution for delete can fail:** If `Projects().Get(name)` returns a not-found error, wrap it with context: `"project %q not found: cannot delete"`.

## Example commands and expected output

```bash
# Create a project
$ rp project create --name "new_project"
ID
42

# Create with explicit entry type
$ rp project create --name "new_project" --entry-type INTERNAL
ID
42

# Update project configuration
$ rp project update my_project --from-file project_config.json
MESSAGE
Project configuration updated successfully.

# Delete a project by name
$ rp project delete old_project
STATUS    MESSAGE
Deleted   Project 'old_project' deleted successfully
```

Example `project_config.json`:
```json
{
  "configuration": {
    "attributes": {
      "job.keepLaunches": "3 months",
      "job.keepLogs": "2 weeks",
      "job.keepScreenshots": "2 weeks",
      "job.interruptJobTime": "1 day"
    }
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

- [ ] `rp project create` with `--name` and optional `--entry-type` flags
- [ ] `rp project update <name> --from-file` with JSON payload
- [ ] `rp project delete <name>` with name-to-ID resolution
- [ ] Existing `list` and `get` subcommands unchanged
- [ ] `DeleteBulkRS` formatted appropriately in table output
- [ ] Unit tests cover create flags, update file handling, delete name resolution, and `DeleteBulkRS` formatting
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
