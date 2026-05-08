# PR 5: Extract Shared CLI Helpers from Pilot Slices

## Goal

Consolidate patterns that appeared in at least two pilot PRs (filter, group, widget) into reusable helpers. This creates the shared foundation that Phase 4 (parallel resource fan-out) will build upon. Extract only what is clearly duplicated — do not speculatively create helpers for patterns that haven't appeared yet.

## Context

After PRs 2–4 land, the following patterns are duplicated:

1. **`--from-file` JSON loading** — both `filter.go` and `widget.go` have nearly identical `readXFromFile` functions that read a path, unmarshal JSON, and return the typed struct.
2. **Numeric ID parsing from positional arguments** — `filter.go`, `group.go`, and `widget.go` all parse a positional arg as `int64` with the same error format.
3. **Operation result printing** — `filter.go` and `group.go` both print `OperationCompletionRS` and `SuccessfulUpdate` (which have the same shape: a message field). The `user.go` file also has `operationResultHeaders`/`operationResultRows`.
4. **Printer/connection/client boilerplate** — every command handler starts with the same ~10 lines (resolve output format, create printer, resolve connection, create client). This preamble is identical across all commands.
5. **Project resolution boilerplate** — project-scoped commands add `resolveProject` after connection resolution.
6. **Test scaffolding** — if the pilot tests introduced duplicated command/setup helpers, extract only the concrete helpers that are reused in at least two test files.

### Prerequisites

- PRs 1–4 have landed
- The patterns described below are visible as concrete duplication in the codebase

## Implementation plan

### 1. Extract `readRequestFromFile[T]` — generic JSON file reader

Create a generic helper in a new file `internal/cli/fromfile.go`:

```go
func readRequestFromFile[T any](path string) (T, error) {
    var zero T

    var (
        data []byte
        err  error
        name = path
    )

    if path == "-" {
        data, err = io.ReadAll(os.Stdin)
        name = "stdin"
    } else {
        data, err = os.ReadFile(path)
    }

    if err != nil {
        return zero, fmt.Errorf("reading file %s: %w", name, err)
    }

    var request T
    if err := json.Unmarshal(data, &request); err != nil {
        return zero, fmt.Errorf("parsing JSON from %s: %w", name, err)
    }

    return request, nil
}
```

Then update `filter.go` and `widget.go` to use it:

```go
// In filter.go
request, err := readRequestFromFile[types.UpdateUserFilterRQ](path)

// In widget.go
request, err := readRequestFromFile[types.WidgetRQ](path)
```

**Important:** Preserve the pilot behavior where `--from-file -` reads from stdin. Keep resource-specific validation (e.g., "filter must have conditions") in the resource file. The shared helper only handles file I/O and JSON parsing. Each resource's create/update command calls the shared reader and then performs its own validation.

### 2. Extract `parseIDArg` — numeric ID parsing from positional args

Create a helper in a new file `internal/cli/args.go`:

```go
func parseIDArg(arg, resourceName string) (int64, error) {
    id, err := strconv.ParseInt(arg, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid %s ID %q: must be a numeric ID", resourceName, arg)
    }

    return id, nil
}
```

Update `filter.go`, `group.go`, and `widget.go` to use it. The `resourceName` parameter produces error messages like `"invalid filter ID \"abc\": must be a numeric ID"`.

**Note:** The existing `parseOptionalInt64Flag` in `item.go` is a different pattern (parses flag values, not positional args, and supports empty-as-absent). Keep it as-is.

### 3. Extract `messageResult` — standardized success-message output

Both `OperationCompletionRS` (has `Message *string`) and `SuccessfulUpdate` (has `Message *string`) are used for mutation confirmation. The existing `operationResultHeaders`/`operationResultRows` in `user.go` already handle `OperationCompletionRS`. Generalize to handle any response with an optional message:

Create a helper in `internal/cli/output_helpers.go` (or extend an existing file):

```go
func messageHeaders() []string {
    return []string{"MESSAGE"}
}

func messageRows(message string) [][]string {
    return [][]string{{message}}
}

func formatMessage(msg *string) string {
    if msg == nil {
        return ""
    }
    return *msg
}
```

Then update:
- `user.go`: replace `operationResultHeaders`/`operationResultRows` with the shared helpers
- `filter.go`: use shared message helpers for update/delete responses
- `group.go`: use shared message helpers for update response and delete synthetic message

### 4. Extract command preamble helper

Every command handler starts with the same boilerplate. Extract into a helper that reduces this repetition:

```go
type cmdContext struct {
    printer    *Printer
    client     *client.Client
    connection *Connection
    logger     *slog.Logger
}

func resolveCmd(cmd *cobra.Command) (*cmdContext, error) {
    logger := loggerFromCmd(cmd)
    outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
    printer := newPrinter(outputFmt)

    conn, err := resolveConnection(cmd)
    if err != nil {
        printer.PrintError(err)
        return nil, err
    }

    clientInstance, err := newClient(conn, logger)
    if err != nil {
        printer.PrintError(err)
        return nil, err
    }

    return &cmdContext{
        printer:    printer,
        client:     clientInstance,
        connection: conn,
        logger:     logger,
    }, nil
}
```

Use a single settled design: store the resolved connection in `cmdContext`, then let project-scoped commands resolve the project through a method:

