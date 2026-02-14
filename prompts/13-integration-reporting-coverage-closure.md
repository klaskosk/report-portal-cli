# Prompt 13: Coverage Closure for Integration/Reporting Controllers (Milestone 11, Part 2)

## Goal

Close practical remaining client coverage gaps for:

- `integration-controller`
- `bug-tracking-system-controller`
- `dashboard-controller`
- `widget-controller`
- `user-filter-controller`

This PR should complete the automation-relevant endpoint surface for these controllers before broader CLI expansion.

## Context

Prompts 08 and 09 introduced broad support for these areas, but the roadmap still calls for closure of remaining CRUD/command/list-variant gaps. This slice should tighten completeness and consistency for reporting and external-system automation workflows.

## Scope and exclusions

### In scope

- Remaining non-async, high-value wrapper gaps in the listed controllers.
- Missing list variants and command operations that users commonly need in automation.
- Ergonomic functional options and helper methods where query surfaces are complex.

### Out of scope

- New CLI commands.
- Rare, plugin-specific endpoints with unstable/opaque contracts unless broadly useful.
- Reworking already-stable method signatures without clear benefit.

## Discovery workflow (required first step)

Build a controller-by-controller coverage map before coding:

1. Extract generated methods for each target controller from `pkg/internal/api/*.gen.go`.
2. Extract currently wrapped methods from:
   - `pkg/client/integrations.go`
   - `pkg/client/bts.go`
   - `pkg/client/dashboard.go`
   - `pkg/client/widget.go`
   - `pkg/client/filter.go`
3. Mark each generated endpoint as:
   - `covered`
   - `missing`
   - `deferred` (with explicit reason)
4. Select a bounded missing set for this PR prioritizing practical automation usage.

Include this map in the PR description and call out what remains deferred.

## Client requirements

Follow project wrapper conventions:

- scope-based accessors from `Client` and `ProjectScope`
- typed request/response contracts from `pkg/types`
- `JSONDefault` error-first response handling
- functional options for pagination/sorting/filtering

### Integration scope completion

In `pkg/client/integrations.go`, close gaps for both:

- instance-level global integrations (`Client.Integrations()`)
- project-level integrations (`Client.Project(name).Integrations()`)

Prioritize:

- list/get variants
- create/update/delete completeness where still missing
- validate/test/execute command paths that are broadly used

### BTS scope completion

In `pkg/client/bts.go`, close practical gaps for:

- retrieving integration metadata needed for issue creation
- issue type/field lookups
- issue retrieval/creation/linking paths that are non-async and high value

If endpoint responses are weakly typed, preserve raw typed payloads from generated models without schema rewriting.

### Dashboard/widget/filter completion

Close remaining wrapper gaps across:

- dashboard lifecycle and widget composition operations
- widget retrieval/update/name-list variants
- filter list/name/query variants and lifecycle operations

Ensure pagination/sorting options are consistently named and match existing option patterns.

## Unit test requirements

Extend these suites:

- `pkg/client/integrations_test.go`
- `pkg/client/bts_test.go`
- `pkg/client/dashboard_test.go`
- `pkg/client/widget_test.go`
- `pkg/client/filter_test.go`
- `pkg/client/export_test.go` (if new constructors/accessors are needed)

Per added method, include:

- success path
- representative error mapping
- request wiring assertions for query/body/path details

Use table-driven tests with `t.Parallel()`.

## Integration test requirements

Add or extend integration coverage in `test/integration/` for stable, realistic workflows:

- integration list/get/validate flows that work in the test stack
- BTS metadata retrieval and safe issue operations where environment allows
- dashboard/widget/filter lifecycle assertions for created resources

Where external dependencies are environment-sensitive (BTS), keep tests resilient and document prerequisites.

## Design considerations

- Keep public method names explicit and predictable (`ListByPlugin`, `GetNames`, etc.).
- Avoid duplicative wrappers for semantically identical generated methods.
- Return canonical data types and avoid introducing client-specific reshaping.
- Favor practical completeness over adding obscure, low-signal methods.

## Validation commands

Run before finalizing:

- `go test ./...`
- `golangci-lint-v2 run --fix`
- `go test -tags=integration ./test/integration/...` (or project-standard integration target)

## Definition of done

- [ ] Coverage map exists for integration/BTS/dashboard/widget/filter controllers
- [ ] Remaining high-value wrapper gaps are closed for instance- and project-scoped integrations
- [ ] Remaining high-value BTS wrapper gaps are closed
- [ ] Remaining high-value dashboard/widget/filter wrapper gaps are closed
- [ ] Unit tests added/extended for all newly wrapped methods
- [ ] Integration tests added/extended for practical workflows
- [ ] Lint and tests pass locally
- [ ] `code-verifier` subagent says your changes are good to go
