# Report Portal Go Client Roadmap (CLI follows)

**Target**: Report Portal **24.1** / API **v5.11** (OpenAPI in `pkg/internal/openapi/`)

**End state**:

- **Go client (`pkg/client`)**: ergonomic, scope-based wrapper covering all high-value and automation-critical API areas needed for practical usage (and eventually broad controller coverage, including admin/system where it makes sense).
- **CLI (`rp`)**: curated day-to-day workflows built on the client; command surface expands after the underlying client areas are stable and well-tested.
- **Long-tail parity track**: remaining low-value/deprecated/admin endpoints can be added incrementally after practical completion.

This document is a high-level, shippable roadmap broken into small, independently completable slices (typically one API tag/controller or endpoint group per PR; CLI follow-ups are separate slices).

## Strategy: client-first execution

- **Ship `pkg/client` first**: all new API surface lands in the wrapper (scopes + options + error handling) with unit tests.
- **CLI is a follow-up**: add/expand `internal/cli` commands only after the underlying client area is solid (usually in a separate PR), so CLI UX doesn’t constrain client API shape.
- **CLI remains curated**: the CLI exposes workflows; the client is where broad API coverage lives.

---

## Current baseline (implemented)

- **Client**:
  - Project-scoped reads: `LaunchScope`, `TestItemScope`, `LogScope`
  - Project-scoped file retrieval: `ProjectScope.DownloadFile`
  - Instance-scoped discovery: `UsersScope` (`GetMyself`, `GetByLogin`, `ListProjects`), `ProjectsScope` (`Get`, `ListNames`)
  - Error handling: `*client.APIError` + predicates (`IsNotFound`, etc.)
- **CLI**:
  - Read commands: `rp get launch`, `rp get item`, `rp get log`, `rp get project`
  - Utility/action commands: `rp download`, `rp whoami`
  - Config resolution: flags > env > config profiles
  - Output modes: `table`, `json`, `go-template=...` with JSON contract stabilization for required keys
- **Testing**:
  - Client: black-box unit tests (`pkg/client/*_test.go`) using `httptest`
  - CLI + client integration tests (Ginkgo) in `test/integration/`
  - End-to-end read workflow integration coverage: launch -> item -> log -> download

---

## Recently completed slices

- [x] **PR: reduce wrapper boilerplate + baseline CLI consistency**
  - Added shared response/error helpers and normalized list/pagination behavior.
- [x] **PR: expand launch read surface**
  - Client + CLI launch read/list/filter/report workflows.
- [x] **PR: expand test-item read surface**
  - Client + CLI test-item read/list workflows aligned to launch navigation.
- [x] **PR: add log read surface** → [`prompts/completed/01-log-read-surface.md`](prompts/completed/01-log-read-surface.md)
  - `LogScope` + `rp get log` with filtering and JSON/table coverage.
- [x] **PR: add attachment download** → [`prompts/completed/02-attachment-download.md`](prompts/completed/02-attachment-download.md)
  - `DownloadFile` + `rp download` for attachment retrieval.
- [x] **PR: lock naming + stabilize end-to-end read flow** → [`prompts/completed/03-stabilize-read-workflow.md`](prompts/completed/03-stabilize-read-workflow.md)
  - Naming conventions documented and workflow integration coverage added.
- [x] **PR: add identity + discovery follow-up** → [`prompts/completed/04-identity-discovery.md`](prompts/completed/04-identity-discovery.md)
  - `UsersScope`/`ProjectsScope` + `rp whoami`/`rp get project`.
- [x] **PR: client write workflows for launches/items/logs (Milestone 6)** → [`prompts/completed/05-client-write-workflows.md`](prompts/completed/05-client-write-workflows.md)
  - Added core launch/item/log mutation workflows in `pkg/client`.
- [x] **PR: client discovery + identity depth (Milestone 7)** → [`prompts/completed/06-project-settings-scope.md`](prompts/completed/06-project-settings-scope.md)
  - Added `ProjectSettingsScope` read summaries and capability-oriented discovery helpers.
- [x] **PR: write endpoint completion pass (Milestone 6 follow-up)** → [`prompts/completed/07-write-endpoints-completion.md`](prompts/completed/07-write-endpoints-completion.md)
  - Completed deferred write endpoints (including bulk and async-style write paths).
