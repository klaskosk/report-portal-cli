# PR 33: Instance-Scoped `activity-event` Search

## Goal

Implement the cross-project audit event search command. This is **instance-scoped** — it searches activity events across all projects without requiring `--project`. It also includes a helper subcommand to retrieve available project subject names.

## Context

Activity events are managed through `ActivityEventsScope` on the client (`client.ActivityEvents()`). This is **instance-scoped** — commands do **not** require `--project`.

**Prerequisite PRs and dependencies:**

- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `mustGetStringFlag`, `formatStringPtr`)
- PR 17 (frozen UX: §4 activity-event filter ergonomics — flag names, pagination model, defaults)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `Search` | `ctx`, `criteria types.SearchCriteriaRQ`, `options ...SearchActivityEventsOption` | `(*types.PagedResponseActivityEventResource, error)` |
| `GetProjectSubjectNames` | `ctx`, `projectName string` | `([]string, error)` |

Search options: `WithEventLimit(int32)`, `WithEventOffset(int32)`, `WithEventSort(string)`, `WithEventOrder(EventSortOrder)`.

Client-provided filter helpers: `NewEventSearchCriteria(...)`, `EventFilterByUser(username)`, `EventFilterByObjectType(objectType)`, `EventFilterByEventName(eventName)`, `EventFilterByProjectName(projectName)`.

### Key types

```go
type SearchCriteriaRQ struct {
    SearchCriterias []SearchCriteria `json:"search_criterias"`
}

type SearchCriteria struct {
    FilterKey string                   `json:"filter_key"`
    Operation *SearchCriteriaOperation `json:"operation,omitempty"`
    Value     string                   `json:"value"`
}

type PagedResponseActivityEventResource struct {
    Items      []ActivityEventResource `json:"items"`
    Limit      int32                   `json:"limit"`
    Offset     int64                   `json:"offset"`
    Order      string                  `json:"order"`
    Sort       string                  `json:"sort"`
    TotalCount int64                   `json:"total_count"`
}
```

`SearchCriteriaOperation` values: `BTW`, `CNT`, `EQ`, `IN`, `NE`, `NON_CNT`.

## Scope

Create `internal/cli/activity_event.go` and `internal/cli/activity_event_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get activity-event [filters] [pagination flags]` | Search cross-project audit events using optional `--user`, `--object-type`, `--event`, `--project-name`, `--limit`, `--offset`, `--sort`, and `--order` flags |
| `rp get activity-event subjects --project-name <name>` | Get available project subject names |

## Implementation

### 1. Create `internal/cli/activity_event.go`

#### `newGetActivityEventCmd()`

```go
func newGetActivityEventCmd() *cobra.Command
```

- `Use`: `"activity-event"`
- `Short`: `"Search cross-project activity events"`
- `Long`: Include examples showing filter flags and pagination. Note this is instance-scoped.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGetActivityEvent)`
- Flags:
  - `--user` (string, optional): filter by subject name
  - `--object-type` (string, optional): filter by object type
  - `--event` (string, optional): filter by event name (e.g., `"createDashboard"`)
  - `--project-name` (string, optional): filter by project name
  - `--limit` (int32, default 20): maximum results
  - `--offset` (int32, default 0): pagination offset
  - `--sort` (string, default `"createdAt"`): sort field (must match the API field name, e.g., `"createdAt"`)
  - `--order` (string, default `"DESC"`): sort direction (ASC or DESC)

**Do NOT** register `--project` / `-p` — activity events are instance-scoped. The `--project-name` flag is a search filter, not a project scope.

The run function:

1. Build search criteria from filter flags:

   ```go
   var criteria []types.SearchCriteria

   if cmd.Flags().Changed("user") {
       criteria = append(criteria, client.EventFilterByUser(user))
   }

   if cmd.Flags().Changed("object-type") {
       criteria = append(criteria, client.EventFilterByObjectType(objectType))
   }

   if cmd.Flags().Changed("event") {
       criteria = append(criteria, client.EventFilterByEventName(eventName))
   }

   if cmd.Flags().Changed("project-name") {
       criteria = append(criteria, client.EventFilterByProjectName(projectName))
   }

   searchCriteria := client.NewEventSearchCriteria(criteria...)
   ```

2. Build pagination options:

   ```go
   options := []client.SearchActivityEventsOption{
       client.WithEventLimit(limit),
       client.WithEventOffset(offset),
   }

   if sort != "" {
       options = append(options, client.WithEventSort(sort))
   }

   if order != "" {
       var eventOrder client.EventSortOrder
       switch order {
       case "ASC":
           eventOrder = client.EventOrderAsc
       case "DESC":
           eventOrder = client.EventOrderDesc
       default:
           return fmt.Errorf("--order must be ASC or DESC, got %q", order)
       }

       options = append(options, client.WithEventOrder(eventOrder))
   }
   ```

3. Resolve connection and client
4. Call `scope.Search(ctx, searchCriteria, options...)`
5. Unwrap the `Items` slice from the response
6. Pass the `[]ActivityEventResource` slice to `printer.Print`

**Output contract:** The value passed to `printer.Print` is the `[]ActivityEventResource` slice, not the `PagedResponseActivityEventResource` wrapper. This is consistent with all other list commands.

**Keep this PR self-contained.** Define local `activityEventHeaders` and `activityEventRow` helpers in `activity_event.go` so PR 33 can be implemented in parallel with PR 32. If PR 32 lands first, match its table shape exactly, but do not take a hard dependency on `activity.go`. Any deduplication of identical formatters belongs in the final fan-in (PR 36), not this Phase 7 track.

```go
func activityEventHeaders() []string {
    return []string{"ID", "EVENT", "OBJECT TYPE", "OBJECT NAME", "SUBJECT", "CREATED AT"}
}

