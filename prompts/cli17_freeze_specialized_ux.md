# PR 17: Freeze Command Shapes for Remaining Special Families

## Goal

Lock down the CLI shape for every specialized command family that still has a "Needs UX decision" marker in `CLI_IMPLEMENTATION_PLAN.md`. After this PR, every Phase 7 implementation PR can proceed without guessing at command names, verb assignments, flag semantics, or resource groupings.

This PR produces **no new runnable commands** — only documentation updates to `CLI_IMPLEMENTATION_PLAN.md` that settle all open design questions.

## Context

Phase 6 exists because the Phase 7 fan-out wave (PRs 18–35) cannot start until the command shapes are settled. Several families still have "Needs UX decision" or "Status: Needs UX decision" annotations. This PR resolves every one of them.

**Prerequisite:** PRs 1–16 must land first (all standard CRUD resources and the Phase 5 normalization).

## Decisions to settle

The following subsections specify the exact decision for each ambiguous area. Apply these decisions to `CLI_IMPLEMENTATION_PLAN.md`.

---

### 1. Plugin commands (PR 18)

**Status change:** "Ready to implement for get/delete. `exec` and `upload` need separate treatment." → **"Settled (PR 17)"**

**Final CLI shape:**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp get plugin` | List all installed plugins | Admin-only. No single-get by ID (API has no GetByID). |
| `rp get plugin --public` | List via public endpoint | No auth required on some deployments. |
| `rp create plugin --file <path>` | Upload a plugin archive | Binary file upload. The archive file is the command's primary input. |
| `rp update plugin <id> --enabled` / `--enabled=false` | Enable or disable a plugin | Maps to `UpdateState`. |
| `rp delete plugin <id>` | Delete a plugin | Standard. |
| `rp exec plugin <plugin-name> <command> [--from-file <body.json>]` | Execute a command against a plugin | Project-scoped (standard resolution). Body is arbitrary JSON (`map[string]any`). `--from-file` is optional; empty body sent if omitted. |
| `rp exec plugin <plugin-name> <command> --public [--from-file <body.json>]` | Execute via public endpoint | `--public` flag switches to `ExecutePublicCommand`; project resolution skipped. |

**Out of scope for PR 18 (stretch / follow-on):**
- `Import` — complex multipart upload with optional `LaunchImportRQ` metadata. Defer to a follow-on PR.
- `GetPublicFile` — raw file download. Defer; low priority.

**Design rationale:**
- `create plugin` uses `--file` because the archive file is the command's primary input mode. `exec plugin` keeps `--from-file` because the file-backed JSON body is an optional alternative to the default empty-body mode.
- `--public` flag on `get plugin` and `exec plugin` switches API paths without creating separate resource names. The public variants are the same conceptual operation with different auth requirements.
- `rp update plugin <id> --enabled` uses the existing `optionalBoolFlag` pattern. The API's `UpdatePluginStateRQ` has a single `Enabled *bool` field. If `--enabled` is not passed, the flag is nil and the request omits the field (which is likely a server error — the flag should effectively be required). Validate that `--enabled` was explicitly set; return a descriptive error if omitted.
- The `exec` verb (scaffolded in PR 2) is instance-scoped at the verb level, but `ExecuteCommand` requires a project name parameter. The `--project` / `-p` flag on `exec plugin` follows the standard project resolution chain (explicit flag → `RP_PROJECT` env → profile `default-project`). This is consistent with how all other project-needing commands work. When `--public` is passed, project resolution is skipped entirely — if a project flag is also provided alongside `--public`, return a descriptive error ("--public and --project are mutually exclusive").

---

### 2. Dashboard↔widget relationship commands (PR 22)

**Status change:** "Needs UX decision on verb (assign vs add)" → **"Settled (PR 17)"**

**Final CLI shape:**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp assign widget --dashboard <dashboard-id> --file <path>` | Add a widget to a dashboard | File contains `AddWidgetRq` JSON (with position, size, options). |
| `rp unassign widget <widget-id> --dashboard <dashboard-id>` | Remove a widget from a dashboard | Simple ID-to-ID removal. |

