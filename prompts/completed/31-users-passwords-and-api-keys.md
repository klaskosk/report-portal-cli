# Prompt 20: Users Passwords and API Keys (Backlog PR 14)

## Goal

Extend `UsersScope` in `pkg/client/users.go` with password management (change, reset, restore) and API key lifecycle (list, create, delete) operations.

## Context

This PR is sequenced **after PR 13** (Prompt 19: Users Admin and Lifecycle). It adds the remaining user-scoped operations that were intentionally split out to keep PR 13 at a manageable size. Both PRs modify `users.go` and `users_test.go`, so they must not be run in parallel.

After this PR lands, `UsersScope` will have full coverage of the `/users/...` API surface.

## Prerequisites

- PR 13 (Prompt 19) must be merged first. This PR builds on the methods already present in `users.go`.

## Endpoints to wrap

### Password management

| HTTP | Path | operationId | Generated method | Response type |
|------|------|-------------|-----------------|---------------|
| POST | `/users/password/change` | `changePassword` | `ChangePasswordWithResponse` | `OperationCompletionRS` (200) |
| POST | `/users/password/restore` | `restorePassword` | `RestorePasswordWithResponse` | `OperationCompletionRS` (200) |
| GET | `/users/password/reset/{uuid}` | `isRestorePasswordBidExist` | `IsRestorePasswordBidExistWithResponse` | Check generated type (200) |
| POST | `/users/password/reset` | `resetPassword` | `ResetPasswordWithResponse` | `OperationCompletionRS` (200) |

### API key management

| HTTP | Path | operationId | Generated method | Response type |
|------|------|-------------|-----------------|---------------|
| GET | `/users/{userId}/api-keys` | `getUsersApiKeys` | `GetUsersApiKeysWithResponse` | `ApiKeysRS` or list of `ApiKeyRS` (200) |
| POST | `/users/{userId}/api-keys` | `createApiKey` | `CreateApiKeyWithResponse` | `ApiKeyRS` (201) |
| DELETE | `/users/{userId}/api-keys/{keyId}` | `deleteApiKey` | `DeleteApiKeyWithResponse` | `OperationCompletionRS` (200) |

## Key types (from `pkg/types`)

**Password types:**

- `ChangePasswordRQ` — `OldPassword` (required string), `NewPassword` (optional `*string`)
- `RestorePasswordRQ` — `Email` (required string)
- `ResetPasswordRQ` — `Password` (required string), `Uuid` (required string)
- `OperationCompletionRS` — generic success with `Message`

**API key types:**

- `ApiKeyRS` — `Id` (int64), `Name` (string), `CreatedAt` (time.Time), `LastUsedAt` (*time.Time), `UserId` (int64), `ApiKey` (*string, only populated on creation)
- `ApiKeysRS` — check the generated type; likely wraps a list of `ApiKeyRS`
- The create body is a `CreateApiKeyJSONRequestBody` — check the generated type for its shape (likely has a `Name` field)

## Implementation plan

### 1) Check generated types and response structs

Before implementing, read the generated code for:

- `ChangePasswordResponse` — confirm it has `JSON200 *OperationCompletionRS`
- `ResetPasswordResponse` — confirm it has `JSON200 *OperationCompletionRS`
- `IsRestorePasswordBidExistResponse` — check what the success field is (could be `JSON200 *IsRestorePasswordBidExist200Response` or similar)
- `RestorePasswordResponse` — confirm it has `JSON200 *OperationCompletionRS`
- `GetUsersApiKeysResponse` — check whether it has `JSON200 *ApiKeysRS` or `JSON200 *[]ApiKeyRS`
- `CreateApiKeyResponse` — check for `JSON201 *ApiKeyRS`
- `DeleteApiKeyResponse` — confirm it has `JSON200 *OperationCompletionRS`
- `CreateApiKeyJSONRequestBody` — check its shape

These are in `pkg/internal/api/api.gen.go`.

### 2) Add password methods to `UsersScope`

Add these methods to `UsersScope` in `pkg/client/users.go`:

- `ChangePassword(ctx, request types.ChangePasswordRQ) (*types.OperationCompletionRS, error)` — wraps `ChangePasswordWithResponse`. Changes the current user's password. Requires old and new password.
- `RestorePassword(ctx, request types.RestorePasswordRQ) (*types.OperationCompletionRS, error)` — wraps `RestorePasswordWithResponse`. Initiates a password restore by email. This sends a reset link to the user's email.
- `IsRestorePasswordBidExist(ctx, uuid string) (bool, error)` — wraps `IsRestorePasswordBidExistWithResponse`. Checks whether a password reset bid exists. Return a boolean for the public API (check the generated response type to determine how to interpret the result).
- `ResetPassword(ctx, request types.ResetPasswordRQ) (*types.OperationCompletionRS, error)` — wraps `ResetPasswordWithResponse`. Completes the password reset using the UUID from the restore email plus the new password.

