# Prompt 01: Log Read Surface (Client + CLI)

## Goal

Add a `LogScope` to the Go client and a `rp get log` CLI command that lets users list and retrieve logs for test items and launches. This is the first half of Milestone 4 (Logs + Attachments).

## Context

The codebase already has two scope implementations to use as patterns:

- `pkg/client/launch.go` — `LaunchScope` with `List`, `GetByID`, `GetByUUID`, and filter options
- `pkg/client/item.go` — `TestItemScope` with `List`, `GetByID`, `GetByUUID`, `GetByIDs`, and filter options

The CLI commands `rp get launch` and `rp get item` in `internal/cli/` follow a consistent pattern: resolve connection, resolve project, dispatch to list vs get-by-id/uuid based on positional args, and print via the shared `Printer`.

Shared infrastructure already exists:

- `pkg/client/response.go` — `handleResponse` generic helper, `operationLogger`
- `pkg/client/errors.go` — `APIError` + predicate functions (`IsNotFound`, etc.)
- `internal/cli/listflags.go` — shared `--page`, `--page-size`, `--sort` flag registration
- `internal/cli/output.go` — `Printer` with table/json/go-template support

## Generated API endpoints available

The generated client in `pkg/internal/api/api.gen.go` already has these log-related methods:

| Generated method | What it does |
|---|---|
| `GetLogsUsingGET1WithResponse` | List logs with filters (item, launch, level, message, pagination) |
| `GetLogUsingGETWithResponse` | Get a single log by numeric ID |
| `GetLogByUuidUsingGETWithResponse` | Get a single log by UUID (deprecated in API v6.0, but current for v5.11) |

The `GetLogsUsingGETParams` struct has these filter fields: `FilterEqItem` (int64), `FilterEqLaunch` (int64), `FilterEqLaunchId` (int64), `FilterEqLevel` (string), `FilterEqMessage` (string), `FilterEqAutoAnalyzed` (bool), plus standard pagination (`PagePage`, `PageSize`, `PageSort`).

The response types are `IterableLogResource` (paginated list) and `LogResource` (single log).

`LogResource` has fields: `id`, `uuid`, `itemId`, `launchId`, `level` (enum: error/warn/info/debug/trace/fatal/unknown), `message`, `binaryContent` (attachment metadata), `thumbnail`.

## Client scope requirements

Create `pkg/client/log.go` with a `LogScope` struct:

- `LogScope` should be accessible from `ProjectScope` via a `Logs()` method (add to `pkg/client/project.go`)
- Methods to implement:
  - `List(ctx, opts...)` — paginated list with functional options
  - `GetByID(ctx, id)` — single log by numeric ID
  - `GetByUUID(ctx, uuid)` — single log by UUID
- Functional options to expose (follow the `WithItem*` pattern from `TestItemScope`):
  - `WithLogItemID(int64)` — filter by test item
  - `WithLogLaunchID(int64)` — filter by launch
  - `WithLogLevel(string)` — filter by log level
  - `WithLogPageNumber`, `WithLogPageSize`, `WithLogSort` — pagination
- Follow the exact same patterns as `TestItemScope` for logging, error handling, and response extraction

## Client tests

Create `pkg/client/log_test.go` (black-box, `package client_test`):

- Use `httptest.Server` to stub API responses (same pattern as `launch_test.go` and `item_test.go`)
- Test cases for `List`: success with filters, empty results, 401/404 error mapping
- Test cases for `GetByID`: success, not-found, unauthorized
- Test cases for `GetByUUID`: success, not-found
- Add test-only constructor in `export_test.go` for `LogScope`
- All tests must use `t.Parallel()`

## CLI command requirements

Create the `rp get log` command in `internal/cli/log.go`:

- Register in `internal/cli/get.go` via `cmd.AddCommand(newGetLogCmd())`
- Command shape: `rp get log [id-or-uuid]`
  - No args → list logs (requires `--item` or `--launch` to scope)
  - Positional arg → get single log by ID or UUID
- Flags:
  - `--project` / `-p` — project name override
  - `--item` — filter by test item ID
  - `--launch` — filter by launch ID
  - `--level` — filter by log level (error, warn, info, debug, trace, fatal)
  - Standard list flags via `registerListFlags(cmd)`
- Table output columns: ID, Level, Item ID, Launch ID, Message (truncated), Has Attachment
- JSON output: full `LogResource` structure
- Error when listing without `--item` or `--launch` (logs are meaningless without scope)

## Integration tests

Add to `test/integration/`:

- `client_log_test.go` — client-level integration tests against the real Report Portal stack:
  - Seed a launch + item + finish them (logs are created during item/launch lifecycle)
  - `List` logs filtered by item ID
  - `GetByID` for a known log
  - `IsNotFound` for a non-existent log ID
- `cli_log_test.go` — CLI integration tests:
  - `rp get log --item <id>` in table and JSON output
  - `rp get log <id>` for a specific log
  - Error cases: missing project, invalid credentials

Integration test seeding: Review `helpers_test.go` for the existing `seedTestLaunches` and `seedTestItems` patterns. Logs are automatically created when test items report results, so seeding items with `FAILED` status should produce error-level logs. You may need to add a `seedTestLogs` helper or extend the existing seeding to explicitly create logs via the API if the auto-generated logs aren't sufficient.

## Definition of done

- [ ] `pkg/client/log.go` with `LogScope` (List, GetByID, GetByUUID)
- [ ] `pkg/client/log_test.go` with comprehensive unit tests
- [ ] `internal/cli/log.go` with `rp get log` command
- [ ] `test/integration/client_log_test.go` with real API tests
- [ ] `test/integration/cli_log_test.go` with CLI integration tests
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-integration` passes
