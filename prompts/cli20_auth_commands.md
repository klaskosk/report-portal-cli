# PR 20: `auth` Commands

## Goal

Implement the `auth` command group — API key management (create, list, delete), password operations (change, reset), and user invitation. API key commands auto-detect the current user's ID via the existing user info endpoint, with a `--user-id` override for admin use cases.

## Context

Auth commands map to `UsersScope` on the client (`client.Users()`). They are **instance-scoped** — commands do **not** require `--project`.

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `parseResourceID`, `mustGetStringFlag`, `formatStringPtr`, `formatInt64Ptr`)

**Design decision:** API key commands auto-detect the current user's ID via `GetMyself` (same call used by `rp whoami`), with a `--user-id` override for admin workflows.

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `CreateAPIKey` | `ctx`, `userID int64`, `request types.ApiKeyRQ` | `(*types.ApiKeyRS, error)` |
| `DeleteAPIKey` | `ctx`, `userID int64`, `keyID int64` | `(*types.OperationCompletionRS, error)` |
| `ListAPIKeys` | `ctx`, `userID int64` | `([]types.ApiKeyRS, error)` |
| `ResetPassword` | `ctx`, `request types.ResetPasswordRQ` | `(*types.OperationCompletionRS, error)` |
| `CreateInvitation` | `ctx`, `request types.CreateUserRQ` | `(*types.CreateUserBidRS, error)` |
| `ChangePassword` | `ctx`, `request types.ChangePasswordRQ` | `(*types.OperationCompletionRS, error)` |

**Notable:** API key operations require a `userID int64` parameter. The CLI auto-detects this from the current user's profile. The existing `whoami` command already calls `client.Users().GetMyself(ctx)` which returns a `*types.UserResource` with `Id int64` — the same mechanism is used here.

### Key types

```go
type ApiKeyRQ struct {
    Name string `json:"name"`
}

type ApiKeyRS struct {
    ApiKey     *string    `json:"api_key,omitempty"`
    CreatedAt  time.Time  `json:"created_at"`
    Id         int64      `json:"id"`
    LastUsedAt *time.Time `json:"last_used_at,omitempty"`
    Name       string     `json:"name"`
    UserId     int64      `json:"user_id"`
}

type ResetPasswordRQ struct {
    Password string `json:"password"`
    Uuid     string `json:"uuid"`
}

type CreateUserRQ struct {
    DefaultProject string `json:"defaultProject"`
    Email          string `json:"email"`
    Role           string `json:"role"`
}

type CreateUserBidRS struct {
    BackLink *string `json:"backLink,omitempty"`
    Bid      *string `json:"bid,omitempty"`
    Message  *string `json:"message,omitempty"`
}

type ChangePasswordRQ struct {
    NewPassword *string `json:"newPassword,omitempty"`
    OldPassword string  `json:"oldPassword"`
}
```

## Scope

Create `internal/cli/auth.go` and `internal/cli/auth_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp auth create-apikey --name <name> [--user-id <id>]` | Create an API key |
| `rp auth list-apikeys [--user-id <id>]` | List API keys |
| `rp auth delete-apikey <key-id> [--user-id <id>]` | Delete an API key |
| `rp auth change-password --old-password <old> --new-password <new>` | Change the current user's password |
| `rp auth reset-password --uuid <uuid> --password <password>` | Reset a password using a reset token |
| `rp auth invite --email <email> --role <role> --default-project <project>` | Invite a new user |

## Implementation

### 1. Create `internal/cli/auth.go`

This file contains all six command constructors, their run functions, and a shared helper for resolving the user ID.

#### User ID resolution helper

```go
func resolveUserID(cmd *cobra.Command, userScope *client.UsersScope) (int64, error) {
    if cmd.Flags().Changed("user-id") {
        userID, _ := cmd.Flags().GetInt64("user-id")
        if userID <= 0 {
            return 0, fmt.Errorf("--user-id must be a positive integer, got %d", userID)
        }

        return userID, nil
    }

    ctx := cmd.Context()
    currentUser, err := userScope.GetMyself(ctx)
    if err != nil {
        return 0, fmt.Errorf("auto-detecting current user ID: %w\n\nTip: use --user-id to specify explicitly", err)
    }

    return currentUser.Id, nil
}
```

