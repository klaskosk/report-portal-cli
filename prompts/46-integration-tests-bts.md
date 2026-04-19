# Prompt 46: Integration Tests for Bug Tracking Systems (BTS)

## Goal

Implement integration tests for instance-level and project-level bug tracking system integrations. The existing `client_bts_test.go` is minimal (34 lines, single test for `GetTicketIdsForProject`). No `client_instance_bts_test.go` exists.

## Context

This is PR 5 of the test coverage improvement plan (Phase 2). It runs in parallel with PRs 4 and 6. Phase 1 must be merged first.

## Prerequisites

- Phase 1 (PRs 1-3) is merged.

## Scopes under test

### `BugTrackingScope` — `Client.Project(name).BugTracking()`

| Method | Signature |
|--------|-----------|
| `GetSystemFields` | `(ctx, integrationID int64) ([]types.PostFormField, error)` |
| `GetAllowableIssueTypes` | `(ctx, integrationID int64) ([]string, error)` |
| `GetTicket` | `(ctx, ticketID, btsURL, btsProject string) (*types.Ticket, error)` |
| `CreateIssue` | `(ctx, integrationID int64, request types.PostTicketRQ) (*types.Ticket, error)` |
| `GetTicketIds` | `(ctx, launchID int64, term string) ([]string, error)` |
| `GetTicketIdsForProject` | `(ctx, term string) ([]string, error)` |

### `InstanceBugTrackingScope` — `Client.InstanceBugTracking()`

| Method | Signature |
|--------|-----------|
| `GetSystemFields` | `(ctx, integrationID int64, issueType string) ([]types.PostFormField, error)` |
| `GetAllowableIssueTypes` | `(ctx, integrationID int64) ([]string, error)` |

## Implementation plan

### 1) Understand the test environment

BTS endpoints require a configured bug tracking integration (e.g., Rally, JIRA). The default Report Portal test instance may not have any BTS integrations configured. Before writing tests:

- Check if the docker-compose test stack includes a BTS integration
- List existing integrations via `c.Integrations().ListGlobal(ctx)` to see if any BTS plugins are available
- If no BTS integration exists, the tests will need to either:
  a) Create a mock BTS integration first (if the API supports it)
  b) Test only the error paths (e.g., "returns not found for a non-existent integration")

### 2) Expand `test/integration/client_bts_test.go`

Build on the existing file. Keep the existing `GetTicketIdsForProject` test and add:

```
Describe("Client: Bug Tracking", Label("client", "bts"))
  BeforeEach: newClient(), create test project
  AfterEach: delete test project

  Context("Ticket IDs")
    It("returns ticket IDs for a project without error") [existing]
    It("returns ticket IDs for a launch")

  Context("System Fields")
    It("returns system fields for a known integration")
    It("returns not found for a non-existent integration")

  Context("Allowable Issue Types")
    It("returns issue types for a known integration")
    It("returns not found for a non-existent integration")

  Context("Tickets")
    It("returns not found for a non-existent ticket")
```

### 3) Create `test/integration/client_instance_bts_test.go`

```
Describe("Client: Instance Bug Tracking", Label("client", "instance-bts"))
  BeforeEach: newClient()

  Context("System Fields")
    It("returns system fields for a known integration")
    It("returns not found for a non-existent integration")

  Context("Allowable Issue Types")
    It("returns issue types for a known integration")
    It("returns not found for a non-existent integration")
```

### 4) Strategy for BTS integration availability

The Report Portal test instance ships with the Rally plugin by default. Check by calling:

```go
plugins, _ := c.Plugins().List(ctx)
```

Find the Rally plugin and its integration. If a global Rally integration exists, use its ID for the BTS tests. If not, create a global integration in `BeforeEach`:

```go
integration, err := c.Integrations().CreateGlobal(ctx, "rally", types.CreateIntegrationRQ{
    Name: "test_rally_<timestamp>",
    // ... minimal config
})
```

If creating a BTS integration is not feasible in the test environment, focus on error-path tests (not-found for invalid integration IDs) and document the limitation.

### 5) Follow existing test conventions

- `NodeTimeout` on setup/teardown nodes
- `By("description")` for multi-step tests
- `Expect(client.IsNotFound(err)).To(BeTrue())` for not-found assertions
- Labels: `Label("client", "bts")` and `Label("client", "instance-bts")`

### 6) Run tests

```bash
make test-integration
make lint
```

## Files touched

| File | Change |
|------|--------|
| `test/integration/client_bts_test.go` | Expand with additional BTS tests |
| `test/integration/client_instance_bts_test.go` | Create — InstanceBugTrackingScope integration tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- Only BTS-related integration tests are affected
