# PR 34: `bts` Commands

## Goal

Implement bug tracking system (BTS) commands for ticket lookup, field inspection, issue type discovery, and issue creation. These are project-scoped commands that interact with BTS integrations (e.g., JIRA, Rally) configured in a project.

## Context

BTS operations are managed through `BugTrackingScope` on the client (`client.Project(name).BugTracking()`). They are **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `readRequestFromFile`, `formatStringPtr`)
- PR 17 (frozen UX: §5 BTS commands under standard verbs)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `GetTicket` | `ctx`, `ticketID string`, `btsURL string`, `btsProject string` | `(*types.Ticket, error)` |
| `GetSystemFields` | `ctx`, `integrationID int64` | `([]types.PostFormField, error)` |
| `GetAllowableIssueTypes` | `ctx`, `integrationID int64` | `([]string, error)` |
| `CreateIssue` | `ctx`, `integrationID int64`, `request types.PostTicketRQ` | `(*types.Ticket, error)` |

### Key types

```go
type Ticket struct {
    Id         *string `json:"id,omitempty"`
    PluginName *string `json:"pluginName,omitempty"`
    Status     *string `json:"status,omitempty"`
    Summary    *string `json:"summary,omitempty"`
    Url        *string `json:"url,omitempty"`
}

type PostFormField struct {
    CommandName   *string         `json:"commandName,omitempty"`
    DefinedValues *[]AllowedValue `json:"definedValues,omitempty"`
    Description   *string         `json:"description,omitempty"`
    FieldName     *string         `json:"fieldName,omitempty"`
    FieldType     *string         `json:"fieldType,omitempty"`
    Id            string          `json:"id"`
    NamedValue    *[]NamedValue   `json:"namedValue,omitempty"`
    Required      bool            `json:"required"`
    Value         *[]string       `json:"value,omitempty"`
}

type PostTicketRQ struct {
    BackLinks          map[string]string `json:"backLinks"`
    Domain             *string           `json:"domain,omitempty"`
    Fields             []PostFormField   `json:"fields"`
    IncludeComments    *bool             `json:"includeComments,omitempty"`
    IncludeData        *bool             `json:"includeData,omitempty"`
    IncludeLogs        *bool             `json:"includeLogs,omitempty"`
    IncludeScreenshots *bool             `json:"includeScreenshots,omitempty"`
    Item               int64             `json:"item"`
    LogQuantity        *int32            `json:"logQuantity,omitempty"`
    Password           *string           `json:"password,omitempty"`
    Token              *string           `json:"token,omitempty"`
    Username           *string           `json:"username,omitempty"`
}
```

## Scope

Create `internal/cli/bts.go` and `internal/cli/bts_test.go`. Register the new commands in `internal/cli/commands.go`.

This file should also define lightweight `bts` grouping commands under both `get` and `create`, with the leaf commands attached beneath them.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp get bts ticket <ticket-id> --bts-url <url> --bts-project <project>` | Look up a BTS ticket |
| `rp get bts fields --integration <id>` | Get available system fields for an integration |
| `rp get bts issue-types --integration <id>` | Get allowable issue types for an integration |
| `rp create bts issue --integration <id> --file <path>` | Create a BTS issue |

## Implementation

### 1. Create `internal/cli/bts.go`

#### `newGetBTSTicketCmd()`

```go
func newGetBTSTicketCmd() *cobra.Command
```

- `Use`: `"ticket <ticket-id>"`
- `Short`: `"Look up a BTS ticket"`
- `Long`: Include example with all required flags.
- `Args`: `cobra.ExactArgs(1)` (the ticket ID string, e.g., `"PROJECT-123"`)
- `RunE`: `withPrintedErrors(runGetBTSTicket)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--bts-url` (string, required): BTS instance URL
  - `--bts-project` (string, required): BTS project name/key

The run function:
1. Read `args[0]` as the ticket ID string
2. Validate `--bts-url` is non-empty
3. Validate `--bts-project` is non-empty
4. Resolve connection, project, and client
5. Call `scope.GetTicket(ctx, ticketID, btsURL, btsProject)`
6. Print the `Ticket` result

**Table formatting for `Ticket`:**

```go
func ticketHeaders() []string {
    return []string{"ID", "SUMMARY", "STATUS", "URL", "PLUGIN"}
}

