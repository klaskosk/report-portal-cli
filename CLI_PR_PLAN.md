# CLI PR Plan

This document turns `CLI_IMPLEMENTATION_PLAN.md` into a concrete PR sequence optimized for delegated agent work. The goal is to keep the sequential spine as short as possible, then fan out by resource family, then fan back in before the next wave.

Guiding rules:

- One resource family or verb group per PR.
- Prefer early scaffolding PRs over repeated edits to shared entrypoints.
- Treat shared helper extraction and unresolved UX choices as explicit fan-in points.
- Keep stretch goals out of the main wave; add them after the core commands land.
- Optimize for mostly-isolated implementation work, not conflict-free merges; shared parent command files will still incur small predictable rebases unless they are scaffolded ahead of time.

This plan assumes three pilot PRs: `filter`, `group`, and `widget`. If a different awkward pilot is chosen, keep the same phase boundaries and swap only that PR.

## Phase 1: Parallelism Prep

Low parallelism. The purpose of this phase is to remove the biggest merge hotspots before multiple agents start landing resource PRs.

- **PR 1: Split parent CUD command builders out of `user.go`.** Move `create`, `update`, and `delete` parent command construction into their own small files so later resource PRs do not all collide in `user.go`.
- **PR 2: Scaffold missing top-level verb groups.** Add empty parents for new verbs such as `start`, `finish`, `assign`, `unassign`, `link`, `unlink`, `analyze`, `compare`, `merge`, `import`, `export`, `exec`, `auth`, `config`, and `generate`.
- **PR 3: Refactor parent command registration for low-risk parallelism.** Reorganize `get`, `create`, `update`, and `delete` registration code so later family PRs make smaller, more predictable registration edits without shipping empty user-visible resource leaves.

## Phase 2: Pilot Slice Spine

No parallelism. These PRs should land in order so later work reuses proven patterns instead of inventing new ones.

- **PR 4: Pilot `filter` full CRUD.** Establish the baseline project-scoped CRUD shape, pagination, single-get behavior, and `--from-file` mutation flow.
- **PR 5: Pilot `group` full CRUD.** Establish the instance-scoped variant of the same pattern without project resolution.
- **PR 6: Pilot `widget` commands.** Lock in the separate-resource treatment for widget definitions versus widget data before the broader wave starts.

## Phase 3: Shared Helper Fan-In

Low parallelism. This phase should only extract patterns that were proven in at least two pilot PRs.

- **PR 7: Extract shared CLI helpers from the pilot slices.** Consolidate project resolution, list-or-single dispatch, ID parsing, `--from-file` handling, table formatting helpers, and reusable test scaffolding where the pilots clearly duplicated them.

## Phase 4: Standard Resource Fan-Out

High parallelism. Each PR should run in its own worktree and stay focused on one resource family. Even after Phase 1, expect light rebasing where multiple PRs touch the same parent command file.

- **PR 8: `dashboard` CRUD.** Add the standard project-scoped CRUD commands for dashboards.
- **PR 9: `log-type` commands.** Implement list plus create, update, and delete for log types.
- **PR 10: Global `integration` CRUD.** Add the instance-scoped admin integration commands with file-based payload support.
- **PR 11: `project-integration` CRUD.** Add the project-scoped integration commands in a separate track from global integrations.
- **PR 12: `project` mutations.** Extend `project` with create, update, and delete while leaving the existing `get project` behavior intact.
- **PR 13: `launch` mutations.** Extend `launch` with update and delete for the existing launch resource.
- **PR 14: `item` mutations.** Extend `item` with update and delete for the existing test item resource.
- **PR 15: `log` mutations.** Extend `log` with create and delete, including the chosen async or batch shape if it is part of the core slice.

## Phase 5: Standard Wave Fan-In

Low parallelism. This is the checkpoint after the first large parallel wave.

- **PR 16: Normalize helpers and UX after the standard resource wave.** Fold in any helper refinements, naming cleanups, help text fixes, or shared test updates that only became obvious after Phase 4 landed.

## Phase 6: Specialized UX Lock

