# Prompt 53: Integration Tests for Test Items (Advanced)

## Goal

Expand integration tests for the `TestItemScope` to cover external issue linking/unlinking, bulk operations, and advanced querying/history. The existing `client_item_test.go` (222 lines) covers basic listing, get-by-ID, get-by-UUID, bulk write operations, write operations, and history, but has gaps in advanced filtering, issue management, and statistics.

## Context

This is PR 12 of the test coverage improvement plan (Phase 4). It runs in parallel with PRs 11 and 13. Phases 1-3 must be merged first.

## Prerequisites

- Phases 1-3 (PRs 1-10) are merged.

## Scope under test

### `TestItemScope` — `Client.Project(name).Items()`

Review existing tests first (read `client_item_test.go`), then add coverage for:

#### External issue linking

| Method | Signature |
|--------|-----------|
| `LinkExternalIssues` | `(ctx, itemID int64, request types.LinkExternalIssueRQ) (*types.OperationCompletionRS, error)` |
| `UnlinkExternalIssues` | `(ctx, itemID int64, request types.UnlinkExternalIssueRQ) (*types.OperationCompletionRS, error)` |

Test scenario:
1. Use a seeded test item
2. Link an external issue (fabricated ticket URL/ID)
3. Verify the link was created (re-fetch the item and check its issue references)
4. Unlink the issue
5. Verify the link was removed

#### Bulk operations

| Method | Signature |
|--------|-----------|
| `BulkUpdate` | `(ctx, request types.BulkUpdateTestItemRQ) (*types.OperationCompletionRS, error)` |

Check if `BulkUpdate` is already tested. If not, add a test that updates the status or description of multiple items at once.

#### Advanced querying

| Method | Options to test |
|--------|----------------|
| `List` | `WithItemPath(path)`, `WithItemParentID(id)`, `WithItemType(type)` |
| `ListV2` | Various `WithItemV2*` options |
| `Search` | `WithSearchItemName`, `WithSearchItemSort` |

Verify which of these are already exercised in integration. Add tests for uncovered options.

#### History

| Method | Options to test |
|--------|----------------|
| `GetHistory` | `WithHistoryFilterID`, `WithHistoryLaunchesLimit`, `WithHistoryType`, `WithHistoryItemName` |

The existing `GetHistory` test may only cover basic history retrieval. Add tests with advanced filter options.

#### Statistics

| Method | Signature |
|--------|-----------|
| `GetStatistics` | `(ctx, options ...StatisticsOption) (*types.StatisticsResource, error)` |
| `GetAttributeKeys` | `(ctx, launchID int64, options ...) ([]string, error)` |
| `GetAttributeKeysForProject` | `(ctx, options ...) ([]string, error)` |

## Implementation plan

### 1) Read existing tests

Read `test/integration/client_item_test.go` thoroughly. Note:
- How test items are seeded (via `seedTestItems` or similar helper)
- What operations are already tested
- The `BeforeEach` pattern (project creation + launch seeding + item seeding)

### 2) Add new Context blocks

```
Describe("Client: Test Items", Label("client", "items"))
  [existing BeforeEach/AfterEach and contexts remain]

  Context("External Issues")
    It("links and unlinks an external issue")
      By("linking an external issue to a test item")
      By("verifying the link exists")
      By("unlinking the external issue")
      By("verifying the link is removed")

  Context("Advanced Filtering")
    It("filters items by type")
    It("filters items by parent ID")
    It("searches items by name")

  Context("History with Options")
    It("retrieves history with a launches limit")
    It("retrieves history filtered by item name")

  Context("Statistics")
    It("returns statistics for a launch")
    It("returns attribute keys for a launch")
    It("returns attribute keys for the project")
```

### 3) Test data requirements

- **External issue tests**: Need a seeded test item with a defect type that allows issue linking. Read the `LinkExternalIssueRQ` type to understand required fields.
- **Statistics**: Need seeded test items with results (passed/failed) to generate meaningful statistics.
- **History**: Need multiple launches with the same test items for history to be meaningful.

### 4) Follow existing test conventions

- `NodeTimeout` on setup/teardown
- `By("description")` for multi-step tests
- Labels: `Label("client", "items")`
- Reuse the existing seeded data from `BeforeEach` where possible

### 5) Run tests

```bash
make test-integration
make lint
```

## Files touched

| File | Change |
|------|--------|
| `test/integration/client_item_test.go` | Add external issue, advanced filtering, history, and statistics tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No other integration test files modified
