# Prompt 07: Write Endpoint Completion Pass (Milestone 6 Follow-up)

## Goal

Finish the remaining write endpoints that were explicitly deferred as optional in earlier slices. This prompt closes the gap between "core write flows implemented" and "write surface complete" for launches, test items, logs, and project settings.

This is a **client-only** prompt. CLI write verbs remain a separate follow-up.

## Context

Prompt 05 delivered the primary write workflows:

- Launch: `Update`, `Delete`, `DeleteBulk`, `ForceFinish`, `Merge`, `Analyze`
- Test item: `Update`, `Delete`, `DefineIssueType`
- Log: `Create`, `Delete`

Prompt 06 delivered project-settings reads:

- `ProjectSettingsScope.Get`
- `ProjectSettingsScope.GetNotifications`

The remaining work is the **deferred write subset** that was left as optional.

## Remaining generated endpoints to wrap

### Launch write endpoints (deferred in Prompt 05)

| Generated method | What it does | Request/Response |
|---|---|---|
| `BulkUpdateUsingPUTWithResponse(ctx, projectName, BulkInfoUpdateRQ)` | Bulk update launch attributes/description | `BulkInfoUpdateRQ` -> `OperationCompletionRS` |
| `BulkForceFinishUsingPUTWithResponse(ctx, projectName, BulkRQLongFinishExecutionRQ)` | Bulk force-finish launches | `BulkRQLongFinishExecutionRQ` -> `OperationCompletionRS` |

### Test item write endpoints (deferred in Prompt 05)

| Generated method | What it does | Request/Response |
|---|---|---|
| `DeleteTestItemsUsingDELETEWithResponse(ctx, projectName, params)` | Bulk delete test items by IDs (query params) | params -> `OperationCompletionRS` |
| `BulkUpdateUsingPUT1WithResponse(ctx, projectName, BulkInfoUpdateRQ)` | Bulk update test item attributes/description | `BulkInfoUpdateRQ` -> `OperationCompletionRS` |

### Log write endpoints (deferred in Prompt 05)

| Generated method | What it does | Request/Response |
|---|---|---|
| `CreateLogUsingPOST1WithBody(ctx, projectName, contentType, body)` | Batch log creation with attachments (multipart) | multipart -> `BatchSaveOperatingRS` |

### Project settings write endpoints (deferred in Prompt 06)

| Generated method | What it does | Request/Response |
|---|---|---|
| `CreateNotificationUsingPOSTWithResponse(ctx, projectName, SenderCaseDTO)` | Create notification rule | `SenderCaseDTO` -> created/result resource |
| `UpdateNotificationUsingPUTWithResponse(ctx, projectName, SenderCaseDTO)` | Update notification rule | `SenderCaseDTO` -> `OperationCompletionRS` or equivalent |
| `DeleteNotificationUsingDELETEWithResponse(ctx, projectName, id int64)` | Delete notification rule | -> `OperationCompletionRS` |
| `CreatePatternTemplateUsingPOSTWithResponse(ctx, projectName, CreatePatternTemplateRQ)` | Create pattern template | `CreatePatternTemplateRQ` -> `EntryCreatedRS` |
| `UpdatePatternTemplateUsingPUTWithResponse(ctx, projectName, id int64, UpdatePatternTemplateRQ)` | Update pattern template | `UpdatePatternTemplateRQ` -> `OperationCompletionRS` |
| `DeletePatternTemplateUsingDELETEWithResponse(ctx, projectName, id int64)` | Delete pattern template | -> `OperationCompletionRS` |
| `CreateProjectIssueSubTypeUsingPOSTWithResponse(ctx, projectName, CreateIssueSubTypeRQ)` | Create issue sub-type | `CreateIssueSubTypeRQ` -> created/result resource |
| `UpdateProjectIssueSubTypeUsingPUTWithResponse(ctx, projectName, UpdateOneIssueSubTypeRQ)` | Update issue sub-type | `UpdateOneIssueSubTypeRQ` -> `OperationCompletionRS` |
| `DeleteProjectIssueSubTypeUsingDELETEWithResponse(ctx, projectName, id int64)` | Delete issue sub-type | -> `OperationCompletionRS` |

## Client requirements

### `LaunchScope` additions (`pkg/client/launch.go`)

Add:

1. `BulkUpdate(ctx, request types.BulkInfoUpdateRQ) (*types.OperationCompletionRS, error)`
2. `BulkForceFinish(ctx, request types.BulkRQLongFinishExecutionRQ) (*types.OperationCompletionRS, error)`

Notes:

- Keep operation names and error text consistent with existing style (`fmt.Errorf("bulk update launches: %w", err)`, etc.).
- Use `handleResponse` with `response.JSONDefault` and the success field from generated response.

### `TestItemScope` additions (`pkg/client/item.go`)

Add:

1. `DeleteBulk(ctx, ids []int64) (*types.OperationCompletionRS, error)`
2. `BulkUpdate(ctx, request types.BulkInfoUpdateRQ) (*types.OperationCompletionRS, error)`

Notes:

- `DeleteTestItemsUsingDELETE` uses query params rather than JSON body. Add a small param-construction helper if it improves readability.