- [x] **PR: dashboards/widgets/filters scopes (Milestone 8)** → [`prompts/completed/08-dashboards-widgets-filters.md`](prompts/completed/08-dashboards-widgets-filters.md)
  - Implemented client CRUD coverage for dashboard, widget, and saved filter surfaces.
- [x] **PR: integrations + BTS scopes (Milestone 9)** → [`prompts/completed/09-integrations-bts.md`](prompts/completed/09-integrations-bts.md)
  - Added integration and bug-tracking-system client workflows with validation-first coverage.
- [x] **PR: activity/audit search scopes (Milestone 10)** → [`prompts/completed/10-activity-audit-scopes.md`](prompts/completed/10-activity-audit-scopes.md)
  - Added activity/audit event listing and practical operator filters.
- [x] **PR: system/admin/internal controllers (Milestone 12)** → [`prompts/completed/11-system-admin-controllers.md`](prompts/completed/11-system-admin-controllers.md)
  - Rounded out remaining system/admin/internal client-only controller coverage.

## Suggested next steps (client coverage first, then CLI)

With the first 11 planned prompt slices now complete, the next priority is to drive the API client to near-complete coverage for the agreed non-async, in-scope controllers before starting further CLI expansion.

- [ ] **PR: coverage closure for core execution controllers (`launch`/`test-item`/`log`)**
  - Close remaining gaps in high-value operations (history/search/compare/import/export/linking/attributes) with unit tests and focused integration coverage.
- [ ] **PR: coverage closure for integration/reporting controllers (`integration`/`bts`/`dashboard`/`widget`/`user-filter`)**
  - Finish remaining CRUD/command/list-variant gaps needed for practical automation completeness.
- [ ] **PR: coverage closure for project/activity surfaces (`project`/`project-settings`/`activity`/`activity-event`)**
  - Complete the agreed project and audit/event endpoints that are useful for operator and automation workflows.
- [ ] **PR: pre-CLI coverage audit + contract lock**
  - Publish a tracked endpoint matrix (`covered`, `deferred`, `excluded`) and lock JSON/error behavior for covered APIs.
- [ ] **PR: release readiness hardening**
  - Expand compatibility notes and integration assertions for automation-facing output stability.
- [ ] **PR: start CLI expansion (Milestone 13)**
  - Begin CLI write/reporting/introspection slices only after the pre-CLI client coverage gate is satisfied.

This sequence preserves the client-first strategy and explicitly gates CLI expansion on non-async client coverage completion.

## Pre-CLI coverage gate (agreed scope)

Target before major CLI expansion:

- **Near-100% for core controllers**: `launch-controller`, `test-item-controller`, `log-controller`
- **High coverage for supporting controllers**: `integration-controller`, `bug-tracking-system-controller`, `project-controller`, `project-settings-controller`, `dashboard-controller`, `widget-controller`, `user-filter-controller`, `activity-controller`, `activity-event-controller`, and practical `file-storage-controller` flows.
- **Explicitly excluded for now**: `launch-async-controller`, `test-item-async-controller`, `log-async-controller`.
- **Default deferred unless needed**: `deprecated-user-controller`, `onboarding-controller`, `demo-data-controller`, `plugin-controller`, `plugin-public-controller`, and low-value or high-risk admin-only edges.

---

## Product / engineering principles (guardrails)

- **No generated client leakage**: all new API surface is exposed via the `pkg/client` wrapper, not `pkg/internal/api`.
- **Scopes model the API**:
  - Prefer **project-scoped** features under `Client.Project(name)`.
  - Add **instance-scoped** scopes on `Client` for non-project endpoints (users, server settings, plugins, etc.).
- **Consistent response handling**: always check `response.JSONDefault` before success fields.
- **Options > param sprawl**: list/search endpoints use functional options; provide a few convenience helpers + allow raw param mutation.
- **TDD default**: add/extend unit tests first; add/extend integration tests for high-value workflows.
- **Client-first sequencing**: new API surface lands in `pkg/client` first; CLI additions are follow-up slices once the client API is proven and stable.
- **CLI is opinionated**: CLI focuses on workflows; the Go client is where “everything” lives.

---

## CLI command taxonomy (proposed)

Keep the existing **verb-first** shape (kubectl-like), and add action verbs only where they improve clarity.

