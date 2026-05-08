# PR 7: Extract Shared CLI Helpers from the Pilot Slices

## Goal

Consolidate duplicated helper functions from the three pilot PRs (filter, group, widget) and existing command files into dedicated shared-helper files. After this PR, the Phase 4 fan-out wave can import well-tested shared utilities instead of re-inventing them per resource.

## Context

After PRs 4–6 land, the CLI codebase contains several classes of duplicated code:

1. **ID parsing** — `parseFilterID`, `parseGroupID`, `parseWidgetID` are structurally identical (`strconv.ParseInt` + resource-specific error message).
2. **`--from-file` handling** — `readFilterRequestFromFile` and `readWidgetRequestFromFile` are identical except for the deserialized target type.
3. **Format helpers** — `formatStringPtr` and `formatInt64Ptr` live in `item.go`, `formatBoolPtr` lives in `user.go`, `formatInt32Ptr` lives in `group.go`. These are used across many command files.
4. **Mutation result formatting** — `operationResultHeaders`/`operationResultRows` live in `user.go` but are used by filter, widget, and user commands. Both filter and widget create commands also need to format `EntryCreatedRS` identically. The group pilot introduces a local formatter for `types.SuccessfulUpdate` which has the same shape (single `Message *string` field) — this is a second message-only result type that should share the same headers.
5. **Flag helpers** — `mustGetStringFlag`, `optionalStringFlag`, `optionalBoolFlag` live in `user.go` but are used by group and will be needed by many Phase 4 resources.
6. **Test helpers** — `commandNames` lives in `user_test.go` but is used from multiple test files. Other test helpers (`newTestCmdWithProjectFlag`, `clearRPEnv`, `newTestCmdWithListFlags`) already live in their respective test files but `commandNames` is misplaced.

This PR extracts only patterns that are **proven duplicated** across at least two families. It does not invent speculative abstractions.

**Prerequisite:** PRs 1–6 must land first.

## Scope

Create the following new files:
- `internal/cli/helpers.go` — shared formatting and ID-parsing helpers
- `internal/cli/helpers_test.go` — tests for the shared helpers
- `internal/cli/fromfile.go` — generic `--from-file` reader
- `internal/cli/fromfile_test.go` — tests for the file reader
- `internal/cli/mutation_result.go` — shared mutation result formatters
- `internal/cli/mutation_result_test.go` — tests for mutation result formatting
- `internal/cli/test_helpers_test.go` — shared test-only utilities

Remove the relocated helpers from their original files.

## Implementation

### 1. Create `internal/cli/helpers.go`

This file consolidates all small shared utilities that don't warrant their own file.

#### Formatting helpers (moved from `item.go` and `user.go` and `group.go`)

```go
package cli

import (
    "fmt"
    "strconv"

    "github.com/spf13/cobra"
)

// formatStringPtr returns the dereferenced string or an empty string if nil.
func formatStringPtr(value *string) string {
    if value == nil {
        return ""
    }

    return *value
}

// formatInt64Ptr returns the formatted int64 or an empty string if nil.
func formatInt64Ptr(value *int64) string {
    if value == nil {
        return ""
    }

    return strconv.FormatInt(*value, 10)
}

// formatInt32Ptr returns the formatted int32 or an empty string if nil.
func formatInt32Ptr(value *int32) string {
    if value == nil {
        return ""
    }

    return strconv.FormatInt(int64(*value), 10)
}

// formatBoolPtr returns the dereferenced bool as a string or an empty string if nil.
func formatBoolPtr(value *bool) string {
    if value == nil {
        return ""
    }

    return strconv.FormatBool(*value)
}
```

#### ID parsing (replaces `parseFilterID`, `parseGroupID`, `parseWidgetID`)

```go
// parseResourceID parses idStr as a base-10 int64. The resourceName parameter is
// used in error messages to identify which resource type was being parsed.
func parseResourceID(idStr, resourceName string) (int64, error) {
    id, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid %s ID %q: must be a numeric ID", resourceName, idStr)
    }

    return id, nil
}
```

#### Flag helpers (moved from `user.go`)