```go
func (c *cmdContext) resolveProject(cmd *cobra.Command) (string, error) {
    project, err := resolveProject(cmd, c.connection)
    if err != nil {
        c.printer.PrintError(err)
        return "", err
    }

    return project, nil
}
```

Project-scoped commands should call `ctx.resolveProject(cmd)` after `resolveCmd(cmd)`. Instance-scoped commands stop at `resolveCmd(cmd)`. Do not introduce a second wrapper such as `resolveCmdWithProject` unless the concrete code proves it is simpler.

**Refactor existing commands** (user, launch, item, log, project, filter, group, widget) to use `resolveCmd`. This is a mechanical refactor that reduces each handler by ~10 lines.

### 5. Move `formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr` to shared location

These are currently in `item.go` and `user.go`. Move them to `internal/cli/output_helpers.go` (or a similar shared file) so all resources can use them without implicit cross-file dependencies. They may already be accessible (same package), but moving them to a clearly-shared file makes the intended use explicit.

### 6. Do NOT extract (yet)

The following are **not** ready for extraction:

- **Pagination helpers:** Groups use offset/limit while filters/launches/items use page/page-size. There's no common abstraction yet. Leave both patterns in place.
- **Table column definitions:** Each resource has different columns. No abstraction needed.
- **Validation logic:** Each resource validates different things. Keep validation in resource files.
- **Test scaffolding:** If the pilot tests share no common setup beyond standard testify patterns, don't extract a test framework. If two or more pilot tests do share a concrete helper, extract only that helper into a small `_test.go` file.

## File layout after this PR

New files:
- `internal/cli/fromfile.go` — `readRequestFromFile[T]`
- `internal/cli/args.go` — `parseIDArg`
- `internal/cli/output_helpers.go` — `messageHeaders`, `messageRows`, `formatMessage`, `formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr`
- `internal/cli/context.go` — `cmdContext`, `resolveCmd`
- `internal/cli/testutil_test.go` — duplicated pilot-test helpers, only if the pilots prove the need

Modified files:
- `internal/cli/filter.go` — use shared helpers
- `internal/cli/group.go` — use shared helpers
- `internal/cli/widget.go` — use shared helpers
- `internal/cli/user.go` — use shared helpers, remove `operationResultHeaders`/`operationResultRows` (moved to shared)
- `internal/cli/launch.go` — use `resolveCmd`
- `internal/cli/item.go` — use `resolveCmd`, move formatters to shared file
- `internal/cli/log.go` — use `resolveCmd`
- `internal/cli/project.go` — use `resolveCmd`
- `internal/cli/download.go` — use `resolveCmd` where applicable
- `internal/cli/whoami.go` — use `resolveCmd`

## Test requirements

### Unit tests

- **`fromfile.go`** — `internal/cli/fromfile_test.go`:
  - Valid JSON file reads correctly (use temp files)
  - Non-existent file returns appropriate error
  - Malformed JSON returns parse error with filename
  - Empty file returns appropriate error
- **`args.go`** — `internal/cli/args_test.go`:
  - Valid numeric string parses correctly
  - Non-numeric string returns error with resource name
  - Empty string returns error
  - Overflow values return error
- **`context.go`** — test that `resolveCmd` returns errors when connection is misconfigured (piggyback on existing `config_test.go` patterns)
- **Existing tests must still pass** — the refactor is mechanical; behavior must not change

### Regression verification

All existing tests in `internal/cli/` must pass unchanged (or with trivial import path adjustments if functions moved between files within the same package). Since everything stays in `package cli`, moves between files don't affect tests.

## Edge cases and constraints

- **Same package:** All files are in `package cli`. Moving functions between files has no import implications. Tests see all unexported symbols regardless of which `.go` file they're in.
- **Don't break behavior:** This is a pure refactoring PR. No user-visible behavior should change. Command outputs, error messages, and flag semantics must remain identical.
- **Generic function requires Go 1.18+:** The `readRequestFromFile[T]` generic helper requires Go 1.18 or later. Verify the `go.mod` specifies at least 1.18 (it almost certainly does, but check).
- **Error message consistency:** After extracting `parseIDArg`, verify that all resources produce consistent error messages. The format should be `"invalid <resource> ID \"<value>\": must be a numeric ID"` everywhere.
- **`resolveCmd` error handling:** When `resolveCmd` returns an error, the error has already been printed (via `printer.PrintError`). Callers should simply return the error without printing again. This matches the existing pattern where handlers call `PrintError` then return.
- **Keep `--from-file` as a required flag:** Even with the shared reader, each command that uses `--from-file` should still mark it required via `cmd.MarkFlagRequired("from-file")`. The shared helper doesn't handle flag registration.

## Validation

Run before finalizing:

```
make lint
make test
```

Since this is a refactoring PR, integration tests should also pass without changes:

```
make test-integration
```

## Definition of done

- [ ] `readRequestFromFile[T]` extracted and used by filter and widget commands
- [ ] `parseIDArg` extracted and used by filter, group, and widget commands
- [ ] Message/operation-result helpers consolidated in shared file
- [ ] `cmdContext`/`resolveCmd` extracted and used by all command handlers
- [ ] Format helpers (`formatStringPtr`, `formatInt64Ptr`, `formatBoolPtr`) in a clearly-shared file
- [ ] All existing tests pass without behavior changes
- [ ] New unit tests for `readRequestFromFile`, `parseIDArg`
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-integration` passes (behavior unchanged)
- [ ] This file moved to `prompts/completed/`