This function checks for an explicit `--user-id` flag first. If not provided, it calls `GetMyself` to auto-detect the current user's numeric ID. The error message includes a tip about `--user-id` so users know how to work around auto-detection failures.

**Verified:** The `whoami` command (`whoami.go`) already uses `client.Users().GetMyself(ctx)`. The method exists on `UsersScope` and returns `(*types.UserResource, error)` where `UserResource.Id` is `int64` (not a pointer).

#### `newAuthCmd()`

The `auth` parent command should already be scaffolded from PR 2. If not, define it here:

```go
func newAuthCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "auth",
        Short: "Authentication and credential management",
    }
}
```

Register subcommands within it.

#### `newAuthCreateAPIKeyCmd()`

```go
func newAuthCreateAPIKeyCmd() *cobra.Command
```

- `Use`: `"create-apikey"`
- `Short`: `"Create an API key"`
- `Long`: Include examples showing `--name` usage and optional `--user-id`. Note the API key value is only shown once on creation.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runAuthCreateAPIKey)`
- Flags:
  - `--name` (string, required): display name for the API key
  - `--user-id` (int64, optional): user ID (auto-detected if omitted)

The run function:
1. Validate `--name` is non-empty
2. Resolve connection and client
3. Resolve user ID using `resolveUserID(cmd, scope)`
4. Build `types.ApiKeyRQ{Name: name}`
5. Call `scope.CreateAPIKey(ctx, userID, request)`
6. Print the `ApiKeyRS` result as a single table row

**Important:** The `ApiKey *string` field in `ApiKeyRS` contains the actual API key token. This is only returned on creation — subsequent `ListAPIKeys` calls do not include it. The table should show the API key value prominently.

#### `newAuthListAPIKeysCmd()`

```go
func newAuthListAPIKeysCmd() *cobra.Command
```

- `Use`: `"list-apikeys"`
- `Short`: `"List API keys"`
- `Long`: Include examples. Note that the actual API key values are not shown in list output.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runAuthListAPIKeys)`
- Flags:
  - `--user-id` (int64, optional): user ID (auto-detected if omitted)

The run function:
1. Resolve connection and client
2. Resolve user ID using `resolveUserID(cmd, scope)`
3. Call `scope.ListAPIKeys(ctx, userID)` — returns `[]types.ApiKeyRS`
4. Pass the slice to `printer.Print`

#### `newAuthDeleteAPIKeyCmd()`

```go
func newAuthDeleteAPIKeyCmd() *cobra.Command
```

- `Use`: `"delete-apikey <key-id>"`
- `Short`: `"Delete an API key"`
- `Long`: Include example showing delete by key ID
- `Args`: `cobra.ExactArgs(1)` (the numeric API key ID)
- `RunE`: `withPrintedErrors(runAuthDeleteAPIKey)`
- Flags:
  - `--user-id` (int64, optional): user ID (auto-detected if omitted)

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "API key")`
2. Resolve connection and client
3. Resolve user ID using `resolveUserID(cmd, scope)`
4. Call `scope.DeleteAPIKey(ctx, userID, keyID)`
5. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newAuthChangePasswordCmd()`

```go
func newAuthChangePasswordCmd() *cobra.Command
```

- `Use`: `"change-password"`
- `Short`: `"Change your password"`
- `Long`: Include example. Note that passwords will be visible in shell history — consider using environment variables or a credentials file in production.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runAuthChangePassword)`
- Flags:
  - `--old-password` (string, required): current password
  - `--new-password` (string, required): new password

The run function:
1. Validate `--old-password` is non-empty
2. Validate `--new-password` is non-empty
3. Build `types.ChangePasswordRQ{OldPassword: oldPassword, NewPassword: &newPassword}`
4. Resolve connection and client
5. Call `scope.ChangePassword(ctx, request)`
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newAuthResetPasswordCmd()`

