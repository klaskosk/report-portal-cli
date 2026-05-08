# PR 16: Normalize Helpers and UX After the Standard Resource Wave

## Goal

Audit all Phase 4 resource files (PRs 8–15), identify local helpers that duplicated each other or could be shared, normalize inconsistent UX patterns, fix help text drift, and consolidate test scaffolding. This is a reactive cleanup PR — it only extracts or changes patterns that are **proven duplicated or inconsistent** across the files that landed in Phase 4.

## Context

Phase 4 (PRs 8–15) added eight resource families in parallel worktrees. By design, those PRs stayed isolated and occasionally redefined similar patterns locally. This PR is the fan-in checkpoint that folds those patterns into the shared infrastructure established in PR 7.

**Prerequisite:** All of PRs 1–15 must have landed.

### Phase 4 file inventory

| PR | File(s) | Scope | Mutation input |
|----|---------|-------|----------------|
| 8 | `dashboard.go`, `dashboard_test.go` | project | inline flags |
| 9 | `log_type.go`, `log_type_test.go` | project | inline flags |
| 10 | `integration.go`, `integration_test.go` | instance | `--from-file` |
| 11 | `project_integration.go`, `project_integration_test.go` | project | `--from-file` |
| 12 | `project.go` (extended), `project_mutation_test.go` | instance | inline + `--from-file` |
| 13 | `launch.go` (extended), `launch_mutation_test.go` | project | `--from-file` |
| 14 | `item.go` (extended), `item_mutation_test.go` | project | `--from-file` |
| 15 | `log.go` (extended), `log_mutation_test.go` | project | `--from-file` |

### Shared infrastructure from PR 7

| File | Contents |
|------|----------|
| `helpers.go` | `formatStringPtr`, `formatInt64Ptr`, `formatInt32Ptr`, `formatBoolPtr`, `parseResourceID`, `mustGetStringFlag`, `optionalStringFlag`, `optionalBoolFlag` |
| `fromfile.go` | `readRequestFromFile[T any]` |
| `mutation_result.go` | `messageResultHeaders`, `operationResultRows`, `successfulUpdateRows`, `entryCreatedHeaders`, `entryCreatedRows` |
| `test_helpers_test.go` | `commandNames` |

## Scope

This PR modifies existing files only — no new resource commands. The changes fall into five categories (sections 1–5), each described below with specific expected findings and how to handle them. Sections 6–10 are verification checks for helpers that should stay local — confirm their placement is correct and move on.

## Implementation

### 1. Consolidate duplicate delete-result structs

**Expected finding:** Both `group.go` (PR 5) and `log_type.go` (PR 9) define local structs for synthesizing a delete result when the API returns no response body:

```go
// group.go
type groupDeleteResult struct {
    Message string `json:"message"`
}

// log_type.go
type logTypeDeleteResult struct {
    Message string `json:"message"`
}
```

Both are used identically: `result := <type>{Message: fmt.Sprintf("<Resource> %d deleted successfully", id)}`, then printed with `messageResultHeaders()` and a single-row slice.

**Action:** Extract a shared helper into `mutation_result.go`:

```go
// deleteResult is a synthesized response for delete operations whose API
// endpoint returns no response body. It provides consistent JSON and table
// output matching the shape of OperationCompletionRS.
type deleteResult struct {
    Message string `json:"message"`
}

// newDeleteResult builds a deleteResult with a standard success message.
func newDeleteResult(resourceName string, id int64) deleteResult {
    return deleteResult{
        Message: fmt.Sprintf("%s %d deleted successfully", resourceName, id),
    }
}

// deleteResultRows formats a deleteResult as a single table row.
func deleteResultRows(result deleteResult) [][]string {
    return [][]string{{result.Message}}
}
```

Update call sites in `group.go` and `log_type.go`:

```go
// group.go delete handler — before:
result := groupDeleteResult{Message: fmt.Sprintf("Group %d deleted successfully", id)}
return printer.Print(result, messageResultHeaders(), [][]string{{result.Message}})

// group.go delete handler — after:
result := newDeleteResult("Group", id)
return printer.Print(result, messageResultHeaders(), deleteResultRows(result))
```

Remove the local `groupDeleteResult` type from `group.go` and `logTypeDeleteResult` type from `log_type.go`.

**Test:** Add to `mutation_result_test.go`:

```go
func TestNewDeleteResult(t *testing.T) {
    t.Parallel()

    result := newDeleteResult("Group", 42)
    assert.Equal(t, "Group 42 deleted successfully", result.Message)
}

func TestDeleteResultRows(t *testing.T) {
    t.Parallel()

    result := newDeleteResult("Widget", 7)
    rows := deleteResultRows(result)
    assert.Equal(t, [][]string{{"Widget 7 deleted successfully"}}, rows)
}

func TestDeleteResult_JSONMarshal(t *testing.T) {
    t.Parallel()

    result := newDeleteResult("Filter", 1)
    data, err := json.Marshal(result)
    require.NoError(t, err)
    assert.JSONEq(t, `{"message":"Filter 1 deleted successfully"}`, string(data))
}
```

