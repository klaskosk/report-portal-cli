# Prompt 17: Start CLI Expansion (Milestone 13 Kickoff)

## Goal

Start curated CLI expansion on top of the now-stable client surface by shipping the first multi-family CLI slice across:

- write workflows
- reporting reads
- operator introspection

This PR should be intentionally bounded, script-friendly, and consistent with existing CLI UX conventions.

## Context

The roadmap gates CLI growth on pre-CLI client coverage and contract hardening. With that gate satisfied, this prompt launches Milestone 13 with practical commands users can run immediately while preserving a kubectl-like command experience.

## Scope and proposed command set

Implement at least one meaningful command family in each category below.

### Write workflows (required)

- `rp start launch`
- `rp finish launch`
- `rp log add`

If scope allows and client methods are stable, include:

- `rp start item`
- `rp finish item`

### Reporting reads (required)

- `rp get dashboard` (list/get)
- `rp get widget` (list/get where practical)
- `rp get filter` (list/get)

### Operator introspection (required)

- `rp get integration` (global/project read flows as supported)
- `rp get bts` (high-signal introspection flows)
- `rp activity list` (time/user/type/project filters)

## UX and command design requirements

- Preserve existing verb-first taxonomy (`rp get ...`, `rp start ...`, `rp finish ...`).
- Keep JSON output machine-stable and aligned with API field names.
- Provide table output with uppercase column headers.
- Prefer minimal, clear flags over large flag sprawl.
- For complex payloads, support file-driven input (`--from-file`) instead of dozens of inline flags.

## Implementation plan

### 1) Wire CLI commands to existing client scopes

- Add/extend command handlers under `internal/cli/`.
- Reuse existing output formatting pipeline (table/json/go-template).
- Ensure command errors map cleanly to existing predicate-style messages.

### 2) Input model and validation

- Add shared request parsing helpers for write commands.
- Validate required fields before API calls.
- Keep error messages lowercase and concise.

### 3) Output contracts

- For each new command, define and test:
  - table columns
  - JSON contract shape
  - representative failure output behavior

## Test requirements

### Unit tests (CLI package level)

Add/extend tests for:

- argument/flag parsing and validation
- client call wiring with expected parameters
- formatter behavior for table/json/template outputs
- representative error mapping

### Integration tests (`test/integration/`)

Add new command-family integration coverage for:

- launch start/finish lifecycle
- log add flow (with practical payload)
- reporting reads for dashboard/widget/filter resources
- introspection reads for integration/BTS/activity

For each command family, include at least:

- one success path with JSON assertions
- one representative error path

## Documentation requirements

- Update README command examples for all newly added commands.
- Add concise examples for script usage (JSON piping, template output, and file-driven input).
- Document any environment prerequisites for introspection commands.

## Design considerations

- Keep this kickoff PR bounded; prefer vertical slices that are complete and tested.
- Do not expose every possible API flag if it harms ergonomics.
- Use aliases only where compatibility demands it.
- If a command would be too complex without file input, make file input mandatory.

## Validation commands

Run before finalizing:

- `go test ./...`
- `golangci-lint-v2 run --fix`
- `go test -tags=integration ./test/integration/...` (or project-standard integration target)

## Definition of done

- [ ] New write commands shipped (`rp start/finish launch`, `rp log add`; item start/finish if included)
- [ ] New reporting read commands shipped (`rp get dashboard/widget/filter`)
- [ ] New introspection commands shipped (`rp get integration`, `rp get bts`, `rp activity list`)
- [ ] Unit tests added/extended for parsing, wiring, output, and errors
- [ ] Integration tests added for each command family with JSON/error assertions
- [ ] README/examples updated for new command usage
- [ ] Lint and tests pass locally
- [ ] `code-verifier` subagent says your changes are good to go
