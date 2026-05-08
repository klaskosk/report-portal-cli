# CLI Implementation Plan

This document outlines the strategy and execution map for implementing the remaining Report Portal CLI commands. The goal is a `kubectl`-like UX, developed through a disciplined sequence of pilot work followed by parallel fan-out.

## Current CLI direction

These are the settled, user-facing conventions. They are intentionally minimal—only rules that are expensive to change later.

- **Noun-first taxonomy.** Top-level commands are resource nouns: `rp launch`, `rp user`, `rp dashboard`. Verbs are subcommands of nouns: `rp launch list`, `rp launch get <id>`. Sub-resources nest under their parent noun: `rp widget data get <id>`.
- **Separate `list` and `get` verbs.** `list` returns multiple resources; `get` returns a single resource by ID. They are always separate subcommands.
- **Instance-scoped resources** (users, projects, groups, integrations, plugins) do **not** require `--project`. Integrations default to global scope; `--project` narrows to project-scoped integrations.
- **Project-scoped resources** (launches, items, logs, dashboards, widgets, filters, log-types, BTS, activity) require project resolution via `--project`, `RP_PROJECT`, or `default-project` in the config profile.
- **JSON output** is the raw API type, passed through unchanged. `--json` is a shortcut for `-o json`.
- **Table output** is human-readable with uppercase headers. Table rows may truncate long fields.
- **Sub-resources over mode flags.** When two operations return conceptually different objects or computed views, prefer a nested sub-resource over a resource-specific flag (e.g., `rp widget data get <id>` rather than `rp widget get <id> --data`).
- **`--from-file` for complex payloads.** When a mutation requires a large or structured body, prefer `--from-file <path>` over dozens of inline flags.

---

## Strategy: Resource-based parallel implementation

The core strategy is unchanged: implement all verbs for a single resource family at once, rather than implementing one verb across all resources.

**Why this works for parallelism:**

1. All logic for a resource lives in its own file (e.g., `internal/cli/dashboard.go`).
2. A subagent only needs the API scope for that resource.
3. Merge conflicts are limited to 1-line `cmd.AddCommand(...)` additions in `root.go`.

**Key constraint:** Parallelism should begin **after** a few representative slices have established the actual shared patterns. Premature fan-out risks parallel tracks inventing incompatible helpers and UX.

---

## Command family inventory

Each family is classified by its real shape, scope, and readiness. Families that are not standard CRUD are explicitly marked.

### Already implemented

**Note:** Existing resources currently use `get` for both list and single retrieval under the old verb-first design. These need refactoring to separate `list` and `get` subcommands and to move to noun-first command registration.

| Family | Scope | Verbs implemented | File |
|--------|-------|-------------------|------|
| user | instance | list, get, create, update, delete | `user.go` |
| launch | project | list, get | `launch.go` |
| item | project | list, get | `item.go` |
| log | project | list, get | `log.go` |
| project | instance | list, get | `project.go` |

Also implemented: `whoami`, `download`.

### Standard CRUD families — ready to implement

These map cleanly to list/get/create/update/delete on a single resource.

| Family | Scope | Client surface | Proposed CLI shape | Notes |
|--------|-------|---------------|-------------------|-------|
| dashboard | project | `DashboardScope.{List, GetByID, Create, Update, Delete}` | `rp dashboard {list,get,create,update,delete}` | `AddWidget`/`RemoveWidget`/`GetConfig`/`CreatePreconfigured` are separate concerns (see associations and specialized sections) |
| filter | project | `UserFilterScope.{List, GetByID, Create, Update, Delete}` | `rp filter {list,get,create,update,delete}` | `BulkUpdate` and `GetByIds` are stretch goals |
| group | instance | `GroupsScope.{List, Get, Create, Update, Delete}` | `rp group {list,get,create,update,delete}` | Group↔project and group↔user associations are sub-resource families (see associations section) |
| log-type | project | `LogTypesScope.{List, Create, Update, Delete}` | `rp log-type {list,create,update,delete}` | No single-get method; `list` only for reads |
| integration | instance / project | `IntegrationsScope.{ListGlobal, GetGlobal, CreateGlobal, UpdateGlobal, DeleteGlobal}`, `ProjectIntegrationsScope.{List, Get, Create, Update, Delete}` | `rp integration {list,get,create,update,delete}` | Scope determined by `--project`: omit for global (admin-only), include for project-scoped. Same data types for both. `--from-file` recommended for create/update. `DeleteAllByType`, `TestGlobalConnection`, and `ExecuteCommand` are stretch goals |
| project (remaining) | instance | `ProjectsScope.{Create, Update, Delete}` | `rp project {create,update,delete}` | `list` and `get` already exist. `update` takes project name; `delete` takes project ID |
| log (remaining) | project | `LogScope.{Create, CreateAsync, CreateBatch, Delete}` | `rp log {create,delete}` | `list` and `get` already exist. Batch create has special semantics |
| launch (remaining) | project | `LaunchScope.{Update, Delete, DeleteBulk}` | `rp launch {update,delete} <id>` | `list` and `get` already exist. Both `update` and `delete` take numeric launch ID. Bulk delete is a stretch goal |
| item (remaining) | project | `TestItemScope.{Update, Delete, DeleteBulk}` | `rp item {update,delete} <id>` | `list` and `get` already exist. Both `update` and `delete` take numeric item ID. Bulk operations are stretch goals |

