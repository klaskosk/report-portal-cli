# CLI PR Plan

This document turns `CLI_IMPLEMENTATION_PLAN.md` into a concrete PR sequence optimized for delegated agent work. The goal is to keep the sequential spine as short as possible, then fan out by resource family, then fan back in before the next wave.

Guiding rules:

- One resource family or verb group per PR.
- Prefer early scaffolding PRs over repeated edits to shared entrypoints.
- Treat shared helper extraction and unresolved UX choices as explicit fan-in points.
- Keep stretch goals out of the main wave; add them after the core commands land.
- With noun-first, each resource is self-contained in its own file. The only shared merge point is 1-line `rootCmd.AddCommand(...)` additions in `root.go`, which keeps conflicts small and predictable.
- For resources with many verb subcommands (especially `launch`), split into sub-files (e.g., `launch_lifecycle.go`, `launch_analysis.go`) that register on the same parent command. This preserves parallelism for Phase 7 tracks that extend the same resource.

This plan assumes three pilot PRs: `filter`, `group`, and `widget`. If a different awkward pilot is chosen, keep the same phase boundaries and swap only that PR.

## Phase 1: Foundation

Low parallelism. Convert the existing CLI from verb-first to noun-first and establish the new conventions before resource fan-out.

- **PR 1: Refactor existing resources to noun-first with separate list/get.** Convert user, launch, item, log, and project from verb-first (`rp get launch`) to noun-first (`rp launch get`). Each resource file defines its own parent command and verb subcommands. Remove parent verb files (`get.go`, etc.). Separate the existing `get` (which handled both list and single) into distinct `list` and `get` subcommands. Add `--json` as a global shortcut for `-o json`.

## Phase 2: Pilot Slice Spine

No parallelism. These PRs should land in order so later work reuses proven patterns instead of inventing new ones.

- **PR 2: Pilot `filter` full CRUD.** Establish the baseline project-scoped CRUD shape: `rp filter {list,get,create,update,delete}`, including pagination, `--from-file` mutation flow, and table formatting.
- **PR 3: Pilot `group` full CRUD.** Establish the instance-scoped variant: `rp group {list,get,create,update,delete}`, without project resolution.
- **PR 4: Pilot `widget` commands.** Lock in the sub-resource pattern: `rp widget get <id>`, `rp widget data get <id>`, `rp widget {create,update}` with `--from-file`.

## Phase 3: Shared Helper Fan-In

Low parallelism. This phase should only extract patterns that were proven in at least two pilot PRs.

- **PR 5: Extract shared CLI helpers from the pilot slices.** Consolidate project resolution, ID parsing, `--from-file` handling, table formatting helpers, and reusable test scaffolding where the pilots clearly duplicated them.

## Phase 4: Standard Resource Fan-Out

High parallelism. Each PR should run in its own worktree and stay focused on one resource family. With noun-first, each resource is self-contained — the only shared merge point is `root.go`.

- **PR 6: `dashboard` CRUD.** `rp dashboard {list,get,create,update,delete}`.
- **PR 7: `log-type` commands.** `rp log-type {list,create,update,delete}`.
- **PR 8: `integration` CRUD.** `rp integration {list,get,create,update,delete}` — unified scope, `--project` determines global vs project-scoped. Same data types for both.
- **PR 9: `project` mutations.** Extend with `rp project {create,update,delete}` while leaving existing `list` and `get` intact.
- **PR 10: `launch` mutations.** Extend with `rp launch {update,delete}` while leaving existing `list` and `get` intact.
- **PR 11: `item` mutations.** Extend with `rp item {update,delete}` while leaving existing `list` and `get` intact.
- **PR 12: `log` mutations.** Extend with `rp log {create,delete}`, including the chosen async or batch shape if it is part of the core slice.
- **PR 13: `plugin` base commands.** `rp plugin {list,delete,update}` for the standard operations. `exec` and `create` (file upload) deferred to Phase 7.

## Phase 5: Standard Wave Fan-In

Low parallelism. This is the checkpoint after the first large parallel wave.

- **PR 14: Normalize helpers and UX after the standard resource wave.** Fold in any helper refinements, naming cleanups, help text fixes, or shared test updates that only became obvious after Phase 4 landed.

## Phase 6: Specialized UX Lock

Low parallelism. This phase settles the remaining command shapes that are still ambiguous before another fan-out.

- **PR 15: Freeze command shapes for the remaining special families.** Lock down the CLI shape for:
  - Dashboard config retrieval and preconfigured-dashboard creation flows
  - `rp activity-event list` filter ergonomics
  - `rp bts ticket get` multi-flag lookup ergonomics
  - `rp plugin exec` and `rp plugin create --file` final shapes
  - `rp launch report get <id>` output format
