# PR 27: Export and Report-Download Commands

## Goal

Implement `export project`, `export user`, and `get launch-report` commands. All three return raw bytes (CSV or binary), using a consistent bytes-output UX: stdout by default with an `--output-file <path>` option for writing to a file.

## Context

Export operations span multiple scopes:
- `ProjectsScope` (`client.Projects()`): backs `rp export project` (instance-scoped)
- `UsersScope` (`client.Users()`): backs `rp export user` (instance-scoped)
- `LaunchScope` (`client.Project(name).Launches()`): backs `rp get launch-report` (project-scoped)

**Prerequisite PRs and dependencies:**
- PR 2 (verb group scaffolding: `export` parent command)
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `parseResourceID`, `mustGetStringFlag`)
- PR 17 (frozen UX: §8–9 export and report bytes-output convention)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `ProjectsScope.Export` | `ctx`, `options ...ExportProjectsOption` | `([]byte, error)` |
| `UsersScope.Export` | `ctx`, `options ...ListUsersOption` | `([]byte, error)` |
| `LaunchScope.GetReport` | `ctx`, `launchID int64` | `([]byte, error)` |

Export options include filters like `WithExportFilterName`, `WithExportFilterType`, `WithUsersRole`, `WithUsersType`, `WithUsersActive`.

## Scope

Create `internal/cli/export.go` and `internal/cli/export_test.go`. Add the `get launch-report` command — this can go in `export.go` alongside the export commands since they share the bytes-output helper. Register in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp export project [--view <view>] [--filter-name <name>] [--filter-type <type>] [--output-file <path>]` | Export projects as CSV |
| `rp export user [--role <role>] [--type <type>] [--active <bool>] [--email <email>] [--output-file <path>]` | Export users as CSV |
| `rp get launch-report <id> [--output-file <path>]` | Download a launch report |

## Implementation

### 1. Create `internal/cli/export.go`

#### Bytes-output helper

```go
func writeBytes(cmd *cobra.Command, data []byte) error {
    outputFile := mustGetStringFlag(cmd, "output-file")

    if outputFile != "" {
        if err := os.WriteFile(outputFile, data, 0o644); err != nil {
            return fmt.Errorf("writing to %s: %w", outputFile, err)
        }

        fmt.Fprintf(os.Stderr, "Written to %s (%d bytes)\n", outputFile, len(data))

        return nil
    }

    _, err := os.Stdout.Write(data)

    return err
}
```

When `--output-file` is set, the data is written to the file and a confirmation message is printed to stderr (so stdout stays clean for piping). When not set, raw bytes go to stdout.

Before adding a new helper, check whether the existing download-path helper can be reused directly. PR 17 explicitly allows either reusing `writeDownloadedContent` if its interface already matches this UX, or extracting a shared bytes-output helper (for example `writeRawOutput` or the local `writeBytes` shown here) if reuse would be awkward.

**Important:** Export commands bypass the `Printer` infrastructure entirely. The global `-o` / `--output` flag (table/json/go-template) is **not applicable** to bytes-output commands. If the user explicitly sets `-o json` or `-o table`, the command should print a warning to stderr and proceed with raw bytes output regardless:

```go
func warnIfFormatFlagSet(cmd *cobra.Command) {
    if cmd.Flags().Changed("output") {
        fmt.Fprintln(os.Stderr, "Note: --output flag is ignored for binary output commands")
    }
}
```

**How this works:** The `"output"` flag is the persistent flag registered on the root command (`-o` / `--output`). Because Cobra propagates persistent flags down the command tree, `cmd.Flags().Changed("output")` returns `true` on any subcommand when the user explicitly passed `-o` at any level. This is the same `Changed` pattern used for optional flags elsewhere in the CLI — it distinguishes "user explicitly set the flag" from "flag has its default value."

Call `warnIfFormatFlagSet(cmd)` at the start of each export run function.

#### `newExportProjectCmd()`

```go
func newExportProjectCmd() *cobra.Command
```

- `Use`: `"project"`
- `Short`: `"Export projects as CSV"`
- `Long`: Include examples showing stdout (piping) and `--output-file`. Mention this is instance-scoped. Explicitly state: "This command produces raw binary output. The -o/--output format flag is ignored."
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runExportProject)`
- Flags:
  - `--output-file` (string, optional): path to write the export file
  - `--view` (string, optional): export view selector
  - `--filter-name` (string, optional): filter by project name
  - `--filter-type` (string, optional): filter by project type (e.g., `INTERNAL`)

**Do NOT** register `--project` — project export is instance-scoped.

The run function:
1. `warnIfFormatFlagSet(cmd)`
2. Build export options from flags:
   - If `--view` is set: add `client.WithExportView(view)`
   - If `--filter-name` is set: add `client.WithExportFilterName(name)`
   - If `--filter-type` is set: add `client.WithExportFilterType(projectType)`
3. Resolve connection and client
4. Call `scope.Export(ctx, options...)`
5. Call `writeBytes(cmd, data)`

#### `newExportUserCmd()`

```go
func newExportUserCmd() *cobra.Command
```

