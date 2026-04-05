# API Client Coverage Backlog

This backlog sketches PR-sized work to reach full public client coverage in `pkg/client` against the current OpenAPI surface, excluding all async endpoints under `/v2/...`.

## Scope Notes

- Out of scope: `Launch Async`, `Test Item Async`, `Log Async`
- In scope: synchronous `/v1/...` gaps, including `/v1/{projectName}/item/v2`
- Likely not a normal wrapper PR: `LaunchScope.Import`, because `pkg/client` currently reports it as unsupported due to the generated client not exposing a corresponding operation from the spec

## Quick Backlog

| ID | Proposed PR | Primary scope/files | Scope summary | Parallelism notes |
| --- | --- | --- | --- | --- |
| 1 | Settings completion | `pkg/client/settings.go`, `settings_test.go` | Add server settings update and resolve the duplicate analytics write variant in the spec. | Safe to run in parallel with most other PRs. |
| 2 | Dashboard completion | `pkg/client/dashboard.go`, `dashboard_test.go` | Add dashboard config retrieval and preconfigured dashboard creation. | Safe to run in parallel with most other PRs. |
| 3 | User filter bulk updates | `pkg/client/filter.go`, `filter_test.go` | Add bulk filter update support. | Safe to run in parallel with most other PRs. |
| 4 | ~~Widget multilevel retrieval~~ | `pkg/client/widget.go`, `widget_test.go` | Already covered by `GetData` (wraps the multilevel endpoint). No new code needed. Widget preview (`POST .../widget/preview`) remains unwrapped. | N/A |
| 5 | Integrations bulk delete-by-type | `pkg/client/integrations.go`, `integrations_test.go` | Add delete-all-by-type for global and project integrations. | Safe to run in parallel with most other PRs. |
| 6 | Log sync navigation gaps | `pkg/client/log.go`, `log_test.go` | Add page lookup, location navigation, location search, and nested logs. | Keep these together to avoid splitting one cohesive log-navigation slice. |
| 7 | Test item sync gaps | `pkg/client/item.go`, `item_test.go` | Add `/v1/{projectName}/item/search` and `/v1/{projectName}/item/v2`. | Safe to run in parallel with Wave 1 PRs. |
| 8 | ProjectsScope completion | `pkg/client/projects.go`, `projects_test.go` | Add export, analyzer status, notification config, preferences, indexing, assignable users, username helpers, and widget-by-code. | Large shared file; prefer one PR rather than multiple parallel PRs. |
| 9 | GroupsScope | `pkg/client/groups.go`, `groups_test.go`, `client.go` | Add a new top-level scope for instance group CRUD, group-user membership, and group-project linking. | Touches `client.go`; avoid overlapping with other new top-level scopes. |
| 10 | Project groups and log types | `pkg/client/project.go` plus new project-scoped files/tests | Add `/projects/{project_name}/groups...` and `/projects/{project_name}/log-types...` wrappers. | Bundle together because both want new `ProjectScope` accessors in `project.go`. |
| 11 | File storage completion | `pkg/client/file.go`, `file_test.go` | Add current-user photo upload/get/delete, project user-photo retrieval, and attachment cleanup by CSV upload. | Good standalone PR, but file upload handling may need extra test care. |
| 12 | Plugin completion | `pkg/client/plugin.go`, `plugin_test.go` | Add plugin upload, project plugin command execution, plugin import, and public plugin command execution. | Good standalone PR; multipart and arbitrary JSON command flows belong together. |
| 13 | Users admin and lifecycle | `pkg/client/users.go`, `users_test.go` | Add admin list/search/edit/delete/export plus invitation and registration flows. | Large shared file; do not parallelize with PR 14. |
| 14 | Users passwords and API keys | `pkg/client/users.go`, `users_test.go` | Add password reset/restore/change flows and API key management. | Sequence after PR 13, or merge both into one larger users PR. |
| 15 | Instance-level BTS completion | New instance-scoped BTS file(s), tests, likely `client.go` | Add `/v1/bts/{integrationId}/...` wrappers for instance-scoped field/type lookups. | Probably wants a new top-level scope; avoid overlapping with PR 9 on `client.go`. |

## Suggested Waves

### Wave 1

Run these in parallel:

- PR 1: Settings completion
- PR 2: Dashboard completion
- PR 3: User filter bulk updates
- PR 5: Integrations bulk delete-by-type
- PR 6: Log sync navigation gaps
- PR 7: Test item sync gaps

### Wave 2

Run these next, with some coordination around shared entry points:

- PR 8: ProjectsScope completion
- PR 9: GroupsScope
- PR 11: File storage completion
- PR 12: Plugin completion

### Wave 3

Finish with the larger or more design-sensitive slices:

- PR 10: Project groups and log types
- PR 13: Users admin and lifecycle
- PR 14: Users passwords and API keys
- PR 15: Instance-level BTS completion

## Merge-Conflict Hotspots

- `pkg/client/client.go`: new top-level scopes will collide here
- `pkg/client/project.go`: new project child scopes will collide here
- `pkg/client/projects.go`: best treated as one PR
- `pkg/client/users.go`: best treated as one serialized sequence

## Recommendation

If the goal is maximum parallel throughput with low rebase churn, assign agents to PRs 1 through 7 first. After that, queue PRs 8, 9, 11, and 12, then finish with 10, 13, 14, and 15.
