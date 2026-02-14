# Prompt 15: Pre-CLI Coverage Audit + Contract Lock (Milestone 11, Gate)

## Goal

Establish a verifiable pre-CLI gate by:

1. Publishing a tracked endpoint coverage matrix (`covered`, `deferred`, `excluded`) for agreed controllers.
2. Locking automation-facing client/CLI response and error behavior with explicit tests and documentation.

This PR is primarily audit, contracts, and tests (minimal feature work only if needed to remove ambiguity).

## Context

Roadmap sequencing requires client coverage confidence before broader CLI expansion. This prompt creates the explicit artifact and contract tests that make the gate objective and repeatable.

## Scope and exclusions

### In scope

- Coverage matrix artifact based on generated OpenAPI methods vs wrapper methods.
- Deferred/excluded rationale documentation.
- Contract tests for JSON output/error behavior of covered workflows.
- Lightweight tooling/scripts used to keep the matrix maintainable.

### Out of scope

- Broad new endpoint implementation (belongs to prompts 12-14 if gaps remain).
- Large CLI command additions.

## Required deliverables

### 1) Coverage matrix document

Create a tracked matrix document (for example `docs/coverage/pre-cli-endpoint-matrix.md`) covering at least:

- `launch-controller`
- `test-item-controller`
- `log-controller`
- `integration-controller`
- `bug-tracking-system-controller`
- `project-controller`
- `project-settings-controller`
- `dashboard-controller`
- `widget-controller`
- `user-filter-controller`
- `activity-controller`
- `activity-event-controller`
- `file-storage-controller` (practical flows)

Each row should include:

- generated endpoint identifier
- wrapper method (or `n/a`)
- status: `covered` | `deferred` | `excluded`
- rationale for deferred/excluded

### 2) Contract lock document

Create/update a contract document (for example `docs/contracts/automation-contracts.md`) that explicitly states:

- JSON output stability policy (field names from API types; `omitempty` behavior)
- error mapping expectations (predicate behavior and representative status mappings)
- scope accessor and naming guarantees relevant to automation users

### 3) Contract tests

Add/extend tests that lock core contracts:

- stable JSON output structure for representative CLI read commands already shipped
- stable error classification behavior for representative client methods
- no field renaming on wrapped response objects used by automation

Use existing integration and unit test suites where possible.

## Audit workflow

1. Enumerate generated endpoints by target controller.
2. Map to current wrappers and current CLI coverage.
3. Produce matrix in sorted, review-friendly format.
4. Resolve inconsistencies:
   - if status says `covered`, ensure wrapper exists and has tests
   - if status says `deferred` or `excluded`, include concrete rationale

If material implementation gaps are discovered, either:

- include small closure fixes in this PR, or
- explicitly link to follow-up prompt/PR and keep status non-covered.

## Testing requirements

### Unit tests

Add/update tests in relevant client packages to verify:

- error predicate behavior for representative status codes
- stable request/response handling around `JSONDefault`

### Integration tests

Add/update integration assertions for existing CLI workflows:

- `rp get launch`
- `rp get item`
- `rp get log`
- `rp get project`
- `rp whoami`

Focus on output contract shape and stable error behavior rather than volatile value snapshots.

## Design considerations

- The matrix should be reviewable in PR diffs and easy to update as coverage evolves.
- Prefer clear rationale text over ambiguous labels for deferred/excluded rows.
- Contract tests should fail only on meaningful behavior regressions, not data timing noise.
- Keep this PR as the objective gate artifact for Milestone 13 readiness.

## Validation commands

Run before finalizing:

- `go test ./...`
- `golangci-lint-v2 run --fix`
- `go test -tags=integration ./test/integration/...` (or project-standard integration target)

## Definition of done

- [ ] Endpoint matrix document is added and includes all agreed pre-CLI controllers
- [ ] Every matrix row is explicitly marked `covered`, `deferred`, or `excluded`
- [ ] Deferred/excluded rows include rationale
- [ ] Contract document for automation-facing JSON/error behavior is added or updated
- [ ] Contract tests added/extended for representative client and CLI surfaces
- [ ] Lint and tests pass locally
- [ ] `code-verifier` subagent says your changes are good to go
