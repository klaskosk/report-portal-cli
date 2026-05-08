# CLI Implementation Plan

This document outlines the strategy and execution map for implementing the remaining Report Portal CLI commands. The goal is a `kubectl`-like UX, developed through a disciplined sequence of pilot work followed by parallel fan-out.

## Current CLI direction

These are the settled, user-facing conventions. They are intentionally minimal—only rules that are expensive to change later.

- **Verb-first taxonomy.** Top-level commands are verbs: `rp get`, `rp create`, `rp update`, `rp delete`. Resources are subcommands of verbs.
- **Instance-scoped resources** (users, projects, groups, global integrations, plugins, server settings) do **not** require `--project`.
- **Project-scoped resources** (launches, items, logs, dashboards, widgets, filters, log-types, project integrations, BTS, activity) require project resolution via `--project`, `RP_PROJECT`, or `default-project` in the config profile.
- **JSON output** is the raw API type, passed through unchanged.
- **Table output** is human-readable with uppercase headers. Table rows may truncate long fields.
- **Subcommands over mode flags.** When two operations return conceptually different objects or computed views, prefer a separate subcommand over a resource-specific flag (e.g., `rp get widget-data <id>` rather than `rp get widget <id> --data`).
- **`--from-file` for complex payloads.** When a mutation requires a large or structured body, prefer `--from-file <path>` over dozens of inline flags.

---

## Strategy: Resource-based parallel implementation

The core strategy is unchanged: implement all verbs for a single resource family at once, rather than implementing one verb across all resources.

**Why this works for parallelism:**

1. All logic for a resource lives in its own file (e.g., `internal/cli/dashboard.go`).
2. A subagent only needs the API scope for that resource.
3. Merge conflicts are limited to 1-line `cmd.AddCommand(...)` additions in the parent verb files.

**Key constraint:** Parallelism should begin **after** a few representative slices have established the actual shared patterns. Premature fan-out risks parallel tracks inventing incompatible helpers and UX.

---

## Command family inventory

Each family is classified by its real shape, scope, and readiness. Families that are not standard CRUD are explicitly marked.

### Already implemented

| Family | Scope | Verbs implemented | File |
|--------|-------|-------------------|------|
| user | instance | get, create, update, delete | `user.go` |
| launch | project | get (list + single) | `launch.go` |
| item | project | get (list + single) | `item.go` |
| log | project | get (list + single) | `log.go` |
| project | instance | get (list + single) | `project.go` |

Also implemented: `whoami`, `download`.

### Standard CRUD families — ready to implement

These map cleanly to list/get/create/update/delete on a single resource.

| Family | Scope | Client surface | Proposed CLI shape | Notes |
|--------|-------|---------------|-------------------|-------|
| dashboard | project | `DashboardScope.{List, GetByID, Create, Update, Delete}` | `rp {get,create,update,delete} dashboard` | `AddWidget`/`RemoveWidget`/`GetConfig`/`CreatePreconfigured` are separate concerns (see associations and specialized sections) |
| filter | project | `UserFilterScope.{List, GetByID, Create, Update, Delete}` | `rp {get,create,update,delete} filter` | `BulkUpdate` and `GetByIds` are stretch goals |
| group | instance | `GroupsScope.{List, Get, Create, Update, Delete}` | `rp {get,create,update,delete} group` | Group↔project and group↔user associations are separate families |
| log-type | project | `LogTypesScope.{List, Create, Update, Delete}` | `rp {get,create,update,delete} log-type` | No single-get method; `get` is list-only |
| integration (global) | instance | `IntegrationsScope.{ListGlobal, GetGlobal, CreateGlobal, UpdateGlobal, DeleteGlobal}` | `rp {get,create,update,delete} integration` | Admin-only. `--from-file` recommended for create/update payloads. `DeleteAllByType` and `TestGlobalConnection` are stretch goals |
| integration (project) | project | `ProjectIntegrationsScope.{List, Get, Create, Update, Delete}` | `rp {get,create,update,delete} project-integration` | Separate from global integrations. `ExecuteCommand` is a specialized operation. `--from-file` recommended |
| project (remaining) | instance | `ProjectsScope.{Create, Update, Delete}` | `rp {create,update,delete} project` | `get project` already exists. `update` takes project name; `delete` takes project ID |
| log (remaining) | project | `LogScope.{Create, CreateAsync, CreateBatch, Delete}` | `rp {create,delete} log` | `get log` already exists. Batch create has special semantics |
| launch (remaining) | project | `LaunchScope.{Update, Delete, DeleteBulk}` | `rp {update,delete} launch <id>` | `get launch` already exists. Both `update` and `delete` take numeric launch ID. Bulk delete is a stretch goal |
| item (remaining) | project | `TestItemScope.{Update, Delete, DeleteBulk}` | `rp {update,delete} item <id>` | `get item` already exists. Both `update` and `delete` take numeric item ID. Bulk operations are stretch goals |

