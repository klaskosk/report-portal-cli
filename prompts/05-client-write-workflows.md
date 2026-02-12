# Prompt 05: Client Write Workflows for Launches, Items, and Logs (Milestone 6)

## Goal

Expand the Go client's mutation surface in `pkg/client` to cover the remaining lifecycle and write endpoints for launches, test items, and logs. The client already has `Start`/`Finish` for launches and `StartRoot`/`StartChild`/`Finish` for test items — this prompt adds **update, delete, merge, force-finish, analyze, and log creation** operations. No new CLI commands are added in this slice; CLI write verbs are a follow-up.

## Context

The client has a working read model (List, GetByID, GetByUUID) and basic lifecycle (Start, Finish) for launches and items. This prompt fills out the remaining write operations that power automation workflows:

- **Launches**: update attributes/description, delete, force-finish (for stuck launches), merge, bulk operations, and auto-analysis trigger
- **Test items**: update (attributes, defect type), delete, bulk defect updates
- **Logs**: create log entries (single and batch), delete

The existing pattern is: thin wrapper per endpoint with structured logging, `handleResponse` for error normalization, and black-box unit tests.

### Already implemented methods

| Scope | Method | Generated API endpoint |
|---|---|---|
| `LaunchScope` | `Start(ctx, StartLaunchRQ)` | `StartLaunchUsingPOST1` |
| `LaunchScope` | `Finish(ctx, launchID, FinishExecutionRQ)` | `FinishLaunchUsingPUT1` |
| `TestItemScope` | `StartRoot(ctx, StartTestItemRQ)` | `StartRootItemUsingPOST1` |
| `TestItemScope` | `StartChild(ctx, parentItemID, StartTestItemRQ)` | `StartChildItemUsingPOST1` |
| `TestItemScope` | `Finish(ctx, itemID, FinishTestItemRQ)` | `FinishTestItemUsingPUT1` |

### Generated API methods available (not yet wrapped)

#### Launch mutations

| Generated method | What it does | Request type |
|---|---|---|
| `UpdateLaunchUsingPUTWithResponse(ctx, projectName, launchId int64, UpdateLaunchRQ)` | Update a single launch's attributes/description/mode | `UpdateLaunchRQ` → `OperationCompletionRS` |
| `DeleteLaunchUsingDELETEWithResponse(ctx, projectName, launchId int64)` | Delete a single launch by numeric ID | — → `OperationCompletionRS` |
| `DeleteLaunchesUsingDELETEWithResponse(ctx, projectName, DeleteBulkRQ)` | Bulk delete launches by IDs | `DeleteBulkRQ` → `DeleteBulkRS` |
| `ForceFinishLaunchUsingPUTWithResponse(ctx, projectName, launchId int64, FinishExecutionRQ)` | Force-finish a stuck launch | `FinishExecutionRQ` → `OperationCompletionRS` |
| `BulkForceFinishUsingPUTWithResponse(ctx, projectName, BulkRQLongFinishExecutionRQ)` | Bulk force-finish launches | `BulkRQLongFinishExecutionRQ` → `OperationCompletionRS` |
| `MergeLaunchesUsingPOST1WithResponse(ctx, projectName, MergeLaunchesRQ)` | Merge multiple launches into one | `MergeLaunchesRQ` → `LaunchResource` |
| `BulkUpdateUsingPUTWithResponse(ctx, projectName, BulkInfoUpdateRQ)` | Bulk update launch attributes/description | `BulkInfoUpdateRQ` → `OperationCompletionRS` |
| `StartLaunchAnalyzerUsingPOSTWithResponse(ctx, projectName, AnalyzeLaunchRQ)` | Trigger auto-analysis on a launch | `AnalyzeLaunchRQ` → `OperationCompletionRS` |

#### Test item mutations

| Generated method | What it does | Request type |
|---|---|---|
| `UpdateTestItemUsingPUTWithResponse(ctx, projectName, itemId int64, UpdateTestItemRQ)` | Update a single test item's attributes/description/defect | `UpdateTestItemRQ` → `OperationCompletionRS` |
| `DeleteTestItemUsingDELETEWithResponse(ctx, projectName, itemId int64)` | Delete a single test item by numeric ID | — → `OperationCompletionRS` |
| `DeleteTestItemsUsingDELETEWithResponse(ctx, projectName, DeleteTestItemsUsingDELETEParams)` | Bulk delete test items by IDs (query param) | query params → `OperationCompletionRS` |
| `DefineTestItemIssueTypeUsingPUTWithResponse(ctx, projectName, DefineIssueRQ)` | Bulk-update defect type on multiple items | `DefineIssueRQ` → `[]Issue` |
| `BulkUpdateUsingPUT1WithResponse(ctx, projectName, BulkInfoUpdateRQ)` | Bulk update test item attributes/description | `BulkInfoUpdateRQ` → `OperationCompletionRS` |

#### Log mutations

