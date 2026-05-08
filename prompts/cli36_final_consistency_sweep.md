# PR 36: Final CLI Consistency Sweep

## Goal

Clean up any remaining help text drift, duplicated code, test gaps, or cross-command inconsistencies that surfaced during the Phase 7 specialized command wave (PRs 18–35). This is the final fan-in checkpoint before the CLI is considered feature-complete.

## Context

After the Phase 7 fan-out, multiple agents implemented commands independently. Even with the frozen UX shapes from PR 17 and the shared helpers from PR 7, drift is expected. This PR is reactive — the agent inspects the codebase as it exists after all Phase 7 PRs have landed and fixes what it finds.

**Prerequisite:** All PRs 1–35 must be merged.

## Inspection checklist

Work through each category below. For each, inspect the relevant files, note any issues, and fix them in place.

### 1. Help text and examples

- **Consistency:** Do all commands use the same phrasing patterns? Check for: consistent capitalization in `Short` descriptions, consistent example formatting in `Long` descriptions, consistent `Use` patterns (e.g., `"resource <id>"` vs `"resource"`).
- **Accuracy:** Do examples in `Long` descriptions match the actual flag names? After renames or flag changes, stale examples may reference old names.
- **Completeness:** Does every command with non-obvious behavior have at least one example in its `Long` text? Priority targets: `rp update issue-sub-type` (locator vs ID), `rp update notification <id> --file` (positional ID injected/validated against the DTO), all file-backed commands (should show sample JSON path), all `--async` commands.

### 2. Flag naming and registration

- **Verify `--project` / `-p` presence:** Every project-scoped command must have it. Every instance-scoped command must NOT have it. Cross-reference against the settled scoping in PR 17.
- **Verify `--file` vs `--from-file`:** Cross-check against PR 17's settled rule. Use `--file` when the file is the command's primary request-body input mode, even if the command also has a positional target ID; use `--from-file` only when the file is one alternative among multiple request-construction modes. Preserve `-f` only where the settled UX for that command family includes it.
- **Verify `--output-file` on all byte-output commands:** `export project`, `export user`, `get launch-report`.
- **Verify no duplicate short flags:** Check that `-f` isn't claimed by two different flags on the same command.

### 3. Formatter consistency

- **Row function signatures:** Reusable per-item/resource formatters should return `([]string, []string)`. Helpers whose job is to build the final one-row `[][]string` payload for `printer.Print` (for example shared result helpers such as `operationResultRows` / `successfulUpdateRows` / `entryCreatedRows`) may return `[][]string`. Avoid mixing the two patterns within the same formatter family unless one helper is the reusable row formatter and the other is the final printer adapter.
- **Header casing:** All table headers should be UPPERCASE. Check for any lowercase headers.
- **Nil handling:** All formatters that receive pointer fields should use the shared `formatStringPtr` / `formatInt64Ptr` / `formatBoolPtr` / `formatInt32Ptr` helpers. Check for inline nil checks that should use the helpers instead.
- **Duplicated formatters:** Check whether `activity.go` and `activity_event.go` keep `ActivityEventResource` table output aligned and whether any duplication should be extracted during this final fan-in. Check whether `integration.go` and `project_integration.go` share any common formatters. If duplicate formatters exist, extract to a shared location or verify they're intentionally separate.

### 4. Error message consistency

- **`parseResourceID` usage:** All numeric ID parsing should use `parseResourceID(arg, "resource-name")` with the correct resource name. Check for manual `strconv.ParseInt` calls that should use the helper.
- **Validation messages:** Check for consistent error message phrasing. Prefer `"--flag is required"` or `"provide a <resource> ID"` patterns.
- **Mutual exclusivity messages:** Commands with mutually exclusive flags (e.g., `rp assign user --group / --project-name`, positional ID + `--item` in `rp get activity`) should have clear error messages explaining the conflict.

### 5. Import tidiness

- **Unused imports:** Check all Phase 7 files for unused imports. Common causes: helpers that were extracted after the file was written, formatter functions that were moved.
- **Import grouping:** Go convention is stdlib, blank line, external packages, blank line, internal packages. Verify consistency.

### 6. Test coverage gaps

- **Formatter tests:** Every formatter function should have at least one test verifying headers and row values, and one test verifying nil/empty field handling (where applicable).
- **Flag registration tests:** Every command constructor should have a test verifying its flag set.
- **Edge case tests:** Commands with validation logic (mutual exclusivity, required flags, positional arg parsing) should have tests for the error paths.
- **Missing tests:** List any commands whose test files are missing or empty.

### 7. Command registration completeness

- **Verify `commands.go`:** Every command constructor created in PRs 18–35 must be registered under the correct parent verb. Cross-reference the `commands.go` file against the prompts.
- **Verify `--help` output:** Run `rp get --help`, `rp create --help`, `rp update --help`, `rp delete --help`, and each specialized verb (`rp start --help`, `rp finish --help`, `rp assign --help`, `rp unassign --help`, `rp link --help`, `rp unlink --help`, `rp analyze --help`, `rp compare --help`, `rp merge --help`, `rp export --help`, `rp generate --help`). Verify every expected subcommand appears.

### 8. Cross-command contract verification

- **Byte-output commands:** Verify all three (`export project`, `export user`, `get launch-report`) use the shared `writeBytes` helper and call `warnIfFormatFlagSet`.
- **Async commands:** Verify all `--async` commands (`start launch`, `start item`, `finish launch`, `finish item`, `merge launch`) consistently select the async variant.
- **File-backed JSON commands:** Verify all commands that read JSON from disk use `readRequestFromFile[T]` with the correct type parameter, regardless of whether the user-facing flag is `--file` or `--from-file`.
- **Pagination commands:** Verify offset-based commands (`rp get activity-event`) use `--limit`/`--offset` and page-based commands use `--page`/`--page-size`. No mixing.

## Implementation

This PR should produce **only bug fixes and cleanups** — no new features, no new commands, no new flags.

### Process

1. Start by building the CLI and running the full test suite to establish a clean baseline:
   ```bash
   make build && make test && make lint
   ```
   If anything fails at baseline, fix it first.

2. Work through the inspection checklist above. For each issue found:
   - Fix the issue in place
   - Run `make build && make test && make lint` after each batch of related fixes
   - Note what was fixed in the PR description

3. After all fixes are applied, do a final verification pass:
   ```bash
   make verify
   ```

### What counts as a fix

- Typos and inconsistencies in help text
- Missing or incorrect flag registrations
- Duplicated code that should use shared helpers
- Missing test cases for existing functionality
- Stale imports or dead code
- Incorrect formatter signatures

### What does NOT count as a fix

- New commands or flags
- Behavior changes to existing commands
- Refactoring that changes public API surface
- Performance optimizations

## Validation

```bash
make verify
```

Additionally, spot-check `--help` output for every top-level verb to confirm all subcommands are visible.

## Definition of done

- [ ] All inspection checklist items reviewed
- [ ] All identified issues fixed
- [ ] No new commands or flags added
- [ ] `make verify` passes
- [ ] `--help` output verified for all verb groups
- [ ] PR description lists every fix applied
- [ ] This file moved to the `completed` directory
