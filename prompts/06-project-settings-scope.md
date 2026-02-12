# Prompt 06: Client Discovery + Identity Depth — Project Settings Scope (Milestone 7)

## Goal

Add a `ProjectSettingsScope` to the Go client for reading project configuration (issue sub-types, pattern templates, notifications). This completes Milestone 7 (discovery + identity depth) — the `UsersScope`, `ProjectsScope`, `rp whoami`, and `rp get project` were already delivered in Prompt 04.

## Context

The remaining work for Milestone 7 is a project-scoped settings read surface. This is useful for:

- **Validation**: verifying that defect sub-types exist before referencing them in test item updates
- **Debugging**: inspecting project configuration when investigating unexpected behavior
- **Automation**: scripts that need to enumerate available issue sub-types or pattern templates

The CLI exposure is optional for this prompt — `rp get project-settings` is a possible follow-up only if the data is useful in table form. Focus on client coverage.

## Generated API endpoints available

The generated client in `pkg/internal/api/api.gen.go` has these project-settings methods:

### Read endpoints (primary — implement these)

| Generated method | What it does | Response type |
|---|---|---|
| `GetProjectSettingsUsingGETWithResponse(ctx, projectName)` | Get all project settings (issue sub-types, etc.) | `ProjectSettingsResource` |
| `GetNotificationsUsingGETWithResponse(ctx, projectName)` | Get notification configuration for the project | notification config response |

### Write endpoints (optional — defer unless straightforward)

| Generated method | What it does | Request type |
|---|---|---|
| `CreateNotificationUsingPOSTWithResponse(ctx, projectName, SenderCaseDTO)` | Create a notification rule | `SenderCaseDTO` |
| `UpdateNotificationUsingPUTWithResponse(ctx, projectName, SenderCaseDTO)` | Update a notification rule | `SenderCaseDTO` |
| `DeleteNotificationUsingDELETEWithResponse(ctx, projectName, id int64)` | Delete a notification rule | — |
| `CreatePatternTemplateUsingPOSTWithResponse(ctx, projectName, CreatePatternTemplateRQ)` | Create a pattern template | `CreatePatternTemplateRQ` |
| `UpdatePatternTemplateUsingPUTWithResponse(ctx, projectName, id int64, UpdatePatternTemplateRQ)` | Update a pattern template | `UpdatePatternTemplateRQ` |
| `DeletePatternTemplateUsingDELETEWithResponse(ctx, projectName, id int64)` | Delete a pattern template | — |
| `CreateProjectIssueSubTypeUsingPOSTWithResponse(ctx, projectName, CreateIssueSubTypeRQ)` | Create a custom issue sub-type | `CreateIssueSubTypeRQ` |
| `UpdateProjectIssueSubTypeUsingPUTWithResponse(ctx, projectName, UpdateOneIssueSubTypeRQ)` | Update an issue sub-type | `UpdateOneIssueSubTypeRQ` |
| `DeleteProjectIssueSubTypeUsingDELETEWithResponse(ctx, projectName, id int64)` | Delete an issue sub-type | — |

### Key types (from `pkg/types/types.gen.go`)

```go
type ProjectSettingsResource struct {
    Project  int64                             `json:"project"`
    SubTypes map[string][]IssueSubTypeResource `json:"subTypes"`
}

type IssueSubTypeResource struct {
    Color     *string `json:"color,omitempty"`
    Id        int64   `json:"id"`
    LongName  string  `json:"longName"`
    Locator   string  `json:"locator"`
    ShortName string  `json:"shortName"`
    TypeRef   string  `json:"typeRef"`
}

type PatternTemplateResource struct {
    Enabled *bool   `json:"enabled,omitempty"`
    Id      int64   `json:"id"`
    Name    *string `json:"name,omitempty"`
    Type    *string `json:"type,omitempty"`
    Value   *string `json:"value,omitempty"`
}
```

## Client requirements

### ProjectSettingsScope

Create `pkg/client/project_settings.go` with a `ProjectSettingsScope` struct:

- `ProjectSettingsScope` should be **project-scoped**, accessible from `ProjectScope` via a `Settings()` method
- Add a `Settings()` method to `ProjectScope` in `pkg/client/project.go` that returns `*ProjectSettingsScope`
- Struct holds `project *ProjectScope` (same pattern as `LaunchScope`, `TestItemScope`, `LogScope`)

