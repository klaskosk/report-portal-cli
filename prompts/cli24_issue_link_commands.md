# PR 24: Issue Link Commands

## Goal

Implement `link` and `unlink` commands for managing external issue associations on test items. These allow users to attach and detach bug tracking tickets (e.g., JIRA issues) to/from test items.

## Context

Issue linking is managed through `TestItemScope` on the client (`client.Project(name).Items()`). Commands are **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 2 (verb group scaffolding: `link` and `unlink` parent commands)
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `readRequestFromFile`, `parseResourceID`, `messageResultHeaders`/`operationResultRows`)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `LinkExternalIssues` | `ctx`, `request types.LinkExternalIssueRQ` | `([]types.OperationCompletionRS, error)` |
| `UnlinkExternalIssues` | `ctx`, `request types.UnlinkExternalIssueRQ` | `([]types.OperationCompletionRS, error)` |

**Notable:** Both methods return `[]types.OperationCompletionRS` (a slice, not a single result). The client internally restructures the `LinkExternalIssueRQ` body to match the API's expected format.

### Key types

```go
type LinkExternalIssueRQ struct {
    Issues      ExternalSystemIssue `json:"issues"`
    TestItemIds []int64             `json:"testItemIds"`
}

type ExternalSystemIssue struct {
    BtsProject string     `json:"btsProject"`
    BtsUrl     string     `json:"btsUrl"`
    PluginName *string    `json:"pluginName,omitempty"`
    SubmitDate *time.Time `json:"submitDate,omitempty"`
    TicketId   string     `json:"ticketId"`
    Url        string     `json:"url"`
}

type UnlinkExternalIssueRQ struct {
    TestItemIds []int64  `json:"testItemIds"`
    TicketIds   []string `json:"ticketIds"`
}
```

## Scope

Create `internal/cli/link.go` and `internal/cli/link_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp link issue --file <path>` | Link an external issue to test items |
| `rp unlink issue --item <id> --ticket <ticket-id> [--ticket <ticket-id>...]` | Unlink external issues from a test item |

## Implementation

### 1. Create `internal/cli/link.go`

#### `newLinkIssueCmd()`

```go
func newLinkIssueCmd() *cobra.Command
```

- `Use`: `"issue"`
- `Short`: `"Link an external issue to test items"`
- `Long`: Include example showing `--file`. Include sample JSON showing a `LinkExternalIssueRQ` with `testItemIds` and `issues` (the external system issue details).
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runLinkIssue)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `LinkExternalIssueRQ`

**Why `--file`?** Per PR 17's file-flag rule, this command uses `--file` because the JSON document is the command's primary input mode. `LinkExternalIssueRQ` contains nested `ExternalSystemIssue` fields plus an array of item IDs, so inline flags would be awkward.

Sample JSON:

```json
{
  "testItemIds": [101, 102],
  "issues": {
    "btsProject": "PROJECT",
    "btsUrl": "https://jira.example.com",
    "ticketId": "PROJECT-123",
    "url": "https://jira.example.com/browse/PROJECT-123"
  }
}
```

The run function:
1. Validate `--file` is non-empty
2. Read and unmarshal using `readRequestFromFile[types.LinkExternalIssueRQ](path)`
3. Resolve connection, project, and client
4. Obtain scope: `scope := clientInstance.Project(project).Items()`
5. Call `scope.LinkExternalIssues(ctx, request)`
6. Print the result — handle the `[]types.OperationCompletionRS` slice

**Result formatting for `[]OperationCompletionRS`:**

```go
func linkResultRows(results []types.OperationCompletionRS) [][]string {
    rows := make([][]string, 0, len(results))
    for i := range results {
        rows = append(rows, []string{formatStringPtr(results[i].Message)})
    }

    return rows
}
```

Print using: `printer.Print(results, messageResultHeaders(), linkResultRows(results))`

#### `newUnlinkIssueCmd()`

```go
func newUnlinkIssueCmd() *cobra.Command
```