### Families that are NOT normal CRUD

#### Widget — resource vs. computed data split

The client has two distinct read operations:

- `WidgetScope.GetByID` — returns the widget resource definition
- `WidgetScope.GetData` — returns computed/rendered widget data

These are conceptually different objects. The widget also has no `Delete` or `List` method in the client.

**Proposed CLI shape:**

- `rp get widget <id>` — the resource itself
- `rp get widget-data <id>` — the computed data
- `rp create widget` / `rp update widget` — standard mutations

**Status:** Needs UX decision on whether `widget-data` is a subcommand of `get widget` or a sibling resource.

**`--from-file` recommended** for create/update (widget definitions are complex JSON).

#### Activity — two unrelated concepts share a name

The client surface reflects two different things:

1. **`ActivityScope`** (project-scoped): read-only activity for a specific item. Methods: `GetByID`, `ListForItem`.
2. **`ActivityEventsScope`** (instance-scoped): cross-project audit event search. Methods: `Search`, `GetProjectSubjectNames`.

These should not be collapsed into one fuzzy "activity" command.

**Proposed split:**

| Family | Scope | Client surface | Proposed CLI shape | Status |
|--------|-------|---------------|-------------------|--------|
| activity | project | `ActivityScope.{GetByID, ListForItem}` | `rp get activity <id>`, `rp get activity --item <id>` | Ready to implement (read-only, no mutations) |
| activity-event | instance | `ActivityEventsScope.{Search, GetProjectSubjectNames}` | `rp get activity-event` with search filters | Needs UX decision on filter ergonomics |

#### Plugin — not CRUD-shaped

The plugin scope is command-oriented, not resource-oriented:

- `List` / `ListPublic` — listing is straightforward
- `Upload` — binary file upload, not a JSON create
- `UpdateState` — enable/disable toggle, not a general update
- `Delete` — straightforward
- `ExecuteCommand` / `ExecutePublicCommand` — RPC-style, takes a command name + arbitrary body
- `Import` / `GetPublicFile` — file operations

**Proposed CLI shape:**

- `rp get plugin` — list plugins
- `rp delete plugin <id>` — delete
- `rp update plugin <id>` — enable/disable state change
- `rp exec plugin <name> -- <command>` — execute command (specialized verb)
- `rp create plugin --file <path>` — upload plugin JAR

**Status:** Ready to implement for get/delete. `exec` and `upload` need separate treatment.

#### BTS (Bug Tracking) — integration-dependent, not standalone CRUD

The bug tracking scope depends on an active integration and does not follow standard resource patterns:

- `GetTicket` requires `ticketID`, `btsURL`, and `btsProject` (not a simple ID lookup)
- `CreateIssue` requires an `integrationID` plus a complex form payload
- `GetSystemFields` / `GetAllowableIssueTypes` are metadata queries on the integration

**Proposed CLI shape:**