Both commands are project-scoped (`--project` / `-p`).

**Design rationale:**
- `assign` / `unassign` verbs are used (not `add` / `remove`) for consistency with all other relationship commands in the CLI (group↔project, group↔user, project↔user).
- Adding a widget requires `WidgetObjectModel` (widget ID, name, position, size, options) — this is a complex payload, and the file is the only request-construction mode here, so use `--file`. The file is the full `AddWidgetRq` object: `{"addWidget": {...}}`.
- Removing a widget is a simple two-ID operation: the dashboard ID from `--dashboard` and the widget ID as a positional arg.
- `--dashboard` uses the flag name (not `-d`) because `-d` is too ambiguous.

---

### 3. Dashboard config and preconfigured-dashboard (PR 23)

**Status change:** New section; previously implied by the "separate concerns" note under dashboard.

**Final CLI shape:**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp get dashboard config <dashboard-id>` | Get the dashboard configuration (widget layout definitions) | Subcommand of `get dashboard`. Returns `DashboardConfigResource`. |
| `rp create dashboard --preconfigured --from-file <path>` | Create a fully configured dashboard with widgets | File contains `DashboardPreconfiguredRq` JSON (name, description, config). |

Both commands are project-scoped (`--project` / `-p`).

**Design rationale:**
- `rp get dashboard config <id>` follows the widget pilot's subcommand pattern. Cobra routes "config" to the subcommand; numeric IDs route to the parent's `RunE` for single-get. The `config` subcommand returns a conceptually different view (widget layout definitions vs. dashboard metadata), satisfying decision rule 1 ("Different object → subcommand").
- `--preconfigured` is a mode flag on `create dashboard` rather than a separate resource name. This preserves the CLI's symmetric verb-resource taxonomy (every resource name that appears under one verb appears under others). A standalone `preconfigured-dashboard` resource would only exist under `create` — no `get`, `update`, or `delete` counterpart — which breaks the pattern. The `--preconfigured` flag switches the expected input mode:
  - Without `--preconfigured`: inline `--name` / `--description` flags (standard create)
  - With `--preconfigured`: requires `--from-file`, uses `DashboardPreconfiguredRq` schema (name, description, and full widget config are all inside the file)
- **Mutual exclusivity validation:** `--preconfigured` is mutually exclusive with `--name` and `--description`. If `--preconfigured` is set without `--from-file`, return a descriptive error. If `--name` or `--description` is set alongside `--preconfigured`, return a descriptive error explaining the two creation modes.

---

### 4. Activity-event search (PR 33)

**Status change:** "Needs UX decision on filter ergonomics" → **"Settled (PR 17)"**

**Final CLI shape:**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp get activity-event` | Search activity events with optional filters | Instance-scoped. Offset-based pagination. |

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--user` | string | Filter by subject name (who performed the activity) |
| `--object-type` | string | Filter by object type (e.g., "launch", "dashboard") |
| `--event` | string | Filter by event name (e.g., "createDashboard") |
| `--project-name` | string | Filter by project name |
| `--limit` | int32 (default 20) | Maximum results |
| `--offset` | int32 (default 0) | Pagination offset |
| `--sort` | string (default "createdAt") | Sort field |
| `--order` | string (default "DESC") | Sort direction (ASC or DESC) |

All filter flags are optional. If none are provided, the search returns recent events across all projects (subject to the pagination defaults).

Multiple filter flags combine with AND semantics (each becomes a `SearchCriteria` entry in the request body). Each filter uses the `EQ` operation.

**`GetProjectSubjectNames` — included as `rp get activity-event subjects`:**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp get activity-event subjects --project-name <name>` | Get available project subject names | Instance-scoped. Useful for discovering valid `--user` filter values. |

This is a simple lookup that helps users discover valid filter values for `rp get activity-event`. `--project-name` is a required parameter (not the project scope flag).