```go
// mustGetStringFlag returns the string value for name, ignoring impossible Cobra access errors.
func mustGetStringFlag(cmd *cobra.Command, name string) string {
    value, _ := cmd.Flags().GetString(name)

    return value
}

// optionalStringFlag returns a pointer to the explicit string flag value, or nil when the flag was omitted.
func optionalStringFlag(cmd *cobra.Command, name string) *string {
    if !cmd.Flags().Changed(name) {
        return nil
    }

    value, _ := cmd.Flags().GetString(name)

    return &value
}

// optionalBoolFlag returns a pointer to the explicit bool flag value, or nil when the flag was omitted.
func optionalBoolFlag(cmd *cobra.Command, name string) *bool {
    if !cmd.Flags().Changed(name) {
        return nil
    }

    value, _ := cmd.Flags().GetBool(name)

    return &value
}
```

### 2. Create `internal/cli/fromfile.go`

This file provides a generic JSON-file reader using Go generics.

```go
package cli

import (
    "encoding/json"
    "fmt"
    "os"
)

// readRequestFromFile reads a JSON file at the given path and unmarshals it into
// the target type T. The path is included in error messages for debugging.
func readRequestFromFile[T any](path string) (T, error) {
    var zero T

    data, err := os.ReadFile(path)
    if err != nil {
        return zero, fmt.Errorf("reading file %s: %w", path, err)
    }

    var request T
    if err := json.Unmarshal(data, &request); err != nil {
        return zero, fmt.Errorf("parsing JSON from %s: %w", path, err)
    }

    return request, nil
}
```

### 3. Create `internal/cli/mutation_result.go`

This file provides formatters for the three common mutation response types.

```go
package cli

import (
    "strconv"

    "github.com/klaskosk/report-portal-cli/pkg/types"
)

// messageResultHeaders returns the table column headers for mutation results
// that only contain a message field. Used for both OperationCompletionRS and
// SuccessfulUpdate responses.
func messageResultHeaders() []string {
    return []string{"MESSAGE"}
}

// operationResultRows formats an OperationCompletionRS as a single table row.
func operationResultRows(result *types.OperationCompletionRS) [][]string {
    return [][]string{{formatStringPtr(result.Message)}}
}

// successfulUpdateRows formats a SuccessfulUpdate as a single table row.
func successfulUpdateRows(result *types.SuccessfulUpdate) [][]string {
    return [][]string{{formatStringPtr(result.Message)}}
}

// entryCreatedHeaders returns the table column headers for creation results
// that return a resource ID (EntryCreatedRS).
func entryCreatedHeaders() []string {
    return []string{"ID"}
}

// entryCreatedRows formats an entry-created response as a single table row.
func entryCreatedRows(result *types.EntryCreatedRS) [][]string {
    id := ""
    if result.Id != nil {
        id = strconv.FormatInt(*result.Id, 10)
    }

    return [][]string{{id}}
}
```

**Migration note for existing call sites:** The old `operationResultHeaders()` is renamed to `messageResultHeaders()` because it now serves both `OperationCompletionRS` and `SuccessfulUpdate`. Update all call sites:
- `user.go` (update/delete user): `operationResultHeaders()` → `messageResultHeaders()`
- `filter.go` (update/delete filter): `operationResultHeaders()` → `messageResultHeaders()`
- `widget.go` (update widget): `operationResultHeaders()` → `messageResultHeaders()`
- `group.go` (update group): replace the local `SuccessfulUpdate` formatter with `messageResultHeaders()` + `successfulUpdateRows(result)`

### 4. Create `internal/cli/test_helpers_test.go`

Move `commandNames` here. This file is visible to all `_test.go` files in the `cli` package.

```go
package cli //nolint:testpackage

import (
    "slices"

    "github.com/spf13/cobra"
)

// commandNames returns the sorted list of immediate child command names for cmd.
func commandNames(cmd *cobra.Command) []string {
    names := make([]string, 0, len(cmd.Commands()))
    for _, child := range cmd.Commands() {
        names = append(names, child.Name())
    }

    slices.Sort(names)

    return names
}
```

**Note:** The sort is load-bearing — existing tests assert exact sorted slices (e.g., `[]string{"item", "launch", "log", "project", "user"}`). If `commandNames` is also used from `commands_test.go` (PR 3), that's fine — all test files in `package cli` share access to test helpers.