### Families that are NOT normal CRUD

#### Widget — resource vs. computed data split

The client has two distinct read operations:

- `WidgetScope.GetByID` — returns the widget resource definition
- `WidgetScope.GetData` — returns computed/rendered widget data

These are conceptually different objects. The widget also has no `Delete` or `List` method in the client.

**Proposed CLI shape:**

- `rp widget get <id>` — the resource itself
- `rp widget data get <id>` — the computed data (nested sub-resource)
- `rp widget create` / `rp widget update` — standard mutations

**`--from-file` recommended** for create/update (widget definitions are complex JSON).

#### Activity — two unrelated concepts share a name

The client surface reflects two different things:

1. **`ActivityScope`** (project-scoped): read-only activity for a specific item. Methods: `GetByID`, `ListForItem`.
2. **`ActivityEventsScope`** (instance-scoped): cross-project audit event search. Methods: `Search`, `GetProjectSubjectNames`.

These should not be collapsed into one fuzzy "activity" command.

**Proposed split:**

| Family | Scope | Client surface | Proposed CLI shape | Status |
|--------|-------|---------------|-------------------|--------|
| activity | project | `ActivityScope.{GetByID, ListForItem}` | `rp activity get <id>`, `rp activity list --item <id>` | Ready to implement (read-only, no mutations) |
| activity-event | instance | `ActivityEventsScope.{Search, GetProjectSubjectNames}` | `rp activity-event list` with search filters | Needs UX decision on filter ergonomics |

#### Plugin — not CRUD-shaped

The plugin scope is command-oriented, not resource-oriented:

- `List` / `ListPublic` — listing is straightforward
- `Upload` — binary file upload, not a JSON create
- `UpdateState` — enable/disable toggle, not a general update
- `Delete` — straightforward
- `ExecuteCommand` / `ExecutePublicCommand` — RPC-style, takes a command name + arbitrary body
- `Import` / `GetPublicFile` — file operations

**Proposed CLI shape:**

- `rp plugin list` — list plugins
- `rp plugin delete <id>` — delete
- `rp plugin update <id>` — enable/disable state change
- `rp plugin exec <name> -- <command>` — execute command (specialized verb)
- `rp plugin create --file <path>` — upload plugin JAR

**Status:** Ready to implement for list/delete. `exec` and `upload` need separate treatment.

#### BTS (Bug Tracking) — integration-dependent, not standalone CRUD

The bug tracking scope depends on an active integration and does not follow standard resource patterns:

- `GetTicket` requires `ticketID`, `btsURL`, and `btsProject` (not a simple ID lookup)
- `CreateIssue` requires an `integrationID` plus a complex form payload
- `GetSystemFields` / `GetAllowableIssueTypes` are metadata queries on the integration

**Proposed CLI shape:**

- `rp bts ticket get <ticket-id> --bts-url <url> --bts-project <project>` — ticket lookup
- `rp bts issue create --integration <id> --from-file <path>` — create issue
- `rp bts field list --integration <id>` — query available fields

**Status:** Needs UX decision. The multi-flag lookup pattern for ticket retrieval is awkward.

#### Settings — grouped sub-resource CRUD

`SettingsScope` (instance) and `ProjectSettingsScope` (project) manage different settings. Project settings includes three sub-resource types: notifications, pattern templates, and issue sub-types. Each has its own CRUD.

**Proposed CLI shape:**

- `rp setting server get` / `rp setting server update` — instance-level server settings (admin-only)
- `rp setting notification {list,create,update,delete}` — project notification rules
- `rp setting pattern {list,create,update,delete}` — project pattern templates
- `rp setting issue-subtype {list,create,update,delete}` — project issue sub-types

Settings scope is determined by sub-resource: `server` is instance-scoped, others are project-scoped (require `--project`).

### Association / relationship commands

These mutate relationships between two resources. They are not base-resource CRUD. Associations use sub-resource nesting: `rp <resource> <sub-resource> <verb>`.

