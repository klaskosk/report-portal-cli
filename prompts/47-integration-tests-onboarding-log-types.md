# Prompt 47: Integration Tests for Onboarding & Log Types

## Goal

Implement integration tests for the onboarding and log types endpoints. Neither has any integration test coverage today — no test files exist for them.

## Context

This is PR 6 of the test coverage improvement plan (Phase 2). It runs in parallel with PRs 4 and 5. Phase 1 must be merged first.

## Prerequisites

- Phase 1 (PRs 1-3) is merged.

## Scopes under test

### `OnboardingScope` — `Client.Onboarding()`

| Method | Signature |
|--------|-----------|
| `GetStatus` | `(ctx, page *string) (map[string]any, error)` |

### `LogTypesScope` — `Client.Project(name).LogTypes()`

| Method | Signature |
|--------|-----------|
| `List` | `(ctx) ([]types.LogTypeResponse, error)` |
| `Create` | `(ctx, request types.LogTypeRequest) (*types.LogTypeResponse, error)` |
| `Update` | `(ctx, logTypeID int64, request types.LogTypeRequest) (*types.SuccessfulUpdate, error)` |
| `Delete` | `(ctx, logTypeID int64) error` |

## Implementation plan

### 1) Create `test/integration/client_onboarding_test.go`

```go
//go:build integration

package integration_test
```

#### Describe block structure

```
Describe("Client: Onboarding", Label("client", "onboarding"))
  BeforeEach: newClient()

  Context("GetStatus")
    It("returns onboarding status without a page filter")
      - Call c.Onboarding().GetStatus(ctx, nil)
      - Assert no error
      - Assert result is a non-nil map

    It("returns onboarding status with a page filter")
      - page := "GENERAL"
      - Call c.Onboarding().GetStatus(ctx, &page)
      - Assert no error
      - Assert result is a non-nil map
```

The onboarding endpoint is simple — a single GET that returns a map. The main value is confirming it works against a real Report Portal instance. Try different `page` values like `"GENERAL"` or `nil` (read the API to understand valid values).

### 2) Create `test/integration/client_log_types_test.go`

```go
//go:build integration

package integration_test
```

#### Describe block structure

```
Describe("Client: Log Types", Label("client", "log-types"))
  var c *client.Client
  var projectName string

  BeforeEach: newClient(), create test project
  AfterEach: delete test project

  Context("CRUD Lifecycle")
    It("creates, reads, updates, and deletes a log type")
      By("creating a log type")
      - Create a log type with a descriptive name
      - Assert no error, assert returned ID > 0

      By("listing log types")
      - List log types for the project
      - Assert the created log type appears in the list

      By("updating the log type")
      - Update with a new name
      - Assert no error

      By("deleting the log type")
      - Delete by ID
      - Assert no error

      By("verifying deletion")
      - List log types again
      - Assert the deleted log type is no longer in the list

  Context("List")
    It("returns an empty list for a new project")
      - Call c.Project(projectName).LogTypes().List(ctx)
      - Assert no error
      - Assert empty slice (not nil)
```

### 3) Determine `LogTypeRequest` fields

Before implementing, read `types.LogTypeRequest` in `pkg/types/` to understand its fields. The request likely includes a name/pattern for the log type. Use a descriptive test value like:

```go
types.LogTypeRequest{
    // fill in actual fields after reading the type definition
}
```

### 4) Follow existing test conventions

- `NodeTimeout(60*time.Second)` on `BeforeEach`
- `NodeTimeout(15*time.Second)` on `AfterEach`
- `By("description")` for documentation
- Project names: `fmt.Sprintf("client_log_types_test_%d", time.Now().UnixNano())`
- Labels: `Label("client", "onboarding")`, `Label("client", "log-types")`

### 5) Run tests

```bash
make test-integration
make lint
```

## Files to create

| File | Action |
|------|--------|
| `test/integration/client_onboarding_test.go` | Create — OnboardingScope integration tests |
| `test/integration/client_log_types_test.go` | Create — LogTypesScope integration tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No existing integration tests modified
