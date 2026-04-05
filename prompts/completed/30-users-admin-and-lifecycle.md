# Prompt 19: Users Admin and Lifecycle (Backlog PR 13)

## Goal

Extend `UsersScope` in `pkg/client/users.go` with administrative CRUD, search, export, and invitation/registration flows for instance-level user management.

## Context

`UsersScope` currently wraps only three endpoints: `GetMyself`, `GetByLogin`, and `ListProjects`. The remaining user management surface — admin listing, search, editing, deletion, export, and the invitation/registration lifecycle — is fully available in the generated client but has no public wrapper. This PR adds all of those operations.

**Important**: This PR shares `users.go` and `users_test.go` with PR 14 (passwords and API keys). Do not parallelize these two PRs. PR 14 should be sequenced after this one.

## Endpoints to wrap

| HTTP | Path | operationId | Generated method | Response type |
|------|------|-------------|-----------------|---------------|
| GET | `/users/all` | `getUsers` | `GetUsersWithResponse` | `PageUserResource` (200) |
| GET | `/users/search` | `findUsers` | `FindUsersWithResponse` | `PageUserResource` (200) |
| GET | `/users/export` | `export` | `ExportWithResponse` | Raw bytes (200, `application/octet-stream` or CSV) |
| POST | `/users` | `createUserByAdmin` | `CreateUserByAdminWithResponse` | `CreateUserRS` (201) |
| PUT | `/users/{login}` | `editUser` | `EditUserWithResponse` | `OperationCompletionRS` (200) |
| DELETE | `/users/{id}` | `deleteUser` | `DeleteUserWithResponse` | `OperationCompletionRS` (200) |
| DELETE | `/users` | `deleteUsers` | `DeleteUsersWithResponse` | `DeleteBulkRS` (200) |
| POST | `/users/bid` | `createUserBid` | `CreateUserBidWithResponse` | `UserBidRS` (201) |
| GET | `/users/registration` | `getUserBidInfo` | `GetUserBidInfoWithResponse` | `UserBidRS` (200) |
| POST | `/users/registration` | `createUser` | `CreateUserWithResponse` | `CreateUserRS` (201) |

## Key types (from `pkg/types`)

**Request types:**

- `CreateUserRQFull` — admin user creation with `Login`, `FullName`, `Email`, `Password`, `AccountRole`, `AccountType`, `DefaultProject`, `ProjectRole`, etc.
- `EditUserRQ` — edit user fields: `FullName` (required), `Email`, `Role`, `Active`, `AccountType`, `ExternalId`
- `CreateUserRQ` — invitation: `Email`, `DefaultProject`, `Role`
- `CreateUserRQConfirm` — registration confirmation: `Login`, `FullName`, `Email`, `Password`

**Response types:**

- `CreateUserRS` — created user with `Id`, `Login`, `FullName`, `Email`, etc.
- `UserBidRS` — invitation bid with `Uuid`, `Email`, `IsActive`
- `OperationCompletionRS` — generic success with `Message`
- `DeleteBulkRS` — bulk delete result

**Pagination / filter params (from generated code):**

- `GetUsersParams` — check the generated struct for available query parameters (likely includes pagination and filter fields)
- `FindUsersParams` — search with `Term` query parameter
- `ExportParams` — check the generated struct for filter/format parameters
- `DeleteUsersParams` — `Ids` parameter for bulk delete

## Implementation plan

### 1) Check generated parameter types

Before implementing, read the generated parameter structs to understand their exact fields:

```
GetUsersParams
FindUsersParams
ExportParams
DeleteUsersParams
GetUserBidInfoParams
CreateUserParams (if query params exist for registration)
```

These are in `pkg/internal/api/api.gen.go`. The exact fields determine what functional options or direct parameters you need to expose.

### 2) Add methods to `UsersScope`

Add these methods to the existing `UsersScope` in `pkg/client/users.go`:

**Admin listing:**

- `List(ctx, ...ListOption) (*types.PageUserResource, error)` — wraps `GetUsersWithResponse`. This is the admin endpoint that returns all users. Use functional options for pagination/filtering.
- `Search(ctx, term string, ...ListOption) (*types.PageUserResource, error)` — wraps `FindUsersWithResponse`. The `term` parameter is the primary search input.

**Admin CRUD:**

- `Create(ctx, request types.CreateUserRQFull) (*types.CreateUserRS, error)` — wraps `CreateUserByAdminWithResponse`. Admin-only user creation.
- `Edit(ctx, login string, request types.EditUserRQ) (*types.OperationCompletionRS, error)` — wraps `EditUserWithResponse`.
- `Delete(ctx, id int64) (*types.OperationCompletionRS, error)` — wraps `DeleteUserWithResponse`. Deletes a single user by numeric ID.
- `DeleteBulk(ctx, ids []int64) (*types.DeleteBulkRS, error)` — wraps `DeleteUsersWithResponse`. Deletes multiple users.

