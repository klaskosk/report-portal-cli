# Prompt 03: Stabilize End-to-End Read Workflow

## Goal

Lock down naming conventions, stabilize JSON output contracts, and add integration tests that exercise the full read workflow: launch → item → log → attachment download. This corresponds to Milestone 5 (Workflow parity + operator UX) and the "lock naming + stabilize end-to-end read flow" PR from the suggested next steps.

## Context

By the time this prompt is executed, the codebase should have:

- `rp get launch` — list/get launches
- `rp get item` — list/get test items
- `rp get log` — list/get logs (from Prompt 01)
- `rp download` — download attachments (from Prompt 02)

This prompt is about polish, consistency, and confidence — not new API surface.

## Naming audit

Review and document canonical resource names throughout the codebase:

1. **CLI nouns**: Confirm `launch`, `item`, and `log` are the canonical nouns. `test-item` should remain only as an alias for `item` (already implemented). No other aliases should be added.
2. **Client scope names**: `LaunchScope`, `TestItemScope`, `LogScope` — verify these are consistent. The `TestItemScope` name is intentional (matches the API domain concept) even though the CLI uses the shorter `item`.
3. **JSON output field names**: Audit the JSON output of each `rp get` command. Field names should match the API resource names exactly (camelCase from Report Portal's API). Do not rename or transform fields in JSON output mode — JSON output is a machine contract.
4. **Table output column names**: These can be human-friendly (e.g., "Launch ID" vs `launchId`). Audit for consistency across all three resource types.
5. **Error messages**: Ensure error messages use consistent resource terminology (e.g., "launch not found" not "Launch not found" or "launch was not found").

Document decisions in a short section added to `CLAUDE.md` under a "## Naming conventions" heading.

## JSON output stability

For each `rp get` command, define and test the output contract:

- `rp get launch <id> -o json` — must include at minimum: `id`, `uuid`, `name`, `status`, `startTime`, `endTime`, `number`, `description`
- `rp get launch -o json` — must return an array of launch objects (same shape as above)
- `rp get item <id> -o json` — must include at minimum: `id`, `uuid`, `name`, `type`, `status`, `startTime`, `endTime`, `launchId`, `parent`
- `rp get item -o json` — must return an array of item objects
- `rp get log <id> -o json` — must include at minimum: `id`, `uuid`, `level`, `message`, `itemId`, `launchId`
- `rp get log --item <id> -o json` — must return an array of log objects

Add integration test assertions that parse the JSON output and verify these fields are present and have the expected types. These tests serve as regression guards for the machine-readable contract.

## End-to-end integration test

Add a new test file `test/integration/workflow_test.go` (or a `Describe` block in an existing file) that exercises the full read workflow:

1. **Seed data**: Create a project, start a launch, add test items (suite + test), finish items with results, finish the launch. If attachments were seeded by Prompt 02, include them.
2. **List launches**: `rp get launch -o json` → parse JSON, verify the seeded launch appears
3. **Get launch by ID**: Extract launch ID from step 2, `rp get launch <id> -o json` → verify fields
4. **List items for launch**: `rp get item --launch <id> -o json` → verify items appear
5. **Get item by ID**: Extract an item ID, `rp get item <id> -o json` → verify fields
6. **List logs for item**: `rp get log --item <id> -o json` → verify logs appear
7. **Get log by ID**: Extract a log ID, `rp get log <id> -o json` → verify fields
8. **Download attachment** (if available): Extract `binaryContent.id` from a log, `rp download <data-id>` → verify non-empty output

This test should be a single `Describe("end-to-end read workflow")` block with ordered `It` steps. Use `Ordered` container from Ginkgo to ensure steps run sequentially and share state.

## Machine-readable error output

Verify that error responses in JSON-output mode are also machine-readable:

- When `--output json` is set and an API error occurs, the error should still be printed as structured JSON to stderr (or at minimum, the exit code should be non-zero and the error message should be parseable)
- Test: `rp get launch 999999 -o json` → non-zero exit, stderr contains error info

## CLI help text audit

Review `--help` output for all commands and ensure:

- Examples are up-to-date and use consistent flag names
- Short descriptions are consistent in tone and capitalization
- Long descriptions mention JSON output and go-template as options

## Definition of done

- [ ] Naming conventions documented in `CLAUDE.md`
- [ ] JSON output assertions for all `rp get` commands in integration tests
- [ ] End-to-end workflow integration test covering launch → item → log → download
- [ ] Machine-readable error output verified
- [ ] CLI help text reviewed and updated
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-integration` passes