### 2. Consolidate integration row formatters (if still duplicated)

**Expected finding:** PR 11 (`project_integration.go`) explicitly defined its own `projectIntegrationHeaders` and `projectIntegrationRow` to avoid coupling with `integration.go` (PR 10) during parallel development. Both have identical logic and identical column layouts (`ID`, `NAME`, `TYPE`, `ENABLED`, `CREATOR`) with the same `IntegrationType.Name` extraction pattern.

**Action:** Evaluate whether consolidation is worth it. The decision rule:

- **If the columns and logic are still identical after Phase 4 landed:** Extract a shared formatter into `integration.go` (or a new `integration_common.go` if the file is already long) and have both files call it. Name the shared functions `integrationResourceHeaders` and `integrationResourceRow` to indicate they format the shared `IntegrationResource` type, regardless of scope.

- **If the columns have diverged** (e.g., project integrations added `ProjectId` to the table): Leave them separate. Note the divergence in a code comment.

Likely action (columns are still identical):

```go
// integration_common.go (or at the top of integration.go)

func integrationResourceHeaders() []string {
    return []string{"ID", "NAME", "TYPE", "ENABLED", "CREATOR"}
}

func integrationResourceRow(integration *types.IntegrationResource) ([]string, []string) {
    typeName := ""
    if integration.IntegrationType != nil {
        typeName = formatStringPtr(integration.IntegrationType.Name)
    }

    return integrationResourceHeaders(), []string{
        formatInt64Ptr(integration.Id),
        formatStringPtr(integration.Name),
        typeName,
        formatBoolPtr(integration.Enabled),
        formatStringPtr(integration.Creator),
    }
}
```

Update `integration.go` and `project_integration.go` to use these shared functions. Remove the local `integrationHeaders`/`integrationRow` from `integration.go` and `projectIntegrationHeaders`/`projectIntegrationRow` from `project_integration.go`.

**Test:** Move any formatter test to `integration_test.go` (or a shared test file) and remove the now-redundant duplicate test from `project_integration_test.go`. Keep at least one test covering the nil `IntegrationType` case and one covering a fully-populated resource.

### 3. Extract required-flag validation helper

**Expected finding:** Multiple commands validate that a string flag is non-empty with a pattern like:

```go
name := mustGetStringFlag(cmd, "name")
if name == "" {
    err := fmt.Errorf("--name is required")
    printer.PrintError(err)
    return err
}
```

This appears in:
- `group.go` create and update (`--name`)
- `dashboard.go` create and update (`--name`)
- `log_type.go` create and update (`--name`)
- `integration.go` create (`--plugin`)
- `project_integration.go` create (`--plugin`)
- `project.go` create (`--name`)
- Any command validating `--from-file` **that uses the manual `mustGetStringFlag` + empty-check pattern** (not `cobra.MarkFlagRequired`)

**Important:** Only convert call sites that use the manual validation pattern (i.e., `mustGetStringFlag` followed by an `if value == ""` check with `printer.PrintError`). If any Phase 4 PR used `cobra.MarkFlagRequired` instead, leave that call site alone — `requireStringFlag` is not a drop-in replacement for Cobra's built-in required flag marking.

**Action:** Extract a reusable validation function into `helpers.go`:

```go
// requireStringFlag returns the value of the named flag, or an error if the
// flag was not provided or is empty.
func requireStringFlag(cmd *cobra.Command, name string) (string, error) {
    value := mustGetStringFlag(cmd, name)
    if value == "" {
        return "", fmt.Errorf("--%s is required", name)
    }

    return value, nil
}
```

Update all call sites to use the new helper. The calling pattern becomes:

```go
name, err := requireStringFlag(cmd, "name")
if err != nil {
    printer.PrintError(err)
    return err
}
```

This does not change behavior — it only eliminates the duplicated validation + error formatting logic. The error message format (`"--<flag> is required"`) is now consistent across all commands.

**Test:** Add to `helpers_test.go`:

```go
func TestRequireStringFlag_Present(t *testing.T) {
    t.Parallel()

    cmd := &cobra.Command{Use: "test"}
    cmd.Flags().String("name", "", "")
    _ = cmd.ParseFlags([]string{"--name", "hello"})

    value, err := requireStringFlag(cmd, "name")
    assert.NoError(t, err)
    assert.Equal(t, "hello", value)
}

func TestRequireStringFlag_Missing(t *testing.T) {
    t.Parallel()

    cmd := &cobra.Command{Use: "test"}
    cmd.Flags().String("name", "", "")

    _, err := requireStringFlag(cmd, "name")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "--name is required")
}

func TestRequireStringFlag_Empty(t *testing.T) {
    t.Parallel()

    cmd := &cobra.Command{Use: "test"}
    cmd.Flags().String("name", "", "")
    _ = cmd.ParseFlags([]string{"--name", ""})

    _, err := requireStringFlag(cmd, "name")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "--name is required")
}
```