### Methods to implement

1. **`Get(ctx) (*types.ProjectSettingsResource, error)`** — get the full project settings (issue sub-types map). This is the core read method.

2. **`GetNotifications(ctx) ([]types.SenderCaseDTO, error)`** — get notification rules for the project. Investigate the exact response shape from the generated client; it may be a list or a wrapped resource. (Note: `SenderCaseDTO` or equivalent — check what the generated response type is.)

Optional (lower priority — implement if time allows):

3. **`CreateIssueSubType(ctx, CreateIssueSubTypeRQ) (*types.IssueSubTypeCreatedRS, error)`** — create a custom issue sub-type
4. **`DeleteIssueSubType(ctx, id int64) (*types.OperationCompletionRS, error)`** — delete an issue sub-type
5. **`CreatePatternTemplate(ctx, CreatePatternTemplateRQ) (*types.EntryCreatedRS, error)`** — create a pattern template
6. **`DeletePatternTemplate(ctx, id int64) (*types.OperationCompletionRS, error)`** — delete a pattern template

### Implementation pattern

Follow the existing scope pattern:

```go
type ProjectSettingsScope struct {
    project *ProjectScope
}

func (s *ProjectSettingsScope) Get(ctx context.Context) (*types.ProjectSettingsResource, error) {
    logger := s.project.operationLogger("GetProjectSettings")
    logger.InfoContext(ctx, "API request")

    response, err := s.project.client.raw.GetProjectSettingsUsingGETWithResponse(
        ctx,
        s.project.projectName,
    )
    if err != nil {
        return nil, fmt.Errorf("get project settings: %w", err)
    }

    logger.DebugContext(ctx, "API response", "status", response.StatusCode())

    return handleResponse(response, "get project settings", response.JSONDefault, response.JSON200)
}
```

## Client tests

Create `pkg/client/project_settings_test.go` (black-box, `package client_test`):

- **Get**: success (200 with sub-types map), unauthorized (401), not found (404 — invalid project)
- **GetNotifications**: success (200), unauthorized (401)
- If optional write methods are implemented:
  - **CreateIssueSubType**: success (201), bad request (400 — duplicate name)
  - **DeleteIssueSubType**: success (200), not found (404)

All tests must:
- Use `t.Parallel()`
- Use `httptest.Server` to stub API responses
- Verify bearer token, path, and response mapping

### `export_test.go`

Add a test-only constructor for `ProjectSettingsScope` in `pkg/client/export_test.go`, following the same pattern as other scope constructors. This is needed because `ProjectSettingsScope` is accessed via `client.Project(name).Settings()`, and tests need to create one directly.

## Integration tests

### `test/integration/client_project_settings_test.go` (new file)

- Use `Label("client", "project-settings")`
- **Get**: verify that the default project has settings with at least the standard issue sub-types (TO_INVESTIGATE, PRODUCT_BUG, AUTOMATION_BUG, SYSTEM_ISSUE, NO_DEFECT)
- **GetNotifications**: verify returns without error (notifications may be empty on a fresh project)
- **Not found**: verify `IsNotFound` for a non-existent project name

### Seeding notes

- Project settings tests don't require seeding launches/items — the default project has built-in issue sub-types
- Use the existing test project that is set up in `BeforeSuite`

## CLI (optional — skip unless requested)

If there's time, add `rp get project-settings` as a read-only CLI command:

- Command shape: `rp get project-settings` (no positional args — uses `--project` flag)
- Table output: list of issue sub-types with columns: ID, SHORT NAME, LONG NAME, TYPE REF, COLOR
- JSON output: full `ProjectSettingsResource`
- Register in `internal/cli/get.go` via `cmd.AddCommand(newGetProjectSettingsCmd())`

This is explicitly optional. The client coverage is the priority.

## Definition of done

- [ ] `pkg/client/project_settings.go` with `ProjectSettingsScope` (Get, GetNotifications)
- [ ] `ProjectScope.Settings()` method added to `pkg/client/project.go`
- [ ] `pkg/client/project_settings_test.go` with unit tests
- [ ] Integration tests in `test/integration/client_project_settings_test.go`
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-integration` passes
