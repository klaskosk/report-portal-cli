# Prompt 21: Instance-Level BTS Completion (Backlog PR 15)

## Goal

Add a new top-level `InstanceBugTrackingScope` on `Client` that wraps the two instance-scoped BTS endpoints: system field retrieval and allowable issue type listing, both keyed by integration ID without a project name.

## Context

The existing `BugTrackingScope` lives under `ProjectScope` and wraps the **project-scoped** BTS endpoints (`/v1/bts/{projectName}/{integrationId}/...`). The OpenAPI spec also defines two **instance-scoped** variants at `/v1/bts/{integrationId}/...` (without `projectName`). These are used when callers need to query BTS metadata for a global integration that isn't tied to a specific project.

This is a small, focused PR — only two endpoints — but it requires a new top-level scope on `Client`, which touches `client.go`. Coordinate with PR 9 (GroupsScope) to avoid merge conflicts if both PRs are in flight.

## Endpoints to wrap

| HTTP | Path | operationId | Generated method | Response type |
|------|------|-------------|-----------------|---------------|
| GET | `/v1/bts/{integrationId}/fields-set` | `getSetOfIntegrationSystemFields_1` | `GetSetOfIntegrationSystemFields1WithResponse` | `[]PostFormField` (200) |
| GET | `/v1/bts/{integrationId}/issue_types` | `getAllowableIssueTypes_1` | `GetAllowableIssueTypes1WithResponse` | `[]string` (200) |

### Parameter details

**`fields-set`**: The generated params type is `GetSetOfIntegrationSystemFields1Params`. Check its fields — the spec marks `issueType` as a required query parameter. The project-scoped variant (`BugTrackingScope.GetSystemFields`) currently passes an empty params struct, which means it omits `issueType`. For the instance-scoped version, decide whether to:

- Accept `issueType` as a required parameter (matching the spec)
- Accept it as optional (matching the project-scoped precedent)

Prefer matching the spec and accepting it as a required parameter, since this is a new method with no backward compatibility concerns.

**`issue_types`**: No query parameters. Only the `integrationId` path parameter (int64).

## Key types (from `pkg/types`)

- `PostFormField` — BTS form field descriptor with `Id` (string), `FieldName` (*string), `FieldType` (*string), `Required` (*bool), `Value` (*[]string), `AllowedValues` (*[]AllowedValue), etc.
- The issue types response is a plain `[]string` — a list of issue type names like `"Bug"`, `"Task"`, `"Story"`.

## Implementation plan

### 1) Check generated types

Read the generated code in `pkg/internal/api/api.gen.go` to confirm:

- `GetSetOfIntegrationSystemFields1Params` — what fields does it have? Confirm `IssueType` is present.
- `GetSetOfIntegrationSystemFields1Response` — confirm `JSON200 *[]PostFormField` and `JSONDefault *ErrorRS` fields exist.
- `GetAllowableIssueTypes1Response` — confirm `JSON200 *[]string` and `JSONDefault *ErrorRS` fields exist.

### 2) Create `pkg/client/instance_bts.go`

Define the scope:

```go
type InstanceBugTrackingScope struct {
    client *Client
}
```

Methods to implement:

- `GetSystemFields(ctx, integrationID int64, issueType string) ([]types.PostFormField, error)` — wraps `GetSetOfIntegrationSystemFields1WithResponse`. Pass `issueType` via the params struct. Return `[]types.PostFormField{}` (not nil) when the result is nil/empty.
- `GetAllowableIssueTypes(ctx, integrationID int64) ([]string, error)` — wraps `GetAllowableIssueTypes1WithResponse`. Return `[]string{}` (not nil) when the result is nil/empty.

Follow the standard instance-level scope pattern (like `IntegrationsScope`):

```go
func (s *InstanceBugTrackingScope) GetSystemFields(
    ctx context.Context,
    integrationID int64,
    issueType string,
) ([]types.PostFormField, error) {
    logger := s.client.logger.With(
        slog.String("operation", "GetInstanceBTSSystemFields"),
        slog.Int64("integrationId", integrationID),
        slog.String("issueType", issueType),
    )
    logger.InfoContext(ctx, "API request")

    response, err := s.client.raw.GetSetOfIntegrationSystemFields1WithResponse(
        ctx,
        integrationID,
        &api.GetSetOfIntegrationSystemFields1Params{
            IssueType: issueType,
        },
    )
    if err != nil {
        return nil, fmt.Errorf("get instance bts system fields: %w", err)
    }

    logger.DebugContext(ctx, "API response", "status", response.StatusCode())

    result, err := handleResponse(response, response.JSONDefault, response.JSON200, response.Body)
    if err != nil {
        return nil, fmt.Errorf("get instance bts system fields: %w", err)
    }

    if result == nil {
        return []types.PostFormField{}, nil
    }

    return *result, nil
}
```

