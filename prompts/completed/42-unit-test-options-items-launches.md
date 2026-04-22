# Prompt 42: Unit Test Functional Options (Items & Launches)

## Goal

Add table-driven unit tests to verify that all untested `With...` functional options in `item.go` and `launch.go` correctly mutate the underlying HTTP request query parameters.

## Context

This is PR 1 of the test coverage improvement plan (Phase 1). It runs in parallel with PRs 2 and 3. The scope is limited to `pkg/client/item_test.go` and `pkg/client/launch_test.go` — files that PR 2 does not touch.

Current unit test coverage is 88.5%. This PR, together with PR 2 (prompt 43), targets > 95% unit coverage by closing functional-option gaps.

## Background

The project uses functional options (e.g. `WithLaunchSort`, `WithHistoryFilterID`) to configure list/query parameters. Each option mutates a generated params struct, which translates to HTTP query parameters. Existing tests cover many options but several remain untested.

## Untested options to cover

### `launch.go`

| Option | Query parameter | Expected value |
|--------|----------------|----------------|
| `WithLaunchSort("startTime,desc")` | `page.sort` | `startTime,desc` |
| `WithClusterSort("id,asc")` | `page.sort` | `id,asc` |

### `item.go`

#### History options (`HistoryOption`)

| Option | Query parameter | Expected value |
|--------|----------------|----------------|
| `WithHistoryFilterID(42)` | `filterId` | `42` |
| `WithHistoryLaunchesLimit(10)` | `launchesLimit` | `10` |
| `WithHistoryType("line")` | `type` | `line` |
| `WithHistoryItemName("login test")` | `filter.eq.name` | `login test` |
| `WithHistoryPageNumber(3)` | `historyPage.page` | `3` |
| `WithHistoryPageSize(25)` | `historyPage.size` | `25` |
| `WithHistoryLatest(true)` | `isLatest` | `true` |

#### Statistics options (`StatisticsOption`)

| Option | Query parameter | Expected value |
|--------|----------------|----------------|
| `WithStatisticsParentID(99)` | `filter.eq.parentId` | `99` |
| `WithStatisticsStatus("PASSED")` | `filter.eq.status` | `PASSED` |
| `WithStatisticsName("suite A")` | `filter.eq.name` | `suite A` |
| `WithStatisticsType("SUITE")` | `filter.eq.type` | `SUITE` |
| `WithStatisticsHasChildren(true)` | `filter.eq.hasChildren` | `true` |

## Implementation plan

### 1) Add launch sort tests in `pkg/client/launch_test.go`

Add a table-driven test following the existing `TestLaunchScope_WithOtherFilters` pattern:

```go
func TestLaunchScope_WithSortOptions(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        option   client.ListLaunchesOption
        expected map[string]string
    }{
        {
            name:   "WithLaunchSort",
            option: client.WithLaunchSort("startTime", "desc"),
            expected: map[string]string{
                "page.sort": "startTime,desc",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                query := r.URL.Query()

                for key, expectedValue := range tt.expected {
                    assert.Equal(t, expectedValue, query.Get(key),
                        "query parameter mismatch for "+key)
                }

                w.Header().Set("Content-Type", "application/json")
                _, _ = w.Write([]byte(`{"content":[],"page":{"number":1,"size":20,"totalElements":0,"totalPages":0}}`))
            }))

            _, err := c.Project("my_project").Launches().List(t.Context(), tt.option)
            require.NoError(t, err)
        })
    }
}
```

Add a similar test for `WithClusterSort` that calls `GetClusters` with the sort option and verifies the `page.sort` query parameter. Follow the existing `TestLaunchScope_GetClusters_Success` pattern for the handler response shape.

### 2) Add history option tests in `pkg/client/item_test.go`

Add a table-driven test for the untested history options:

```go
func TestTestItemScope_GetHistory_WithAdditionalOptions(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        option   client.HistoryOption
        expected map[string]string
    }{
        {
            name:   "WithHistoryFilterID",
            option: client.WithHistoryFilterID(42),
            expected: map[string]string{
                "filterId": "42",
            },
        },
        {
            name:   "WithHistoryLaunchesLimit",
            option: client.WithHistoryLaunchesLimit(10),
            expected: map[string]string{
                "launchesLimit": "10",
            },
        },
        // ... remaining options from the table above
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                query := r.URL.Query()

                for key, expectedValue := range tt.expected {
                    assert.Equal(t, expectedValue, query.Get(key),
                        "query parameter mismatch for "+key)
                }

                w.Header().Set("Content-Type", "application/json")
                _, _ = w.Write([]byte(`{"content":[]}`))
            }))

            _, err := c.Project("my_project").Items().GetHistory(t.Context(), []int64{1}, tt.option)
            require.NoError(t, err)
        })
    }
}
```

### 3) Add statistics option tests in `pkg/client/item_test.go`

Add a table-driven test for the untested statistics options:

```go
func TestTestItemScope_GetStatistics_WithOptions(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        option   client.StatisticsOption
        expected map[string]string
    }{
        {
            name:   "WithStatisticsParentID",
            option: client.WithStatisticsParentID(99),
            expected: map[string]string{
                "filter.eq.parentId": "99",
            },
        },
        // ... remaining options from the table above
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                query := r.URL.Query()

                for key, expectedValue := range tt.expected {
                    assert.Equal(t, expectedValue, query.Get(key),
                        "query parameter mismatch for "+key)
                }

                w.Header().Set("Content-Type", "application/json")
                _, _ = w.Write([]byte(`{}`))
            }))

            _, err := c.Project("my_project").Items().GetStatistics(t.Context(), tt.option)
            require.NoError(t, err)
        })
    }
}
```

### 4) Verify option signatures before writing tests

Before implementing, read the actual option function signatures in `item.go` and `launch.go` to confirm:

- Parameter types (e.g. `WithLaunchSort` takes variadic `...string` and joins with comma)
- The exact query parameter key each option sets (read the function body)
- The API method each option applies to (e.g. `List`, `GetHistory`, `GetStatistics`, `GetClusters`)

Adjust the test code if the actual signatures differ from the table above.

### 5) Run tests and lint

```bash
go test -v -run 'TestLaunchScope_WithSortOptions|TestTestItemScope_GetHistory_WithAdditionalOptions|TestTestItemScope_GetStatistics_WithOptions' ./pkg/client/
make test
make lint
```

## Files touched

| File | Change |
|------|--------|
| `pkg/client/launch_test.go` | Add `TestLaunchScope_WithSortOptions`, cluster sort test |
| `pkg/client/item_test.go` | Add `TestTestItemScope_GetHistory_WithAdditionalOptions`, `TestTestItemScope_GetStatistics_WithOptions` |

## What this PR does NOT change

- No production code modified
- No integration tests modified
- No new dependencies