Low parallelism. This phase settles the remaining command shapes that are still ambiguous before another fan-out.

- **PR 17: Freeze command shapes for the remaining special families.** Lock down the CLI shape for `dashboard`↔`widget` relationship commands, dashboard config and preconfigured-dashboard flows, `activity-event`, `bts`, project settings grouping, `plugin`, top-level settings reads, and `get launch-report` so the next wave can implement without guessing.
- After PR 17 lands, all remaining "Needs UX decision" markers in `CLI_IMPLEMENTATION_PLAN.md` should be resolved. Phase 7 agents should implement the settled shapes directly rather than making new UX decisions.

## Phase 7: Specialized Command Fan-Out

High parallelism. These PRs are intentionally grouped by verb family or domain-shaped command set rather than CRUD resource shape.

- **PR 18: `plugin` commands.** Add the agreed plugin surface, keeping the upload or exec flows in scope only if the final UX still fits a reviewable slice.
- **PR 19: Lifecycle commands.** Add `start` and `finish` flows for launches and items.
- **PR 20: `auth` commands.** Add API key management, password flows, and user invitation commands.
- **PR 21: Instance-scoped assignment commands.** Add `assign` and `unassign` flows for group-project, group-user, and project-user relationships.
- **PR 22: Dashboard-widget relationship commands.** Add the project-scoped `dashboard`↔`widget` association commands after the verb choice is settled.
- **PR 23: Dashboard extras commands.** Add the non-CRUD dashboard flows such as config retrieval and preconfigured dashboard creation after their final command shape is settled.
- **PR 24: Issue link commands.** Add `link` and `unlink` flows for external issues on items.
- **PR 25: Launch analysis commands.** Add `analyze` and `compare` for launches.
- **PR 26: Launch merge commands.** Add `merge launch` once the async and `--from-file` behavior is finalized.
- **PR 27: Export and report-download commands.** Add `export project`, `export user`, and `get launch-report`, which all need a consistent bytes-output UX.
- **PR 28: Settings commands (reads and mutations).** Add `rp config get-server`, `rp config get-project`, `rp config set-server`, and `rp config set-analytics` under the `config` verb.
- **PR 29: Notification settings commands.** Add the notification-rule sub-resource commands under whichever grouping Phase 6 selects.
- **PR 30: Pattern-template settings commands.** Add the pattern-template sub-resource commands under whichever grouping Phase 6 selects.
- **PR 31: Issue-sub-type settings commands.** Add the issue sub-type sub-resource commands under whichever grouping Phase 6 selects.
- **PR 32: Project-scoped `activity` commands.** Add the read-only activity commands for item-specific project activity.
- **PR 33: Instance-scoped `activity-event` search.** Add the cross-project audit search commands after the filter ergonomics are settled.
- **PR 34: `bts` commands.** Add ticket lookup, field inspection, and issue creation for bug tracking integrations.
- **PR 35: `generate demo-data`.** Add the isolated demo-data generation command.

## Phase 8: Final Fan-In

Low parallelism. This phase is the final merge-and-polish checkpoint.

- **PR 36: Final CLI consistency sweep.** Clean up any remaining help text drift, duplicated examples, test gaps, or cross-command inconsistencies that surfaced during the specialized wave.

## Deferred / Blocked Follow-On

- **Deferred PR: `launch import`.** Keep `rp import launch` out of the main sequence until the generated client and OpenAPI surface actually support the import operation; it should be its own follow-on PR rather than bundled into the first-pass data movement work.

## Notes

- The highest conflict-risk PRs are the ones that extend existing files such as `project.go`, `launch.go`, `item.go`, and `log.go`; each should stay tightly scoped to one family.
- Shared parent files such as `get.go` and the CUD parent command builders are still expected rebase points, but Phase 1 should keep those conflicts small and predictable.
- If Phase 4 lands cleanly, Phase 5 can stay very small. The same applies to the final fan-in in Phase 8.
- Stretch goals should stay out of the main sequence. That includes bulk delete variants, filter bulk update, integration test or execute extras beyond the core slice, plugin public-command extras, and other non-essential follow-on features.
