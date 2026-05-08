# PR 29: Notification Settings Commands

## Goal

Implement CRUD commands for notification rules (the `notification` resource). Notifications are a sub-resource of project settings, but per the UX decision in PR 17, they are exposed as a flat resource name under standard verbs.

## Context

Notification rules are managed through `ProjectSettingsScope` on the client (`client.Project(name).Settings()`). They are **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `readRequestFromFile`, `parseResourceID`, `formatInt64Ptr`, `formatBoolPtr`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`)
- PR 17 (frozen UX: §6b notification rules as flat resource under standard verbs)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `GetNotifications` | `ctx` | `([]types.SenderCaseDTO, error)` |
| `CreateNotification` | `ctx`, `request types.SenderCaseDTO` | `(*types.EntryCreatedRS, error)` |
| `UpdateNotification` | `ctx`, `request types.SenderCaseDTO` | `(*types.OperationCompletionRS, error)` |
| `DeleteNotification` | `ctx`, `id int64` | `(*types.OperationCompletionRS, error)` |

**Notable:** `GetNotifications` returns all notifications (no pagination). Both `CreateNotification` and `UpdateNotification` take the full `SenderCaseDTO` — the same type is used for both create and update. For update, the CLI still accepts a positional notification ID and then injects or validates the DTO `Id` field before calling the client.

### Key types

```go
type SenderCaseDTO struct {
    Attributes         *[]ItemAttributeResource        `json:"attributes,omitempty"`
    AttributesOperator SenderCaseDTOAttributesOperator `json:"attributesOperator"`
    Enabled            *bool                           `json:"enabled,omitempty"`
    Id                 *int64                          `json:"id,omitempty"`
    LaunchNames        *[]string                       `json:"launchNames,omitempty"`
    Recipients         *[]string                       `json:"recipients,omitempty"`
    RuleDetails        *map[string]interface{}         `json:"ruleDetails,omitempty"`
    RuleName           string                          `json:"ruleName"`
    SendCase           SenderCaseDTOSendCase           `json:"sendCase"`
    Type               *string                         `json:"type,omitempty"`
}
```

`SenderCaseDTOAttributesOperator` values: `AND`, `OR`.
`SenderCaseDTOSendCase` values: `always`, `failed`, `more10`, `more20`, `more50`, `toInvestigate`.

## Scope

Create `internal/cli/notification.go` and `internal/cli/notification_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get notification` | List all notification rules in a project |
| `rp create notification --file <path>` | Create a notification rule |
| `rp update notification <id> --file <path>` | Update a notification rule |
| `rp delete notification <id>` | Delete a notification rule by numeric ID |

## Implementation

### 1. Create `internal/cli/notification.go`

#### `newGetNotificationCmd()`

```go
func newGetNotificationCmd() *cobra.Command
```

- `Use`: `"notification"`
- `Short`: `"List notification rules in a project"`
- `Long`: Include examples. No single-get-by-ID — always lists all.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGetNotification)`
- Flags:
  - `--project` / `-p` (string): project name override

No pagination flags — the API returns all notifications at once.

The run function:
1. Resolve connection, project, and client
2. Call `scope.GetNotifications(ctx)` — returns `[]types.SenderCaseDTO`
3. Build headers and rows for table output, then call `printer.Print`:
   ```go
   headers := notificationHeaders()
   rows := make([][]string, 0, len(notifications))

   for i := range notifications {
       _, row := notificationRow(&notifications[i])
       rows = append(rows, row)
   }

   return printer.Print(notifications, headers, rows)
   ```
   `printer.Print` takes `(value any, tableHeaders []string, tableRows [][]string)`. For JSON, it uses the first argument (`notifications` slice) directly. For table, it uses the pre-built headers and rows. The caller is responsible for iterating the slice and building rows — the printer does not call the row formatter itself.

#### `newCreateNotificationCmd()`

```go
func newCreateNotificationCmd() *cobra.Command
```

- `Use`: `"notification"`
- `Short`: `"Create a notification rule"`
- `Long`: Include example with `--file`. Include sample JSON.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateNotification)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `SenderCaseDTO`

**Why `--file`?** Per PR 17's file-flag rule, notification create uses `--file` because the DTO file is the command's primary input mode. `SenderCaseDTO` carries arrays, nested rule details, and enum fields, so inline flags would be unwieldy.

The run function:
1. Validate `--file` is non-empty
2. Read and unmarshal using `readRequestFromFile[types.SenderCaseDTO](path)`
3. Resolve connection, project, and client
4. Call `scope.CreateNotification(ctx, request)`
5. Print the `EntryCreatedRS` result using `entryCreatedHeaders()` and `entryCreatedRows(result)`

#### `newUpdateNotificationCmd()`

```go
func newUpdateNotificationCmd() *cobra.Command
```

