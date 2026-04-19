# Prompt 43: Unit Test Functional Options (Projects & Users)

## Goal

Add table-driven unit tests to verify that all untested `With...` functional options in `projects.go`, `users.go`, `dashboard.go`, and `filter.go` correctly mutate the underlying HTTP request query parameters. Together with PR 1 (prompt 42), this should push unit test coverage above 95%.

## Context

This is PR 2 of the test coverage improvement plan (Phase 1). It runs in parallel with PRs 1 and 3. The scope is limited to `pkg/client/projects_test.go`, `pkg/client/users_test.go`, `pkg/client/dashboard_test.go`, and `pkg/client/filter_test.go` — files that PR 1 does not touch.

## Untested options to cover

### `projects.go`

| Option | Applies to | Query parameter | Expected value |
|--------|-----------|----------------|----------------|
| `WithProjectType("INTERNAL")` | `List` | `filter.eq.type` | `INTERNAL` |
| `WithProjectSort("name,asc")` | `List` | `page.sort` | `name,asc` |
| `WithUserSort("login,desc")` | `ListUsers` | `page.sort` | `login,desc` |
| `WithExportFilterLastRun(date)` | `Export` | `filter.eq.last_run` | date string |
| `WithSearchUsernamesSort("login,asc")` | `SearchUsernames` | `page.sort` | `login,asc` |

### `users.go`

| Option | Applies to | Query parameter | Expected value |
|--------|-----------|----------------|----------------|
| `WithUsersSort("login,desc")` | `List` | `page.sort` | `login,desc` |
| `WithUsersType("INTERNAL")` | `List` | `filter.eq.type` | `INTERNAL` |
| `WithUsersActive(true)` | `List` | `filter.eq.expired` | `false` |
| `WithUsersEmail("user@example.com")` | `List` | `filter.eq.email` | `user@example.com` |
| `WithSearchUsersPageNumber(2)` | `Search` | `page.page` | `2` |
| `WithSearchUsersSort("login,asc")` | `Search` | `page.sort` | `login,asc` |

### `dashboard.go`

| Option | Applies to | Query parameter | Expected value |
|--------|-----------|----------------|----------------|
| `WithDashboardSort("name,asc")` | `List` | `page.sort` | `name,asc` |

### `filter.go`

| Option | Applies to | Query parameter | Expected value |
|--------|-----------|----------------|----------------|
| `WithFilterSort("name,asc")` | `List` | `page.sort` | `name,asc` |

## Implementation plan

### 1) Verify option signatures

Before writing tests, read the actual option function signatures in each file to confirm:

- Parameter types (variadic `...string` joined with comma, or single string)
- The exact query parameter key each option sets
- The API method each option applies to
- Special cases like `WithUsersActive(true)` mapping to `filter.eq.expired` = `false` (inverted boolean)

### 2) Add project option tests in `pkg/client/projects_test.go`

Follow the existing `TestProjectsScope_List_WithOptions` pattern. Add a new table-driven test for the untested options:

```go
func TestProjectsScope_List_WithAdditionalOptions(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        option   client.ListProjectsOption
        expected map[string]string
    }{
        {
            name:   "WithProjectType",
            option: client.WithProjectType("INTERNAL"),
            expected: map[string]string{
                "filter.eq.type": "INTERNAL",
            },
        },
        {
            name:   "WithProjectSort",
            option: client.WithProjectSort("name", "asc"),
            expected: map[string]string{
                "page.sort": "name,asc",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // ... handler asserts query params, returns empty page JSON
        })
    }
}
```

Add separate tests for `WithUserSort` (on `ListUsers`), `WithExportFilterLastRun` (on `Export`), and `WithSearchUsernamesSort` (on `SearchUsernames`), each calling the appropriate API method with the option and asserting the query parameter in the handler.

### 3) Add user option tests in `pkg/client/users_test.go`

Add a table-driven test for the untested `ListUsersOption` options (`WithUsersSort`, `WithUsersType`, `WithUsersActive`, `WithUsersEmail`). Pay special attention to `WithUsersActive` — read the implementation to confirm the query key and value mapping (it may invert the boolean to `filter.eq.expired`).

Add a separate test for the `SearchUsersOption` options (`WithSearchUsersPageNumber`, `WithSearchUsersSort`). This calls `Users().Search(ctx, "term", option)` and asserts query parameters.

### 4) Add dashboard sort test in `pkg/client/dashboard_test.go`

Add a single test for `WithDashboardSort` calling `Dashboards().List(ctx, option)` and asserting `page.sort`.

### 5) Add filter sort test in `pkg/client/filter_test.go`

Add a single test for `WithFilterSort` calling `Filters().List(ctx, option)` and asserting `page.sort`.

### 6) Run tests and lint

```bash
go test -v ./pkg/client/ -run 'WithAdditionalOptions|WithSort|WithUsersType|WithUsersActive|WithUsersEmail|WithSearchUsers'
make test
make lint
```

## Test conventions

- Package: `package client_test` (black-box)
- Every test and subtest calls `t.Parallel()`
- Use `newTestClient(t, handler)` to create the test client
- Handler returns valid JSON for the expected response shape (empty page, empty slice, etc.)
- Use `assert.Equal` for query parameter assertions with descriptive messages
- Use `require.NoError` for the API call result

## Files touched

| File | Change |
|------|--------|
| `pkg/client/projects_test.go` | Add tests for `WithProjectType`, `WithProjectSort`, `WithUserSort`, `WithExportFilterLastRun`, `WithSearchUsernamesSort` |
| `pkg/client/users_test.go` | Add tests for `WithUsersSort`, `WithUsersType`, `WithUsersActive`, `WithUsersEmail`, `WithSearchUsersPageNumber`, `WithSearchUsersSort` |
| `pkg/client/dashboard_test.go` | Add test for `WithDashboardSort` |
| `pkg/client/filter_test.go` | Add test for `WithFilterSort` |

## What this PR does NOT change

- No production code modified
- No integration tests modified
- No new dependencies
