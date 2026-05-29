# PR 22: Item Issue Linking

## Goal

Add `rp item issue {link,unlink}` as sub-resource commands under item for linking/unlinking external BTS issues to test items.

## Context

Phase 7. The item command already has list/get/update/delete. The `TestItemScope` client already has `LinkExternalIssues` and `UnlinkExternalIssues`.

### Client API surface

**TestItemScope** (via `client.Project(name).TestItems()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `LinkExternalIssues` | `LinkExternalIssues(ctx, types.LinkExternalIssueRQ)` | `[]types.OperationCompletionRS` |
| `UnlinkExternalIssues` | `UnlinkExternalIssues(ctx, types.UnlinkExternalIssueRQ)` | `[]types.OperationCompletionRS` |

### Relevant types

```go
type LinkExternalIssueRQ struct {
    Issues      types.ExternalSystemIssue `json:"issues"`
    TestItemIds []int64                   `json:"testItemIds"`
}

type ExternalSystemIssue struct {
    BtsProject *string `json:"btsProject,omitempty"`
    BtsUrl     *string `json:"btsUrl,omitempty"`
    TicketId   *string `json:"ticketId,omitempty"`
    Url        *string `json:"url,omitempty"`
}

type UnlinkExternalIssueRQ struct {
    TestItemIds []int64 `json:"testItemIds"`
    TicketIds   []string `json:"ticketIds"`
}
```

Note: Check `pkg/types/` for the exact struct definitions — they may differ slightly.

## Implementation plan

### 1. Create `internal/cli/item_issue.go`

#### `rp item issue link`

- No positional args
- Required flags:
  - `--item-id` (int64 slice, repeatable) — test item IDs to link
  - `--ticket-id` (string) — external ticket ID (e.g., "PROJ-123")
  - `--url` (string) — URL of the external ticket
- Optional flags:
  - `--bts-url` (string) — BTS instance URL
  - `--bts-project` (string) — BTS project key
  - `--from-file` (string) — reads full request from JSON
- Build `types.LinkExternalIssueRQ` from flags
- Call `scope.LinkExternalIssues(ctx, request)`
- Output: Print each `OperationCompletionRS` message

#### `rp item issue unlink`

- No positional args
- Required flags:
  - `--item-id` (int64 slice, repeatable) — test item IDs to unlink from
  - `--ticket-id` (string slice, repeatable) — ticket IDs to unlink
- Optional: `--from-file` (string) — reads full request from JSON
- Build `types.UnlinkExternalIssueRQ` from flags
- Call `scope.UnlinkExternalIssues(ctx, request)`
- Output: Print each `OperationCompletionRS` message

### 2. Register in item.go

Add to `newItemCmd()`:
```go
cmd.AddCommand(newItemIssueCmd())
```

## Test requirements

### Unit tests (`internal/cli/item_issue_test.go`)

- **Link:** `--item-id`, `--ticket-id`, `--url` required (unless `--from-file`)
- **Unlink:** `--item-id`, `--ticket-id` required (unless `--from-file`)
- **From-file:** overrides flags
- **Command wiring:** item has `issue` subcommand with `link` and `unlink`

## Edge cases

- **Project-scoped:** Inherits `-p`/`--project` from item parent
- **Multiple items:** `--item-id` can be repeated to link/unlink multiple items at once
- **`--from-file` takes precedence**
- **Returns slice:** Both operations return `[]OperationCompletionRS` — iterate and print each message

## Example commands

```bash
# Link an issue to test items
$ rp item issue link --item-id 100 --item-id 101 --ticket-id "PROJ-456" --url "https://jira.example.com/browse/PROJ-456" -p my_project
MESSAGE
External issue has been linked successfully.
External issue has been linked successfully.

# Unlink issues from test items
$ rp item issue unlink --item-id 100 --ticket-id "PROJ-456" -p my_project
MESSAGE
External issue has been unlinked successfully.
```

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [ ] `internal/cli/item_issue.go` with link and unlink commands
- [ ] Registered in `newItemCmd()`
- [ ] Unit tests cover flag validation and command wiring
- [ ] Existing item subcommands unchanged
- [ ] `make lint` and `make test` pass
- [ ] This file moved to `prompts/completed/`