### `LogScope` addition (`pkg/client/log.go`)

Add:

1. `CreateBatch(ctx, entries ...BatchLogEntry) (*types.BatchSaveOperatingRS, error)` (or equivalent ergonomic API)

Where `BatchLogEntry` is a small wrapper type you define to represent:

- the log payload fields
- optional attachment bytes and filename

Requirements:

- Build multipart payload internally
- Set proper `Content-Type` with boundary
- Keep caller-facing API typed and ergonomic (avoid requiring callers to assemble raw multipart)
- Keep single-entry `Create` method unchanged

### `ProjectSettingsScope` write additions (`pkg/client/project_settings.go`)

Add methods to complete the deferred mutation surface:

1. `CreateNotification(ctx, request types.SenderCaseDTO) (...)`
2. `UpdateNotification(ctx, request types.SenderCaseDTO) (...)`
3. `DeleteNotification(ctx, id int64) (*types.OperationCompletionRS, error)`
4. `CreatePatternTemplate(ctx, request types.CreatePatternTemplateRQ) (*types.EntryCreatedRS, error)`
5. `UpdatePatternTemplate(ctx, id int64, request types.UpdatePatternTemplateRQ) (*types.OperationCompletionRS, error)`
6. `DeletePatternTemplate(ctx, id int64) (*types.OperationCompletionRS, error)`
7. `CreateIssueSubType(ctx, request types.CreateIssueSubTypeRQ) (...)`
8. `UpdateIssueSubType(ctx, request types.UpdateOneIssueSubTypeRQ) (*types.OperationCompletionRS, error)`
9. `DeleteIssueSubType(ctx, id int64) (*types.OperationCompletionRS, error)`

Use exact return types from generated response structs where creation endpoints differ.

## Implementation pattern

Follow existing wrapper style consistently:

- operation logger per method
- `InfoContext` before the API call and `DebugContext` after with status code
- transport errors wrapped as `fmt.Errorf("verb noun: %w", err)`
- typed `handleResponse(...)` usage with `JSONDefault` checked first
- nil-safe handling for list responses where appropriate

## Client tests

### `pkg/client/launch_test.go` (extend)

- **BulkUpdate**: success (200), bad request (400), unauthorized (401)
- **BulkForceFinish**: success (200), bad request (400), unauthorized (401)

### `pkg/client/item_test.go` (extend)

- **DeleteBulk**: success (200), not found/partial semantics as returned by API, unauthorized (401)
- **BulkUpdate**: success (200), bad request (400), unauthorized (401)

### `pkg/client/log_test.go` (extend)

- **CreateBatch**:
  - success with one entry (200)
  - success with attachment payload
  - bad request (400) for malformed/invalid input
  - unauthorized (401)
- Validate multipart request shape (boundary present, expected fields/files present).

### `pkg/client/project_settings_test.go` (extend)

- **Create/Update/DeleteNotification**: success + representative errors
- **Create/Update/DeletePatternTemplate**: success + representative errors
- **Create/Update/DeleteIssueSubType**: success + representative errors

Test expectations:

- all tests in `package client_test`
- `t.Parallel()` everywhere
- verify path/method/token/body mappings in handlers
- use table-driven style for error cases

## Integration tests

### Launch / item / log write completion

Extend existing integration suites:

- `test/integration/client_launch_test.go`
  - bulk update launches
  - bulk force-finish for started launches
- `test/integration/client_item_test.go`
  - bulk delete items
  - bulk update items
- `test/integration/client_log_test.go`
  - batch log creation (with and without attachment)

### Project settings write flows

Add/extend:

- `test/integration/client_project_settings_test.go`
  - notification create/update/delete lifecycle
  - pattern template create/update/delete lifecycle
  - issue sub-type create/update/delete lifecycle

Seeding and cleanup notes:

- use unique names to avoid collisions between parallel tests
- clean up created resources in `AfterEach`/`AfterAll` when deletion is not part of test flow
- keep project-settings mutation tests isolated from baseline read assertions

## Design considerations

- **Multipart complexity**: keep multipart encoding encapsulated in `LogScope`; do not leak low-level request construction to callers.
- **Bulk endpoint semantics**: preserve API response semantics exactly (do not rewrite field names or coerce server responses).
- **Error predicates**: tests should continue using `client.IsBadRequest`, `client.IsUnauthorized`, `client.IsNotFound`, etc.
- **No CLI in this slice**: CLI verbs remain follow-up work after client write surface is complete and stable.

## Definition of done

- [ ] `LaunchScope.BulkUpdate` and `LaunchScope.BulkForceFinish` implemented
- [ ] `TestItemScope.DeleteBulk` and `TestItemScope.BulkUpdate` implemented
- [ ] `LogScope.CreateBatch` (multipart batch logs) implemented
- [ ] `ProjectSettingsScope` deferred write methods implemented (notifications, pattern templates, issue sub-types)
- [ ] Unit tests added/extended for all new methods
- [ ] Integration tests added/extended for high-value write workflows
- [ ] `code-verifier` subagent says your changes are good to go