- `rp get bts-ticket <ticket-id> --bts-url <url> --bts-project <project>` — ticket lookup
- `rp create bts-issue --integration <id> --from-file <path>` — create issue
- `rp get bts-fields --integration <id>` — query available fields

**Status:** Needs UX decision. The multi-flag lookup pattern is awkward. Consider whether this belongs in a separate `rp bts` verb group.

#### Project settings — sub-resource CRUD

`ProjectSettingsScope` manages three sub-resource types within a project's settings: notifications, pattern templates, and issue sub-types. Each has its own CRUD.

**Proposed CLI shape:** Needs UX decision. Options include:

- `rp get project-setting notification` / `rp create project-setting notification`
- `rp config get-notification --project <p>` / `rp config set-notification --project <p>`

**Status:** Needs UX decision on whether these live under `project-setting`, `config`, or another grouping.

### Association / relationship commands

These mutate relationships between two resources. They are not base-resource CRUD.

| Association | Scope | Client surface | Proposed CLI shape | Status |
|------------|-------|---------------|-------------------|--------|
| group ↔ project | instance | `GroupsScope.{AddProject, RemoveProject, ListProjects, GetProject}` | `rp assign project <id> --group <id>` / `rp unassign project <id> --group <id>` | Ready. Mirror also exists via `ProjectGroupsScope` (project side) |
| group ↔ user | instance | `GroupsScope.{AddUser, RemoveUser, ListUsers, GetUser}` | `rp assign user <id> --group <id>` / `rp unassign user <id> --group <id>` | Ready |
| project ↔ user | instance | `ProjectsScope.{AssignUsers, UnassignUsers}` | `rp assign user <login> --project <name>` / `rp unassign user <login> --project <name>` | Ready |
| dashboard ↔ widget | project | `DashboardScope.{AddWidget, RemoveWidget}` | `rp assign widget <id> --dashboard <id>` / `rp unassign widget <id> --dashboard <id>` | Needs UX decision on verb (assign vs add) |
| item ↔ external issue | project | `TestItemScope.{LinkExternalIssues, UnlinkExternalIssues}` | `rp link issue <item-id> <ticket-id>` / `rp unlink issue <item-id> <ticket-id>` | Ready |

### Lifecycle / workflow commands

These represent execution lifecycle operations, not resource CRUD.

| Operation | Scope | Client surface | Proposed CLI shape | Notes |
|-----------|-------|---------------|-------------------|-------|
| start launch | project | `LaunchScope.{Start, StartAsync}` | `rp start launch` | `--from-file` recommended; `--async` flag for async variant |
| finish launch | project | `LaunchScope.{Finish, FinishAsync, ForceFinish}` | `rp finish launch <uuid>` | `--force` flag for force-finish |
| start item | project | `TestItemScope.{StartRoot, StartChild, StartRootAsync, StartChildAsync}` | `rp start item` | `--parent <uuid>` for child items |
| finish item | project | `TestItemScope.{Finish, FinishAsync}` | `rp finish item <uuid>` | |
| bulk force-finish | project | `LaunchScope.BulkForceFinish` | `rp finish launch --bulk --force` | Stretch goal |

### Specialized / derived operations

These are domain-specific operations that don't fit standard patterns.

| Operation | Scope | Client surface | Proposed CLI shape | Status |
|-----------|-------|---------------|-------------------|--------|
| analyze launch | project | `LaunchScope.Analyze` | `rp analyze launch <id>` | Ready |
| compare launches | project | `LaunchScope.Compare` | `rp compare launch <id1> <id2>` | Ready |
| merge launches | project | `LaunchScope.{Merge, MergeAsync}` | `rp merge launch <ids...>` | `--from-file` recommended; `--async` flag |
| import launch | project | `LaunchScope.Import` | `rp import launch <file>` | Client method is currently a stub |
| export project | instance | `ProjectsScope.Export` | `rp export project` | Returns CSV bytes |
| export users | instance | `UsersScope.Export` | `rp export user` | Returns CSV bytes |
| generate demo-data | project | `DemoDataScope.Generate` | `rp generate demo-data` | |
| get server settings | instance | `SettingsScope.GetServerSettings` | `rp config get-server` | Admin-only |
| get project settings | project | `ProjectSettingsScope.Get` | `rp config get-project` | |
| get launch report | project | `LaunchScope.GetReport` | `rp get launch-report <id>` | Returns bytes; needs UX decision |

