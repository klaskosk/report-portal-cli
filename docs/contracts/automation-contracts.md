# Automation Contracts

This document defines the stability guarantees for automation-facing behavior
in the Report Portal CLI and client library. Breaking these contracts requires
a major version bump and migration guidance.

## JSON output stability

### Field names

JSON output field names are derived directly from the generated API types in
`pkg/types/types.gen.go`. The CLI serializes API response structs with
`encoding/json` using the struct tags defined in the generated code.

Field names use **camelCase** and match the Report Portal REST API response
schema. Examples: `id`, `uuid`, `name`, `status`, `startTime`, `endTime`,
`launchId`, `projectName`, `userId`.

Renaming a JSON field in CLI output is a breaking change.

### omitempty behavior

Optional fields carry `omitempty` in their JSON struct tags. This means:

- Fields with zero values (empty strings, nil pointers, zero integers) may be
  absent from JSON output rather than present as `null` or `""`.
- Automation consumers must treat absent fields as equivalent to their zero
  value.
- Adding `omitempty` to a previously always-present field, or removing it, is a
  breaking change to the output contract.

### Output structure by command

| Command | List output | Single-resource output |
|---|---|---|
| `rp get launch` | JSON array of launch objects | JSON object (single launch) |
| `rp get item` | JSON array of item objects | JSON object (single item) |
| `rp get log` | JSON array of log objects | JSON object (single log) |
| `rp get project` | JSON array of project name strings | JSON object (project details) |
| `rp whoami` | n/a (always single) | JSON object (user resource) |

### Contracted field sets

The following fields are guaranteed present (non-nil, non-zero) in JSON output
for successfully retrieved resources. Fields not listed here may appear but are
not part of the stability contract.

**Launch** (`rp get launch -o json`):

- `id` (int64): numeric launch identifier
- `uuid` (string): unique launch identifier
- `name` (string): launch name
- `status` (string): execution status
- `startTime` (int64): epoch milliseconds
- `number` (int64): launch sequence number

**Test item** (`rp get item -o json`):

- `id` (int64): numeric item identifier
- `uuid` (string): unique item identifier
- `name` (string): item name
- `type` (string): item type (SUITE, TEST, STEP, etc.)
- `status` (string): execution status
- `startTime` (int64): epoch milliseconds
- `launchId` (int64): parent launch identifier

**Log** (`rp get log -o json`):

- `id` (int64): numeric log identifier
- `uuid` (string): unique log identifier

**Project** (`rp get project <name> -o json`):

- `projectName` (string): project identifier

**User** (`rp whoami -o json`):

- `userId` (string): login name
- `email` (string): email address

## Error output stability

### Error format

All CLI commands follow the same error output conventions:

- **Table/template mode**: errors are written to stderr as
  `Error: <message>\n`.
- **JSON mode** (`-o json`): errors are written to stderr as a JSON object
  with a single `error` key: `{"error": "<message>"}`. Stdout remains empty.
- Exit code is non-zero for any error.

### Error JSON schema

```json
{
  "error": "string — human-readable error message"
}
```

This schema is stable. Additional keys will not be added without a major
version bump.

## Client error mapping

### Error types

The client library (`pkg/client`) returns two error types:

- `*APIError`: the server returned a JSON error body (`JSONDefault`). Exposes
  `StatusCode()`, `ErrorCode()`, and `Message()`.
- `*UnexpectedStatusError`: the server returned a non-2xx status without a
  parseable JSON error body (e.g. proxy HTML page). Exposes `StatusCode()`.

Both types implement the `error` interface and the internal `statusCoder`
interface, which allows `HasStatusCode` to match either type uniformly.

These types represent the **cause** of an error (the HTTP failure itself).
Operation context (what the caller was trying to do) is added by each API
method via `fmt.Errorf("operation: %w", err)`. Predicates like `IsNotFound`
use `errors.As` internally, so they work correctly through any number of
wrapping layers.

### Error predicates

| Predicate | HTTP status | Behavior |
|---|---|---|
| `IsNotFound(err)` | 404 | Matches both `APIError` and `UnexpectedStatusError`. |
| `IsUnauthorized(err)` | 401 | Matches both error types. |
| `IsForbidden(err)` | 403 | Matches both error types. |
| `IsBadRequest(err)` | 400 | Matches both error types. |
| `IsConflict(err)` | 409 | Matches both error types. |
| `HasStatusCode(err, code)` | any | General-purpose escape hatch; matches both error types. |
| `HasErrorCode(err, code)` | n/a | Matches `APIError` only (RP application error code). |