### 5. Remove relocated definitions from source files

#### From `item.go`:

Remove the definitions of:
- `formatStringPtr`
- `formatInt64Ptr`

Leave `parseOptionalInt64Flag` in `item.go` — it is item-specific (3-return-value pattern for optional flag parsing) and not duplicated by the pilots.

#### From `user.go`:

Remove the definitions of:
- `operationResultHeaders` (replaced by `messageResultHeaders` in `mutation_result.go`)
- `operationResultRows` (moved to `mutation_result.go`)
- `formatBoolPtr`
- `mustGetStringFlag`
- `optionalStringFlag`
- `optionalBoolFlag`

Update the user update/delete call sites from `operationResultHeaders()` to `messageResultHeaders()`.

#### From `group.go`:

Remove the definitions of:
- `formatInt32Ptr`
- Any local `SuccessfulUpdate` formatting helper (e.g., inline headers/rows logic or a local `groupUpdateResultRows` function)

Replace the update command's result printing with:
```go
return printer.Print(result, messageResultHeaders(), successfulUpdateRows(result))
```

Replace the delete command's result printing (which uses the local `groupDeleteResult` struct and `operationResultHeaders()`) with `messageResultHeaders()`:
```go
return printer.Print(result, messageResultHeaders(), [][]string{{result.Message}})
```

The `groupDeleteResult` struct itself stays in `group.go` — it is group-specific (synthesized because `Delete` returns no body) and not duplicated elsewhere.

#### From `filter.go`:

Remove the definition of:
- `readFilterRequestFromFile`

Replace its call site with `readRequestFromFile[types.UpdateUserFilterRQ](path)`.

Also remove `parseFilterID` and replace its call sites with `parseResourceID(idStr, "filter")`.

#### From `widget.go`:

Remove the definition of:
- `readWidgetRequestFromFile`

Replace its call site with `readRequestFromFile[types.WidgetRQ](path)`.

Also remove `parseWidgetID` and replace its call sites with `parseResourceID(idStr, "widget")`.

#### From `group.go`:

Remove `parseGroupID` (if it existed as a standalone function) and replace call sites with `parseResourceID(idStr, "group")`.

#### From `user_test.go`:

Remove the `commandNames` helper function definition. It now lives in `test_helpers_test.go`.

### 6. Update `filter.go` call sites

Replace:
```go
id, err := parseFilterID(args[0])
```
with:
```go
id, err := parseResourceID(args[0], "filter")
```

Replace:
```go
request, err := readFilterRequestFromFile(path)
```
with:
```go
request, err := readRequestFromFile[types.UpdateUserFilterRQ](path)
```

For create result formatting, if the filter pilot had inline formatting of `EntryCreatedRS`, replace with:
```go
return printer.Print(result, entryCreatedHeaders(), entryCreatedRows(result))
```

For update/delete result formatting, replace `operationResultHeaders()` with `messageResultHeaders()`:
```go
return printer.Print(result, messageResultHeaders(), operationResultRows(result))
```

### 7. Update `widget.go` call sites

Replace:
```go
id, err := parseWidgetID(args[0])
```
with:
```go
id, err := parseResourceID(args[0], "widget")
```

Replace:
```go
request, err := readWidgetRequestFromFile(path)
```
with:
```go
request, err := readRequestFromFile[types.WidgetRQ](path)
```

For create result formatting, replace with:
```go
return printer.Print(result, entryCreatedHeaders(), entryCreatedRows(result))
```

For update result formatting, replace `operationResultHeaders()` with `messageResultHeaders()`:
```go
return printer.Print(result, messageResultHeaders(), operationResultRows(result))
```

### 8. Update `group.go` call sites

Replace any local `parseGroupID(args[0])` with:
```go
id, err := parseResourceID(args[0], "group")
```

Replace the update command's local `SuccessfulUpdate` formatting with:
```go
return printer.Print(result, messageResultHeaders(), successfulUpdateRows(result))
```

Replace the delete command's `operationResultHeaders()` call with `messageResultHeaders()`:
```go
return printer.Print(result, messageResultHeaders(), [][]string{{result.Message}})
```

