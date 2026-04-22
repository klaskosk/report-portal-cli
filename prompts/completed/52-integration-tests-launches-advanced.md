# Prompt 52: Integration Tests for Launches (Advanced)

## Goal

Expand integration tests for the `LaunchScope` to cover advanced filtering, bulk force finishing, merging, and comparing launches. The existing `client_launch_test.go` (647 lines) has solid baseline coverage for basic CRUD and listing, but lacks coverage for advanced operations.

## Context

This is PR 11 of the test coverage improvement plan (Phase 4). It runs in parallel with PRs 12 and 13. Phases 1-3 must be merged first. Phase 1's helper refactoring (PR 3) makes test data setup easier.

## Prerequisites

- Phases 1-3 (PRs 1-10) are merged.

## Scope under test

### `LaunchScope` — `Client.Project(name).Launches()`

Review the existing tests first (read `client_launch_test.go`) to understand what's already covered. Then add tests for these areas:

#### Advanced filtering

| Method | Options to test |
|--------|----------------|
| `List` | `WithLaunchAttribute("key", "value")`, `WithLaunchAttributeKey("key")`, `WithLaunchNumber(n)`, `WithLaunchMode("DEBUG")` |
| `GetLatest` | Combination of multiple filter options |

The existing tests cover basic `List` with name/status filters and `GetLatest`. Verify whether attribute-based filtering and launch-number filtering are exercised in integration. If not, add tests.

#### Bulk force finish

| Method | Signature |
|--------|-----------|
| `BulkForceFinish` | `(ctx, request types.BulkRQ) ([]types.OperationCompletionRS, error)` |

Test scenario:
1. Seed multiple launches (some may already be finished — check seeded data state)
2. Start new launches that remain in "IN_PROGRESS" state
3. Call `BulkForceFinish` with those launch IDs
4. Verify the launches are now in "STOPPED" or "FAILED" state

#### Merge

| Method | Signature |
|--------|-----------|
| `Merge` | `(ctx, request types.MergeLaunchesRQ) (*types.LaunchResource, error)` |

Test scenario:
1. Seed at least 2 launches
2. Call `Merge` to combine them into one
3. Verify the merged launch exists and the original launches are consumed

**Note**: `Merge` uses `handleV2Response` due to content negotiation. It may have quirks — read the implementation before writing the test.

#### Compare

| Method | Signature |
|--------|-----------|
| `Compare` | `(ctx, ids []int64) (map[string]any, error)` |

The existing test covers basic comparison. Expand with:
- Comparing more than 2 launches
- Comparing launches with different statuses

#### Clusters

| Method | Signature |
|--------|-----------|
| `GetClusters` | `(ctx, launchID int64, options ...ListClustersOption) (*types.ClusterPage, error)` |

Test scenario:
1. Use a seeded launch
2. Call `GetClusters` with pagination options
3. Assert the response structure is valid (may return empty clusters if the analyzer hasn't run)

## Implementation plan

### 1) Read existing tests

Read `test/integration/client_launch_test.go` thoroughly to understand:
- What's already covered
- How test data is seeded (`seedTestLaunches`)
- The `BeforeEach`/`AfterEach` pattern

### 2) Add new Context blocks

```
Describe("Client: Launches", Label("client", "launches"))
  [existing BeforeEach/AfterEach with project creation and launch seeding]

  [existing contexts remain unchanged]

  Context("Advanced Filtering")
    It("filters by launch number")
    It("filters by mode")
    It("filters by attribute key-value pair")
    It("filters by attribute key only")

  Context("Bulk Force Finish")
    It("force finishes multiple launches")
      By("starting new in-progress launches")
      By("calling BulkForceFinish")
      By("verifying launches are stopped")

  Context("Merge")
    It("merges two launches into one")
      By("listing launches before merge")
      By("merging two launches")
      By("verifying the merged launch exists")

  Context("Clusters")
    It("returns clusters for a launch")
    It("supports pagination options")
```

### 3) Test data setup for bulk operations

For `BulkForceFinish`, you need launches in "IN_PROGRESS" state. The seeded launches may already be finished. Either:
- Start new launches via `c.Project(projectName).Launches().Start(ctx, request)` and leave them unfinished
- Or check the seeded data state

For `Merge`, you need at least 2 launch IDs. Use the seeded launches.

### 4) Follow existing test conventions

- `NodeTimeout(60*time.Second)` on `BeforeEach`
- `By("description")` for multi-step tests
- Use `expectContent` helper for paginated results
- Labels: `Label("client", "launches")`

### 5) Run tests

```bash
make test-integration
make lint
```

## Files touched

| File | Change |
|------|--------|
| `test/integration/client_launch_test.go` | Add advanced filtering, bulk force finish, merge, and cluster tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No other integration test files modified
