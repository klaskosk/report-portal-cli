# Prompt 49: Integration Tests for Projects Expansion

## Goal

Expand the integration tests for the `ProjectsScope` beyond basic fetching. The existing `client_projects_test.go` (183 lines) covers `Get`, `ListNames`, `List`, `SearchNames`, `GetInfo`, `Create`/`Delete`, `Update`, and `ListUsers`. This PR adds tests for project preferences, indexing, analyzer status, notification config, user assignment, and other underexercised methods.

## Context

This is PR 8 of the test coverage improvement plan (Phase 3). It runs in parallel with PRs 7, 9, and 10. Phase 2 must be merged first.

## Prerequisites

- Phases 1-2 (PRs 1-6) are merged.

## Scope under test

### `ProjectsScope` — `Client.Projects()`

Methods to add or expand integration tests for:

| Method | Signature | Status |
|--------|-----------|--------|
| `Get` | `(ctx, projectName string) (*types.ProjectResource, error)` | Exists |
| `List` | `(ctx, options ...) (*types.PageProjectInfoResource, error)` | Exists |
| `ListNames` | `(ctx) ([]string, error)` | Exists |
| `SearchNames` | `(ctx, term string) ([]string, error)` | Exists |
| `GetInfo` | `(ctx, projectName string, options ...) (*types.ProjectInfoResource, error)` | Exists |
| `Create` / `Delete` | | Exists |
| `Update` | | Exists |
| `ListUsers` | | Exists |
| `Export` | `(ctx, options ...) ([]byte, error)` | **New** |
| `GetAnalyzerStatus` | `(ctx) (map[string]bool, error)` | **New** |
| `AssignUsers` | `(ctx, projectName string, request types.AssignUsersRQ) (*types.OperationCompletionRS, error)` | **New** |
| `UnassignUsers` | `(ctx, projectName string, request types.UnassignUsersRQ) (*types.OperationCompletionRS, error)` | **New** |
| `GetPreference` | `(ctx, projectName, login string) (*, error)` | **New** |
| `AddPreference` | `(ctx, projectName, login string, request ...) (*, error)` | **New** |
| `RemovePreference` | `(ctx, projectName, login string, preferenceID int64) error` | **New** |
| `UpdateNotificationConfig` | `(ctx, projectName string, request ...) (*, error)` | **New** |
| `DeleteIndex` | `(ctx, projectName string) (*, error)` | **New** |
| `ListAssignableUsers` | `(ctx, projectName, term string, options ...) (*, error)` | **New** |
| `ListUsernames` | `(ctx, projectName string, options ...) (*, error)` | **New** |
| `SearchUsernames` | `(ctx, projectName, term string, options ...) (*, error)` | **New** |

## Implementation plan

### 1) Read method signatures

Before implementing, read the actual signatures in `pkg/client/projects.go` for all methods listed above. Pay attention to:

- Request and response types
- Whether methods operate on the `ProjectsScope` (instance-level) or need a project name parameter
- Option types for listing methods

### 2) Expand `test/integration/client_projects_test.go`

Add new Context blocks. Structure:

```
Describe("Client: Projects", Label("client", "projects"))
  [existing contexts remain unchanged]

  Context("Export")
    It("returns CSV data for projects")
      - Call c.Projects().Export(ctx)
      - Assert no error
      - Assert non-empty byte slice

  Context("Analyzer Status")
    It("returns analyzer status")
      - Call c.Projects().GetAnalyzerStatus(ctx)
      - Assert no error
      - Assert result is a non-nil map

  Context("Assignable Users")
    It("returns assignable users for a project")
      - Call c.Projects().ListAssignableUsers(ctx, projectName, "")
      - Assert no error

  Context("Usernames")
    It("lists usernames for a project")
    It("searches usernames in a project")

Describe("Client: Projects Write", Label("client", "projects"))
  [existing write contexts remain unchanged]

  Context("User Assignment")
    It("assigns and unassigns a user")
      By("creating a test user")
      By("assigning the user to the project")
      By("verifying the user appears in project users")
      By("unassigning the user")
      By("cleaning up the test user")

  Context("Preferences")
    It("adds, gets, and removes a project preference")
      By("adding a preference")
      By("getting the preference")
      By("removing the preference")

  Context("Notification Config")
    It("updates notification configuration")

  Context("Index Management")
    It("deletes and optionally reindexes project data")
      - Call c.Projects().DeleteIndex(ctx, projectName)
      - Assert no error
```

### 3) Test data requirements

- **User assignment tests**: Create a temporary user via `c.Users().Create()`, assign to project, then clean up. Use `DeferCleanup` for user deletion.
- **Preferences**: These are per-user project preferences (like dashboard filters). Read the `AddPreference` request type to understand what to pass.
- **Notification config**: Read `UpdateNotificationConfig` request type. Use minimal/default configuration.

### 4) Follow existing test conventions

- `NodeTimeout` on setup/teardown
- Labels: `Label("client", "projects")`
- `By("description")` for multi-step tests
- Use the refactored `createProject` helper from PR 3

### 5) Run tests

```bash
make test-integration
make lint
```

## Files touched

| File | Change |
|------|--------|
| `test/integration/client_projects_test.go` | Expand with export, analyzer, assignment, preference, notification, and index tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No other integration test files modified