**Design rationale:**
- Instance-scoped: no `--project` / `-p` flag (the search spans all projects). `--project-name` is a search filter, not a scope selector.
- Offset/limit pagination (same model as groups) because the underlying API uses `GetActivitiesParams{Limit, Offset, Sort, Order}`.
- Flag names mirror the filter semantics: `--user` for `subjectName`, `--object-type` for `objectType`, `--event` for `eventName`, `--project-name` for `projectName`.
- Default limit is 20 (matches the client's default). Default sort is `createdAt` DESC (matches the client's hardcoded defaults).
- No single-get-by-ID mode. Activity events are query-only.
- `subjects` is nested under `activity-event` rather than exposed as a separate hyphenated sibling resource because it is a discovery helper for the search family, not a distinct top-level resource.

---

### 5. BTS commands (PR 34)

**Status change:** "Needs UX decision. The multi-flag lookup pattern is awkward. Consider whether this belongs in a separate `rp bts` verb group." → **"Settled (PR 17). BTS commands stay under standard verbs and are namespaced under `bts`."**

**Final CLI shape:**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp get bts ticket <ticket-id> --bts-url <url> --bts-project <bts-project>` | Look up a BTS ticket | Project-scoped. All three flags required. |
| `rp get bts fields --integration <id>` | Get available system fields for a BTS integration | Project-scoped. Returns `[]PostFormField`. |
| `rp get bts issue-types --integration <id>` | Get allowable issue types for a BTS integration | Project-scoped. Returns `[]string`. |
| `rp create bts issue --integration <id> --file <path>` | Create an issue in the BTS | Project-scoped. File contains `PostTicketRQ` JSON. |

All commands are project-scoped (`--project` / `-p`).

**Design rationale:**
- Commands stay under standard `get` / `create` verbs rather than a separate top-level `rp bts` verb group. The `bts` namespace sits one level lower as a grouping subcommand under those verbs.
- `ticket`, `fields`, `issue-types`, and `issue` are subcommands under `bts` rather than hyphenated standalone resource names. This keeps the domain visible without inventing another top-level verb family.
- `--bts-url` and `--bts-project` are required flags on `get bts ticket` (not positional args) because they are configuration-like values that identify the BTS instance, not the resource being fetched. The positional arg is the ticket ID (the thing being looked up).
- `PostTicketRQ` is complex (backLinks, fields array, include-flags, item ID, credentials), and the file is the only request input mode here, so `create bts issue` uses `--file`.
- `GetTicketIds` and `GetTicketIdsForProject` are autocomplete/search utilities. Defer to a follow-on PR.

---

### 6. Project settings grouping (PRs 28–31)

**Status change:** "Needs UX decision. Options include: `rp get project-setting notification` / `rp config get-notification --project <p>`" → **"Settled (PR 17)"**

The settings family is split into two categories:

#### 6a. Read-only settings views (PR 28)

These are diagnostic/inspection commands that dump configuration state. They live under the `config` verb.

| Command | Description | Notes |
|---------|-------------|-------|
| `rp config get-server` | Get server settings (key-value map) | Instance-scoped. Admin-only. Returns `map[string]string`. |
| `rp config get-project` | Get project settings (issue sub-type definitions) | Project-scoped (`--project` / `-p`). Returns `ProjectSettingsResource`. |

**Design rationale for `config`:** These are read-only dumps of configuration state, not resource CRUD. They don't have create/update/delete counterparts exposed via the same path. The `config` verb (scaffolded in PR 2) groups them logically. `get-server` and `get-project` are subcommands of `config`, using the Cobra subcommand pattern (`rp config get-server`, not `rp get config-server`).

**Mixed scoping within `config` is intentional.** Unlike other verbs in the CLI (where all subcommands share the same scoping model), `config` mixes instance-scoped commands (`get-server`, `set-server`, `set-analytics`) with a project-scoped command (`get-project`). This is acceptable because: (a) `config` is a diagnostic/administrative verb, not a resource-CRUD verb — its subcommands are grouped by purpose ("inspect or mutate system configuration") rather than by scope; (b) the `--project` flag on `get-project` makes the scope explicit at the call site; (c) `rp config --help` should group subcommands with a brief annotation indicating which require `--project`. The implementing agent should include scope annotations in the subcommand `Short` descriptions (e.g., `"Get project settings (requires --project)"`).

#### 6b. CRUD sub-resources (PRs 29–31)

Notification rules, pattern templates, and issue sub-types are standard CRUD resources that happen to live under the project settings API surface. In the CLI, they are promoted to top-level resources under standard verbs.

**Notification rules (PR 29):**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp get notification` | List notification rules for the project | No single-get-by-ID (API returns full list). |
| `rp create notification --file <path>` | Create a notification rule | File contains `SenderCaseDTO` JSON. |
| `rp update notification <id> --file <path>` | Update a notification rule | File contains `SenderCaseDTO` JSON. CLI injects or validates the payload ID — see note below. |
| `rp delete notification <id>` | Delete a notification rule | Standard. |