### 4. Audit help text for consistency

Perform a manual audit of all `Short`, `Long`, and `Example` strings across the Phase 4 files. Ensure:

#### a. `Short` descriptions follow a consistent pattern

All existing commands use imperative mood, present tense, no trailing period:

- `"List dashboards or get a specific dashboard"` (get)
- `"Create a dashboard"` (create)
- `"Update a dashboard"` (update)
- `"Delete a dashboard"` (delete)

Verify every Phase 4 resource uses this pattern. Fix any that deviate.

#### b. `Long` descriptions include usage examples

Every get command should show at least:
- Basic list usage
- Get-by-ID usage (if applicable)
- Project override (if project-scoped)
- Output format switch (`-o json`)

Every mutation command should show:
- The required flags or `--from-file` usage
- Project override (if project-scoped)

Verify example command names match the registered `Use` string (e.g., `rp get log-type`, not `rp get logtype`).

#### c. Project-scoped commands mention project resolution

The `Long` description for project-scoped commands should mention the project resolution chain: `--project` flag → `RP_PROJECT` env var → `default-project` in the config profile. Verify this is stated consistently. Instance-scoped commands should explicitly state they do not require `--project`.

#### d. Admin-only commands are marked

Global integrations (PR 10) and any other admin-only resources should mention "admin" or "administrator" in the `Short` or `Long` description. Verify this is present.

### 5. Audit test consistency

Review all Phase 4 test files for consistent patterns:

#### a. Package declaration

All test files should use:

```go
package cli //nolint:testpackage
```

Verify none use `package cli_test`.

#### b. Parallel tests

All test functions and subtests should call `t.Parallel()`. Verify none are missing.

#### c. Naming conventions

Verify test function names follow the established pattern:
- `TestNew<Resource>Cmd_<Aspect>` for command construction tests
- `Test<Resource>Row_<Scenario>` for table formatting tests
- `Test<Helper>_<Scenario>` for helper function tests

#### d. `commandNames` usage

If any Phase 4 test file defines its own version of `commandNames` or a similar helper, move it to `test_helpers_test.go` (or remove the duplicate if it is already there).

#### e. Assertion libraries

All tests should use `testify` (`assert` and `require`). Verify none use raw `if` + `t.Errorf` patterns.

### 6. Verify `commands.go` registration completeness

Verify that `buildCommandTree` in `commands.go` (created by PR 3) has registrations for every command from PRs 4–15:

**Expected `get` registrations:**
- `newGetFilterCmd()`
- `newGetGroupCmd()`
- `newGetWidgetCmd()`
- `newGetDashboardCmd()`
- `newGetLogTypeCmd()`
- `newGetIntegrationCmd()`
- `newGetProjectIntegrationCmd()`

(Existing: `newGetLaunchCmd`, `newGetItemCmd`, `newGetLogCmd`, `newGetProjectCmd`, `newGetUserCmd`)

**Expected `create` registrations:**
- `newCreateFilterCmd()`
- `newCreateGroupCmd()`
- `newCreateWidgetCmd()`
- `newCreateDashboardCmd()`
- `newCreateLogTypeCmd()`
- `newCreateIntegrationCmd()`
- `newCreateProjectIntegrationCmd()`
- `newCreateProjectCmd()`
- `newCreateLogCmd()`

(Existing: `newCreateUserCmd`)

**Expected `update` registrations:**
- `newUpdateFilterCmd()`
- `newUpdateGroupCmd()`
- `newUpdateWidgetCmd()`
- `newUpdateDashboardCmd()`
- `newUpdateLogTypeCmd()`
- `newUpdateIntegrationCmd()`
- `newUpdateProjectIntegrationCmd()`
- `newUpdateProjectCmd()`
- `newUpdateLaunchCmd()`
- `newUpdateItemCmd()`

(Existing: `newUpdateUserCmd`)

**Expected `delete` registrations:**
- `newDeleteFilterCmd()`
- `newDeleteGroupCmd()`
- `newDeleteDashboardCmd()`
- `newDeleteLogTypeCmd()`
- `newDeleteIntegrationCmd()`
- `newDeleteProjectIntegrationCmd()`
- `newDeleteProjectCmd()`
- `newDeleteLaunchCmd()`
- `newDeleteItemCmd()`
- `newDeleteLogCmd()`

(Existing: `newDeleteUserCmd`)

