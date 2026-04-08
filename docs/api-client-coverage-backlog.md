# API Client Coverage Backlog

This backlog has been fully implemented. The original slices below are now better read as a completion ledger for the handwritten public client in `pkg/client` against the `5.15.0` OpenAPI surface, still excluding all async endpoints under `/v2/...`.

## Verification Summary

- All numbered backlog items below now have corresponding wrapper code in `pkg/client`.
- The new top-level and project-scoped surfaces that used to be missing are present: `GroupsScope`, `ProjectGroupsScope`, `LogTypesScope`, `FileStorageScope`, `PluginScope`, `UsersScope`, and `InstanceBugTrackingScope`.
- The remaining synchronous spec gap is still `POST /v1/{projectName}/widget/preview`, which this backlog explicitly left out when item 4 was closed as already covered by `WidgetScope.GetData`.
- `LaunchScope.Import` is still unsupported, but that is not a backlog miss because `pkg/internal/openapi/openapi.yaml` does not define a launch import endpoint.

## Scope Notes

- Out of scope: `Launch Async`, `Test Item Async`, `Log Async`
- Completed backlog scope: the numbered synchronous wrapper gaps listed below
- Remaining non-backlog synchronous gap: widget preview

## Completed Backlog

| ID | Backlog slice | Current status | Implemented in |
| --- | --- | --- | --- |
| 1 | Settings completion | Done | `SettingsScope` in `pkg/client/settings.go` |
| 2 | Dashboard completion | Done | `DashboardScope` in `pkg/client/dashboard.go` |
| 3 | User filter bulk updates | Done | `UserFilterScope.BulkUpdate` in `pkg/client/filter.go` |
| 4 | Widget multilevel retrieval | Already done when logged | `WidgetScope.GetData` in `pkg/client/widget.go` |
| 5 | Integrations bulk delete-by-type | Done | `IntegrationsScope.DeleteAllByType` and `ProjectIntegrationsScope.DeleteAllByType` in `pkg/client/integrations.go` |
| 6 | Log sync navigation gaps | Done | `LogScope` navigation helpers in `pkg/client/log.go` |
| 7 | Test item sync gaps | Done | `TestItemScope.Search` and `TestItemScope.ListV2` in `pkg/client/item.go` |
| 8 | ProjectsScope completion | Done | `ProjectsScope` in `pkg/client/projects.go` |
| 9 | GroupsScope | Done | `GroupsScope` in `pkg/client/groups.go` and `Client.Groups()` in `pkg/client/client.go` |
| 10 | Project groups and log types | Done | `ProjectGroupsScope` and `LogTypesScope` in `pkg/client/project_groups.go`, `pkg/client/log_types.go`, and `pkg/client/project.go` |
| 11 | File storage completion | Done | `FileStorageScope` and project file helpers in `pkg/client/file.go` |
| 12 | Plugin completion | Done | `PluginScope` in `pkg/client/plugin.go` |
| 13 | Users admin and lifecycle | Done | `UsersScope` in `pkg/client/users.go` |
| 14 | Users passwords and API keys | Done | `UsersScope` password and API-key helpers in `pkg/client/users.go` |
| 15 | Instance-level BTS completion | Done | `InstanceBugTrackingScope` in `pkg/client/instance_bts.go` and `Client.InstanceBugTracking()` in `pkg/client/client.go` |

## Historical Note

This document is now primarily historical. For the current snapshot of spec coverage, see `docs/api-client-implementation-status.md`.
