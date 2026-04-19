# Prompt 44: Refactor Raw HTTP Calls in Integration Tests

## Goal

Replace raw `http.NewRequest` and `httpClient.Do` calls in `test/integration/helpers_test.go`, `test/integration/client_widget_test.go`, and `test/integration/client_launch_test.go` with their respective `pkg/client` wrapper methods.

## Context

This is PR 3 of the test coverage improvement plan (Phase 1). It runs in parallel with PRs 1 and 2. Several integration test helpers bypass the Go client and make raw HTTP calls to Report Portal. This creates a maintenance burden and means those code paths in `pkg/client` don't get exercised during integration tests. By replacing raw calls with client wrappers, we both simplify the helpers and increase integration coverage of the client library.

## Prerequisites

None — this PR is independent of PRs 1 and 2.

## Scope of raw HTTP calls to replace

### `helpers_test.go`

#### `createProject` (lines ~526–563)

Currently uses raw `http.NewRequestWithContext` POST to `/api/v1/project`. Replace with:

```go
func createProject(ctx context.Context, c *client.Client, projectName string) error {
    _, err := c.Projects().Create(ctx, types.CreateProjectRQ{
        ProjectName: projectName,
        EntryType:   "INTERNAL",
    })
    return err
}
```

This changes the function signature — it takes a `*client.Client` instead of `rpBaseURL, token string`. Update all call sites in the integration tests. The `BeforeEach` blocks already have a `c *client.Client` variable from `newClient()`, so pass that through.

**Important**: `deleteProject` also uses raw HTTP. Check if `Projects().Delete()` takes a project ID (int64) or project name. If it takes an ID, you may need to look up the project first. If the signature makes it awkward, leave `deleteProject` as raw HTTP for now and add a TODO comment.

#### `seedTestLogEntries` (lines ~944+)

Currently uses raw `http.NewRequestWithContext` POST to `/api/v1/{project}/log/entry`. Replace with:

```go
_, err := c.Project(projectName).Logs().CreateBatch(ctx, entries)
```

Read the `CreateBatch` signature in `pkg/client/log.go` to confirm the request type. The current raw call sends a JSON array of log entry objects. Confirm that `CreateBatch` accepts the same shape.

**Note**: `CreateBatch` has a known spec mismatch (it declares `JSON200` but the API returns 201). The client method handles this internally. Verify that the refactored call works correctly by running the integration tests.

#### `seedLogWithAttachment` (lines ~814+)

Currently builds a multipart body with `json_request_part` + `file` parts and POSTs to `/api/v1/{project}/log`. Replace with:

```go
_, err := c.Project(projectName).Logs().CreateBatch(ctx, request)
```

Read the `CreateBatch` method to see if it supports file attachments. If `CreateBatch` only handles JSON (not multipart), check if there's a separate method for multipart log creation. If no wrapper exists for multipart log creation, leave this helper as-is and add a TODO comment noting the gap.

### `client_widget_test.go`

#### `GetNames` raw HTTP call (lines ~194–226)

Currently builds URL `{baseURL}/api/v1/{project}/widget/names/all`, does a raw GET, and decodes the JSON response. Replace with:

```go
result, err := c.Project(projectName).Widgets().GetNames(ctx)
```

Verify that `GetNames()` exists on `WidgetScope` and returns the same data shape. Confirm the test assertions still hold with the wrapper return type.

### `client_launch_test.go`

#### `Compare` raw HTTP test

There are two tests under `Context("Compare")`:

1. **`"compares seeded launches by ID"`** — already uses `c.Project(projectName).Launches().Compare(ctx, ids)`. No change needed.

2. **`"returns epoch millis startTime in raw response"`** — uses raw `http.NewRequestWithContext` GET to `/api/v1/{project}/launch/compare?ids=...`. This test intentionally inspects the raw JSON response structure (checking whether `startTime` is `float64` or `string`). **Evaluate whether this can use the client wrapper.**

   If the test's purpose is to validate raw JSON field types (epoch millis vs ISO), it must stay as raw HTTP because the client wrapper would parse the response into typed structs, losing the raw type information. In that case, leave it as-is and add a comment explaining why it uses raw HTTP.

   If the test is actually just verifying the Compare response content and the typed wrapper preserves the relevant data, convert it to use the wrapper.

## Implementation plan

### 1) Read current helper signatures and call sites

Before making changes, read:

- `helpers_test.go` — full `createProject`, `deleteProject`, `seedTestLogEntries`, `seedLogWithAttachment` functions
- All `BeforeEach`/`AfterEach` blocks that call these helpers (search for `createProject(` across `test/integration/`)
- `pkg/client/projects.go` — `Create` and `Delete` method signatures
- `pkg/client/log.go` — `CreateBatch` method signature
- `pkg/client/widget.go` — `GetNames` method (if exists)
- `client_launch_test.go` — the raw `Compare` test

### 2) Refactor `createProject`

Change the signature to accept `*client.Client` and use `Projects().Create()`. Update all call sites across the integration test suite. This is a mechanical find-and-replace of the function arguments.

If `deleteProject` can be similarly refactored (check if `Projects().Delete()` accepts a name or can be easily adapted), do so. Otherwise, leave it as raw HTTP.

### 3) Refactor `seedTestLogEntries`

Replace the raw HTTP loop with `Logs().CreateBatch()`. Confirm the request and response types match. The polling/wait logic after creation should remain unchanged.

### 4) Evaluate `seedLogWithAttachment`

Check if the client has a multipart-capable log creation method. If yes, refactor. If no, leave as-is with a TODO.

### 5) Refactor `client_widget_test.go` GetNames

Replace the raw HTTP call with `Widgets().GetNames()`. Update assertions to work with the typed return value.

### 6) Evaluate `client_launch_test.go` Compare

Make a judgment call on the raw Compare test as described above. Document the decision with a comment.

### 7) Run integration tests

```bash
make test-integration
```

All integration tests must pass. Pay special attention to tests that seed log entries, create projects, or test widget names.

### 8) Run linter

```bash
make lint
```

After refactoring, some imports may become unused (`"net/http"`, `"strings"`, `"encoding/json"` in helpers or test files). Clean them up.

## Files touched

| File | Change |
|------|--------|
| `test/integration/helpers_test.go` | Refactor `createProject`, `seedTestLogEntries`, possibly `seedLogWithAttachment` and `deleteProject` |
| `test/integration/client_widget_test.go` | Replace raw `GetNames` HTTP call with client wrapper |
| `test/integration/client_launch_test.go` | Evaluate and possibly refactor raw `Compare` test |
| Various `test/integration/client_*_test.go` | Update `createProject` call sites to pass `*client.Client` |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No new endpoints or API surface