### Auth / credential operations

| Operation | Scope | Client surface | Proposed CLI shape | Status |
|-----------|-------|---------------|-------------------|--------|
| create API key | instance | `UsersScope.CreateAPIKey` | `rp auth create-apikey` | Ready |
| delete API key | instance | `UsersScope.DeleteAPIKey` | `rp auth delete-apikey` | Ready |
| list API keys | instance | `UsersScope.ListAPIKeys` | `rp auth list-apikeys` | Ready |
| reset password | instance | `UsersScope.ResetPassword` | `rp auth reset-password` | Ready |
| invite user | instance | `UsersScope.CreateInvitation` | `rp auth invite` | Ready |
| change password | instance | `UsersScope.ChangePassword` | `rp auth change-password` | Ready |

---

## Decision rules for ambiguous command design

When the shape of a command is unclear, apply these rules:

1. **Different object → subcommand.** If an operation returns a conceptually different object or computed view from the base resource, prefer a subcommand over a flag. Example: `rp get widget-data <id>` rather than `rp get widget <id> --data`.

2. **Relationship mutation → association command.** If the operation mutates a relationship between two resources, treat it as an association command (`assign`/`unassign`, `link`/`unlink`), not base-resource CRUD.

3. **Complex payload → `--from-file`.** If a workflow requires structured or deeply nested payloads, prefer `--from-file <path>` over many inline flags.

4. **Dual scope → separate families.** If an area spans both instance and project scopes with different semantics, do not force them into one command family unless the UX is clearly coherent.

5. **Admin operations → explicit flag or separate group.** Operations that require admin privileges should be clearly marked in help text. Consider grouping under `rp admin` if there are enough.

---

## Implementation phases

### Phase 0: Freeze minimal guardrails

Define and document (in this plan and in code comments / `CLAUDE.md` if needed) only the hard-to-undo rules:

- Verb-first taxonomy (already settled)
- Instance vs. project scoping rules (already settled)
- Output contract: JSON = raw API type, table = human-readable uppercase headers
- Input policy: flags for simple fields, `--from-file` for complex payloads
- Subcommands preferred over mode flags for distinct concepts

**This phase is complete.** The conventions above capture the current state.

### Phase 1: Scaffold parent verbs

Create the empty parent commands for any standard verbs that don't yet exist, so subagents have a place to register subcommands.

Currently existing:
- `get` — exists with subcommands
- `create` — exists (user only)
- `update` — exists (user only)
- `delete` — exists (user only)

May need scaffolding:
- `start` / `finish` — lifecycle verbs
- `assign` / `unassign` — association verbs
- `link` / `unlink` — issue linking verbs
- Other specialized verbs as needed (`analyze`, `compare`, `merge`, `import`, `export`, `exec`, `auth`, `config`, `generate`)

Each scaffold is a small file defining the parent `cobra.Command` and registering it in `root.go`. This can be done incrementally as needed.

### Phase 2: Build 2–3 pilot slices sequentially

Before broad fan-out, implement a small number of intentionally different families to learn what shared helpers are actually needed.

**Recommended pilot mix:**

1. **One straightforward read+write family** — `dashboard` or `filter`
   - Clean CRUD shape, project-scoped
   - Will establish patterns for: project resolution, list pagination, single-get-by-ID, create/update with `--from-file`, delete, table formatting

2. **One instance-scoped mutation family** — `group`
   - Instance-scoped, no `--project`
   - Will establish patterns for: instance-scoped commands, different table layouts

