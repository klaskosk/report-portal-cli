# Prompt 12: Coverage Closure for Core Execution Controllers (Milestone 11, Part 1)

## Goal

Close the remaining non-async client coverage gaps for the three core execution controllers in `pkg/client`:

- `launch-controller`
- `test-item-controller`
- `log-controller`

This PR is client-first and must prioritize practical automation completeness over exhaustive low-value parity.

## Context

Core read and write workflows already exist from earlier slices, but the roadmap still identifies closure gaps in high-value operations (history/search/compare/import/export/linking/attributes). This prompt is the first pre-CLI gate slice and should leave only explicitly deferred or excluded endpoints.

## Scope and exclusions

### In scope

- Non-async endpoints under launch/item/log that are useful for daily operators and CI/CD automation.
- Missing list variants, query/filter combinations, and high-signal command endpoints.
- Missing request/response wrappers needed to make existing capabilities scriptable from the client.

### Out of scope

- `launch-async-controller`, `test-item-async-controller`, `log-async-controller`.
- One-off legacy/deprecated edges with low operator value unless required by dependencies.
- New CLI commands (this is still client-only).

## Discovery workflow (required first step)

Before writing code, produce a short gap list from generated endpoints to current wrappers.

1. Enumerate generated methods for the three controllers in `pkg/internal/api/*.gen.go`.
2. Enumerate existing methods in:
   - `pkg/client/launch.go`
   - `pkg/client/item.go`
   - `pkg/client/log.go`
3. Map generated endpoints to wrapper coverage with statuses:
   - `covered`
   - `missing`
   - `deferred` (with reason)
4. Use that mapping to select a bounded set of missing endpoints for this PR (must be the highest-value missing set).

Include the gap list in the PR description.

## Client requirements

Use existing wrapper conventions consistently:

- scope-based API only (`Client` -> `ProjectScope` -> resource scope)
- `response.JSONDefault` checked before success payload usage
- error wrapping style: lowercase, no trailing punctuation
- functional options for list/search endpoints, not long parameter lists

### `LaunchScope` completion (`pkg/client/launch.go`)

Close missing high-value methods in these categories (as applicable from your gap list):

- history and comparison endpoints
- import/export/report retrieval endpoints
- link/attribute mutation helpers that are practical for automation
- list/search variants currently reachable only through generated client

Prefer typed return values from `pkg/types` and preserve API field names without renaming.

### `TestItemScope` completion (`pkg/client/item.go`)

Close missing high-value methods in these categories:

- history/retry/navigation helpers
- linking and issue-related operations useful for triage workflows
- bulk/list/search variants needed for operational tooling

Add/extend functional options for query-heavy list/search operations if missing.

### `LogScope` completion (`pkg/client/log.go`)

Close missing high-value methods in these categories:

- search/list variants with practical filters
- endpoint variants needed for attachment-aware workflows
- bulk retrieval helpers for script automation

If the API exposes multiple response variants for similar operations, pick one canonical wrapper and document why.

## Unit test requirements

Extend the existing suites (do not create white-box tests):

- `pkg/client/launch_test.go`
- `pkg/client/item_test.go`
- `pkg/client/log_test.go`
- `pkg/client/export_test.go` (test constructors if new scopes/helpers are added)

For each added method, cover:

- success response
- at least one representative mapped API error (`401`, `403`, or `404`)
- request wiring (path/query/body) where relevant

Use table-driven tests and keep `t.Parallel()` everywhere.

## Integration test requirements

Add focused workflows under `test/integration/` for the highest-value newly wrapped behavior:

- at least one end-to-end launch workflow using newly added launch method(s)
- at least one item workflow using newly added item method(s)
- at least one log workflow using newly added log method(s)

Keep integration scenarios deterministic and avoid brittle assertions against volatile metadata.

## Design considerations

- Prefer small wrapper methods over over-generalized abstractions.
- If an endpoint has weak typing, return the best available generated type; avoid inventing custom schema translations.
- Keep method naming aligned with existing scope conventions (`Get`, `List`, `Create`, `Update`, `Delete`, action verbs only when clear).
- Do not leak generated client types from `pkg/internal/api` into public signatures.

## Validation commands

Run these before finalizing:

- `go test ./...`
- `golangci-lint-v2 run --fix`
- `go test -tags=integration ./test/integration/...` (or project-standard integration target)

## Definition of done

- [ ] Gap list between generated and wrapped launch/item/log endpoints is produced and used to scope the PR
- [ ] Missing high-value non-async launch methods are wrapped with idiomatic client APIs
- [ ] Missing high-value non-async item methods are wrapped with idiomatic client APIs
- [ ] Missing high-value non-async log methods are wrapped with idiomatic client APIs
- [ ] Unit tests added/extended for all new methods
- [ ] Focused integration tests added for newly wrapped high-value workflows
- [ ] Lint and tests pass locally
- [ ] `code-verifier` subagent says your changes are good to go
