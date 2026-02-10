# Report Portal CLI + Go Client Roadmap

**Target**: Report Portal **24.1** / API **v5.11** (OpenAPI in `pkg/internal/openapi/`)

**End state**:

- **Go client (`pkg/client`)**: ergonomic, scope-based wrapper covering all high-value and automation-critical API areas needed for practical usage.
- **CLI (`rp`)**: supports most day-to-day workflows end-to-end; some API areas remain client-only (admin/system/internal/UI-only).
- **Long-tail parity track**: remaining low-value/deprecated/admin endpoints can be added incrementally after practical completion.

This document is a high-level, shippable roadmap broken into small, independently completable slices (typically one endpoint group or one command family per PR).

---

## Current baseline (what exists today)

- **Client**: `Client → ProjectScope → LaunchScope`
  - Launches: `List`, `GetByID`, `GetByUUID`
  - Error handling: `*client.APIError` + predicates (`IsNotFound`, etc.)
- **CLI**:
  - `rp get launch [id-or-uuid]`
  - Config resolution: flags > env > config profiles
  - Output modes: `table`, `json`, `go-template=...`
- **Testing**:
  - Client: black-box unit tests (`pkg/client/*_test.go`) using `httptest`
  - CLI + client integration tests (Ginkgo) in `test/integration/`

---

## Suggested next steps (next 4–6 PRs)

If you want the fastest path to a customer-ready MVP while still building toward broad API coverage:

- [x] **PR: reduce wrapper boilerplate + baseline CLI consistency**
  - Add shared helpers for logging + `JSONDefault` checking, then refactor `LaunchScope` to use them.
  - Normalize common list/pagination/sorting behavior in both client options and CLI flags.
- [ ] **PR: expand launch read surface**
  - Client: list/get/search launch endpoints + report/export retrieval where relevant.
  - CLI: strengthen `rp get launch` for list/get/filter/report workflows and stable machine-readable output.
- [ ] **PR: expand test-item read surface**
  - Client: get/list/search test item endpoints (including hierarchy/navigation support where available).
  - CLI: `rp get item` for list/get workflows aligned with launch navigation patterns.
- [ ] **PR: add log + attachment read flows**
  - Client: list/get logs, resolve/download attachments/files where API supports it.
  - CLI: `rp get log` and attachment download command shape (`rp download` / `rp file get`).
- [ ] **PR: lock naming + stabilize end-to-end read flow**
  - Decide and document canonical resource names (`item` as canonical; optional aliases only for compatibility).
  - Add one full integration workflow: launch get/list -> item get/list -> log get/list -> attachment download.
- [ ] **PR: add identity + discovery follow-up**
  - Client: `UsersScope.GetMyself` + `ProjectsScope.List/Get`.
  - CLI: `rp whoami` + `rp get project`.

---

## Product / engineering principles (guardrails)

- **No generated client leakage**: all new API surface is exposed via the `pkg/client` wrapper, not `pkg/internal/api`.
- **Scopes model the API**:
  - Prefer **project-scoped** features under `Client.Project(name)`.
  - Add **instance-scoped** scopes on `Client` for non-project endpoints (users, server settings, plugins, etc.).
- **Consistent response handling**: always check `response.JSONDefault` before success fields.
- **Options > param sprawl**: list/search endpoints use functional options; provide a few convenience helpers + allow raw param mutation.
- **TDD default**: add/extend unit tests first; add/extend integration tests for high-value workflows.
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

## Roadmap milestones (incremental and shippable, MVP-first order)

### Milestone 1: Foundations (reduce future effort per endpoint)

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

### Milestone 2: Launch read model (core MVP workflow, part 1)

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

### Milestone 3: Test items read model (core MVP workflow, part 2)

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

### Milestone 4: Logs + attachments (retrieval/downloads)

**Client (`LogScope`, `FileStorageScope`)**

- List/get logs for item/launch (with practical filtering)
- Download files where API supports it
- Resolve attachment metadata needed to fetch related files

**CLI**

- `rp get log` (scoped to item/launch)
- `rp download` (or `rp file get`) for attachments

Notes:

- Some file-storage endpoints may be too low-level for CLI; prefer high-signal retrieval flows for logs and attachments.

---

### Milestone 5: Workflow parity + operator UX (definition of practical completion)

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

### Milestone 6: Write workflows (post-MVP action parity)

**Client + CLI**

- Launch lifecycle actions: start/finish/update/delete
- Test item lifecycle actions: start/finish/update and related mutation flows
- Log mutation flows: add logs, upload/attach files, and bulk/async variants

Notes:

- Implement mutation endpoints after the read-first MVP is stable and customer-ready.

---

### Milestone 7: Project discovery + identity (post-MVP usability)

**Client**

- Instance-level:
  - `UsersScope`: `GetMyself` (whoami), (optional) admin CRUD
  - `ProjectsScope`: list/get projects (and later create/delete under admin)
- Project-level:
  - `ProjectSettingsScope`: read settings summary (useful for validation/debug)

**CLI**

- `rp whoami` (or `rp get me`)
- `rp get project` (list/get)

Notes:

- Admin-only operations (create/delete users, some project operations) should exist in the client, and may be gated behind `rp admin ...` if exposed.