- After PR 15 lands, all remaining "Needs UX decision" markers in `CLI_IMPLEMENTATION_PLAN.md` should be resolved. Phase 7 agents should implement the settled shapes directly rather than making new UX decisions.

## Phase 7: Specialized Command Fan-Out

High parallelism. These PRs add non-CRUD verbs, sub-resources, and new grouping nouns to the CLI. PRs that extend the same resource file should use separate sub-files (e.g., `launch_lifecycle.go`) to avoid conflicts.

- **PR 16: `plugin` specialized commands.** Add `rp plugin exec` and `rp plugin create --file` once the final shapes are settled.
- **PR 17: Lifecycle commands.** Add `rp launch {start,finish}` and `rp item {start,finish}` in separate sub-files (`launch_lifecycle.go`, `item_lifecycle.go`).
- **PR 18: `auth` commands.** Add `rp auth apikey {list,create,delete}`, `rp auth {reset-password,change-password,invite}`.
- **PR 19: Group associations.** Add `rp group user {list,get,add,remove}` and `rp group project {list,get,add,remove}` as sub-resources of group.
- **PR 20: Dashboard widget association.** Add `rp dashboard widget {add,remove}` as a sub-resource of dashboard.
- **PR 21: Dashboard extras.** Add non-CRUD dashboard flows such as config retrieval and preconfigured dashboard creation after their final command shape is settled.
- **PR 22: Item issue linking.** Add `rp item issue {link,unlink}` as a sub-resource of item.
- **PR 23: Project user assignment.** Add `rp project user {assign,unassign}` as a sub-resource of project.
- **PR 24: Launch analysis commands.** Add `rp launch {analyze,compare}` in a sub-file (`launch_analysis.go`).
- **PR 25: Launch merge.** Add `rp launch merge` in a sub-file (`launch_merge.go`) once the async and `--from-file` behavior is finalized.
- **PR 26: Export and report commands.** Add `rp project export`, `rp user export`, and `rp launch report get <id>`, which all produce bytes output and need a consistent download UX.
- **PR 27: Settings — server.** Add `rp setting server {get,update}` for instance-level server settings (admin-only).
- **PR 28: Settings — notifications.** Add `rp setting notification {list,create,update,delete}` for project notification rules.
- **PR 29: Settings — patterns.** Add `rp setting pattern {list,create,update,delete}` for project pattern templates.
- **PR 30: Settings — issue sub-types.** Add `rp setting issue-subtype {list,create,update,delete}` for project issue sub-types.
- **PR 31: `activity` commands.** Add the read-only `rp activity {get,list}` for project-scoped item activity.
- **PR 32: `activity-event` search.** Add `rp activity-event list` with cross-project audit search filters.
- **PR 33: `bts` commands.** Add `rp bts ticket get`, `rp bts field list`, and `rp bts issue create` as sub-resources under the `bts` grouping noun.
- **PR 34: `demo-data` generation.** Add `rp demo-data generate`.

## Phase 8: Final Fan-In

Low parallelism. This phase is the final merge-and-polish checkpoint.

- **PR 35: Final CLI consistency sweep.** Clean up any remaining help text drift, duplicated examples, test gaps, or cross-command inconsistencies that surfaced during the specialized wave.

## Deferred / Blocked Follow-On

- **Deferred PR: `launch import`.** Keep `rp launch import` out of the main sequence until the generated client and OpenAPI surface actually support the import operation; it should be its own follow-on PR rather than bundled into the first-pass data movement work.

## Notes

- **Noun-first eliminates verb-file hotspots.** With verb-first, parent verb files (`get.go`, `create.go`, etc.) were merge hotspots because every resource PR added subcommands to them. With noun-first, each resource is self-contained and only touches `root.go` (1-line additions), which keeps merge conflicts minimal.
- **Resource file conflicts in Phase 7.** Multiple Phase 7 PRs extend the same resource file: `launch.go` is touched by lifecycle, analysis, merge, and report; `item.go` by lifecycle and issue linking; `dashboard.go` by widget association and extras; `project.go` by user assignment and export. Use sub-files (e.g., `launch_lifecycle.go`) to keep these tracks parallel.
- The highest conflict-risk PRs in Phase 4 are the ones that extend existing files (`project.go`, `launch.go`, `item.go`, `log.go`); each should stay tightly scoped to one family.
- If Phase 4 lands cleanly, Phase 5 can stay very small. The same applies to the final fan-in in Phase 8.
- Stretch goals should stay out of the main sequence. That includes bulk delete variants, filter bulk update, integration test or execute extras beyond the core slice, plugin public-command extras, and other non-essential follow-on features.