All predicates use `errors.As` internally, so they work correctly with wrapped
errors (`fmt.Errorf("context: %w", err)`).

### Predicate stability guarantees

- The named predicates (`IsNotFound`, etc.) will not be removed or have their
  status code mapping changed.
- `HasStatusCode` and `HasErrorCode` remain the escape hatches for codes not
  covered by a named predicate.
- Predicates are safe to call with `nil` — they return `false`.
- Predicates are safe to call with non-API errors — they return `false`.

### APIError message format

`APIError.Error()` returns a cause-only string in one of two formats:

- With response body: `"HTTP <status>: [<errorCode>] <message>"`
- Without response body: `"HTTP <status>"`

When returned from an API method, the error is wrapped with operation context,
producing strings like `"get launch by id: HTTP 404: [40410] Launch not found"`.

This format is informational and should not be parsed by automation. Use the
accessor methods and predicate functions instead.

## Scope accessor naming

### Instance-level scopes

| Accessor | Scope type | Purpose |
|---|---|---|
| `Client.Users()` | `*UsersScope` | User operations (GetMyself, GetByLogin, ListProjects). |
| `Client.Projects()` | `*ProjectsScope` | Project CRUD and user assignment. |
| `Client.Integrations()` | `*IntegrationsScope` | Global integration management. |
| `Client.Settings()` | `*SettingsScope` | Server settings. |
| `Client.Plugins()` | `*PluginScope` | Plugin management. |
| `Client.Onboarding()` | `*OnboardingScope` | First-run onboarding status. |
| `Client.ActivityEvents()` | `*ActivityEventsScope` | Instance-wide activity event search. |

### Project-level scopes

Accessed via `Client.Project(name)` which returns `*ProjectScope`:

| Accessor | Scope type | Purpose |
|---|---|---|
| `.Launches()` | `*LaunchScope` | Launch lifecycle and queries. |
| `.Items()` | `*TestItemScope` | Test item lifecycle and queries. |
| `.Logs()` | `*LogScope` | Log entry management. |
| `.Dashboards()` | `*DashboardScope` | Dashboard CRUD. |
| `.Widgets()` | `*WidgetScope` | Widget CRUD and data retrieval. |
| `.Filters()` | `*UserFilterScope` | Saved filter CRUD. |
| `.Integrations()` | `*ProjectIntegrationsScope` | Project-level integration management. |
| `.BugTracking()` | `*BugTrackingScope` | BTS ticket and issue operations. |
| `.Activities()` | `*ActivityScope` | Activity audit log queries. |
| `.Settings()` | `*ProjectSettingsScope` | Project settings, notifications, patterns. |
| `.DemoData()` | `*DemoDataScope` | Demo data generation. |
| `.DownloadFile()` | (direct method) | File/attachment download by data ID. |

### Naming guarantees

- Scope type names and accessor method names are part of the public API and
  will not be renamed without a major version bump.
- The `Client.Project(name)` pattern for obtaining project-scoped operations
  is stable.

## Contract test coverage

The following test suites lock the contracts described above:

| Contract | Test location | What it verifies |
|---|---|---|
| Error predicates | `pkg/client/errors_test.go` | All predicates match correct status codes, reject others, handle wrapping and nil. |
| Error type accessors | `pkg/client/errors_test.go` | `APIError` and `UnexpectedStatusError` expose stable accessor methods. |
| Response handling | `pkg/client/response_test.go` | `handleResponse` returns correct error types for JSON and non-JSON error bodies. |
| Error contract stability | `pkg/client/contract_test.go` | Cross-predicate isolation, extended status codes, interface conformance. |
| Launch JSON output | `test/integration/cli_launch_test.go` | Field presence and types for list and single-launch JSON. |
| Item JSON output | `test/integration/cli_item_test.go` | Field presence and types for list and single-item JSON. |
| Log JSON output | `test/integration/cli_log_test.go` | Field presence and types for list and single-log JSON. |
| Project JSON output | `test/integration/cli_project_test.go` | Project name list and detail JSON structure. |
| Whoami JSON output | `test/integration/cli_whoami_test.go` | User resource field presence. |
| JSON error format | `test/integration/cli_*_test.go` | `{"error": "..."}` format on stderr for all commands. |
| CLI contract stability | `test/integration/contract_test.go` | Comprehensive field set verification, value type stability, and error format for all CLI commands. |
| Client error predicates (integration) | `test/integration/client_errors_test.go` | Predicates against live server: IsNotFound, IsUnauthorized, HasStatusCode, exclusivity, wrapping. |