### 3) Add API key methods to `UsersScope`

- `ListAPIKeys(ctx, userID int64) ([]types.ApiKeyRS, error)` — wraps `GetUsersApiKeysWithResponse`. Returns the list of API keys for the given user. Unwrap from the response container type so the public API returns `[]types.ApiKeyRS` directly. Return empty slice (not nil) when no keys exist.
- `CreateAPIKey(ctx, userID int64, request ...) (*types.ApiKeyRS, error)` — wraps `CreateApiKeyWithResponse`. The request type depends on the generated body type (check `CreateApiKeyJSONRequestBody`). The response includes the `ApiKey` field which is only available at creation time.
- `DeleteAPIKey(ctx, userID int64, keyID int64) (*types.OperationCompletionRS, error)` — wraps `DeleteApiKeyWithResponse`.

### 4) Logging

Use the instance-level logging pattern: `u.client.logger.With(slog.String("operation", "..."))`. Add relevant attributes like `slog.Int64("userId", userID)` or `slog.Int64("keyId", keyID)`.

### 5) Password flow design

The password lifecycle has three distinct flows. Name methods clearly:

1. **Change** (authenticated user changes own password): `ChangePassword`
2. **Restore** (unauthenticated user requests reset link via email): `RestorePassword`
3. **Reset** (unauthenticated user completes reset with UUID + new password): `ResetPassword`
4. **Check bid** (verify a reset UUID is valid): `IsRestorePasswordBidExist`

## Test requirements

### Unit tests

Add tests to `pkg/client/users_test.go` for every new method.

For **each** method, write at minimum:

- **Success case**: httptest handler asserting method, path, token, and request body. Return realistic JSON and assert parsed fields.
- **Error case**: At least one error path per method.

### Password test expectations

Expected paths:

- `/api/users/password/change` (POST) — assert request body contains `oldPassword` and `newPassword`
- `/api/users/password/restore` (POST) — assert request body contains `email`
- `/api/users/password/reset/{uuid}` (GET) — assert UUID in path
- `/api/users/password/reset` (POST) — assert request body contains `password` and `uuid`

Error scenarios:

- `ChangePassword` with wrong old password → expect `IsBadRequest` or similar
- `IsRestorePasswordBidExist` with invalid UUID → expect `IsNotFound`

### API key test expectations

Expected paths:

- `/api/users/{userId}/api-keys` (GET) — list keys for user
- `/api/users/{userId}/api-keys` (POST) — create key, assert body
- `/api/users/{userId}/api-keys/{keyId}` (DELETE) — delete specific key

Test scenarios:

- List API keys success (multiple keys)
- List API keys empty
- Create API key success — assert the `ApiKey` field is present
- Delete API key success
- Delete API key not found

Use `t.Parallel()` on every test. Follow existing patterns from the file.

## Files to modify

| File | Action |
|------|--------|
| `pkg/client/users.go` | Modify — add ~7 new methods to `UsersScope` |
| `pkg/client/users_test.go` | Modify — add unit tests for all new methods |

## Design considerations

- `ChangePassword` is an authenticated operation (current user changes their own password). `RestorePassword` and `ResetPassword` are part of an unauthenticated flow, but the client always sends the bearer token. The server may or may not require authentication for these — the client should send the token regardless, as it does for all operations.
- The `IsRestorePasswordBidExist` endpoint is a simple existence check. Expose it as a `bool` return. If the bid doesn't exist, the API likely returns a 404 — map that to `false, nil` rather than an error. This is a design choice; alternatively return the error and let the caller use `IsNotFound`. Pick the approach that is more consistent with the rest of the client (the client generally does propagate errors, so returning the error is likely more consistent).
- API key operations use `userId` (numeric int64), not `login` (string). Make sure the method signatures use `int64`.
- The `ApiKey` field on `ApiKeyRS` is `*string` and is only non-nil when the key is first created. Document this in the method's godoc comment for `CreateAPIKey`.

## Validation commands

Run before finalizing:

- `make lint`
- `make test`

## Definition of done

- [ ] Password change/restore/reset/bid-check methods added
- [ ] API key list/create/delete methods added
- [ ] Unit tests for every new method with success and error paths
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