All are project-scoped (`--project` / `-p`).

**Why `update notification` takes a positional ID:** The client method `UpdateNotification(ctx, request SenderCaseDTO)` still takes only the request body, but the CLI should accept `rp update notification <id> --file <path>` so the target is explicit at the command line. When loading the `SenderCaseDTO`, the CLI should inject the positional ID into the payload if the file omits it. If the file already contains an `Id` and it disagrees with the positional arg, return a descriptive error. This keeps the UX aligned with the rest of the CLI without hiding API constraints.

**Pattern templates (PR 30):**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp create pattern-template --name <name> --type <type> --value <value> [--enabled=true|false]` | Create a pattern template | Simple fields → inline flags. `--enabled` defaults to true. |
| `rp update pattern-template <id> --name <name> --enabled=true|false` | Update a pattern template | Only name and enabled are updatable. `--enabled` is required on update because the API requires the field and there is no read endpoint to preserve it implicitly. |
| `rp delete pattern-template <id>` | Delete a pattern template | Standard. |

No `rp get pattern-template` command — the API has no List endpoint for pattern templates (see edge case note below).

All are project-scoped (`--project` / `-p`).

**Issue sub-types (PR 31):**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp get issue-sub-type` | List issue sub-types for the project | Calls `ProjectSettingsScope.Get()` and flattens the `SubTypes` map. |
| `rp create issue-sub-type --color <hex> --long-name <name> --short-name <name> --type-ref <ref>` | Create an issue sub-type | All fields required. Inline flags (4 simple strings). |
| `rp update issue-sub-type --locator <loc> --color <hex> --long-name <name> --short-name <name> --type-ref <ref>` | Update an issue sub-type | All fields required. Uses `--locator` (not positional ID) — see edge case below. |
| `rp delete issue-sub-type <id>` | Delete an issue sub-type by numeric ID | Standard positional ID. |

All are project-scoped (`--project` / `-p`).

**Design rationale:**
- Sub-resources are promoted to top-level resources because they have full CRUD semantics. This is more kubectl-like than nesting them (`rp get project-setting notification` would be verbose and inconsistent with how every other resource is accessed).
- `SenderCaseDTO` is complex (attributes, launch names, recipients, rule details, send case enum, attributes operator enum), and the file is the only request input mode here, so `--file` is required for notification create/update.
- `CreatePatternTemplateRQ` is simple (4 fields, all scalars) — inline flags work.
- `CreateIssueSubTypeRQ` is simple (4 string fields) — inline flags work.
- `UpdateOneIssueSubTypeRQ` adds `locator` (5 strings total) — still manageable with inline flags.

**Edge case: no `rp get pattern-template` command.** The `ProjectSettingsResource` returned by `ProjectSettingsScope.Get()` contains only `SubTypes` (issue sub-types), NOT pattern templates. There is no client method that lists pattern templates — `GetNotifications()` exists for notifications, but pattern templates have only Create/Update/Delete (no List/Get). Pattern templates can be created, updated, and deleted, but not listed. This follows the widget precedent where a missing API operation means the CLI command is simply not offered.

**Edge case: issue sub-type update uses locator, not numeric ID.** The `UpdateOneIssueSubTypeRQ` requires a `Locator` string field (not a numeric ID). The update command takes `--locator` as a required flag rather than a positional arg. This is different from other resources but matches the API's expectations. The `delete` still uses a numeric ID (the client's `DeleteIssueSubType` takes `int64`).

