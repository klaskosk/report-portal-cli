# PR 27-30: Settings Commands

## Goal

Add the full `rp setting` command tree:
- `rp setting server {get,update}` — instance-level server settings (admin-only)
- `rp setting notification {list,create,update,delete}` — project notification rules
- `rp setting pattern {list,create,update,delete}` — project pattern templates
- `rp setting issue-subtype {list,create,update,delete}` — project issue sub-types

These are combined because they all share the `setting` parent command and would conflict if implemented separately.

## Context

Phase 7. The client already has:
- `SettingsScope` with `GetServerSettings`, `UpdateServerSettings`
- `ProjectSettingsScope` with notifications, patterns, and issue sub-type CRUD

### Client API surface

**SettingsScope** (via `client.Settings()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `GetServerSettings` | `GetServerSettings(ctx)` | `map[string]string` |
| `UpdateServerSettings` | `UpdateServerSettings(ctx, types.UpdateSettingsRq)` | `*types.OperationCompletionRS` |

**ProjectSettingsScope** (via `client.Project(name).Settings()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `Get` | `Get(ctx)` | `*types.ProjectSettingsResource` |
| `GetNotifications` | `GetNotifications(ctx)` | `[]types.SenderCaseDTO` |
| `CreateNotification` | `CreateNotification(ctx, types.SenderCaseDTO)` | `*types.EntryCreatedRS` |
| `UpdateNotification` | `UpdateNotification(ctx, types.SenderCaseDTO)` | `*types.OperationCompletionRS` |
| `DeleteNotification` | `DeleteNotification(ctx, notificationID int64)` | `*types.OperationCompletionRS` |
| `CreatePatternTemplate` | `CreatePatternTemplate(ctx, types.CreatePatternTemplateRQ)` | `*types.EntryCreatedRS` |
| `UpdatePatternTemplate` | `UpdatePatternTemplate(ctx, id int64, types.UpdatePatternTemplateRQ)` | `*types.OperationCompletionRS` |
| `DeletePatternTemplate` | `DeletePatternTemplate(ctx, id int64)` | `*types.OperationCompletionRS` |
| `CreateIssueSubType` | `CreateIssueSubType(ctx, types.CreateIssueSubTypeRQ)` | `*types.IssueSubTypeCreatedRS` |
| `UpdateIssueSubType` | `UpdateIssueSubType(ctx, types.UpdateOneIssueSubTypeRQ)` | `*types.OperationCompletionRS` |
| `DeleteIssueSubType` | `DeleteIssueSubType(ctx, id int64)` | `*types.OperationCompletionRS` |

### Relevant types

Look up in `pkg/types/`:
- `UpdateSettingsRq` — has Key and Value fields for server settings
- `SenderCaseDTO` — notification rule (has Id, SendCase, Recipients, etc.)
- `CreatePatternTemplateRQ` — pattern template creation
- `UpdatePatternTemplateRQ` — pattern template update
- `CreateIssueSubTypeRQ` — issue sub-type creation (Color, LongName, ShortName, TypeRef)
- `UpdateOneIssueSubTypeRQ` — issue sub-type update
- `IssueSubTypeCreatedRS` — has Id field
- `ProjectSettingsResource` — full settings including IssueSubTypes map

## Implementation plan

### 1. Create `internal/cli/setting.go`

Parent command and server settings:

```
rp setting
├── server
│   ├── get
│   └── update
├── notification
│   ├── list
│   ├── create
│   ├── update
│   └── delete
├── pattern
│   ├── list
│   ├── create
│   ├── update
│   └── delete
└── issue-subtype
    ├── list
    ├── create
    ├── update
    └── delete
```

#### `rp setting server get`
- No args, no project flag (instance-scoped, admin-only)
- Call `client.Settings().GetServerSettings(ctx)`
- Table: KEY, VALUE columns (iterate map)
- JSON: raw map

#### `rp setting server update`
- No positional args
- Required flags: `--key` (string), `--value` (string)
- Or: `--from-file` for full `UpdateSettingsRq`
- Call `client.Settings().UpdateServerSettings(ctx, request)`
- Output: `OperationCompletionRS` message

### 2. Create `internal/cli/setting_notification.go`

#### `rp setting notification list`
- No args. Project-scoped (needs `-p`/`--project`)
- Call `scope.Settings().GetNotifications(ctx)`
- Table columns: ID, SEND CASE, RECIPIENTS (or whatever key fields exist)
- JSON: raw slice

#### `rp setting notification create`
- `--from-file` required (SenderCaseDTO is complex)
- Call `scope.Settings().CreateNotification(ctx, dto)`
- Output: `EntryCreatedRS` (ID)

#### `rp setting notification update`
- `--from-file` required (SenderCaseDTO includes the ID)
- Call `scope.Settings().UpdateNotification(ctx, dto)`
- Output: `OperationCompletionRS` message

#### `rp setting notification delete <id>`
- One positional arg: notification ID (numeric)
- Call `scope.Settings().DeleteNotification(ctx, id)`
- Output: `OperationCompletionRS` message

### 3. Create `internal/cli/setting_pattern.go`

#### `rp setting pattern list`
- No args. Project-scoped.
- Use `scope.Settings().Get(ctx)` to get full settings, extract pattern templates
- Table columns: ID, NAME, TYPE, VALUE (key fields from pattern templates)
- JSON: the pattern templates array

#### `rp setting pattern create`
- Flags OR `--from-file`:
  - `--name` (string, required)
  - `--type` (string, required) — pattern type (e.g., "STRING")
  - `--value` (string, required) — the pattern value
  - `--enabled` (bool, default true)
- Call `scope.Settings().CreatePatternTemplate(ctx, request)`
- Output: `EntryCreatedRS` (ID)

#### `rp setting pattern update <id>`
- One positional arg: pattern template ID
- Flags OR `--from-file`:
  - `--name`, `--type`, `--value`, `--enabled` (all optional, at least one required without file)
- Call `scope.Settings().UpdatePatternTemplate(ctx, id, request)`
- Output: `OperationCompletionRS` message

#### `rp setting pattern delete <id>`
- One positional arg: pattern template ID
- Call `scope.Settings().DeletePatternTemplate(ctx, id)`
- Output: `OperationCompletionRS` message

### 4. Create `internal/cli/setting_issue_subtype.go`

#### `rp setting issue-subtype list`
- No args. Project-scoped.
- Use `scope.Settings().Get(ctx)` to get full settings, extract issue sub-types
- Table columns: ID, LOCATOR, LONG NAME, SHORT NAME, COLOR
- JSON: the issue sub-types from ProjectSettingsResource

#### `rp setting issue-subtype create`
- Flags OR `--from-file`:
  - `--long-name` (string, required)
  - `--short-name` (string, required)
  - `--color` (string, required) — hex color
  - `--type-ref` (string, required) — parent type reference
- Call `scope.Settings().CreateIssueSubType(ctx, request)`
- Output: `IssueSubTypeCreatedRS` (ID)

#### `rp setting issue-subtype update`
- `--from-file` required (UpdateOneIssueSubTypeRQ is complex)
- Call `scope.Settings().UpdateIssueSubType(ctx, request)`
- Output: `OperationCompletionRS` message

#### `rp setting issue-subtype delete <id>`
- One positional arg: issue sub-type ID
- Call `scope.Settings().DeleteIssueSubType(ctx, id)`
- Output: `OperationCompletionRS` message

### 5. Register in root.go

Add `cmd.AddCommand(newSettingCmd())` in `newRootCmd()`.

The `setting` parent gets a persistent `--project`/`-p` flag for the project-scoped subcommands. The `server` subcommand ignores it.

## Test requirements

### Unit tests (split across files matching implementation)

- Server get/update: command wiring, update requires --key and --value
- Notification: list needs project, create/update need --from-file, delete needs ID
- Pattern: create needs --name/--type/--value, update needs ID + at least one flag or file
- Issue sub-type: create needs --long-name/--short-name/--color/--type-ref, delete needs ID
- Command wiring: full tree structure verified

## Edge cases

- **Mixed scoping:** `server` is instance-scoped (admin), everything else is project-scoped
- **Pattern/issue-subtype list from Get:** The API doesn't have a dedicated list endpoint for patterns or issue sub-types. Use `Settings().Get()` and extract the relevant section.
- **Complex types prefer `--from-file`:** Notifications and issue-subtype updates are complex enough that `--from-file` is the primary input method
- **Project flag placement:** Put `-p`/`--project` as persistent on the `setting` parent so notification/pattern/issue-subtype inherit it. Server commands just ignore it.

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [ ] `internal/cli/setting.go` — parent command + server get/update
- [ ] `internal/cli/setting_notification.go` — notification CRUD
- [ ] `internal/cli/setting_pattern.go` — pattern CRUD
- [ ] `internal/cli/setting_issue_subtype.go` — issue sub-type CRUD
- [ ] `rp setting` registered in `root.go`
- [ ] Unit tests cover all subcommands
- [ ] `make lint` and `make test` pass
- [ ] This file moved to `prompts/completed/`