Note: There is no `newDeleteWidgetCmd` (widget has no delete API).

If any registration is missing, add it. If there is a `commands_test.go` that asserts subcommand names, update the expected lists.

### 7. Verify `containsInt64` placement

PR 12 (`project.go`) introduces a `containsInt64` helper for checking `DeleteBulkRS` slices. Verify:

- If `containsInt64` is only used in `project.go`, leave it there.
- If any other Phase 4 file also defined a `containsInt64` or equivalent, consolidate into `helpers.go`.

Most likely it stays in `project.go` — this is a single-use utility specific to `DeleteBulkRS` handling.

### 8. Verify `logEntryCreatedHeaders`/`logEntryCreatedRows` placement

PR 15 defines local formatters for `EntryCreatedAsyncRS` (which has `Id *string` instead of `*int64`). Verify:

- These are only used in `log.go` — leave them there.
- If any other resource also returns `EntryCreatedAsyncRS`, move to `mutation_result.go`.

Most likely they stay in `log.go` — only log create uses the async create response.

### 9. Verify `buildLogTypeStyle` and `--level` Changed validation

PR 9 introduces the `buildLogTypeStyle` helper and uses `cmd.Flags().Changed("level")` to distinguish omitted flags from zero-value flags. Verify:

- `buildLogTypeStyle` is only used in `log_type.go` — leave it there.
- No other Phase 4 resource uses `Changed`-based validation for `int32` flags. If one does, assess whether to extract a shared `requireInt32Flag` helper.

### 10. Verify `formatProjectDeleteResult` placement

PR 12 introduces `formatProjectDeleteResult` in `project.go` to handle the complex `DeleteBulkRS` response. Verify:

- This is project-specific — leave it in `project.go`.
- No other resource uses `DeleteBulkRS`. If one does, consider extraction.

## Design notes

- **Only extract what is proven duplicated.** The audit may find fewer duplications than expected. If a pattern is only used once, leave it local. Do not speculatively extract.
- **Preserve error message content.** When replacing inline validation with `requireStringFlag`, verify the resulting error messages are equivalent. Users may have scripts that parse error output.
- **Integration formatter consolidation is optional.** If the columns have diverged between `integration.go` and `project_integration.go`, skip this step. The duplication is tolerable and the files should remain independently maintainable.
- **This PR should be small.** If the Phase 4 wave landed cleanly (consistent patterns, no drift), the changes in this PR will be minimal — possibly just the delete-result consolidation, the required-flag helper, and minor help text fixes. Do not manufacture work.

## What NOT to do

- **Do not add new commands.** This is a refactor-only PR. No new user-facing commands or flags.
- **Do not change command behavior.** All commands should produce identical output before and after this PR.
- **Do not extract helpers that are only used once.** If a pattern appears in exactly one file, leave it there.
- **Do not rename command `Use` strings, flag names, or subcommand names.** Those are part of the user-facing CLI surface and any change is a breaking change.
- **Do not refactor the printer or config infrastructure.** Those are out of scope.
- **Do not add integration tests.** This is a pure refactor.
- **Do not touch `listflags.go`** or the group-specific offset-based pagination helpers unless a second offset-based resource appeared in Phase 4 (none are expected).

## Validation

```bash
make build
make test
make lint
```

After the refactor, all existing commands must behave identically:

- Run `rp get --help` and verify all resource subcommands are listed
- Run `rp create --help`, `rp update --help`, `rp delete --help` and verify the same
- Spot-check help text for 2–3 resources to confirm examples render correctly
- Verify no unused imports or declarations remain in modified files (the linter will catch these)

## Definition of done

- [ ] Delete-result structs (`groupDeleteResult`, `logTypeDeleteResult`) consolidated into shared `deleteResult` / `newDeleteResult` / `deleteResultRows` in `mutation_result.go`
- [ ] All call sites updated to use the shared delete-result helpers
- [ ] `mutation_result_test.go` updated with tests for the new shared helpers
- [ ] Integration row formatters consolidated if columns are still identical (or documented as intentionally separate if diverged)
- [ ] `requireStringFlag` extracted into `helpers.go` with tests in `helpers_test.go`
- [ ] All call sites using the `mustGetStringFlag` + empty-check + error pattern updated to use `requireStringFlag`
- [ ] Help text audited for consistent `Short`/`Long` patterns, examples, project-resolution mentions, and admin markers
- [ ] Test files audited for consistent package declaration, `t.Parallel()`, naming conventions, and `testify` usage
- [ ] `commands.go` registration verified complete — all Phase 4 commands registered
- [ ] Single-use helpers (`containsInt64`, `logEntryCreatedHeaders`, `buildLogTypeStyle`, `formatProjectDeleteResult`) confirmed to be in their correct files
- [ ] No behavioral changes — all commands produce identical output
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