func ticketRow(ticket *types.Ticket) ([]string, []string) {
    return ticketHeaders(), []string{
        formatStringPtr(ticket.Id),
        formatStringPtr(ticket.Summary),
        formatStringPtr(ticket.Status),
        formatStringPtr(ticket.Url),
        formatStringPtr(ticket.PluginName),
    }
}
```

#### `newGetBTSFieldsCmd()`

```go
func newGetBTSFieldsCmd() *cobra.Command
```

- `Use`: `"fields"`
- `Short`: `"Get available system fields for a BTS integration"`
- `Long`: Include example. Explain this is useful for discovering field structure before creating an issue.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGetBTSFields)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--integration` (int64, required): integration ID

The run function:
1. Validate `--integration` was explicitly provided via `cmd.Flags().Changed("integration")` and is positive
2. Resolve connection, project, and client
3. Call `scope.GetSystemFields(ctx, integrationID)` — returns `[]types.PostFormField`
4. Pass the slice to `printer.Print`

**Table formatting for `PostFormField`:**

```go
func btsFieldHeaders() []string {
    return []string{"ID", "FIELD NAME", "FIELD TYPE", "REQUIRED", "DESCRIPTION"}
}

func btsFieldRow(field *types.PostFormField) ([]string, []string) {
    return btsFieldHeaders(), []string{
        field.Id,
        formatStringPtr(field.FieldName),
        formatStringPtr(field.FieldType),
        strconv.FormatBool(field.Required),
        formatStringPtr(field.Description),
    }
}
```

`DefinedValues`, `NamedValue`, and `Value` are complex nested fields omitted from the table. Use `-o json` for full details.

#### `newGetBTSIssueTypesCmd()`

```go
func newGetBTSIssueTypesCmd() *cobra.Command
```

- `Use`: `"issue-types"`
- `Short`: `"Get allowable issue types for a BTS integration"`
- `Long`: Include example.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGetBTSIssueTypes)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--integration` (int64, required): integration ID

The run function:
1. Validate `--integration` was explicitly provided and is positive
2. Resolve connection, project, and client
3. Call `scope.GetAllowableIssueTypes(ctx, integrationID)` — returns `[]string`
4. Print the result

**Table formatting for `[]string`:**

```go
func issueTypeHeaders() []string {
    return []string{"ISSUE TYPE"}
}

func issueTypeRows(types []string) [][]string {
    rows := make([][]string, len(types))
    for i, t := range types {
        rows[i] = []string{t}
    }

    return rows
}
```

#### `newCreateBTSIssueCmd()`

```go
func newCreateBTSIssueCmd() *cobra.Command
```

- `Use`: `"issue"`
- `Short`: `"Create a BTS issue"`
- `Long`: Include example showing `--integration` and `--file`. Note the JSON structure includes form fields, back links, and test item reference.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runCreateBTSIssue)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--integration` (int64, required): integration ID
  - `--file` / `-f` (string, required): path to JSON file containing `PostTicketRQ`

**Why `--file`?** Per PR 17's file-flag rule, `create bts issue` uses `--file` because the request JSON is the command's primary input mode. `PostTicketRQ` contains `Fields []PostFormField`, `BackLinks`, and related issue-construction details that do not fit ergonomic inline flags.

Sample JSON (include in `Long` help text):

```json
{
  "item": 42,
  "backLinks": {
    "https://reportportal.example.com": "superadmin_personal"
  },
  "fields": [
    {
      "id": "summary",
      "fieldName": "Summary",
      "fieldType": "string",
      "required": true,
      "value": ["Test failure in login flow"]
    },
    {
      "id": "issuetype",
      "fieldName": "Issue Type",
      "fieldType": "string",
      "required": true,
      "value": ["Bug"]
    }
  ],
  "includeLogs": true,
  "includeScreenshots": false
}
```

The run function:
1. Validate `--integration` was explicitly provided and is positive
2. Validate `--file` is non-empty
3. Read and unmarshal using `readRequestFromFile[types.PostTicketRQ](path)`
4. Resolve connection, project, and client
5. Call `scope.CreateIssue(ctx, integrationID, request)`
6. Print the `Ticket` result using `ticketRow`

