# Prompt 54: Integration Tests for Logs (Advanced)

## Goal

Expand integration tests for the `LogScope` to cover log searching, nested log retrieval, and edge cases in batch creation. The existing `client_log_test.go` (216 lines) covers basic listing, get-by-ID, batch write, write operations, search, and `GetLogsUnder`, but there are gaps in advanced filtering, pagination edge cases, and batch creation error handling.

## Context

This is PR 13 of the test coverage improvement plan (Phase 4). It runs in parallel with PRs 11 and 12. Phases 1-3 must be merged first.

## Prerequisites

- Phases 1-3 (PRs 1-10) are merged.

## Scope under test

### `LogScope` — `Client.Project(name).Logs()`

Review the existing tests first (read `client_log_test.go`), then expand coverage for:

#### Advanced search

| Method | Signature |
|--------|-----------|
| `Search` | `(ctx, request types.SearchRQ) (*types.SearchLogPage, error)` |

Test scenarios:
- Search with different filter criteria
- Search returning empty results
- Search with pagination

#### Nested log retrieval

| Method | Signature |
|--------|-----------|
| `GetLogsUnder` | `(ctx, parentID int64, options ...ListLogsOption) ([]types.PagedLogResource, error)` |

Test scenarios:
- Retrieve logs under a specific test item
- Verify log hierarchy (parent-child relationships)
- Test with different log levels

#### Batch creation edge cases

| Method | Signature |
|--------|-----------|
| `CreateBatch` | `(ctx, request types.SaveLogRQ) (*types.BatchSaveOperatingRS, error)` |

Test scenarios:
- Batch create logs with attachments (multipart)
- Large batch creation (multiple log entries)
- Batch creation with different log levels (ERROR, WARN, INFO, DEBUG, TRACE)

#### Log listing with filters

| Method | Options to test |
|--------|----------------|
| `List` | `WithLogItemID`, `WithLogLaunchID`, `WithLogLevel`, pagination options |

Verify which filter options are already exercised. Add tests for uncovered options.

#### Page number retrieval

| Method | Signature |
|--------|-----------|
| `GetPageNumber` | `(ctx, logID, itemID int64, options ...ListLogsOption) (map[string]any, error)` |

Test scenarios:
- Get page number for a known log entry
- Verify the response contains pagination info

#### Log locations

| Method | Signature |
|--------|-----------|
| `GetLogLocations` | `(ctx, logID int64) ([]types.PagedLogResource, error)` |

Test scenarios:
- Get locations for a known log
- Verify the response contains location data

## Implementation plan

### 1) Read existing tests

Read `test/integration/client_log_test.go` thoroughly. Note:
- How logs are seeded (via `seedTestLogEntries`, `seedLogWithAttachment`)
- What operations are already tested
- The `BeforeEach` pattern

### 2) Add new Context blocks

```
Describe("Client: Logs", Label("client", "logs"))
  [existing BeforeEach/AfterEach and contexts remain]

  Context("Advanced Search")
    It("searches logs with specific criteria")
    It("returns empty results for non-matching search")

  Context("Nested Logs")
    It("retrieves logs under a parent item")
    It("returns empty for an item with no logs")

  Context("Batch Creation with Levels")
    It("creates logs with different log levels")
      By("creating log entries with ERROR, WARN, INFO levels")
      By("listing logs and verifying levels")

  Context("Page Number")
    It("returns the page number for a known log")

  Context("Log Locations")
    It("returns locations for a known log")

  Context("Advanced Filtering")
    It("filters logs by level")
    It("filters logs by launch ID")
```

### 3) Test data requirements

- **Search tests**: Need seeded logs with distinctive messages for search to find
- **Nested logs**: Need test items with child items that have their own logs
- **Batch with levels**: Use different log levels in the `SaveLogRQ` entries
- **Page number / locations**: Need at least enough logs to span multiple pages

### 4) Follow existing test conventions

- `NodeTimeout` on setup/teardown
- `By("description")` for multi-step tests
- Labels: `Label("client", "logs")`
- Use `Eventually` with polling for log creation visibility (logs may not appear immediately)

### 5) Run tests

```bash
make test-integration
make lint
```

## Files touched

| File | Change |
|------|--------|
| `test/integration/client_log_test.go` | Add advanced search, nested logs, batch levels, page number, locations, and filtering tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No other integration test files modified