| Generated method | What it does | Request type |
|---|---|---|
| `CreateLogEntryUsingPOST1WithResponse(ctx, projectName, SaveLogRQ)` | Create a single log entry (JSON body) | `SaveLogRQ` → `EntryCreatedRS` |
| `CreateLogUsingPOST1WithBody(ctx, projectName, contentType, body)` | Batch log creation with file attachments (multipart) | multipart → `BatchSaveOperatingRS` |
| `DeleteLogUsingDELETEWithResponse(ctx, projectName, logId int64)` | Delete a single log entry by numeric ID | — → `OperationCompletionRS` |

### Key request/response types (from `pkg/types/types.gen.go`)

- `UpdateLaunchRQ`: `Attributes *[]ItemAttributesRQ`, `Description *string`, `Mode *UpdateLaunchRQMode`
- `UpdateTestItemRQ`: `Attributes *[]ItemAttributesRQ`, `Description *string`, `Issue *Issue`, `Status *string`
- `DeleteBulkRQ`: `Ids *[]int64`
- `DeleteBulkRS`: `Errors *[]ErrorRS`, `NotFound *[]int64`, `SuccessfullyDeleted *[]int64`
- `MergeLaunchesRQ`: complex — has `Attributes`, `Description`, `EndTime`, `ExtendSuitesDescription`, `Launches` ([]int64), `MergeType`, `Mode`, `Name`, `StartTime`, `Tags`
- `AnalyzeLaunchRQ`: `AnalyzeItemsMode`, `AnalyzerMode`, `AnalyzerTypeName`, `LaunchId`
- `BulkInfoUpdateRQ`: `Attributes *[]UpdateItemAttributeRQ`, `Description *Description`, `Ids *[]int64`
- `BulkRQLongFinishExecutionRQ`: `Entities map[string]FinishExecutionRQ`
- `DefineIssueRQ`: `Issues []IssueDefinition`
- `SaveLogRQ`: `File *File`, `ItemUuid *string`, `LaunchUuid string`, `Level *SaveLogRQLevel`, `Message *string`, `Time EpochMillis`, `Uuid *string`
- `EntryCreatedRS`: `Id *int64`
- `OperationCompletionRS`: `Message *string`

## Client requirements

### LaunchScope additions (add to `pkg/client/launch.go`)

Add these methods to the existing `LaunchScope`:

1. **`Update(ctx, launchID int64, request UpdateLaunchRQ) (*types.OperationCompletionRS, error)`** — update a single launch
2. **`Delete(ctx, launchID int64) (*types.OperationCompletionRS, error)`** — delete a single launch
3. **`DeleteBulk(ctx, ids []int64) (*types.DeleteBulkRS, error)`** — bulk delete launches
4. **`ForceFinish(ctx, launchID int64, request FinishExecutionRQ) (*types.OperationCompletionRS, error)`** — force-finish a stuck launch
5. **`Merge(ctx, request MergeLaunchesRQ) (*types.LaunchResource, error)`** — merge multiple launches
6. **`Analyze(ctx, request AnalyzeLaunchRQ) (*types.OperationCompletionRS, error)`** — trigger auto-analysis

Optional (lower priority):
- `BulkUpdate(ctx, request BulkInfoUpdateRQ) (*types.OperationCompletionRS, error)` — bulk update attributes/description
- `BulkForceFinish(ctx, request BulkRQLongFinishExecutionRQ) (*types.OperationCompletionRS, error)` — bulk force-finish

### TestItemScope additions (add to `pkg/client/item.go`)

1. **`Update(ctx, itemID int64, request UpdateTestItemRQ) (*types.OperationCompletionRS, error)`** — update a single item
2. **`Delete(ctx, itemID int64) (*types.OperationCompletionRS, error)`** — delete a single item
3. **`DefineIssueType(ctx, request DefineIssueRQ) ([]types.Issue, error)`** — bulk-update defect classification

Optional (lower priority):
- `DeleteBulk(ctx, ids []int64) (*types.OperationCompletionRS, error)` — bulk delete items
- `BulkUpdate(ctx, request BulkInfoUpdateRQ) (*types.OperationCompletionRS, error)` — bulk update attributes/description

### LogScope additions (add to `pkg/client/log.go`)

1. **`Create(ctx, request SaveLogRQ) (*types.EntryCreatedRS, error)`** — create a single log entry (JSON body)
2. **`Delete(ctx, logID int64) (*types.OperationCompletionRS, error)`** — delete a single log entry

Optional (lower priority):
- Batch log creation with file attachments (multipart) — this is significantly more complex. Only implement if you can cleanly abstract the multipart encoding. Consider deferring to a follow-up PR.

### Implementation patterns

Follow the exact patterns from the existing `Start`/`Finish` methods:

```go
func (l *LaunchScope) Update(
    ctx context.Context,
    launchID int64,
    request types.UpdateLaunchRQ,
) (*types.OperationCompletionRS, error) {
    logger := l.project.operationLogger("UpdateLaunch", slog.Int64("launchId", launchID))
    logger.InfoContext(ctx, "API request")

    response, err := l.project.client.raw.UpdateLaunchUsingPUTWithResponse(
        ctx,
        l.project.projectName,
        launchID,
        request,
    )
    if err != nil {
        return nil, fmt.Errorf("update launch: %w", err)
    }

    logger.DebugContext(ctx, "API response", "status", response.StatusCode())

    return handleResponse(response, "update launch", response.JSONDefault, response.JSON200)
}
```