- `Use`: `"issue"`
- `Short`: `"Unlink external issues from a test item"`
- `Long`: Include examples showing `--item` and `--ticket` usage. Note that `--ticket` can be repeated.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runUnlinkIssue)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--item` (int64 slice, required): test item IDs to unlink from
  - `--ticket` (string slice, required): ticket IDs to unlink

**Why inline flags for unlink?** `UnlinkExternalIssueRQ` has only two array fields of simple types (int64 IDs and string ticket IDs). Repeatable flags are a natural fit: `--item 101 --item 102 --ticket PROJECT-123`.

The run function:
1. Read `--item` as `[]int64` via `cmd.Flags().GetInt64Slice("item")` — note: `testItemIds` in the request type is plural; `--item` can be repeated to target multiple items at once
2. Read `--ticket` as `[]string` via `cmd.Flags().GetStringSlice("ticket")` — also repeatable
3. Validate at least one item ID and one ticket ID are provided
4. Build `types.UnlinkExternalIssueRQ{TestItemIds: itemIDs, TicketIds: ticketIDs}`
5. Resolve connection, project, and client
6. Obtain scope: `scope := clientInstance.Project(project).Items()`
7. Call `scope.UnlinkExternalIssues(ctx, request)`
8. Print via `printer.Print(results, messageResultHeaders(), linkResultRows(results))`

### 2. Register commands in `commands.go`

```go
// link
link.AddCommand(newLinkIssueCmd())

// unlink
unlink.AddCommand(newUnlinkIssueCmd())
```

### 3. Create `internal/cli/link_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewLinkIssueCmd_Flags`** — Verify `--file` / `-f` and `--project` / `-p` flags. Verify `cobra.NoArgs`.

2. **`TestNewUnlinkIssueCmd_Flags`** — Verify `--item`, `--ticket`, and `--project` / `-p` flags. Verify `cobra.NoArgs`.

3. **`TestLinkResultRows_MultipleResults`** — Build a `[]OperationCompletionRS` with two entries, verify row output.

4. **`TestLinkResultRows_Empty`** — Verify empty slice produces empty rows.

5. **`TestLinkResultRows_NilMessages`** — Verify nil messages produce empty strings.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **`--file` for link, inline flags for unlink.** Linking uses `--file` because the request JSON is the only input mode. Unlinking requires only two simple lists (item IDs, ticket IDs), so repeatable flags are cleaner.
- **`testItemIds` is plural — multiple items can be linked at once.** The `--item` flag for unlink is repeatable (e.g., `--item 101 --item 102`), and the `testItemIds` array in the link JSON can contain multiple IDs. Each operation applies to all listed items.
- **Slice result handling.** Both methods return `[]OperationCompletionRS`. Each entry in the slice corresponds to one operation's result. The table shows one row per result.
- **Route all output through `printer.Print`.** Use `printer.Print(results, messageResultHeaders(), linkResultRows(results))`. This ensures JSON, Go-template, and table modes work correctly.
- **Reuse shared helpers.** Use `readRequestFromFile[types.LinkExternalIssueRQ]`, `messageResultHeaders`, `formatStringPtr`, `withPrintedErrors`.
- **Command registration follows PR 3's centralized command-tree layout.** Shared mutation/result helpers assume PR 7's naming conventions.

## What NOT to do

- **Do not validate ticket ID format.** Different BTS systems use different formats.
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp link --help` shows `issue` as a subcommand
- `rp unlink --help` shows `issue` as a subcommand
- `rp link issue --help` shows `--file` and `--project`
- `rp unlink issue --help` shows `--item`, `--ticket`, and `--project`

## Definition of done

- [ ] `internal/cli/link.go` created
- [ ] `internal/cli/link_test.go` created
- [ ] Commands registered under `link` and `unlink` parents
- [ ] Both commands use `--project` / `-p`
- [ ] Link uses `--file` for the request payload
- [ ] Unlink uses repeatable `--item` and `--ticket` flags
- [ ] Slice results formatted with one row per `OperationCompletionRS`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
