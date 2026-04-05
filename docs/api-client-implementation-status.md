# High-Level Summary

Against the `5.15.0` OpenAPI spec in `pkg/internal/openapi/openapi.yaml`, the public client in `pkg/client` remains strongest on project-scoped reporting APIs. Core day-to-day workflows are covered across launches, test items, logs, dashboards, widgets, filters, project settings, integrations, demo data, onboarding, and activity APIs.

The biggest remaining gaps are still instance-admin APIs (`User`, `File Storage`, parts of `Project`, `Settings`, and `Plugin`) and, new in the 5.15 spec, the `groups` and `projects` tags for instance groups, project-group membership, and project log types.

## Scope Of This Document

This status tracks the handwritten public client in `pkg/client`. The generated client in `pkg/internal/api` contains additional operations from the spec, but those endpoints do not count as implemented here until they are wrapped in the higher-level scope API.

## Architectural Note: Sync Vs. Async

Report Portal 5.15 splits reporting APIs into synchronous `v1` tags (`Launch`, `Test Item`, `Log`) and asynchronous `v2` tags (`Launch Async`, `Test Item Async`, `Log Async`). The public client currently wraps only the synchronous `v1` operations. None of the `/v2/{projectName}/...` async endpoints are exposed through `pkg/client`.

## Fully Implemented Tags

- `Activity` (`ActivityScope`): Get activity by ID and list activities for a test item.
- `Activity Event` (`ActivityEventsScope`): Search activities and get project subject names.
- `Demo Data` (`DemoDataScope`): Generate demo data for a project.
- `Onboarding` (`OnboardingScope`): Get onboarding status.
- `Project Settings` (`ProjectSettingsScope`): Get project settings plus full CRUD for notifications, pattern templates, and custom issue sub-types.
- `User Filter` (`UserFilterScope`): Create/get/list/update/delete, get names, get by IDs, and bulk update.

## Mostly Implemented Tags

- `Bug Tracking System` (`BugTrackingScope`):
  - Implemented: Project-scoped system fields, issue types, ticket lookup, issue creation, and ticket ID autocomplete.
  - Not implemented: Instance-scoped BTS field/type endpoints (`/v1/bts/{integrationId}/...` without `projectName`).
- `Dashboard` (`DashboardScope`):
  - Implemented: List/get/create/update/delete dashboards plus add/remove widgets.
  - Not implemented: Dashboard config retrieval and preconfigured dashboard creation.
- `Integration` (`IntegrationsScope`, `ProjectIntegrationsScope`):
  - Implemented: List/get/create/update/delete, connection testing, and project command execution for global and project integrations.
  - Not implemented: Bulk delete-by-type endpoints.
- `Launch` (`LaunchScope`):
  - Implemented: Start/finish/get/list/update/delete, bulk operations, compare, analyze, latest/debug listings, report retrieval, clusters, names, owners, statuses, and attribute autocomplete.
  - Not implemented: `Launch Async` wrappers. `LaunchScope.Import` currently returns an unsupported error because the 5.15 OpenAPI spec does not define a launch import operation.
- `Log` (`LogScope`):
  - Implemented: Create single/batch, get by ID/UUID, list, delete, search, and get logs under an item hierarchy.
  - Not implemented: Log page numbers, location navigation, location search, nested logs, and all `Log Async` wrappers.
- `Project` (`ProjectsScope`):
  - Implemented: Create/get/update/delete/list, list/search names, get project info, assign/unassign users, and list project users.
  - Not implemented: Export, analyzer status, notification config, preferences, indexing, assignable users, usernames helpers, and widget-by-code.
- `Settings` (`SettingsScope`):
  - Implemented: Get server settings and update analytics settings.
  - Not implemented: Update server settings and the duplicate analytics write variant in the spec.
- `Test Item` (`TestItemScope`):
  - Implemented: Start root/child items, finish, get by ID/UUID/IDs, list, update/delete, bulk issue/update/delete operations, attribute autocomplete, history, statistics, suggestions, suggest choice handling, and external issue link/unlink.
  - Not implemented: `/v1/{projectName}/item/search`, `/v1/{projectName}/item/v2`, and all `Test Item Async` wrappers.
- `Widget` (`WidgetScope`):
  - Implemented: Get/create/update widgets, preview widget data, and get widget names.
  - Not implemented: Multilevel widget retrieval.

## Partially Implemented Tags

- `File Storage` (`ProjectScope.DownloadFile`):
  - Implemented: File download by data ID.
  - Not implemented: User photo upload/get/delete and attachment cleanup endpoints.
- `Plugin` / `Plugin Public` (`PluginScope`):
  - Implemented: List private/public plugins, update plugin state, delete plugins, and fetch public plugin files.
  - Not implemented: Plugin upload plus import/command execution endpoints.
- `User` (`UsersScope`):
  - Implemented: `GetMyself`, `GetByLogin`, and `ListProjects`.
  - Not implemented: Administrative CRUD, registration/invitation, export, password workflows, and API key management.

## Missing 5.15 Tags

- `groups`: No `GroupsScope`; instance group CRUD, user membership, and group-project linking are not wrapped in `pkg/client`.
- `projects`: No wrappers for `/projects/{project_name}/groups...` or `/projects/{project_name}/log-types...`, so project-group links and project log type management are entirely absent from the public client.
- `Launch Async`, `Test Item Async`, `Log Async`: No async wrappers anywhere in `pkg/client`.