### 9. Create `internal/cli/helpers_test.go`

```go
package cli //nolint:testpackage

import (
    "testing"

    "github.com/spf13/cobra"
    "github.com/stretchr/testify/assert"
)

func TestFormatStringPtr(t *testing.T) {
    t.Parallel()

    t.Run("nil returns empty", func(t *testing.T) {
        t.Parallel()
        assert.Equal(t, "", formatStringPtr(nil))
    })

    t.Run("non-nil returns value", func(t *testing.T) {
        t.Parallel()
        s := "hello"
        assert.Equal(t, "hello", formatStringPtr(&s))
    })
}

func TestFormatInt64Ptr(t *testing.T) {
    t.Parallel()

    t.Run("nil returns empty", func(t *testing.T) {
        t.Parallel()
        assert.Equal(t, "", formatInt64Ptr(nil))
    })

    t.Run("non-nil returns formatted", func(t *testing.T) {
        t.Parallel()
        v := int64(42)
        assert.Equal(t, "42", formatInt64Ptr(&v))
    })
}

func TestFormatInt32Ptr(t *testing.T) {
    t.Parallel()

    t.Run("nil returns empty", func(t *testing.T) {
        t.Parallel()
        assert.Equal(t, "", formatInt32Ptr(nil))
    })

    t.Run("non-nil returns formatted", func(t *testing.T) {
        t.Parallel()
        v := int32(7)
        assert.Equal(t, "7", formatInt32Ptr(&v))
    })
}

func TestFormatBoolPtr(t *testing.T) {
    t.Parallel()

    t.Run("nil returns empty", func(t *testing.T) {
        t.Parallel()
        assert.Equal(t, "", formatBoolPtr(nil))
    })

    t.Run("true", func(t *testing.T) {
        t.Parallel()
        v := true
        assert.Equal(t, "true", formatBoolPtr(&v))
    })

    t.Run("false", func(t *testing.T) {
        t.Parallel()
        v := false
        assert.Equal(t, "false", formatBoolPtr(&v))
    })
}

func TestParseResourceID_Valid(t *testing.T) {
    t.Parallel()

    tests := []struct {
        input    string
        expected int64
    }{
        {"1", 1},
        {"42", 42},
        {"9999999", 9999999},
    }

    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            t.Parallel()
            id, err := parseResourceID(tt.input, "test-resource")
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, id)
        })
    }
}

func TestParseResourceID_Invalid(t *testing.T) {
    t.Parallel()

    tests := []struct {
        input        string
        resourceName string
    }{
        {"abc", "filter"},
        {"12.5", "widget"},
        {"", "group"},
        {"9999999999999999999999", "dashboard"},
    }

    for _, tt := range tests {
        t.Run(tt.input+"_"+tt.resourceName, func(t *testing.T) {
            t.Parallel()
            _, err := parseResourceID(tt.input, tt.resourceName)
            assert.Error(t, err)
            assert.Contains(t, err.Error(), tt.resourceName)
            assert.Contains(t, err.Error(), tt.input)
        })
    }
}

func TestMustGetStringFlag(t *testing.T) {
    t.Parallel()

    t.Run("returns value when set", func(t *testing.T) {
        t.Parallel()
        cmd := &cobra.Command{Use: "test"}
        cmd.Flags().String("name", "", "")
        _ = cmd.ParseFlags([]string{"--name", "hello"})
        assert.Equal(t, "hello", mustGetStringFlag(cmd, "name"))
    })

    t.Run("returns default when not set", func(t *testing.T) {
        t.Parallel()
        cmd := &cobra.Command{Use: "test"}
        cmd.Flags().String("name", "default-val", "")
        assert.Equal(t, "default-val", mustGetStringFlag(cmd, "name"))
    })
}

func TestOptionalStringFlag_NotChanged(t *testing.T) {
    t.Parallel()

    cmd := &cobra.Command{Use: "test"}
    cmd.Flags().String("name", "", "")
    assert.Nil(t, optionalStringFlag(cmd, "name"))
}

func TestOptionalStringFlag_Changed(t *testing.T) {
    t.Parallel()

    cmd := &cobra.Command{Use: "test"}
    cmd.Flags().String("name", "", "")
    _ = cmd.ParseFlags([]string{"--name", "hello"})
    result := optionalStringFlag(cmd, "name")
    assert.NotNil(t, result)
    assert.Equal(t, "hello", *result)
}

func TestOptionalStringFlag_ChangedEmpty(t *testing.T) {
    t.Parallel()

    cmd := &cobra.Command{Use: "test"}
    cmd.Flags().String("name", "", "")
    _ = cmd.ParseFlags([]string{"--name", ""})
    result := optionalStringFlag(cmd, "name")
    assert.NotNil(t, result)
    assert.Equal(t, "", *result)
}

func TestOptionalBoolFlag_NotChanged(t *testing.T) {
    t.Parallel()

    cmd := &cobra.Command{Use: "test"}
    cmd.Flags().Bool("active", false, "")
    assert.Nil(t, optionalBoolFlag(cmd, "active"))
}

func TestOptionalBoolFlag_Changed(t *testing.T) {
    t.Parallel()

    cmd := &cobra.Command{Use: "test"}
    cmd.Flags().Bool("active", false, "")
    _ = cmd.ParseFlags([]string{"--active"})
    result := optionalBoolFlag(cmd, "active")
    assert.NotNil(t, result)
    assert.True(t, *result)
}
```

