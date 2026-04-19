# Prompt 48: Integration Tests for Users Expansion

## Goal

Expand the integration tests for the `UsersScope` beyond basic fetching. The existing `client_users_test.go` is minimal (52 lines) and only covers `GetMyself`, `GetByLogin`, and `ListProjects`. This PR adds tests for user creation, editing, deletion, API key management, and advanced listing operations.

## Context

This is PR 7 of the test coverage improvement plan (Phase 3). It runs in parallel with PRs 8, 9, and 10. Phase 2 must be merged first.

## Prerequisites

- Phases 1-2 (PRs 1-6) are merged.

## Scope under test

### `UsersScope` — `Client.Users()`

Methods to add integration tests for:

| Method | Signature | Status |
|--------|-----------|--------|
| `GetMyself` | `(ctx) (*types.OwnUserResource, error)` | Already tested |
| `GetByLogin` | `(ctx, login string) (*types.UserResource, error)` | Already tested |
| `ListProjects` | `(ctx, login string) (map[string]types.AssignedProject, error)` | Already tested |
| `List` | `(ctx, options ...ListUsersOption) (*types.PageUserResource, error)` | **New** |
| `Search` | `(ctx, term string, options ...SearchUsersOption) (*types.PageUserResource, error)` | **New** |
| `Export` | `(ctx, options ...ListUsersOption) ([]byte, error)` | **New** |
| `Create` | `(ctx, request types.CreateUserRQ) (*types.CreateUserRS, error)` | **New** |
| `Edit` | `(ctx, login string, request types.EditUserRQ) (*types.OperationCompletionRS, error)` | **New** |
| `Delete` | `(ctx, login string) (*types.OperationCompletionRS, error)` | **New** |
| `ListAPIKeys` | `(ctx) ([]types.ApiKeyRS, error)` | **New** |
| `CreateAPIKey` | `(ctx, request types.ApiKeyRQ) (*types.ApiKeyRS, error)` | **New** |
| `DeleteAPIKey` | `(ctx, keyID int64) error` | **New** |

Methods excluded from this PR (require email/SMTP configuration):
- `CreateInvitation`, `GetInvitation`, `Register`
- `ChangePassword`, `RestorePassword`, `IsRestorePasswordBidExist`, `ResetPassword`

## Implementation plan

### 1) Expand `test/integration/client_users_test.go`

Keep the existing tests and add new Contexts:

```
Describe("Client: Users", Label("client", "users"))
  BeforeEach: newClient()

  [existing contexts remain unchanged]

  Context("List")
    It("returns at least one user")
      - Call c.Users().List(ctx)
      - Assert no error, assert content not empty

    It("supports pagination options")
      - Call c.Users().List(ctx, client.WithUsersPageSize(1))
      - Assert page size is 1

    It("filters by role")
      - Call c.Users().List(ctx, client.WithUsersRole("ADMINISTRATOR"))
      - Assert all returned users have admin role

  Context("Search")
    It("finds users matching a term")
      - Call c.Users().Search(ctx, "admin")
      - Assert results contain the admin user

    It("returns empty for a non-matching term")
      - Call c.Users().Search(ctx, "nonexistent_user_xyz_99999")
      - Assert empty content

  Context("Export")
    It("returns CSV data")
      - Call c.Users().Export(ctx)
      - Assert no error
      - Assert returned bytes are non-empty
      - Assert content starts with CSV header (first line contains column names)

  Context("User Lifecycle")
    It("creates, edits, and deletes a user")
      By("creating a test user")
      - username := fmt.Sprintf("test_user_%d", time.Now().UnixNano())
      - Call c.Users().Create(ctx, types.CreateUserRQ{...})
      - Assert no error

      By("verifying the user exists")
      - Call c.Users().GetByLogin(ctx, username)
      - Assert no error, assert login matches

      By("editing the user")
      - Call c.Users().Edit(ctx, username, types.EditUserRQ{...})
      - Assert no error

      By("deleting the user")
      - Call c.Users().Delete(ctx, username)
      - Assert no error

      By("verifying the user is deleted")
      - Call c.Users().GetByLogin(ctx, username)
      - Assert client.IsNotFound(err)

  Context("API Keys")
    It("creates, lists, and deletes an API key")
      By("creating an API key")
      - Call c.Users().CreateAPIKey(ctx, types.ApiKeyRQ{Name: "test-key-<timestamp>"})
      - Assert no error, capture key ID

      By("listing API keys")
      - Call c.Users().ListAPIKeys(ctx)
      - Assert the created key appears in the list

      By("deleting the API key")
      - Call c.Users().DeleteAPIKey(ctx, keyID)
      - Assert no error

      By("verifying the key is deleted")
      - Call c.Users().ListAPIKeys(ctx)
      - Assert the key no longer appears
```

### 2) Determine request type fields

Before implementing, read the type definitions:

- `types.CreateUserRQ` — likely includes `Login`, `Password`, `Email`, `Role`, `ProjectName`
- `types.EditUserRQ` — likely includes `Role`, `Email`, or other editable fields
- `types.ApiKeyRQ` — likely includes `Name`

Use realistic test values. For user creation, the new user needs a valid password, email, and likely needs to be assigned to a project. Check if the default project works.

### 3) Follow existing test conventions

- Use `NodeTimeout` on setup/teardown nodes
- Labels: `Label("client", "users")`
- Clean up created users in `DeferCleanup` or at the end of the test to avoid leaking test data
- Use `By("description")` for multi-step lifecycle tests

### 4) Run tests

```bash
make test-integration
make lint
```

## Files touched

| File | Change |
|------|--------|
| `test/integration/client_users_test.go` | Expand with lifecycle, listing, search, export, and API key tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No other integration test files modified
- Password/invitation endpoints are out of scope (require SMTP)