- `Use`: `"notification <id>"`
- `Short`: `"Update a notification rule"`
- `Long`: Include example showing the positional ID plus `--file`. Explain that the positional argument is the source of truth for the target notification. If the JSON omits `id`, the CLI injects it; if the JSON includes a different `id`, the command should fail with a descriptive error.
- `Args`: `cobra.ExactArgs(1)` (the numeric notification ID)
- `RunE`: `withPrintedErrors(runUpdateNotification)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `SenderCaseDTO`

**ID handling:** The API method takes only `SenderCaseDTO`, but the frozen CLI UX still uses a positional target: `rp update notification <id> --file <path>`. After loading the DTO:
- If `request.Id` is nil, set it from the positional ID.
- If `request.Id` is non-nil and matches the positional ID, proceed.
- If `request.Id` is non-nil and does not match, return a descriptive error explaining the mismatch.

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "notification")`
2. Validate `--file` is non-empty
3. Read and unmarshal using `readRequestFromFile[types.SenderCaseDTO](path)`
4. Inject or validate the DTO `Id` against the positional ID
5. Resolve connection, project, and client
6. Call `scope.UpdateNotification(ctx, request)`
7. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newDeleteNotificationCmd()`

```go
func newDeleteNotificationCmd() *cobra.Command
```

- `Use`: `"notification <id>"`
- `Short`: `"Delete a notification rule"`
- `Args`: `cobra.ExactArgs(1)` (the numeric notification ID)
- `RunE`: `withPrintedErrors(runDeleteNotification)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "notification")`
2. Resolve connection, project, and client
3. Call `scope.DeleteNotification(ctx, id)`
4. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### Table formatting

```go
func notificationHeaders() []string {
    return []string{"ID", "RULE NAME", "SEND CASE", "ENABLED", "RECIPIENTS"}
}

func notificationRow(n *types.SenderCaseDTO) ([]string, []string) {
    recipients := ""
    if n.Recipients != nil {
        recipients = strings.Join(*n.Recipients, ", ")
    }

    return notificationHeaders(), []string{
        formatInt64Ptr(n.Id),
        n.RuleName,
        string(n.SendCase),
        formatBoolPtr(n.Enabled),
        recipients,
    }
}
```

### 2. Register commands in `commands.go`

```go
// get
get.AddCommand(newGetNotificationCmd())

// create
create.AddCommand(newCreateNotificationCmd())

// update
update.AddCommand(newUpdateNotificationCmd())

// delete
del.AddCommand(newDeleteNotificationCmd())
```

### 3. Create `internal/cli/notification_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetNotificationCmd_Flags`** — Verify `--project` / `-p`. No pagination flags. `cobra.NoArgs`.

2. **`TestNewCreateNotificationCmd_Flags`** — Verify `--file` / `-f` and `--project` / `-p`.

3. **`TestNewUpdateNotificationCmd_Flags`** — Verify `--file` / `-f` and `--project` / `-p`. Verify it accepts exactly 1 positional arg.

4. **`TestNewDeleteNotificationCmd_Flags`** — Verify `--project` / `-p`. Verify 1 arg.

5. **`TestUpdateNotificationRequestIDInjection`** — Verify update injects the positional ID when the file omits `Id`, and errors when the file `Id` disagrees with the positional arg.

6. **`TestNotificationRow_Formatting`** — Verify row output for a sample `SenderCaseDTO`.

7. **`TestNotificationRow_NilFields`** — Verify handling of nil `Id`, nil `Enabled`, nil `Recipients`.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Flat resource name.** `notification` is used directly under standard verbs, not nested under `project-setting` or `config`.
- **`--file` for create and update.** Per PR 17, the JSON file is the primary input mode for both commands.
- **Update still takes a positional ID.** Even though `UpdateNotification` only accepts a DTO, the CLI uses `rp update notification <id> --file <path>` so the target is explicit at the call site. The implementation should inject the positional ID into the DTO when omitted, and reject mismatches when the file already includes a conflicting `Id`.
- **No single-get-by-ID.** The API only supports listing all notifications. `cobra.NoArgs` for get.
- **No pagination.** All notifications returned at once.
- **Reuse shared helpers.** Use `parseResourceID`, `readRequestFromFile[types.SenderCaseDTO]`, `formatInt64Ptr`, `formatBoolPtr`, `entryCreatedHeaders`/`entryCreatedRows`, `messageResultHeaders`/`operationResultRows`.

## What NOT to do

- **Do not add pagination flags.**
- **Do not validate `sendCase` or `attributesOperator` values locally.**
- **Do not make update JSON-only.** Keep the positional notification ID in the CLI shape.
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` shows `notification`
- `rp create --help` shows `notification`
- `rp update --help` shows `notification`
- `rp delete --help` shows `notification`
- `rp create notification --help` shows `--file`
- `rp update notification --help` shows a positional ID plus `--file`

## Definition of done

- [ ] `internal/cli/notification.go` created
- [ ] `internal/cli/notification_test.go` created
- [ ] Commands registered in `commands.go`
- [ ] All commands use `--project` / `-p`
- [ ] Create and update use `--file`
- [ ] Update takes a positional ID and injects or validates `SenderCaseDTO.Id`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