### 10. Create `internal/cli/fromfile_test.go`

```go
package cli //nolint:testpackage

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// testPayload is a simple struct used only by tests to exercise the generic reader.
type testPayload struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}

func TestReadRequestFromFile_ValidJSON(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    path := filepath.Join(dir, "payload.json")
    require.NoError(t, os.WriteFile(path, []byte(`{"name":"hello","count":5}`), 0o644))

    result, err := readRequestFromFile[testPayload](path)
    require.NoError(t, err)
    assert.Equal(t, "hello", result.Name)
    assert.Equal(t, 5, result.Count)
}

func TestReadRequestFromFile_MissingFile(t *testing.T) {
    t.Parallel()

    _, err := readRequestFromFile[testPayload]("/nonexistent/path.json")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "/nonexistent/path.json")
    assert.Contains(t, err.Error(), "reading file")
}

func TestReadRequestFromFile_InvalidJSON(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    path := filepath.Join(dir, "bad.json")
    require.NoError(t, os.WriteFile(path, []byte(`{not valid json`), 0o644))

    _, err := readRequestFromFile[testPayload](path)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), path)
    assert.Contains(t, err.Error(), "parsing JSON")
}

func TestReadRequestFromFile_EmptyFile(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    path := filepath.Join(dir, "empty.json")
    require.NoError(t, os.WriteFile(path, []byte(``), 0o644))

    _, err := readRequestFromFile[testPayload](path)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "parsing JSON")
}
```

### 11. Create `internal/cli/mutation_result_test.go`

```go
package cli //nolint:testpackage

import (
    "testing"

    "github.com/klaskosk/report-portal-cli/pkg/types"
    "github.com/stretchr/testify/assert"
)

func TestMessageResultHeaders(t *testing.T) {
    t.Parallel()

    assert.Equal(t, []string{"MESSAGE"}, messageResultHeaders())
}

func TestOperationResultRows(t *testing.T) {
    t.Parallel()

    t.Run("with message", func(t *testing.T) {
        t.Parallel()
        msg := "Operation completed successfully"
        result := &types.OperationCompletionRS{Message: &msg}
        rows := operationResultRows(result)
        assert.Equal(t, [][]string{{"Operation completed successfully"}}, rows)
    })

    t.Run("nil message", func(t *testing.T) {
        t.Parallel()
        result := &types.OperationCompletionRS{Message: nil}
        rows := operationResultRows(result)
        assert.Equal(t, [][]string{{""}}, rows)
    })
}

func TestSuccessfulUpdateRows(t *testing.T) {
    t.Parallel()

    t.Run("with message", func(t *testing.T) {
        t.Parallel()
        msg := "Group updated"
        result := &types.SuccessfulUpdate{Message: &msg}
        rows := successfulUpdateRows(result)
        assert.Equal(t, [][]string{{"Group updated"}}, rows)
    })

    t.Run("nil message", func(t *testing.T) {
        t.Parallel()
        result := &types.SuccessfulUpdate{Message: nil}
        rows := successfulUpdateRows(result)
        assert.Equal(t, [][]string{{""}}, rows)
    })
}

func TestEntryCreatedHeaders(t *testing.T) {
    t.Parallel()

    assert.Equal(t, []string{"ID"}, entryCreatedHeaders())
}

func TestEntryCreatedRows(t *testing.T) {
    t.Parallel()

    t.Run("with id", func(t *testing.T) {
        t.Parallel()
        id := int64(42)
        result := &types.EntryCreatedRS{Id: &id}
        rows := entryCreatedRows(result)
        assert.Equal(t, [][]string{{"42"}}, rows)
    })

    t.Run("nil id", func(t *testing.T) {
        t.Parallel()
        result := &types.EntryCreatedRS{Id: nil}
        rows := entryCreatedRows(result)
        assert.Equal(t, [][]string{{""}}, rows)
    })
}
```

## Design notes

- **Only extract what is duplicated.** This PR extracts helpers that appear in at least two resource families. It does not speculatively add helpers for patterns that might appear later. If Phase 4 reveals more shared patterns, those are addressed in PR 16 (Phase 5 fan-in).

- **`parseResourceID` takes a `resourceName` parameter.** This preserves resource-specific error messages (`"invalid filter ID"` vs `"invalid widget ID"`) without per-resource wrapper functions. Callers pass the resource name as a literal string at each call site.

- **`readRequestFromFile` uses Go generics.** The project targets Go 1.26. The generic approach eliminates the need for per-type wrapper functions while preserving type safety at compile time. The function uses `json.Unmarshal` (not `json.NewDecoder`) because the entire file is read into memory anyway and `Unmarshal` produces clearer error messages for malformed JSON.

- **`messageResultHeaders` unifies both message-only result types.** The API has two structurally identical message-only responses: `OperationCompletionRS` (used by filter/widget/user update/delete) and `SuccessfulUpdate` (used by group update). Both have a single `Message *string` field. Rather than two header functions, a single `messageResultHeaders()` returns `[]string{"MESSAGE"}` and both `operationResultRows` and `successfulUpdateRows` produce the row. This keeps the headers DRY while preserving type safety for the row formatters (callers pass the concrete type they received from the API).

- **`entryCreatedHeaders`/`entryCreatedRows` are new shared helpers.** Both the filter and widget pilots format `EntryCreatedRS` identically. Rather than each resource defining inline formatting logic, a shared helper makes this consistent and eliminates duplication for all future create commands that return `EntryCreatedRS`.

- **File layout rationale:**
  - `helpers.go` — small, stateless formatting and parsing functions (the "utility drawer")
  - `fromfile.go` — isolated because it has `os` and `json` dependencies and a distinct responsibility
  - `mutation_result.go` — isolated because it depends on `pkg/types` and provides a cohesive set of formatters for mutation responses
  - `test_helpers_test.go` — test-only utilities shared across all `*_test.go` files in the package

- **`parseOptionalInt64Flag` stays in `item.go`.** Its 3-return-value signature (value, present, error) is specific to the item/log pattern of optional filter flags. It is not duplicated by the pilots and would be misleading as a "shared" helper.

- **No shared "list-or-single dispatch" helper.** The `if len(args) == 1 { single } else { list }` pattern is a 3-line branch that varies significantly between resources (different return types, different list options, different single-get methods). Extracting it into a shared function would require excessive generics or interfaces for minimal DRY benefit. It remains a documented convention, not extracted code.

- **No shared project-resolution changes.** `resolveProject` already lives in `config.go` and works correctly for all project-scoped commands. No refinement was needed during the pilots.

- **Unused `strconv` import cleanup.** After moving `formatInt64Ptr` out of `item.go`, verify that `item.go` still imports `strconv` (it does — `strconv.ParseInt` is used by `parseOptionalInt64Flag`). Similarly, after moving `formatBoolPtr` and flag helpers out of `user.go`, verify `user.go` still compiles (it will — `strconv` is used elsewhere in user formatting code).

## What NOT to do

- **Do not create abstractions for patterns not yet duplicated.** If only one resource uses a pattern, leave it local. This PR is descriptive (extracting proven duplication), not prescriptive (guessing future needs).
- **Do not refactor command structure.** This PR only moves helper functions and updates call sites. No command registration, flag registration, or run function logic changes beyond swapping local helpers for shared ones.
- **Do not change the behavior of any helper.** The extracted functions must be semantically identical to the originals. Error messages should remain the same (with the parameterized resource name being the only variation for `parseResourceID`).
- **Do not introduce interfaces or abstraction layers** for things like "resource formatter" or "CRUD command builder." Phase 4 resources will follow patterns by convention, not by forced inheritance.
- **Do not move `resolveConnection`, `resolveProject`, `newClient`, or `initConfig`.** These already live in `config.go` and are not duplicated.
- **Do not move `registerListFlags` / `resolveListFlags`.** These already live in `listflags.go` and are not duplicated.
- **Do not move `registerGroupListFlags` / `resolveGroupListFlags` / `groupListFlags`** out of `group.go`. These are group-specific (offset-based pagination) and not duplicated. If Phase 4 reveals another offset-based resource, extraction can happen in PR 16.
- **Do not add integration tests.** This is a pure refactor with no behavioral changes.

## Validation

```bash
make build
make test
make lint
```

After the refactor, all existing commands must behave identically. Specifically:

- `rp get filter 1` (single-get) — same output format and error behavior
- `rp create filter --from-file <path>` — same parsing and error messages
- `rp get group` — same pagination behavior
- `rp get widget 1` — same output
- `rp create widget --from-file <path>` — same parsing
- `rp create user`, `rp update user`, `rp delete user` — unchanged
- `rp get launch`, `rp get item`, `rp get log` — unchanged

Verify no unused imports or declarations remain in the source files after extraction. The linter (`make lint`) will catch these.

## Definition of done

- [ ] `internal/cli/helpers.go` created with `formatStringPtr`, `formatInt64Ptr`, `formatInt32Ptr`, `formatBoolPtr`, `parseResourceID`, `mustGetStringFlag`, `optionalStringFlag`, `optionalBoolFlag`
- [ ] `internal/cli/helpers_test.go` created with tests for all helpers
- [ ] `internal/cli/fromfile.go` created with generic `readRequestFromFile[T any]`
- [ ] `internal/cli/fromfile_test.go` created with tests for valid, missing, invalid, and empty file cases
- [ ] `internal/cli/mutation_result.go` created with `messageResultHeaders`, `operationResultRows`, `successfulUpdateRows`, `entryCreatedHeaders`, `entryCreatedRows`
- [ ] `internal/cli/mutation_result_test.go` created with tests for all three result types including nil fields
- [ ] `internal/cli/test_helpers_test.go` created with `commandNames` (moved from `user_test.go`, preserving `slices.Sort`)
- [ ] `formatStringPtr` and `formatInt64Ptr` removed from `item.go`
- [ ] `formatBoolPtr`, `operationResultHeaders`, `operationResultRows`, `mustGetStringFlag`, `optionalStringFlag`, `optionalBoolFlag` removed from `user.go`; user update/delete call sites use `messageResultHeaders()`
- [ ] `formatInt32Ptr` removed from `group.go`
- [ ] `readFilterRequestFromFile` and `parseFilterID` removed from `filter.go`; call sites updated
- [ ] `readWidgetRequestFromFile` and `parseWidgetID` removed from `widget.go`; call sites updated
- [ ] `parseGroupID` (if present) removed from `group.go`; call sites updated
- [ ] Local `SuccessfulUpdate` formatter removed from `group.go`; update command uses `messageResultHeaders()` + `successfulUpdateRows()`
- [ ] Group delete command's `operationResultHeaders()` call updated to `messageResultHeaders()`
- [ ] `commandNames` removed from `user_test.go`
- [ ] All pilot command call sites updated to use shared helpers
- [ ] Filter and widget create commands use `entryCreatedHeaders`/`entryCreatedRows`
- [ ] No unused imports remain in any modified file
- [ ] All existing tests pass unchanged (same assertions, same behavior)
- [ ] New tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