- **Read operations**: `rp get <resource> [id]` (table/json/template output)
- **MVP priority operations**:
  - `rp get launch`
  - `rp get item`
  - `rp get log`
  - `rp download` / `rp file get` (attachments)
- **Post-MVP actions**:
  - `rp start launch` / `rp finish launch`
  - `rp start item` / `rp finish item`
  - `rp log add`
- **Admin (optional / selective)**: `rp admin <resource> ...`

This keeps the CLI predictable while allowing the Go client to remain the “everything is possible” surface.

---

## Roadmap milestones (client-first, shippable slices)

Milestones 1–5 delivered the read-only MVP (client + minimal CLI). Milestones 6–12 expanded the client surface; the current focus is **closing non-async in-scope client coverage gaps first**, then expanding CLI workflows on top of that completed client baseline.

### Milestone 1: Foundations (reduce future effort per endpoint) — completed

- **Client boilerplate reduction**
  - Add shared helpers for: logging, error wrapping, and “check JSONDefault then return”
  - Standardize “list” option patterns (paging/sorting) across scopes
- **CLI ergonomics**
  - Standard flags for pagination/sorting where relevant (`--page`, `--page-size`, `--sort`)
  - Consistent resource identification flags (`--id`, `--uuid`) where needed (avoid ambiguous positional args when actions grow)
- **Docs / DX**
  - Document the “new scope” checklist and patterns in `pkg/client/`
  - Ensure `make generate`, `make lint`, `make test`, `make test-integration` stay green

Deliverable slice examples:

- “Introduce shared response helper + refactor `LaunchScope`”
- “Add standard paging flags to `rp get launch`”

---

### Milestone 2: Launch read model (core MVP workflow, part 1) — completed

**Client**

- List launches with practical filters and paging
- Get launch by ID/UUID with consistent shape
- Launch “report/export” retrieval endpoints (PDF/HTML/XLS if present in spec)
- Any launch metadata retrieval endpoints needed for reliable drill-down UX

**CLI**

- Expand and harden `rp get launch`:
  - list/get/filter/paging
  - stable JSON output for automation
  - report retrieval/export shape where it maps to files/stdout

Notes:

- Keep command UX read-first and script-friendly; defer mutation flags/verbs to post-MVP milestones.

---

### Milestone 3: Test items read model (core MVP workflow, part 2) — completed

**Client (`TestItemScope`)**

- Get test item by ID/UUID
- List/search test items (with filters + tree/leaf traversal)
- Tree/navigation traversal primitives for suite/test hierarchy

**CLI**

- `rp get item` (list/get)
- Optional UX: `rp item tree` for hierarchical views

Notes:

- The CLI should focus on the “operator” experience (finding failing items quickly) rather than exposing every filter field.
- Canonical CLI noun is `item`. If `test-item` is provided, treat it as an alias for compatibility, not a second primary surface.

---

### Milestone 4: Logs + attachments (retrieval/downloads) — completed

**Client (`LogScope` + `ProjectScope.DownloadFile`)**

- List/get logs for item/launch (with practical filtering)
- Download files where API supports it
- Resolve attachment metadata needed to fetch related files

**CLI**

- `rp get log` (scoped to item/launch)
- `rp download` for attachments

Notes:

- Some file-storage endpoints may be too low-level for CLI; prefer high-signal retrieval flows for logs and attachments.

---

### Milestone 5: Workflow parity + operator UX (definition of practical completion) — completed

**Goal**

- Ensure one complete, scriptable happy path is excellent and stable:
  - launch get/list -> item get/list -> log get/list -> attachment download

**Client + CLI**

- Fill missing read/query options for core workflow commands (filters, selectors, paging, sorting)
- Consistent machine-readable errors and stable JSON output for automation
- Add integration tests for full flow success and representative failures
- Add README examples for end-to-end read-only usage (single run and scripted automation usage)

Notes:

- This milestone is the primary MVP completion gate. Remaining API parity work is secondary.

---

### Milestone 6: Client write workflows (post-MVP action parity) — completed (client)

**Client (primary)**

- Launch lifecycle actions: start/finish/update/delete
- Test item lifecycle actions: start/finish/update and related mutation flows
- Log mutation flows: add logs, upload/attach files, and bulk/async variants

**CLI (follow-up slices)**