**Export:**

- `Export(ctx, ...ListOption) ([]byte, error)` — wraps `ExportWithResponse`. Returns raw bytes. The response is likely CSV. Check the generated response type to see if there is a parsed body or just raw bytes.

**Invitation/registration:**

- `CreateInvitation(ctx, request types.CreateUserRQ) (*types.UserBidRS, error)` — wraps `CreateUserBidWithResponse`. Creates an invitation bid.
- `GetInvitation(ctx, uuid string) (*types.UserBidRS, error)` — wraps `GetUserBidInfoWithResponse`. Retrieves bid info by UUID query parameter.
- `Register(ctx, uuid string, request types.CreateUserRQConfirm) (*types.CreateUserRS, error)` — wraps `CreateUserWithResponse`. Activates an invitation and creates the user. The UUID is a query parameter.

### 3) Add functional options for listing

Follow the existing conventions. For example, look at how `LaunchScope` defines list options. Create options specific to the user list endpoints, such as:

- Page size / page number options
- Any filter options that `GetUsersParams` and `FindUsersParams` support

Name them with a `WithUser` prefix or similar to avoid collisions with other option types.

### 4) Handle the Export response

The export endpoint likely returns raw bytes (CSV/octet-stream), not JSON. Check the generated `ExportResponse` type. If it only has `Body []byte` and no `JSON200`, handle it differently from the standard `handleResponse` pattern:

- Check `JSONDefault` for error bodies
- Validate the status code is 2xx
- Return `response.Body` directly

This may need a small inline handling block rather than using `handleResponse`.

### 5) Logging pattern

For instance-level scopes, use `u.client.logger.With(slog.String("operation", "..."))` directly, not `operationLogger` (which is a `ProjectScope` method). Follow the existing pattern from `GetMyself` and other `UsersScope` methods.

## Test requirements

### Unit tests

Extend `pkg/client/users_test.go` with tests for every new method.

For **each** method, write at minimum:

- **Success case**: Full handler with method/path/token assertions and realistic JSON response. Assert parsed fields.
- **Error case**: At least one error path (forbidden for admin ops, not-found where applicable).
- **Edge cases**:
  - `List` and `Search` returning empty page
  - `DeleteBulk` with multiple IDs
  - `Export` returning raw bytes
  - `GetInvitation` not found

Expected paths to assert:

- `/api/users/all` (list)
- `/api/users/search` (search — check for `term` query param)
- `/api/users/export` (export)
- `/api/users` (create by admin — POST, delete bulk — DELETE)
- `/api/users/{login}` (edit — PUT)
- `/api/users/{id}` (delete — DELETE)
- `/api/users/bid` (create invitation — POST)
- `/api/users/registration` (get bid info — GET, register — POST; check `uuid` query param)

### Test patterns

Follow the existing test style in `users_test.go`. For write operations, also assert the request body:

```go
body, _ := io.ReadAll(r.Body)
var req types.CreateUserRQFull
_ = json.Unmarshal(body, &req)
assert.Equal(t, "newuser", req.Login)
```

Use `t.Parallel()` on every test. Use `newTestClient`, `assertBearerToken`, `assertMethod`, `assertPath`.

## Files to modify

| File | Action |
|------|--------|
| `pkg/client/users.go` | Modify — add ~10 new methods to `UsersScope` |
| `pkg/client/users_test.go` | Modify — add unit tests for all new methods |
| `pkg/client/export_test.go` | Possibly modify — no new constructors needed since `UsersScope` already exists |

## Design considerations

- The `deleteUser` endpoint takes a numeric `{id}` path parameter, while `editUser` takes a string `{login}`. Make sure the method signatures reflect the actual API parameter types.
- `DeleteUsersParams` uses `Ids` as a query parameter for bulk deletion. Check the generated code to confirm whether this is `[]int64` or a comma-separated string.
- The invitation flow is a two-step process: `createUserBid` creates the invitation, then `createUser` activates it with user details. Name methods clearly to reflect this lifecycle.
- The `export` endpoint may return different content types. Return raw `[]byte` and let the caller decide what to do with it.
- Some admin endpoints require `ADMINISTRATOR` role. Error tests should include `IsForbidden` assertions.

## Validation commands

Run before finalizing:

- `make lint`
- `make test`

## Definition of done

- [ ] Admin list/search/export methods added
- [ ] Admin create/edit/delete/bulk-delete methods added
- [ ] Invitation create/get and registration methods added
- [ ] Functional options for list pagination/filtering
- [ ] Export raw-bytes handling implemented
- [ ] Unit tests for every new method with success and error paths
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