3. **One awkward/special family** — `widget`, `integration`, or `log`
   - Widget: establishes the subcommand-over-flag pattern for computed data
   - Integration: dual scope (global + project), `--from-file` pattern
   - Log: create with async/batch variants

**Each pilot should be built sequentially** so discoveries from one inform the next.

### Phase 3: Extract shared helpers

After the pilot slices, extract patterns that appeared in at least two families. Likely candidates based on the current codebase:

- **Project resolution helper** — already exists (`resolveProject`), but may need refinement for new command patterns
- **Single-or-list dispatch** — the `cobra.MaximumNArgs(1)` + branch-on-arg-count pattern used in `get launch`, `get item`, etc.
- **ID-or-UUID parsing** — the try-int-then-treat-as-UUID pattern
- **Mutation request helpers** — reading `--from-file` JSON, applying flag overrides
- **Table row formatting** — `formatStringPtr`, `formatInt64Ptr` already exist; may need more
- **Common test scaffolding** — shared setup for CLI command tests

Do not extract helpers speculatively. Only extract what the pilot slices clearly justify.

### Phase 4: Fan out standard families in parallel

Once conventions are proven and helpers exist, parallelize the remaining standard CRUD families. Each track is a single resource in its own worktree.

**Parallel tracks (one subagent each):**

| Track | Family | File | Dependencies |
|-------|--------|------|-------------|
| 1 | dashboard | `dashboard.go` | Phase 2 pilot (if not already the pilot) |
| 2 | filter | `filter.go` | Phase 2 pilot (if not already the pilot) |
| 3 | group | `group.go` | Phase 2 pilot (if not already the pilot) |
| 4 | log-type | `log_type.go` | None beyond Phase 3 helpers |
| 5 | integration (global) | `integration.go` | None beyond Phase 3 helpers |
| 6 | integration (project) | `project_integration.go` | None beyond Phase 3 helpers |
| 7 | project (remaining) | `project.go` (extend) | None beyond Phase 3 helpers |
| 8 | launch (remaining) | `launch.go` (extend) | None beyond Phase 3 helpers |
| 9 | item (remaining) | `item.go` (extend) | None beyond Phase 3 helpers |
| 10 | log (remaining) | `log.go` (extend) | None beyond Phase 3 helpers |
| 11 | widget | `widget.go` | Pilot decision on subcommand pattern |
| 12 | plugin | `plugin.go` | None beyond Phase 3 helpers |

Tracks that extend existing files should be carefully scoped to avoid conflicts with existing code.

### Phase 5: Implement domain-specific / specialized families

These are more isolated and can be parallelized once Phase 4 conventions are stable.

**Separate tracks:**

| Track | Family | File |
|-------|--------|------|
| A | Lifecycle (start/finish) | `lifecycle.go` |
| B | Auth / credentials | `auth.go` |
| C | Associations (assign/unassign) | `assign.go` |
| D | Issue linking | `link.go` |
| E | Analysis (analyze, compare) | `analysis.go` |
| F | Data management (merge, import, export) | `data.go` |
| G | Config / settings | `config_cmd.go` |
| H | Activity (project-scoped) | `activity.go` |
| I | Activity events (instance-scoped) | `activity_event.go` |
| J | BTS | `bts.go` |
| K | Demo data | `demo_data.go` |

---

## Summary

The plan follows a **pilot → extract → parallelize** model:

1. **Freeze** the minimal user-facing conventions (done).
2. **Scaffold** parent verbs as needed.
3. **Build** 2–3 deliberately different pilot slices sequentially to discover real patterns.
4. **Extract** shared helpers only after concrete duplication appears.
5. **Fan out** standard CRUD families in parallel worktrees.
6. **Implement** domain-specific families in a final parallel wave.

The inventory above is honest about which families are settled and which still need UX decisions. Implementers should consult the decision rules when they encounter ambiguity rather than guessing.