Key patterns:
- Operation logger with relevant identifiers
- `InfoContext` before the call, `DebugContext` after
- `fmt.Errorf("verb noun: %w", err)` for transport errors
- `handleResponse` with operation name, `JSONDefault`, and success field
- Success status may be `JSON200` or `JSON201` depending on endpoint — check the generated response struct

## Client tests

### `pkg/client/launch_test.go` (extend existing file)

Add test cases for all new launch methods. Follow the existing test patterns in this file:

- **Update**: success (200), not found (404), unauthorized (401)
- **Delete**: success (200), not found (404), unauthorized (401)
- **DeleteBulk**: success (200) with partial failures (some IDs not found), unauthorized
- **ForceFinish**: success (200), not found (404)
- **Merge**: success (200), bad request (400 — e.g. invalid merge configuration)
- **Analyze**: success (200), not found (404)

### `pkg/client/item_test.go` (extend existing file)

- **Update**: success (200), not found (404), unauthorized (401)
- **Delete**: success (200), not found (404)
- **DefineIssueType**: success (200), bad request (400)

### `pkg/client/log_test.go` (extend existing file)

- **Create**: success (201), bad request (400 — missing required fields), unauthorized (401)
- **Delete**: success (200), not found (404)

### Test patterns

All tests must:
- Use `t.Parallel()`
- Be in `package client_test` (black-box)
- Use `httptest.Server` to stub API responses
- Assert the correct HTTP method, path, and request body are sent
- Verify bearer token is forwarded (`assertBearerToken`)
- Verify the correct path is called (`assertPath`)
- Use table-driven subtests where there are multiple error scenarios

### `export_test.go`

If any new unexported helpers are needed for tests, expose them in `pkg/client/export_test.go`. The existing write methods (`Start`, `Finish`, `StartRoot`, `StartChild`) don't require special exports, so you likely won't need new ones here.

## Integration tests

### `test/integration/client_launch_test.go` (extend)

Add a new `Context("write operations")` or `Describe` block:

- **Update**: start a launch, finish it, update its description/attributes, verify the change by re-reading
- **Delete**: start+finish a launch, delete it, verify `IsNotFound` on re-read
- **ForceFinish**: start a launch (don't finish it), force-finish it, verify it's finished
- **Merge**: start+finish two launches, merge them, verify merged launch exists with expected name

### `test/integration/client_item_test.go` (extend)

- **Update**: seed items, update an item's description, verify by re-reading
- **Delete**: seed items, delete one, verify `IsNotFound`

### `test/integration/client_log_test.go` (extend)

- **Create**: seed a launch+item, create a log entry via `LogScope.Create`, verify it appears in `List`
- **Delete**: create a log, delete it, verify `IsNotFound`

### Seeding notes

- Write tests need their own isolated launches/items to avoid interference with read tests. Create unique project names or launch names per write test context.
- Use `BeforeAll` or `BeforeEach` in Ginkgo ordered containers to seed data, and clean up in `AfterAll` where possible.
- The existing `seedTestLaunches` and `seedTestItems` helpers in `helpers_test.go` can be reused, but write tests may need to leave launches in "started" state (not finished) to test force-finish.

## Design considerations

- **No CLI in this PR**: This prompt is client-only. CLI write commands (`rp start launch`, `rp finish launch`, `rp log add`, etc.) will come in a follow-up prompt after the client surface is stable.
- **Bulk operations**: Bulk methods (`DeleteBulk`, `BulkUpdate`, `BulkForceFinish`) are optional/lower-priority. Implement them if time allows, but the single-entity methods are the priority.
- **Multipart log creation**: The batch log creation endpoint (`CreateLogUsingPOST1`) uses multipart encoding for file attachments. This is complex. Implement `Create` (JSON single log) first. Batch/multipart can be a separate follow-up.
- **Response type differences**: Some write endpoints return `JSON200`, others return `JSON201`. Check each generated response struct's fields. For example, `StartLaunchUsingPOST1` returns `JSON201`, while `UpdateLaunchUsingPUT` returns `JSON200`.
- **`launchID` parameter types**: The generated API uses `string` for UUID-based IDs (used in Finish, which takes a launch UUID) and `int64` for numeric IDs (used in Update, Delete). Make sure wrapper method signatures match what the generated client expects.

## Definition of done

- [ ] `LaunchScope.Update`, `Delete`, `DeleteBulk`, `ForceFinish`, `Merge`, `Analyze` implemented
- [ ] `TestItemScope.Update`, `Delete`, `DefineIssueType` implemented
- [ ] `LogScope.Create`, `Delete` implemented
- [ ] Unit tests for all new methods in `pkg/client/*_test.go`
- [ ] Integration tests for key write workflows in `test/integration/`
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-integration` passes