- `Use`: `"user"`
- `Short`: `"Export users as CSV"`
- `Long`: Include examples showing stdout and `--output-file`. Mention this is instance-scoped. Explicitly state: "This command produces raw binary output. The -o/--output format flag is ignored."
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runExportUser)`
- Flags:
  - `--output-file` (string, optional): path to write the export file
  - `--role` (string, optional): filter by role (e.g., `ADMINISTRATOR`, `USER`)
  - `--type` (string, optional): filter by account type (e.g., `INTERNAL`, `SCIM`)
  - `--active` (bool, optional): filter by active status
  - `--email` (string, optional): filter by email address

The run function:
1. `warnIfFormatFlagSet(cmd)`
2. Build options from flags:
   - If `--role` is set: add `client.WithUsersRole(role)`
   - If `--type` is set: add `client.WithUsersType(accountType)`
   - If `--active` is set: add `client.WithUsersActive(active)`
   - If `--email` is set: add `client.WithUsersEmail(email)`
3. Resolve connection and client
4. Call `scope.Export(ctx, options...)`
5. Call `writeBytes(cmd, data)`

#### `newGetLaunchReportCmd()`

```go
func newGetLaunchReportCmd() *cobra.Command
```

- `Use`: `"launch-report <id>"`
- `Short`: `"Download a launch report"`
- `Long`: Include example showing `--output-file` usage. Note this is project-scoped. Explicitly state: "This command produces raw binary output. The -o/--output format flag is ignored."
- `Args`: `cobra.ExactArgs(1)` (the numeric launch ID)
- `RunE`: `withPrintedErrors(runGetLaunchReport)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--output-file` (string, optional): path to write the report file

The run function:
1. `warnIfFormatFlagSet(cmd)`
2. Parse `args[0]` as int64 using `parseResourceID(args[0], "launch")`
3. Resolve connection, project, and client
4. Call `scope.GetReport(ctx, launchID)`
5. Call `writeBytes(cmd, data)`

### 2. Register commands in `commands.go`

```go
// export
export.AddCommand(newExportProjectCmd())
export.AddCommand(newExportUserCmd())

// get
get.AddCommand(newGetLaunchReportCmd())
```

### 3. Create `internal/cli/export_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewExportProjectCmd_Flags`** — Verify `--output-file`, `--view`, `--filter-name`, `--filter-type`. No `--project`. Verify `cobra.NoArgs`.

2. **`TestNewExportUserCmd_Flags`** — Verify `--output-file`, `--role`, `--type`, `--active`, `--email`. Verify `cobra.NoArgs`.

3. **`TestNewGetLaunchReportCmd_Flags`** — Verify `--output-file` and `--project` / `-p`. Verify 1 arg.

4. **`TestWriteBytes_ToFile`** — Write bytes to a temp file via `writeBytes`, verify contents.

5. **`TestWriteBytes_NoFile`** — Verify `writeBytes` with empty `output-file` writes to stdout (redirect stdout in test).

6. **`TestNoProjectFlagOnExportCommands`** — Verify export project and export user do not have `--project`.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Stdout by default, `--output-file` to redirect.** This matches the kubectl convention for raw data output. Stdout allows piping (`rp export project | head`).
- **Bypasses `Printer` infrastructure.** Export commands produce raw bytes, not structured data. The `-o` / `--output` format flag is not applicable.
- **Warning for format flag.** If the user explicitly sets `-o json`, they get a warning rather than a silent ignore. This prevents confusion.
- **Confirmation to stderr.** When writing to a file, the "Written to..." message goes to stderr so it doesn't corrupt piped output.
- **Prefer reusing the existing download-path helper when compatible.** PR 17 allows either reusing `writeDownloadedContent` or extracting a shared helper for bytes-output commands. Avoid creating a second near-duplicate helper if the existing one already fits.
- **Filter flags are optional.** Without filters, all projects/users are exported.
- **Match the frozen flag names exactly.** `export project` uses `--view`, `--filter-name`, and `--filter-type`. `export user` uses `--role`, `--type`, `--active`, and `--email`.
- **Defer `--filter-last-run`.** PR 17 leaves date-based export filtering as a follow-on because date parsing ergonomics need separate treatment.
- **`launch-report` is project-scoped.** It requires project resolution, unlike the instance-scoped exports.
- **Reuse shared helpers.** Use `parseResourceID`, `mustGetStringFlag`.

## What NOT to do

- **Do not add `--project` to export project or export user.** They are instance-scoped.
- **Do not attempt to format bytes as table or JSON.** These are raw byte streams.
- **Do not implement `launch import`.** That is deferred per the PR plan.
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp export --help` shows `project` and `user`
- `rp get --help` shows `launch-report`
- `rp export project --help` shows `--output-file` and filter flags
- `rp export user --help` shows `--role`, `--type`, `--active`, `--email`, and `--output-file`
- `rp get launch-report --help` shows `--output-file` and `--project`

## Definition of done

- [ ] `internal/cli/export.go` created with all three commands and the `writeBytes` helper
- [ ] `internal/cli/export_test.go` created
- [ ] Commands registered in `commands.go`
- [ ] Bytes-output UX: stdout default, `--output-file` for file output
- [ ] Format flag warning when `-o` is explicitly set
- [ ] Export commands are instance-scoped (no `--project`)
- [ ] `get launch-report` is project-scoped
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