| Association | Scope | Client surface | Proposed CLI shape | Status |
|------------|-------|---------------|-------------------|--------|
| group ↔ project | instance | `GroupsScope.{AddProject, RemoveProject, ListProjects, GetProject}` | `rp group project {list,get,add,remove}` | Ready. Mirror also exists via `ProjectGroupsScope` (project side) |
| group ↔ user | instance | `GroupsScope.{AddUser, RemoveUser, ListUsers, GetUser}` | `rp group user {list,get,add,remove}` | Ready |
| project ↔ user | instance | `ProjectsScope.{AssignUsers, UnassignUsers}` | `rp project user {assign,unassign}` | Ready |
| dashboard ↔ widget | project | `DashboardScope.{AddWidget, RemoveWidget}` | `rp dashboard widget {add,remove}` | Ready |
| item ↔ external issue | project | `TestItemScope.{LinkExternalIssues, UnlinkExternalIssues}` | `rp item issue {link,unlink}` | Ready |

### Lifecycle / workflow commands

These represent execution lifecycle operations, not resource CRUD.

| Operation | Scope | Client surface | Proposed CLI shape | Notes |
|-----------|-------|---------------|-------------------|-------|
| start launch | project | `LaunchScope.{Start, StartAsync}` | `rp launch start` | `--from-file` recommended; `--async` flag for async variant |
| finish launch | project | `LaunchScope.{Finish, FinishAsync, ForceFinish}` | `rp launch finish <uuid>` | `--force` flag for force-finish |
| start item | project | `TestItemScope.{StartRoot, StartChild, StartRootAsync, StartChildAsync}` | `rp item start` | `--parent <uuid>` for child items |
| finish item | project | `TestItemScope.{Finish, FinishAsync}` | `rp item finish <uuid>` | |
| bulk force-finish | project | `LaunchScope.BulkForceFinish` | `rp launch finish --bulk --force` | Stretch goal |

### Specialized / derived operations

These are domain-specific operations that don't fit standard patterns.

| Operation | Scope | Client surface | Proposed CLI shape | Status |
|-----------|-------|---------------|-------------------|--------|
| analyze launch | project | `LaunchScope.Analyze` | `rp launch analyze <id>` | Ready |
| compare launches | project | `LaunchScope.Compare` | `rp launch compare <id1> <id2>` | Ready |
| merge launches | project | `LaunchScope.{Merge, MergeAsync}` | `rp launch merge <ids...>` | `--from-file` recommended; `--async` flag |
| import launch | project | `LaunchScope.Import` | `rp launch import <file>` | Client method is currently a stub |
| export project | instance | `ProjectsScope.Export` | `rp project export` | Returns CSV bytes |
| export users | instance | `UsersScope.Export` | `rp user export` | Returns CSV bytes |
| generate demo-data | project | `DemoDataScope.Generate` | `rp demo-data generate` | |
| get launch report | project | `LaunchScope.GetReport` | `rp launch report get <id>` | Returns bytes; needs UX decision |

### Auth / credential operations

| Operation | Scope | Client surface | Proposed CLI shape | Status |
|-----------|-------|---------------|-------------------|--------|
| create API key | instance | `UsersScope.CreateAPIKey` | `rp auth apikey create` | Ready |
| delete API key | instance | `UsersScope.DeleteAPIKey` | `rp auth apikey delete` | Ready |
| list API keys | instance | `UsersScope.ListAPIKeys` | `rp auth apikey list` | Ready |
| reset password | instance | `UsersScope.ResetPassword` | `rp auth reset-password` | Ready |
| invite user | instance | `UsersScope.CreateInvitation` | `rp auth invite` | Ready |
| change password | instance | `UsersScope.ChangePassword` | `rp auth change-password` | Ready |

---

## Decision rules for ambiguous command design

When the shape of a command is unclear, apply these rules:

1. **Different object → sub-resource.** If an operation returns a conceptually different object or computed view from the base resource, prefer a nested sub-resource over a flag. Example: `rp widget data get <id>` rather than `rp widget get <id> --data`.

2. **Relationship mutation → sub-resource commands.** If the operation mutates a relationship between two resources, nest it as a sub-resource of the owning resource with `add`/`remove` or `link`/`unlink` verbs, not base-resource CRUD.

3. **Complex payload → `--from-file`.** If a workflow requires structured or deeply nested payloads, prefer `--from-file <path>` over many inline flags.

4. **Dual scope → separate families unless types match.** If an area spans both instance and project scopes with different semantics, keep them as separate families. If the types and operations are identical (as with integrations), unify under one command and let `--project` determine scope.