- `rp start launch` / `rp finish launch` (and related update/delete workflows where UX is clear)
- `rp start item` / `rp finish item`
- `rp log add`

Notes:

- Client mutation foundations were delivered and completed through the write-workflow slices.
- CLI verbs remain a follow-up in Milestone 13.

---

### Milestone 7: Client discovery + identity depth (post-MVP usability) — completed

**Client (primary)**

- Instance-level:
  - `UsersScope`: `GetMyself` (whoami), (optional) admin CRUD
  - `ProjectsScope`: list/get projects (and later create/delete under admin)
- Project-level:
  - `ProjectSettingsScope`: read settings summary (useful for validation/debug)

**CLI (stable baseline, optional follow-up)**

- Already delivered: `rp whoami`, `rp get project` (list/get)
- Optional follow-up: `rp get project-settings` (read-only) if it materially helps troubleshooting

Notes:

- Delivered slices include `UsersScope`, `ProjectsScope`, `ProjectSettingsScope`, `rp whoami`, and `rp get project`.
- Admin-only operations (create/delete users, some project operations) should exist in the client, and may be gated behind `rp admin ...` if exposed.

---

### Milestone 8: Dashboards, widgets, and filters (reporting UX) — completed (client)

**Client (primary)**

- `DashboardScope`: list/get/create/update/delete
- `WidgetScope`: list/get/create/update/delete
- `UserFilterScope`: list/get/create/update/delete (saved filters)

**CLI (follow-up slices)**

- Likely **read-heavy** first:
  - `rp get dashboard`, `rp get widget`, `rp get filter`
- Create/update may be supported later, but only if the CLI can represent the payload ergonomically (YAML/JSON input files help).

Notes:

- Client coverage is in place; CLI remains best as read-first with file-driven workflows for complex payloads.

---

### Milestone 9: Integrations + Bug Tracking Systems (BTS) — completed (client)

**Client (primary)**

- `IntegrationScope`: list available integrations, configure for project, validate connection
- `BugTrackingSystemScope`: create/update BTS settings, link/unlink issues, query defects

**CLI (follow-up slices)**

- Start with operational introspection:
  - `rp get integration`, `rp get bts`
- Consider supporting “validate integration config” in CLI (good UX win)

Notes:

- Client support is in place; CLI exposure should remain pragmatic due to external-system and admin-heavy complexity.

---

### Milestone 10: Activity / audit events + search — completed (client)

**Client (primary)**

- `ActivityScope` / `ActivityEventScope`: list events, filter by time/object/user

**CLI (follow-up slices)**

- `rp activity list` with filters like `--since`, `--until`, `--user`, `--type`

Notes:

- Client support is in place and remains high value for debugging “what changed”; CLI support stays optional/pragmatic.

---

### Milestone 11: In-scope client coverage completion + compatibility hardening (pre-CLI gate)

**Client (priority)**

- Close remaining non-async gaps for agreed in-scope controllers, especially `launch`, `test-item`, and `log`
- Track endpoint-level coverage status (`covered`, `deferred`, `excluded`) in a maintained matrix
- Ensure representative integration coverage for high-value workflows and common failure mappings

**Compatibility hardening (before broad CLI expansion)**

- Add lightweight capability checks/version notes for known Report Portal differences where they affect command behavior
- Define and test stable error/response expectations for automation-facing client and CLI surfaces
- Ensure release hygiene:
  - changelog/release notes include command and output contract changes
  - shell completion/docs generation is part of release verification (if supported)

Notes:

- This milestone is the gate for broad CLI expansion and reduces breakage risk for automation users.

---

### Milestone 12: System / admin / internal controllers (client complete; CLI selective) — completed (client)

**Client (full coverage)**

- `SettingsScope` (server settings)
- `PluginScope` + `PluginPublicScope`
- `OnboardingScope`
- `DemoDataScope`
- Deprecated endpoints (for completeness, but clearly marked as such)

**CLI (selective / likely client-only)**

- Usually **client-only**:
  - onboarding, demo-data, deprecated endpoints
  - plugin marketplace style endpoints
  - low-level/system settings that aren’t safe/pleasant in CLI
- Optional `rp admin ...` commands only if you want to explicitly support admin workflows.

---

### Milestone 13: CLI expansion (after client completion; curated workflows)

**CLI**

- Write workflows on top of Milestone 6:
  - `rp start|finish launch`, `rp start|finish item`, `rp log add`