---

### Milestone 8: Dashboards, widgets, and filters (reporting UX)

**Client**

- `DashboardScope`: list/get/create/update/delete
- `WidgetScope`: list/get/create/update/delete
- `UserFilterScope`: list/get/create/update/delete (saved filters)

**CLI**

- Likely **read-heavy** first:
  - `rp get dashboard`, `rp get widget`, `rp get filter`
- Create/update may be supported later, but only if the CLI can represent the payload ergonomically (YAML/JSON input files help).

Notes:

- These endpoints are important for “reporting as code”, but can be painful as flags. Plan on `--from-file` workflows if implemented in CLI.

---

### Milestone 9: Integrations + Bug Tracking Systems (BTS)

**Client**

- `IntegrationScope`: list available integrations, configure for project, validate connection
- `BugTrackingSystemScope`: create/update BTS settings, link/unlink issues, query defects

**CLI**

- Start with operational introspection:
  - `rp get integration`, `rp get bts`
- Consider supporting “validate integration config” in CLI (good UX win)

Notes:

- Many endpoints are admin-like and/or depend on external systems. Client should fully support them; CLI support should be pragmatic.

---

### Milestone 10: Activity / audit events + search

**Client**

- `ActivityScope` / `ActivityEventScope`: list events, filter by time/object/user

**CLI**

- `rp activity list` with filters like `--since`, `--until`, `--user`, `--type`

Notes:

- This is high value for debugging “what changed” but not always a primary CLI use case; treat as optional CLI milestone if time is limited.

---

### Milestone 11: Release readiness + compatibility

**Client + CLI**

- Add lightweight capability checks/version notes for known Report Portal differences where they affect command behavior
- Define JSON output stability expectations for automation-facing commands and test them in integration coverage
- Ensure release hygiene:
  - changelog/release notes include command and output contract changes
  - shell completion/docs generation is part of release verification (if supported)

Notes:

- This milestone reduces breakage risk for automation users and clarifies support expectations.

---

### Milestone 12: System / admin / internal controllers (client complete; CLI selective)

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

## API coverage matrix (client vs CLI)

The OpenAPI spec in `pkg/internal/openapi/openapi.yaml` is tagged by controller. Use this matrix to decide which areas get CLI exposure.

| OpenAPI tag / area | Go client | CLI | Notes |
| --- | --- | --- | --- |
| `launch-controller` | Yes (full) | Yes (full) | MVP focuses on read/list/export first; lifecycle actions can follow |
| `test-item-controller` | Yes (full) | Yes (most) | MVP focuses on read/navigation/search first; mutations can follow |
| `log-controller` | Yes (full) | Yes (most) | MVP focuses on read/log retrieval first; log add can follow |
| `file-storage-controller` | Yes (full) | Partial | MVP favors download/read flows; avoid raw low-level file APIs unless useful |
| `project-controller` | Yes (full) | Yes (most) | Create/delete may be admin-only; still valuable for automation |
| `project-settings-controller` | Yes (full) | Partial | Read-only CLI is often enough |
| `user-controller` | Yes (full) | Partial | CLI: `whoami`, maybe `users list` under admin |
| `user-filter-controller` | Yes (full) | Partial | CLI best with `--from-file` for create/update |
| `dashboard-controller` | Yes (full) | Partial | Same as filters/widgets: read-heavy first |
| `widget-controller` | Yes (full) | Partial | Read-heavy first |
| `integration-controller` | Yes (full) | Partial | List/validate are good CLI candidates |
| `bug-tracking-system-controller` | Yes (full) | Partial | Useful but complex; keep CLI pragmatic |
| `activity-controller` | Yes (full) | Optional | Great for audits; not required for MVP CLI |
| `activity-event-controller` | Yes (full) | Optional | Same as above |
| `settings-controller` | Yes (full) | Likely no | System-level, often admin-only and risky in CLI |
| `plugin-controller` | Yes (full) | Likely no | Client-only by default; expose only if you want explicit admin tooling |
| `plugin-public-controller` | Yes (full) | Likely no | UI/marketplace-like; client-only |
| `launch-async-controller` | Yes (full) | No (explicitly) | CLI may use internally; no need for separate commands |
| `test-item-async-controller` | Yes (full) | No (explicitly) | Same |
| `log-async-controller` | Yes (full) | No (explicitly) | Same |
| `onboarding-controller` | Yes (full) | No | Client-only |
| `demo-data-controller` | Yes (full) | No | Client-only |
| `deprecated-user-controller` | Yes (for completeness) | No | Client-only (and clearly labeled deprecated) |

---

## How to slice work into small PRs (repeatable template)

For each API area (tag/controller), treat the work as a small sequence you can repeat:

Prefer endpoint-group or vertical-slice PRs when a full controller is too large (for example, launch list/get as one slice, export as another, lifecycle actions later).

- **Client**
  - Add a new scope type (instance-scoped or project-scoped)
  - Implement a thin wrapper per endpoint (logging + `JSONDefault` handling + typed returns)
  - Add black-box unit tests for success + common errors (401/403/404) + query param wiring
  - Add integration tests for 1–3 “happy path” workflows (only where it’s worth the runtime)
- **CLI (if applicable)**
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