```go
func newAuthResetPasswordCmd() *cobra.Command
```

- `Use`: `"reset-password"`
- `Short`: `"Reset a password using a reset token"`
- `Long`: Include example. Explain that `--uuid` is the password reset token received via email, not a user UUID.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runAuthResetPassword)`
- Flags:
  - `--uuid` (string, required): password reset token from the reset email
  - `--password` (string, required): new password

The run function:
1. Validate `--uuid` is non-empty
2. Validate `--password` is non-empty
3. Build `types.ResetPasswordRQ{Password: password, Uuid: uuid}`
4. Resolve connection and client
5. Call `scope.ResetPassword(ctx, request)`
6. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newAuthInviteCmd()`

```go
func newAuthInviteCmd() *cobra.Command
```

- `Use`: `"invite"`
- `Short`: `"Invite a new user"`
- `Long`: Include examples showing all required flags. Note valid role values.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runAuthInvite)`
- Flags:
  - `--email` (string, required): email address of the user to invite
  - `--role` (string, required): role to assign (e.g., `USER`, `ADMINISTRATOR`)
  - `--default-project` (string, required): project to assign as the user's default

The run function:
1. Validate `--email` is non-empty
2. Validate `--role` is non-empty
3. Validate `--default-project` is non-empty
4. Build `types.CreateUserRQ{Email: email, Role: role, DefaultProject: defaultProject}`
5. Resolve connection and client
6. Call `scope.CreateInvitation(ctx, request)`
7. Print the `CreateUserBidRS` result

**Result formatting for `CreateUserBidRS`:**

```go
func inviteHeaders() []string {
    return []string{"BID", "BACK LINK", "MESSAGE"}
}

func inviteRows(result *types.CreateUserBidRS) [][]string {
    return [][]string{{formatStringPtr(result.Bid), formatStringPtr(result.BackLink), formatStringPtr(result.Message)}}
}
```

The `BackLink` field contains the URL the invited user can follow to complete registration. It is always shown in table output alongside the `Bid`.

#### Table formatting for API keys

```go
func apiKeyHeaders() []string {
    return []string{"ID", "NAME", "API KEY", "CREATED AT", "LAST USED AT"}
}