5. **Admin operations → explicit flag or separate group.** Operations that require admin privileges should be clearly marked in help text. Consider grouping under `rp admin` if there are enough.

---

## Implementation phases

### Phase 0: Freeze minimal guardrails

Define and document (in this plan and in code comments / `CLAUDE.md` if needed) only the hard-to-undo rules:

- Noun-first taxonomy (already settled)
- Separate list and get verbs (already settled)
- Instance vs. project scoping rules (already settled)
- Output contract: JSON = raw API type, table = human-readable uppercase headers, `--json` shortcut
- Input policy: flags for simple fields, `--from-file` for complex payloads
- Sub-resources preferred over mode flags for distinct concepts

**This phase is complete.** The conventions above capture the current state.

### Phase 1: Scaffold resource commands

Create the empty parent commands for each resource noun, so subagents have a place to register verb subcommands.

Currently existing:
- `user` — exists with list, get, create, update, delete
- `launch` — exists with list, get
- `item` — exists with list, get
- `log` — exists with list, get
- `project` — exists with list, get

May need scaffolding:
- Standard CRUD resources: `dashboard`, `filter`, `group`, `log-type`, `integration`, `widget`, `plugin`
- Grouping nouns: `bts`, `setting`, `auth`, `demo-data`, `activity`, `activity-event`

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
   - Widget: establishes the sub-resource pattern for computed data
   - Integration: unified scope (global + project via `--project`), `--from-file` pattern
   - Log: create with async/batch variants

**Each pilot should be built sequentially** so discoveries from one inform the next.

### Phase 3: Extract shared helpers

After the pilot slices, extract patterns that appeared in at least two families. Likely candidates based on the current codebase:

- **Project resolution helper** — already exists (`resolveProject`), but may need refinement for new command patterns
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
| 5 | integration | `integration.go` | None beyond Phase 3 helpers |
| 6 | project (remaining) | `project.go` (extend) | None beyond Phase 3 helpers |
| 7 | launch (remaining) | `launch.go` (extend) | None beyond Phase 3 helpers |
| 8 | item (remaining) | `item.go` (extend) | None beyond Phase 3 helpers |
| 9 | log (remaining) | `log.go` (extend) | None beyond Phase 3 helpers |
| 10 | widget | `widget.go` | Pilot decision on sub-resource pattern |
| 11 | plugin | `plugin.go` | None beyond Phase 3 helpers |

Tracks that extend existing files should be carefully scoped to avoid conflicts with existing code.

### Phase 5: Implement domain-specific / specialized families

These add non-CRUD verbs, sub-resources, and new grouping nouns. They can be parallelized once Phase 4 conventions are stable.

**Separate tracks:**

| Track | Operations | File(s) | Notes |
|-------|-----------|---------|-------|
| A | Lifecycle: `rp launch {start,finish}`, `rp item {start,finish}` | `launch.go`, `item.go` (extend) | Additional verbs on existing resources |
| B | Auth / credentials | `auth.go` | New grouping noun with `apikey` sub-resource |
| C | Group associations: `rp group {user,project} {list,get,add,remove}` | `group.go` (extend) | Sub-resources of group |
| D | Dashboard associations: `rp dashboard widget {add,remove}` | `dashboard.go` (extend) | Sub-resource of dashboard |
| E | Item + project associations: `rp item issue {link,unlink}`, `rp project user {assign,unassign}` | `item.go`, `project.go` (extend) | Sub-resources of existing nouns |
| F | Launch extras: analyze, compare, merge, import, report | `launch.go` (extend) | Additional verbs and sub-resource |
| G | Export: `rp project export`, `rp user export` | `project.go`, `user.go` (extend) | Additional verb on existing resources |
| H | Settings | `setting.go` | New grouping noun with server/notification/pattern/issue-subtype sub-resources |
| I | Activity (project-scoped) | `activity.go` | New resource |
| J | Activity events (instance-scoped) | `activity_event.go` | New resource |
| K | BTS | `bts.go` | New grouping noun with ticket/issue/field sub-resources |
| L | Demo data | `demo_data.go` | New resource |

---

## Summary

The plan follows a **pilot → extract → parallelize** model:

1. **Freeze** the minimal user-facing conventions (done).
2. **Scaffold** resource commands as needed.
3. **Build** 2–3 deliberately different pilot slices sequentially to discover real patterns.
4. **Extract** shared helpers only after concrete duplication appears.
5. **Fan out** standard CRUD families in parallel worktrees.
6. **Implement** domain-specific families in a final parallel wave.

The inventory above is honest about which families are settled and which still need UX decisions. Implementers should consult the decision rules when they encounter ambiguity rather than guessing.