### 2. Register commands in `commands.go`

```go
// get
getBTS := newGetBTSCmd()
getBTS.AddCommand(newGetBTSTicketCmd())
getBTS.AddCommand(newGetBTSFieldsCmd())
getBTS.AddCommand(newGetBTSIssueTypesCmd())
get.AddCommand(getBTS)

// create
createBTS := newCreateBTSCmd()
createBTS.AddCommand(newCreateBTSIssueCmd())
create.AddCommand(createBTS)
```

### 3. Create `internal/cli/bts_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGetBTSTicketCmd_Flags`** — Verify `--bts-url`, `--bts-project`, `--project` / `-p`. Verify 1 arg.

2. **`TestNewGetBTSFieldsCmd_Flags`** — Verify `--integration` and `--project` / `-p`. Verify `cobra.NoArgs`.

3. **`TestNewGetBTSIssueTypesCmd_Flags`** — Verify `--integration` and `--project` / `-p`. Verify `cobra.NoArgs`.

4. **`TestNewCreateBTSIssueCmd_Flags`** — Verify `--integration`, `--file` / `-f`, and `--project` / `-p`.

5. **`TestTicketRow_Formatting`** — Verify row output for a sample `Ticket`.

6. **`TestTicketRow_NilFields`** — Verify nil handling.

7. **`TestBTSFieldRow_Formatting`** — Verify row output for a sample `PostFormField`.

8. **`TestIssueTypeRows_WithData`** — Verify row output for a sample `[]string`.

9. **`TestIssueTypeRows_Empty`** — Verify empty slice.

10. **`TestCreateBTSIssue_FileUnmarshal`** — Write a minimal `PostTicketRQ` JSON to a temp file, call `readRequestFromFile[types.PostTicketRQ](path)`, and verify the unmarshalled struct has the expected `Item`, `BackLinks`, and `Fields` values. This catches JSON structure mismatches early since `PostTicketRQ` is deeply nested and the implementer won't see real API errors until integration.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Grouped under `bts` beneath standard verbs.** Per PR 17, BTS commands stay under `get` and `create`, but are namespaced one level lower as `bts` subcommands rather than flat `bts-*` resource names.
- **Multi-flag ticket lookup.** `GetTicket` requires three parameters (ticket ID, BTS URL, BTS project). The ticket ID is a positional arg; the other two are flags. This is the most natural mapping.
- **`--integration` uses `Changed` validation.** Same pattern as other int64 flag validation.
- **`--file` for issue creation.** The request JSON is the primary input mode for `create bts issue`.
- **`GetTicketIds` and `GetTicketIdsForProject` are not implemented.** These are autocomplete helpers, not primary user-facing commands. They can be added later if needed.
- **Reuse shared helpers.** Use `readRequestFromFile`, `formatStringPtr`. Note: `parseResourceID` is not needed here (ticket ID is a string, not int64; integration ID is read via `cmd.Flags().GetInt64`). `formatBoolPtr` is not needed (the `Required` field on `PostFormField` is `bool`, not `*bool`, so use `strconv.FormatBool` directly).

## What NOT to do

- **Do not implement `GetTicketIds` or `GetTicketIdsForProject`.** Those are autocomplete utilities.
- **Do not validate BTS URLs or ticket ID formats locally.**
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp get --help` shows `bts`
- `rp create --help` shows `bts`
- `rp get bts --help` shows `ticket`, `fields`, `issue-types`
- `rp create bts --help` shows `issue`
- `rp get bts ticket --help` shows `--bts-url`, `--bts-project`, `--project`
- `rp get bts fields --help` shows `--integration`, `--project`
- `rp create bts issue --help` shows `--integration`, `--file`, `--project`

## Definition of done

- [ ] `internal/cli/bts.go` created
- [ ] `internal/cli/bts_test.go` created
- [ ] Commands registered in `commands.go` under `get bts` / `create bts`
- [ ] All commands use `--project` / `-p`
- [ ] Ticket lookup uses positional ID + `--bts-url` + `--bts-project`
- [ ] Fields and issue types use `--integration`
- [ ] Issue creation uses `--integration` + `--file`
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