- Reporting UX (read-first):
  - `rp get dashboard|widget|filter` (prefer file-driven create/update only if ergonomic)
- Operator introspection:
  - `rp get integration|bts` and “validate” style commands where useful
  - `rp activity list` for audit/debug with time filters

Notes:

- Keep the CLI workflow-focused; avoid turning complex payloads into dozens of flags.
- Start this milestone only after Milestone 11 coverage targets are met.

---

## API coverage matrix (pre-CLI target)

The OpenAPI spec in `pkg/internal/openapi/openapi.yaml` is tagged by controller. Use this matrix to decide pre-CLI client coverage targets and what should be deferred.

| OpenAPI tag / area | Pre-CLI client target | CLI timing | Notes |
| --- | --- | --- | --- |
| `launch-controller` | Near-100% (priority) | After coverage gate | Core execution workflow controller |
| `test-item-controller` | Near-100% (priority) | After coverage gate | Core execution workflow controller |
| `log-controller` | Near-100% (priority) | After coverage gate | Core execution workflow controller |
| `file-storage-controller` | High (practical flows) | Selective | Focus on attachment/file retrieval tied to logs/items |
| `project-controller` | High (in-scope subset) | Selective | Prioritize automation-relevant project operations |
| `project-settings-controller` | High to near-100% | Selective | Strong troubleshooting/validation value |
| `user-controller` | Selective only | Selective | Keep mainly identity/discovery unless admin workflows are required |
| `user-filter-controller` | High to near-100% | After coverage gate | Important for saved-filter automation |
| `dashboard-controller` | High to near-100% | After coverage gate | Reporting workflows; read-first CLI |
| `widget-controller` | High to near-100% | After coverage gate | Reporting workflows; read-first CLI |
| `integration-controller` | High (priority) | After coverage gate | External-system automation and validation |
| `bug-tracking-system-controller` | High (priority) | After coverage gate | Defect and BTS integration workflows |
| `activity-controller` | High | Optional/after gate | Audit and operator debugging value |
| `activity-event-controller` | High | Optional/after gate | Audit and operator debugging value |
| `settings-controller` | Deferred by default | Likely no | System-level and frequently admin-only |
| `plugin-controller` | Deferred by default | Likely no | Usually client-only admin/internal tooling |
| `plugin-public-controller` | Deferred by default | Likely no | Public/plugin marketplace style surfaces |
| `launch-async-controller` | Excluded for now | No (explicitly) | Out of current pre-CLI target |
| `test-item-async-controller` | Excluded for now | No (explicitly) | Out of current pre-CLI target |
| `log-async-controller` | Excluded for now | No (explicitly) | Out of current pre-CLI target |
| `onboarding-controller` | Deferred by default | No | Low-value for primary automation workflows |
| `demo-data-controller` | Deferred by default | No | Low-value for production automation workflows |
| `deprecated-user-controller` | Deferred by default | No | Legacy/deprecated coverage only if required |

---

## How to slice work into small PRs (repeatable template)

For each API area (tag/controller), treat the work as a small sequence you can repeat:

Prefer endpoint-group or vertical-slice PRs when a full controller is too large (for example, launch list/get as one slice, export as another, lifecycle actions later). Default order is **client-first**: land `pkg/client` + tests, then add CLI commands (if any) as a follow-up slice.

- **Client (primary)**
  - Add a new scope type (instance-scoped or project-scoped)
  - Implement a thin wrapper per endpoint (logging + `JSONDefault` handling + typed returns)
  - Add black-box unit tests for success + common errors (401/403/404) + query param wiring
  - Add integration tests for 1–3 “happy path” workflows (only where it’s worth the runtime)
- **CLI (optional follow-up, only for clear workflows)**
  - Add `rp get ...` (read-only) first
  - Add action commands next (`start`, `finish`, `delete`, `update`) only when UX is good
  - Add integration tests per command family, covering JSON output + error cases
- **Docs**
  - Update README examples as soon as a workflow is “good enough for humans”

Definition of done (minimum):

- Unit tests cover at least: 200 + one representative error mapping
- Integration tests cover at least one real workflow (when the feature is user-facing)
- For user-facing command changes, include at least one integration assertion for JSON output shape/fields
- code-verifier subagent passes without any failures