### 3) Add accessor on `Client`

In `pkg/client/client.go`, add:

```go
// InstanceBugTracking returns an InstanceBugTrackingScope for instance-level BTS operations.
func (c *Client) InstanceBugTracking() *InstanceBugTrackingScope {
    return &InstanceBugTrackingScope{client: c}
}
```

Place it near the other top-level scope accessors. The name `InstanceBugTracking` distinguishes it from the project-scoped `BugTracking()` on `ProjectScope`.

### 4) Add test constructor in `export_test.go`

```go
var NewTestInstanceBugTrackingScope = func(c *Client) *InstanceBugTrackingScope {
    return c.InstanceBugTracking()
}
```

## Test requirements

### Unit tests

Create `pkg/client/instance_bts_test.go` in package `client_test`.

For **each** method:

**`GetSystemFields`:**

- **Success**: Handler asserts path `/api/v1/bts/{integrationId}/fields-set`, method GET, bearer token, and `issueType` query parameter. Return JSON array of `PostFormField` objects. Assert parsed field count and field properties.
- **Empty result**: Return `[]` or `null`. Assert empty slice returned.
- **Not found**: Return 404 with error body. Assert `client.IsNotFound`.
- **Unauthorized**: Return 401. Assert `client.IsUnauthorized`.

**`GetAllowableIssueTypes`:**

- **Success**: Handler asserts path `/api/v1/bts/{integrationId}/issue_types`, method GET, bearer token. Return `["Bug", "Task", "Story"]`. Assert parsed values.
- **Empty result**: Return `[]`. Assert empty slice.
- **Not found**: Return 404. Assert `client.IsNotFound`.

Expected paths to assert:

- `/api/v1/bts/1/fields-set?issueType=Bug` (fields — note the query param)
- `/api/v1/bts/1/issue_types` (issue types)

### Comparison with project-scoped tests

The existing tests in `bts_test.go` test paths like `/api/v1/bts/my_project/1/fields-set`. Instance-scoped tests should use `/api/v1/bts/1/fields-set` (no project name segment).

Use `t.Parallel()` on every test. Use `newTestClient`, `assertBearerToken`, `assertMethod`, `assertPath`.

## Files to create or modify

| File | Action |
|------|--------|
| `pkg/client/instance_bts.go` | Create — `InstanceBugTrackingScope` with two methods |
| `pkg/client/instance_bts_test.go` | Create — unit tests for both methods |
| `pkg/client/client.go` | Modify — add `InstanceBugTracking()` accessor |
| `pkg/client/export_test.go` | Modify — add test constructor |

## Design considerations

- **Naming**: `InstanceBugTrackingScope` makes the scope's level clear and avoids confusion with the project-scoped `BugTrackingScope`. The accessor `Client.InstanceBugTracking()` follows the pattern of `Client.Integrations()` for instance-level scopes.
- **`issueType` parameter**: The spec marks it as required on the fields-set endpoint. Accept it as a required `string` parameter rather than making it optional. This is safer and matches the spec. The project-scoped version passes an empty struct — that may be a latent bug, but fixing it is out of scope for this PR.
- **Merge conflicts**: This PR adds a method to `client.go`. If PR 9 (GroupsScope) also modifies `client.go`, coordinate the merge order or resolve conflicts.
- **Keep it small**: This PR is intentionally bounded to just two endpoints. Don't expand scope to include project-scoped BTS fixes or other BTS operations.

## Validation commands

Run before finalizing:

- `make lint`
- `make test`

## Definition of done

- [ ] `InstanceBugTrackingScope` implemented with `GetSystemFields` and `GetAllowableIssueTypes`
- [ ] `Client.InstanceBugTracking()` accessor added to `client.go`
- [ ] Test constructor added to `export_test.go`
- [ ] Unit tests for both methods with success, empty, and error cases
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
