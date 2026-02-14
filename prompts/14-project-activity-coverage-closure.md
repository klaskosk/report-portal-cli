# Prompt 14: Coverage Closure for Project/Activity Surfaces (Milestone 11, Part 3)

## Goal

Complete the remaining practical client coverage gaps for:

- `project-controller`
- `project-settings-controller`
- `activity-controller`
- `activity-event-controller`

This PR should finish the operator-focused project and audit/event surface needed before CLI growth.

## Context

Project discovery and settings foundations already exist, and activity scopes were planned in earlier milestone work. The roadmap now calls for closure of remaining useful endpoints for automation, troubleshooting, and audit workflows.

## Scope and exclusions

### In scope

- Missing high-value non-async project and project-settings endpoints.
- Missing high-value activity and activity-event wrappers with practical filters.
- Functional options and helper builders needed for common query patterns.

### Out of scope

- Admin-heavy edges with low automation value (unless required by dependencies).
- New CLI commands in this PR.
- Async controllers.

## Discovery workflow (required first step)

Produce a scoped coverage map before coding:

1. Enumerate generated methods for the four target controllers.
2. Compare against current wrappers in:
   - `pkg/client/projects.go`
   - `pkg/client/project_settings.go`
   - any existing `activity*.go` files (create if absent)
3. Classify endpoints as `covered`, `missing`, `deferred`.
4. Select highest-value missing endpoints for this PR.

Add the map summary to the PR description.

## Client requirements

Follow existing client patterns:

- `Client` and `ProjectScope` accessors for scope ownership
- functional options for list/search methods
- `JSONDefault` checks and predicate-friendly error handling
- typed `pkg/types` contracts with no field renaming

### Project coverage completion (`pkg/client/projects.go` + `pkg/client/project.go`)

Close remaining high-value project operations relevant to automation:

- discovery/list/get variants
- project-level metadata/configuration actions that are practical and non-destructive
- any missing subset operations required by downstream scopes

If adding project-scoped methods, place them on `ProjectScope` with clear naming.

### Project settings completion (`pkg/client/project_settings.go`)

Close practical settings gaps still needed for:

- notification and rules workflows
- issue-type/template settings management
- other settings read/write paths used by automation or troubleshooting

Keep payload and response types exactly aligned with generated models.

### Activity scope completion (`pkg/client/activity.go`)

Implement or complete project-scoped activity APIs:

- get by ID and list/query methods
- filter/pagination/sort options for common operator use
- list helpers for item/launch-oriented activity lookups where available

### Activity event scope completion (`pkg/client/activity_events.go`)

Implement or complete instance-scoped activity event APIs:

- structured search method(s)
- helper builders for common filters (project, user, object type, event name, time)
- subject lookup helpers when provided by API

Keep filter helper APIs composable and typed.

## Unit test requirements

Add or extend:

- `pkg/client/projects_test.go`
- `pkg/client/project_settings_test.go`
- `pkg/client/activity_test.go` (new or extend)
- `pkg/client/activity_events_test.go` (new or extend)
- `pkg/client/export_test.go`

Per new method, cover:

- success path
- representative error mapping
- query/body/path wiring assertions

Use table-driven black-box tests with `t.Parallel()`.

## Integration test requirements

Add focused integration coverage for:

- project/settings workflows that are stable in the test stack
- activity generation and retrieval after seeded launch/item/log actions
- activity-event searches with scoped filters to avoid flakiness

Use deterministic seed/setup patterns and cleanup where needed.

## Design considerations

- Activity data can be eventually consistent; polling/retry may be needed in integration tests.
- Keep helper names explicit (`FilterByProjectName`, `FilterByUser`, etc.) and reusable.
- Avoid making settings mutation tests destructive for shared resources.
- Do not expose generated-client internals in public APIs.

## Validation commands

Run before finalizing:

- `go test ./...`
- `golangci-lint-v2 run --fix`
- `go test -tags=integration ./test/integration/...` (or project-standard integration target)

## Definition of done

- [ ] Coverage map exists for project/project-settings/activity/activity-event controllers
- [ ] Remaining high-value project and project-settings wrapper gaps are closed
- [ ] Activity scope is implemented/completed with practical query options
- [ ] Activity-event scope is implemented/completed with practical search helpers
- [ ] Unit tests added/extended for all new methods
- [ ] Integration tests added/extended for seeded project/audit workflows
- [ ] Lint and tests pass locally
- [ ] `code-verifier` subagent says your changes are good to go
