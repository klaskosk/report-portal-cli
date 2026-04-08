# High-Level Summary

Against the `5.15.0` OpenAPI spec in `pkg/internal/openapi/openapi.yaml`, the handwritten public client in `pkg/client` now covers effectively the entire synchronous API surface. The old documentation was stale: the wrapper now includes the previously missing instance-level admin surfaces (`User`, `File Storage`, `Plugin`, instance BTS), the new `groups` and `projects` tags, and the remaining `Project`, `Settings`, `Dashboard`, `Log`, and `Test Item` gaps from the backlog.

At this point, the only synchronous spec endpoint still not wrapped in `pkg/client` is `POST /v1/{projectName}/widget/preview`. The async `/v2/{projectName}/...` tags remain intentionally out of scope for the public wrapper API.

## Scope Of This Document

This status tracks the handwritten public client in `pkg/client`. The generated client in `pkg/internal/api` contains additional operations from the spec, but those endpoints do not count as implemented here until they are exposed through the higher-level scope API.

## Architectural Note: Sync Vs. Async

Report Portal 5.15 splits reporting APIs into synchronous `v1` tags (`Launch`, `Test Item`, `Log`) and asynchronous `v2` tags (`Launch Async`, `Test Item Async`, `Log Async`). The public client currently wraps only the synchronous `v1` operations. None of the `/v2/{projectName}/...` async endpoints are exposed through `pkg/client`.

## Fully Implemented Synchronous Tags

- `Activity` (`ActivityScope`): Get an activity by ID and list activities for a test item.
- `Activity Event` (`ActivityEventsScope`): Search activities and get project subject names.
- `Bug Tracking System` (`BugTrackingScope`, `InstanceBugTrackingScope`): Project-scoped ticket lookup/creation plus instance-scoped field and issue-type lookup.
- `Dashboard` (`DashboardScope`): List/get/create/update/delete dashboards, add/remove widgets, get dashboard config, and create preconfigured dashboards.
- `Demo Data` (`DemoDataScope`): Generate demo data for a project.
- `File Storage` (`FileStorageScope`, `ProjectScope.DownloadFile`, `ProjectScope.GetUserPhoto`): Download project files, get project user photos, and manage the current user's photo plus attachment cleanup.
- `Integration` (`IntegrationsScope`, `ProjectIntegrationsScope`): Full CRUD, list-by-plugin, connection testing, command execution, and delete-all-by-type for global and project integrations.
- `Launch` (`LaunchScope`): All synchronous spec-backed launch operations, including CRUD, compare/analyze, names, owners, statuses, report retrieval, clusters, and bulk operations.
- `Log` (`LogScope`): Create single/batch, get by ID/UUID, list, search, delete, page lookup, location navigation/search, nested logs, and logs-under traversal.
- `Onboarding` (`OnboardingScope`): Get onboarding status.
- `Plugin` / `Plugin Public` (`PluginScope`): List private/public plugins, update state, delete, upload, execute commands, import through plugins, execute public commands, and fetch public plugin files.
- `Project` (`ProjectsScope`): Full project CRUD plus export, analyzer status, notification config, preferences, indexing, assignable users, usernames helpers, and widget-by-code.
- `Project Settings` (`ProjectSettingsScope`): Get project settings plus full CRUD for notifications, pattern templates, and custom issue sub-types.
- `Settings` (`SettingsScope`): Get server settings, update server settings, and update analytics settings.
- `Test Item` (`TestItemScope`): All synchronous test-item operations, including search, `/item/v2`, history, statistics, suggestions, bulk operations, and external issue linking.
- `User` (`UsersScope`): Current user, admin list/search/export/CRUD, invitation and registration flows, password workflows, API key management, and user-project lookup.
- `User Filter` (`UserFilterScope`): Full CRUD, names, lookup by IDs, and bulk update.
- `groups` (`GroupsScope`): Instance group CRUD plus group-user and group-project membership management.
- `projects` (`ProjectGroupsScope`, `LogTypesScope`): Project-group links and project log-type management.

## Partially Implemented Synchronous Tags

- `Widget` (`WidgetScope`):
  - Implemented: Get/create/update widgets, get widget data via the multilevel endpoint, and get widget names.
  - Not implemented: `POST /v1/{projectName}/widget/preview`. This has been low priority because the preview response is not strongly typed in the spec or generated client, so it is a weak fit for the otherwise typed wrapper API.

## Spec Notes And Non-Gaps

- `LaunchScope.Import` exists in `pkg/client`, but it intentionally returns an unsupported error because `pkg/internal/openapi/openapi.yaml` does not define a launch import endpoint. It is not a spec coverage gap.
- `WidgetScope` does not currently wrap `POST /v1/{projectName}/widget/preview`, but that endpoint returns an untyped object (`map[string]any` in the generated client) and has not been important enough to justify adding a low-value, weakly typed helper so far.
- The spec contains a few duplicate or alias operations, such as trailing-slash variants and both `POST` and `PUT` forms of `/v1/settings/analytics`. Those do not require distinct high-level wrapper methods when they are functionally identical.

## Intentionally Unwrapped Tags

- `Launch Async`
- `Test Item Async`
- `Log Async`