This creates a UX asymmetry: updates identify by locator string, deletes identify by numeric ID. Both identifiers are visible in the `rp get issue-sub-type` output (table shows both ID and LOCATOR columns). The implementing agent must:
1. Include both ID and LOCATOR in the table output so users can find either value.
2. Add a note in the `Long` help text for both update and delete explaining which identifier each expects and why they differ.
3. Include examples in help text that show the full workflow: list sub-types (shows both identifiers) → update by locator → delete by ID.

---

### 7. Top-level settings reads — server settings detail (PR 28)

The `SettingsScope` also has `UpdateAnalytics` and `UpdateServerSettings`. These are admin mutation commands. Their CLI shape:

| Command | Description | Notes |
|---------|-------------|-------|
| `rp config set-server --key <key> --value <value>` | Update a single server setting | Admin-only. Instance-scoped. Maps to `UpdateServerSettings`. |
| `rp config set-analytics --enabled [--type <type>]` | Update analytics configuration | Admin-only. Instance-scoped. Maps to `UpdateAnalytics`. |

These are included under `config` because they are key-value settings mutations, not resource CRUD.

**Design rationale:**
- `set-server` / `set-analytics` mirror `get-server` / `get-project` — the `config` verb groups all settings reads and writes.
- `UpdateSettingsRq` is trivial (key + value strings) — inline flags.
- `AnalyticsResource` is trivial (type string + enabled bool) — inline flags. `--type` defaults to the standard analytics type if the server only supports one; the flag allows future extensibility.

---

### 8. Export and report-download bytes-output UX (PR 27)

**Status change:** For `get launch-report`: "Returns bytes; needs UX decision" → **"Settled (PR 17)"**

All commands that return raw bytes (not JSON resources) follow the same output pattern established by the existing `download` command:

- Default: write raw bytes to stdout (pipeable to other commands or file redirection)
- `--output-file <path>`: write to a named file and print a confirmation to stderr

**Final CLI shape:**

| Command | Description | Notes |
|---------|-------------|-------|
| `rp get launch-report <launch-id>` | Download a launch report | Project-scoped. Returns raw bytes. |
| `rp export project [--view <view>] [--filter-name <name>] [--filter-type <type>]` | Export project data | Instance-scoped. Returns Jasper report bytes (CSV/PDF). |
| `rp export user [--role <role>] [--type <type>] [--active <bool>] [--email <email>]` | Export user list | Instance-scoped. Returns CSV bytes. |

All three commands accept `--output-file <path>` to write to a file instead of stdout.

**Design rationale:**
- `rp get launch-report` stays under `get` (not `export` or `download`) because it fetches a single computed resource by ID. It is project-scoped.
- `rp export project` and `rp export user` use the `export` verb (scaffolded in PR 2) because they produce bulk data exports, not single-resource reads.
- The `--output-file` pattern is reused from `download.go`. The implementing PRs should extract a shared `writeRawOutput(cmd *cobra.Command, data []byte)` helper (or reuse `writeDownloadedContent` if its interface is compatible). If extraction is needed, it belongs in PR 27 itself as a local helper.
- `export project` filter flags map to `ExportProjectsOption`: `--view`, `--filter-name`, `--filter-type`. The `--filter-last-run` option (which takes an `openapi_types.Date`) is deferred as a stretch goal because date parsing ergonomics need separate consideration.
- `export user` filter flags map to a subset of `ListUsersOption`: `--role`, `--type`, `--active`, `--email`. Pagination fields are not exposed (export produces the full dataset).

---

### 9. `get launch-report` scope detail

- **Scope:** Project-scoped (`--project` / `-p`)
- **Args:** `cobra.ExactArgs(1)` — the numeric launch ID
- **Flags:** `--project` / `-p`, `--output-file`
- **Output behavior:** Bypasses the normal `printer.Print` path entirely (raw bytes are not JSON-formattable or table-formattable). Use direct stdout write or file write.
- **`-o json` / `-o table` interaction:** If the user explicitly passes an `-o` flag, the command prints a warning to stderr: `"Note: --output flag is ignored for binary output commands"` and then proceeds normally with raw byte output. The command does NOT error. This applies to all byte-output commands (`get launch-report`, `export project`, `export user`). The `Long` help text for each of these commands must mention: "This command produces raw binary output. The -o/--output format flag is ignored."

