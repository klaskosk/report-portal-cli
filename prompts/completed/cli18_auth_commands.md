# PR 18: Auth Commands

## Goal

Add `rp auth apikey {list,create,delete}` and `rp auth {change-password,reset-password,invite}` as a new resource grouping for authentication-related operations.

## Context

This PR runs in Phase 7. The user/identity client methods already exist. This PR creates a new `auth` noun in the CLI that groups authentication operations that are conceptually different from user CRUD.

### Prerequisites

- PRs 1–15 have landed
- `pkg/client/users.go` already has: `ListAPIKeys`, `CreateAPIKey`, `DeleteAPIKey`, `ChangePassword`, `RestorePassword`, `ResetPassword`, `CreateInvitation`

### Client API surface

**UsersScope** (via `client.Users()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `ListAPIKeys` | `ListAPIKeys(ctx, userID int64)` | `[]types.ApiKeyRS` |
| `CreateAPIKey` | `CreateAPIKey(ctx, userID int64, types.ApiKeyRQ)` | `*types.ApiKeyRS` |
| `DeleteAPIKey` | `DeleteAPIKey(ctx, userID int64, keyID int64)` | `*types.OperationCompletionRS` |
| `ChangePassword` | `ChangePassword(ctx, types.ChangePasswordRQ)` | `*types.OperationCompletionRS` |
| `RestorePassword` | `RestorePassword(ctx, types.RestorePasswordRQ)` | `*types.OperationCompletionRS` |
| `CreateInvitation` | `CreateInvitation(ctx, types.CreateUserRQ)` | `*types.CreateUserBidRS` |

Also need `GetMyself` to get the current user's ID for API key operations.

### Relevant types

```go
type ApiKeyRS struct {
    ApiKey     *string    `json:"api_key,omitempty"`
    CreatedAt  *time.Time `json:"created_at,omitempty"`
    Id         *int64     `json:"id,omitempty"`
    Name       *string    `json:"name,omitempty"`
    UserId     *int64     `json:"user_id,omitempty"`
}

type ApiKeyRQ struct {
    Name string `json:"name"`
}

type ChangePasswordRQ struct {
    NewPassword string `json:"newPassword"`
    OldPassword string `json:"oldPassword"`
}

type RestorePasswordRQ struct {
    Email string `json:"email"`
}

type CreateUserRQ struct {
    DefaultProject *string `json:"defaultProject,omitempty"`
    Email          string  `json:"email"`
    Role           *string `json:"role,omitempty"`
}

type CreateUserBidRS struct {
    Backlink *string `json:"backlink,omitempty"`
    Bid      *string `json:"bid,omitempty"`
    Message  *string `json:"message,omitempty"`
}
```

## Implementation plan

### 1. Create `internal/cli/auth.go`

#### Parent command structure

```
rp auth
├── apikey
│   ├── list
│   ├── create
│   └── delete
├── change-password
├── reset-password
└── invite
```

#### `rp auth apikey list`

- No positional arguments
- Optional flag: `--user-id` (int64) — target user ID. If not provided, use current user (via `GetMyself`)
- Call `scope.ListAPIKeys(ctx, userID)`
- Table columns: ID, NAME, CREATED AT
- JSON output: raw `[]ApiKeyRS`
- Note: The `ApiKey` field (the actual token value) is NOT shown in list (it's only available at creation time)

#### `rp auth apikey create`

- No positional arguments
- Required flag: `--name` (string) — key name
- Optional flag: `--user-id` (int64) — target user ID. If not provided, use current user
- Call `scope.CreateAPIKey(ctx, userID, types.ApiKeyRQ{Name: name})`
- Table columns: ID, NAME, API KEY (the raw key value)
- JSON output: raw `*types.ApiKeyRS`
- IMPORTANT: The API key value is only returned once at creation. Make this clear in help text.

#### `rp auth apikey delete <key-id>`

- Exactly one positional argument: numeric key ID
- Optional flag: `--user-id` (int64) — target user ID. If not provided, use current user
- Parse key ID using `parseIDArg`
- Call `scope.DeleteAPIKey(ctx, userID, keyID)`
- Table/JSON output: `OperationCompletionRS` message

#### `rp auth change-password`

- No positional arguments
- Required flags: `--old-password`, `--new-password` (both strings)
- Call `scope.ChangePassword(ctx, types.ChangePasswordRQ{OldPassword: old, NewPassword: new})`
- Table/JSON output: `OperationCompletionRS` message

#### `rp auth reset-password`

- No positional arguments
- Required flag: `--email` (string)
- Call `scope.RestorePassword(ctx, types.RestorePasswordRQ{Email: email})`
- Table/JSON output: `OperationCompletionRS` message
- Note: This initiates the password reset flow (sends email). It does NOT complete the reset.

#### `rp auth invite`

- No positional arguments
- Required flag: `--email` (string)
- Optional flags:
  - `--default-project` (string) — default project for the invited user
  - `--role` (string) — role for the invited user
- Call `scope.CreateInvitation(ctx, types.CreateUserRQ{Email: email, DefaultProject: dp, Role: role})`
- Table columns: BID, MESSAGE
- JSON output: raw `*types.CreateUserBidRS`

### 2. Register in root.go

Add `cmd.AddCommand(newAuthCmd())` in `newRootCmd()`.

### 3. Get current user helper

Create a helper to resolve user ID:
```go
func resolveUserID(cmd *cobra.Command, cl *client.Client) (int64, error) {
    if cmd.Flags().Changed("user-id") {
        id, _ := cmd.Flags().GetInt64("user-id")
        return id, nil
    }
    myself, err := cl.Users().GetMyself(cmd.Context())
    if err != nil {
        return 0, fmt.Errorf("resolving current user: %w", err)
    }
    if myself.UserId == nil {
        return 0, fmt.Errorf("current user has no ID")
    }
    return int64(*myself.UserId), nil
}
```

Note: Check the actual type of `UserResource.UserId` — it may be `*int32` or `*int64`.

## Test requirements

### Unit tests (`internal/cli/auth_test.go`)

- **API key list:** Verify `--user-id` flag is optional
- **API key create:** Verify `--name` is required
- **API key delete:** Verify exactly 1 positional arg, non-numeric rejected
- **Change password:** Verify both `--old-password` and `--new-password` required
- **Reset password:** Verify `--email` required
- **Invite:** Verify `--email` required, `--default-project` and `--role` optional
- **Command wiring:** Verify `newAuthCmd()` has correct subcommand tree

## Edge cases and constraints

- **No `--project` flag:** Auth commands are instance-scoped (no project context needed)
- **User ID resolution:** API key commands need a user ID. Default to current user via `GetMyself` if `--user-id` is not provided.
- **API key secrecy:** The raw API key is only available at creation time. Table output for `create` shows it; `list` does not.
- **Password flags are sensitive:** Consider noting in help text that passwords will be visible in shell history. Do NOT attempt to read from stdin or prompt — keep it simple with flags.
- **`UserResource.UserId` type:** Check if it's `*int32` — if so, cast to `int64` for the API call.

## Example commands

```bash
# List my API keys
$ rp auth apikey list
ID    NAME          CREATED AT
1     ci-token      2024-01-15T10:30:00Z
2     dev-token     2024-03-20T14:00:00Z

# Create an API key
$ rp auth apikey create --name "new-ci-key"
ID    NAME          API KEY
3     new-ci-key    rp_abc123...xyz

# Delete an API key
$ rp auth apikey delete 3
MESSAGE
API key has been deleted successfully.

# Change password
$ rp auth change-password --old-password "old123" --new-password "new456"
MESSAGE
Password has been changed successfully.

# Reset password (sends email)
$ rp auth reset-password --email user@example.com
MESSAGE
Password reset link has been sent.

# Invite a user
$ rp auth invite --email newuser@example.com --default-project my_project --role USER
BID                                   MESSAGE
a1b2c3d4-e5f6-7890-abcd-ef1234567890  Invitation has been sent.
```

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [ ] `internal/cli/auth.go` implements the full auth command tree
- [ ] `rp auth` registered in `root.go`
- [ ] API key commands default to current user when `--user-id` not provided
- [ ] Unit tests cover flag validation, command wiring, and ID resolution
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] This file moved to `prompts/completed/`