func apiKeyRow(key *types.ApiKeyRS) ([]string, []string) {
    apiKey := ""
    if key.ApiKey != nil {
        apiKey = *key.ApiKey
    }

    lastUsed := ""
    if key.LastUsedAt != nil {
        lastUsed = key.LastUsedAt.Format(time.RFC3339)
    }

    return apiKeyHeaders(), []string{
        strconv.FormatInt(key.Id, 10),
        key.Name,
        apiKey,
        key.CreatedAt.Format(time.RFC3339),
        lastUsed,
    }
}
```

The `API KEY` column will be empty for list results (the API does not return the key value after creation) and populated only on the create response.

### 2. Register commands in `commands.go`

Add this to `buildCommandTree` in `internal/cli/commands.go`:

```go
// auth
auth.AddCommand(newAuthCreateAPIKeyCmd())
auth.AddCommand(newAuthListAPIKeysCmd())
auth.AddCommand(newAuthDeleteAPIKeyCmd())
auth.AddCommand(newAuthChangePasswordCmd())
auth.AddCommand(newAuthResetPasswordCmd())
auth.AddCommand(newAuthInviteCmd())
```

### 3. Create `internal/cli/auth_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewAuthCreateAPIKeyCmd_Flags`** — Verify `--name` and `--user-id` flags are registered.

2. **`TestNewAuthListAPIKeysCmd_Flags`** — Verify `--user-id` flag is registered. Verify `Args` is `cobra.NoArgs`.

3. **`TestNewAuthDeleteAPIKeyCmd_Flags`** — Verify `--user-id` flag is registered. Verify exactly 1 arg is accepted.

4. **`TestNewAuthChangePasswordCmd_Flags`** — Verify `--old-password` and `--new-password` flags are registered.

5. **`TestNewAuthResetPasswordCmd_Flags`** — Verify `--uuid` and `--password` flags are registered.

6. **`TestNewAuthInviteCmd_Flags`** — Verify `--email`, `--role`, and `--default-project` flags are registered.

7. **`TestApiKeyRow_Formatting`** — Verify `apiKeyRow` produces expected headers and values, including a non-nil `ApiKey` value.

8. **`TestApiKeyRow_NilOptionalFields`** — Verify `apiKeyRow` handles nil `ApiKey` and nil `LastUsedAt` (produces empty strings).

9. **`TestInviteRows_Formatting`** — Verify `inviteRows` produces expected output for a fully-populated `CreateUserBidRS` (Bid, BackLink, Message).

10. **`TestInviteRows_NilFields`** — Verify `inviteRows` handles nil `Bid`, nil `BackLink`, and nil `Message`.

11. **`TestNoProjectFlagOnAuthCommands`** — Verify no auth subcommand has a `--project` flag.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Instance-scoped means no `--project` flag.** Auth operations are user-level, not project-level.
- **Auto-detect user ID for API key operations.** The common case is managing your own keys. Requiring `--user-id` every time would be tedious. The auto-detection uses the same endpoint as `rp whoami`. The `--user-id` override supports admin workflows.
- **Inline flags for all commands.** All request types have simple scalar fields (1–3 fields). `--from-file` would be overkill.
- **Password visibility warning.** The help text for `change-password` and `reset-password` should note that passwords passed via command-line flags are visible in shell history. This is a known trade-off — for production use, consider environment variables.
- **`CreateUserBidRS` is an invitation response, not a user creation response.** The `Bid` field is the invitation token. The user is not created until they accept the invitation.
- **API key token shown only on create.** The `ApiKey` field in `ApiKeyRS` is only populated when creating a key. List and delete responses do not include it. The table column will be empty for list output.
- **Reuse shared helpers from PR 7.** Use `parseResourceID`, `mustGetStringFlag`, `formatStringPtr`, `messageResultHeaders`/`operationResultRows`.
- **Route all output through `printer.Print`.** Single-item results use `printer.Print(obj, headers, [][]string{row})`. List results use `printer.Print(slice, headers, rows)`. This ensures JSON, Go-template, and table output modes all work correctly.
- **Use client error predicates.** When handling API errors, use the predicate functions (`IsNotFound`, `IsUnauthorized`, etc.) from the client package where relevant, rather than type assertions.
- **Command registration follows PR 3's centralized command-tree layout.** All subcommands are registered in `buildCommandTree` in `commands.go`.

## What NOT to do

- **Do not add `--project` / `-p` flags** to any auth command.
- **Do not hash or validate passwords locally.** Pass through to the API.
- **Do not validate email format locally.** Let the API validate.
- **Do not validate role values locally.** Valid roles may vary between Report Portal versions.
- **Do not store or cache user IDs.** Each invocation either uses the explicit `--user-id` or makes a fresh `GetMyself` call.
- **Do not add integration tests.** Unit tests are sufficient for this PR.

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp auth --help` shows all six subcommands
- `rp auth create-apikey --help` shows `--name` and `--user-id` flags
- `rp auth list-apikeys --help` shows `--user-id` flag
- `rp auth delete-apikey --help` shows `--user-id` flag and accepts a key ID arg
- `rp auth change-password --help` shows `--old-password` and `--new-password`
- `rp auth invite --help` shows `--email`, `--role`, `--default-project`
- No existing commands are affected

## Definition of done

- [ ] `internal/cli/auth.go` created with all six command constructors and run functions
- [ ] `internal/cli/auth_test.go` created with unit tests
- [ ] Commands registered under `auth` parent in `commands.go`
- [ ] No `--project` flag on any auth command
- [ ] API key commands auto-detect user ID with `--user-id` override
- [ ] `resolveUserID` helper uses `GetMyself` for auto-detection (returns `int64`, no nil check needed)
- [ ] Table output for API keys shows ID, NAME, API KEY, CREATED AT, LAST USED AT
- [ ] Invite result shows BID, BACK LINK, MESSAGE
- [ ] `reset-password` command accepts `--uuid` and `--password`, calls `ResetPassword`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