---

## Implementation

### 1. Update `CLI_IMPLEMENTATION_PLAN.md`

Apply the following changes to the plan document:

#### Section: "Widget — resource vs. computed data split"
Already settled by PR 6. Verify the status says "Settled (PR 6)" and the form is `rp get widget data <id>`. If the plan still shows the hyphenated `rp get widget-data <id>` form anywhere (the top-level conventions bullet, decision rule 1 example, and the widget subsection itself), update all occurrences to the subcommand form `rp get widget data <id>`. PR 6's prompt instructed this update, but apply it here as a fallback if it was missed.

#### Section: "Activity — two unrelated concepts share a name"
Update the `activity-event` row:
- Status: "Needs UX decision on filter ergonomics" → "Settled (PR 17)"
- Proposed CLI shape: update to show the finalized flag set from decision 4 above, and add the helper subcommand `rp get activity-event subjects --project-name <name>`

#### Section: "Plugin — not CRUD-shaped"
- Status: "Ready to implement for get/delete. `exec` and `upload` need separate treatment." → "Settled (PR 17)"
- Update the proposed CLI shape table to match decision 1 above
- Add note that `Import` and `GetPublicFile` are deferred

#### Section: "BTS (Bug Tracking) — integration-dependent, not standalone CRUD"
- Status: "Needs UX decision" → "Settled (PR 17). Commands stay under standard verbs and are namespaced under `bts`."
- Update proposed CLI shape to match decision 5 above
- Remove the "Consider whether this belongs in a separate `rp bts` verb group" note

#### Section: "Project settings — sub-resource CRUD"
- Status: "Needs UX decision" → "Settled (PR 17)"
- Replace the two option examples with the final settled shape from decision 6 above
- Note the split: `config` for read-only settings views, standard verbs for sub-resource CRUD
- Note the `rp get pattern-template` limitation (no list endpoint)

#### Section: "Association / relationship commands"
Update the dashboard↔widget row:
- Status: "Needs UX decision on verb (assign vs add)" → "Settled (PR 17)"
- Update proposed CLI shape to match decision 2 above

#### Section: "Specialized / derived operations"
Update the following rows:
- `get server settings`: shape → `rp config get-server`, status → "Settled (PR 17)"
- `get project settings`: shape → `rp config get-project`, status → "Settled (PR 17)"
- `get launch report`: shape → `rp get launch-report <id>`, status → "Settled (PR 17)"
- `export project`: shape → `rp export project`, status → "Settled (PR 17)"
- `export users`: shape → `rp export user`, status → "Settled (PR 17)"

#### Add new section: "Bytes-output convention"
After the "Decision rules for ambiguous command design" section, add a new rule:

> 6. **Raw bytes → stdout or `--output-file`.** Commands that produce raw bytes (reports, exports, file downloads) write to stdout by default and accept `--output-file <path>` to write to a named file. These commands bypass `printer.Print`. If `-o` is explicitly set, print a warning to stderr and proceed with raw output.

#### Add new section: "Settings command grouping"
Within the project settings area, document the split:

> **Settings grouping rule:** Read-only configuration dumps use the `config` verb (`rp config get-server`, `rp config get-project`, `rp config set-server`, `rp config set-analytics`). CRUD operations on settings sub-resources (notifications, pattern templates, issue sub-types) use standard verbs (`rp get notification`, `rp create notification`, etc.) because they are full lifecycle resources despite living under the settings API surface.

### 2. Update `CLI_PR_PLAN.md`

#### Expand PR 28 description

The current PR 28 description reads: "Top-level settings read commands. Add the server-level and project-level settings read commands under whichever grouping Phase 6 selects."

Replace it with: "Settings commands (reads and mutations). Add `rp config get-server`, `rp config get-project`, `rp config set-server`, and `rp config set-analytics` under the `config` verb."