func activityEventRow(e *types.ActivityEventResource) ([]string, []string) {
    return activityEventHeaders(), []string{
        strconv.FormatInt(e.Id, 10),
        e.EventName,
        e.ObjectType,
        e.ObjectName,
        e.SubjectName,
        e.CreatedAt.Format(time.RFC3339),
    }
}
```

#### `newGetActivityEventSubjectsCmd()`

```go
func newGetActivityEventSubjectsCmd() *cobra.Command
```

- `Use`: `"subjects"`
- `Short`: `"Get available project subject names"`
- `Long`: Include example. Note this is useful for discovering valid values for the `--user` filter on `rp get activity-event`, and more generally for understanding which subject names appear within a given project.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGetActivityEventSubjects)`
- Flags:
  - `--project-name` (string, required): project name to query subjects for

**Do NOT** register `--project` / `-p` — this is instance-scoped. The `--project-name` is a required parameter for this specific query, not a project scope flag.

The run function:

1. Validate `--project-name` is non-empty
2. Resolve connection and client
3. Call `scope.GetProjectSubjectNames(ctx, projectName)` — returns `[]string`
4. Print the result

**Table formatting for `[]string`:**

```go
func subjectNameHeaders() []string {
    return []string{"SUBJECT NAME"}
}

func subjectNameRows(names []string) [][]string {
    rows := make([][]string, len(names))
    for i, name := range names {
        rows[i] = []string{name}
    }

    return rows
}
```

For JSON output, pass the `[]string` through unchanged.

Register `subjects` as a child subcommand of `activity-event`, not as a sibling under `get`:

### 2. Register commands in `commands.go`

```go
// get
activityEvent := newGetActivityEventCmd()
activityEvent.AddCommand(newGetActivityEventSubjectsCmd())
get.AddCommand(activityEvent)
```

### 3. Create `internal/cli/activity_event_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetActivityEventCmd_Flags`** — Verify `--user`, `--object-type`, `--event`, `--project-name`, `--limit`, `--offset`, `--sort`, `--order` flags. No `--project`.

2. **`TestNewGetActivityEventCmd_NoArgs`** — Verify `cobra.NoArgs`.

3. **`TestNewGetActivityEventCmd_DefaultValues`** — Verify default values: limit=20, offset=0, sort="createdAt", order="DESC".

4. **`TestNewGetActivityEventSubjectsCmd_Flags`** — Verify `--project-name`. No `--project`.

5. **`TestSubjectNameRows_WithData`** — Verify row output for a sample `[]string`.

6. **`TestSubjectNameRows_Empty`** — Verify empty slice produces empty rows.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Instance-scoped, not project-scoped.** Activity events are a cross-project audit trail. No `--project` flag.
- **`--project-name` is a search filter, not a scope.** This is different from `--project` / `-p` which selects the project scope for API calls.
- **Offset-based pagination.** Activity events use offset/limit, not page-number pagination. Do not reuse `registerListFlags`.
- **Inline filter flags mapped to client helpers.** The client provides `EventFilterByUser`, `EventFilterByObjectType`, etc. Mapping these to `--user`, `--object-type`, `--event`, `--project-name` is natural. Note that `--event` (not `--event-name`) matches the settled UX from PR 17.
- **The public CLI shape should show pagination controls explicitly.** PR 17 freezes not just the filter names but also the offset-based pagination and sort defaults (`--limit`, `--offset`, `--sort`, `--order`). Keep those visible in the shape/examples so implementers do not treat them as secondary details.
- **Keep the `ActivityEventResource` formatter local in this PR.** PR 32 formats the same type for `rp get activity --item`, but PR 33 should not depend on PR 32 if the specialized wave is meant to stay parallel. Match the same table columns so PR 36 can deduplicate later if desired.
- **Reuse shared helpers.** Use `mustGetStringFlag`, `formatStringPtr`.

## What NOT to do

- **Do not add `--project` / `-p`.** Activity events are instance-scoped.
- **Do not confuse `--project-name` with `--project`.** They serve different purposes.
- **Do not reuse `registerListFlags`** — activity events use offset-based pagination.
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:

- `rp get --help` shows `activity-event`
- `rp get activity-event --help` shows filter and pagination flags, no `--project`
- `rp get activity-event subjects --help` shows `--project-name`

## Definition of done

- [ ] `internal/cli/activity_event.go` created
- [ ] `internal/cli/activity_event_test.go` created
- [ ] `activity-event` registered under `get` with nested `subjects` subcommand
- [ ] No `--project` / `-p` flag (instance-scoped)
- [ ] Search filters mapped to client helper functions
- [ ] Offset-based pagination with validation
- [ ] `--order` validated as ASC or DESC
- [ ] Subject names command requires `--project-name`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