This ensures the implementing agent for PR 28 knows they are responsible for both the read and write commands.

#### Add Phase 6 completion note

Add a one-line note under Phase 6:

> After this PR lands, all "Needs UX decision" markers in `CLI_IMPLEMENTATION_PLAN.md` should be resolved. Phase 7 agents should reference the settled shapes in the plan rather than making independent design choices.

---

## Design notes

- **No new code in this PR.** The deliverable is exclusively documentation. This keeps the PR small, reviewable, and conflict-free.
- **All decisions are grounded in existing patterns.** Each decision references the decision rules already documented in the plan or patterns established by the pilot PRs (filter, group, widget).
- **"Deferred" is explicit.** For each family, anything not in the core slice is explicitly marked as deferred or a stretch goal. This prevents Phase 7 agents from scope-creeping.
- **The `config` verb now has a clear purpose.** PR 2 noted it was "provisional." This PR settles its scope: settings inspection and key-value settings mutations, mixing instance and project scopes by design.
- **File flag names follow input-mode context, not payload type alone.** Use `--from-file` when a file is one alternative among multiple request-construction modes (for example, `create dashboard --preconfigured --from-file` versus inline flags, or `exec plugin ... --from-file` versus an empty body). Use `--file` when the file itself is the primary or only input to the command (for example, `create plugin --file`, `assign widget --file`, `create notification --file`, `create bts issue --file`). Document this rule in the updated plan.
- **Mode flags over asymmetric resource names.** Where a creation variant produces the same resource type but from different input, a flag (`--preconfigured`) is preferred over a one-off resource name that would only appear under `create`. This preserves verb-resource symmetry across the CLI.

---

## What NOT to do

- **Do not implement any commands.** This PR only updates documentation.
- **Do not change existing command files.** No Go code should be modified.
- **Do not update `commands.go`.** No registration changes.
- **Do not create new Go files.**
- **Do not resolve stretch goals.** Only document what is deferred; do not design their UX.
- **Do not add or remove PRs from the Phase 7 structure in `CLI_PR_PLAN.md`.** The only PR plan change is expanding PR 28's description to include the `set-*` mutations alongside the reads.

---

## Validation

```bash
make build
make test
make lint
```

Since this PR only changes markdown files, all three targets should pass unchanged. Run them to confirm no accidental code changes were introduced.

Additionally, verify that a text search for "Needs UX decision" in `CLI_IMPLEMENTATION_PLAN.md` returns zero results after the update.

---

## Definition of done

- [ ] Every "Needs UX decision" marker in `CLI_IMPLEMENTATION_PLAN.md` is replaced with "Settled (PR 17)" and the finalized command shape
- [ ] Plugin section updated with full command table and stretch/deferred items
- [ ] Dashboard↔widget relationship settled (assign/unassign verbs, `--file` for add)
- [ ] Dashboard config/preconfigured flows settled (`get dashboard config <id>`, `create dashboard --preconfigured --from-file`)
- [ ] Activity-event filter ergonomics settled (flat flags, offset/limit pagination)
- [ ] BTS commands settled (stay under standard verbs, nested `bts` namespace)
- [ ] Project settings grouping settled (`config` for reads/writes, standard verbs for sub-resource CRUD)
- [ ] Pattern template "no list" limitation documented
- [ ] Notification update positional-ID injection/validation rationale documented
- [ ] Issue sub-type update-by-locator edge case documented with UX guidance (table shows both ID and LOCATOR, help text clarifies)
- [ ] Bytes-output convention added as decision rule 6
- [ ] Contextual `--file` vs `--from-file` naming rule documented
- [ ] Settings command grouping rule documented
- [ ] `CLI_PR_PLAN.md` PR 28 description expanded to include `set-server` and `set-analytics` mutations
- [ ] `CLI_PR_PLAN.md` has a note confirming all UX decisions are settled after this PR
- [ ] No Go code changed
- [ ] `make build`, `make test`, `make lint` pass unchanged
- [ ] This file moved to the `completed` directory
