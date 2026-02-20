# Pre-CLI Endpoint Coverage Matrix

This document tracks the mapping between generated OpenAPI endpoints and the
high-level client wrapper methods in `pkg/client`. It serves as the objective
gate artifact for Milestone 13 readiness.

Each endpoint is derived from the `ClientWithResponsesInterface` in
`pkg/internal/api/api.gen.go`. `WithBody` variants (alternate encoding of
the same endpoint) are omitted; only the canonical operation is listed.

## Status definitions

| Status | Meaning |
|---|---|
| `covered` | Wrapper method exists and is exercised by unit or integration tests. |
| `deferred` | Not wrapped. May be added in a future prompt if automation demand justifies it. |
| `excluded` | Intentionally omitted — destructive, UI-only, or outside the client's scope. |

## Summary

| Controller | Covered | Deferred | Excluded | Total |
|---|--:|--:|--:|--:|
| launch-controller | 19 | 7 | 0 | 26 |
| test-item-controller | 15 | 12 | 0 | 27 |
| log-controller | 8 | 3 | 0 | 11 |
| integration-controller | 15 | 0 | 2 | 17 |
| bug-tracking-system-controller | 4 | 2 | 0 | 6 |
| project-controller | 11 | 13 | 0 | 24 |
| project-settings-controller | 11 | 0 | 0 | 11 |
| dashboard-controller | 7 | 0 | 0 | 7 |
| widget-controller | 5 | 1 | 0 | 6 |
| user-filter-controller | 7 | 1 | 0 | 8 |
| activity-controller | 2 | 0 | 0 | 2 |
| activity-event-controller | 2 | 0 | 0 | 2 |
| file-storage-controller | 1 | 0 | 5 | 6 |
| **Total** | **107** | **39** | **7** | **153** |

Coverage rate: 107/153 (70%) overall, 107/146 (73%) excluding intentional exclusions.

---

## launch-controller

Wrapper scope: `LaunchScope` (project-scoped via `Client.Project(name).Launches()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `StartLaunchUsingPOST1` | `Start` | covered | |
| `FinishLaunchUsingPUT1` | `Finish` | covered | |
| `GetLaunchByUuidUsingGET` | `GetByUUID` | covered | |
| `GetLaunchUsingGET` | `GetByID` | covered | |
| `GetProjectLaunchesUsingGET` | `List` | covered | |
| `GetLatestLaunchesUsingGET` | `GetLatest` | covered | |
| `GetDebugLaunchesUsingGET` | `GetDebug` | covered | |
| `UpdateLaunchUsingPUT` | `Update` | covered | |
| `DeleteLaunchUsingDELETE` | `Delete` | covered | |
| `DeleteLaunchesUsingDELETE` | `DeleteBulk` | covered | |
| `ForceFinishLaunchUsingPUT` | `ForceFinish` | covered | |
| `MergeLaunchesUsingPOST1` | `Merge` | covered | |
| `StartLaunchAnalyzerUsingPOST` | `Analyze` | covered | |
| `BulkUpdateUsingPUT` | `BulkUpdate` | covered | |
| `BulkForceFinishUsingPUT` | `BulkForceFinish` | covered | |
| `GetAllLaunchNamesUsingGET` | `GetNames` | covered | |
| `CompareLaunchesUsingGET` | `Compare` | covered | |
| `ImportLaunchUsingPOST` | `Import` | covered | |
| `GetLaunchReportUsingGET` | `GetReport` | covered | |
| `CreateClustersUsingPOST` | n/a | deferred | ML clustering feature; low automation demand. |
| `GetClustersUsingGET` | n/a | deferred | ML clustering feature; low automation demand. |
| `GetAllOwnersUsingGET` | n/a | deferred | UI autocomplete for launch owners; low automation value. |
| `GetAttributeKeysUsingGET` | n/a | deferred | UI autocomplete for attribute keys; low automation value. |
| `GetAttributeValuesUsingGET` | n/a | deferred | UI autocomplete for attribute values; low automation value. |
| `GetStatusesUsingGET` | n/a | deferred | Launch status summary widget; low automation value. |
| `UpdateLaunchesUsingPUT` | n/a | deferred | Bulk attribute update on multiple launches; `BulkUpdate` covers the primary use case. |

## test-item-controller

Wrapper scope: `TestItemScope` (project-scoped via `Client.Project(name).Items()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `StartRootItemUsingPOST1` | `StartRoot` | covered | |
| `StartChildItemUsingPOST1` | `StartChild` | covered | |
| `FinishTestItemUsingPUT1` | `Finish` | covered | |
| `GetTestItemUsingGET` | `GetByID` | covered | |
| `GetTestItemByUuidUsingGET` | `GetByUUID` | covered | |
| `GetTestItemsUsingGET2` | `List` | covered | |
| `GetTestItemsUsingGET` | `GetByIDs` | covered | |
| `UpdateTestItemUsingPUT` | `Update` | covered | |
| `DeleteTestItemUsingDELETE` | `Delete` | covered | |
| `DefineTestItemIssueTypeUsingPUT` | `DefineIssueType` | covered | |
| `DeleteTestItemsUsingDELETE` | `DeleteBulk` | covered | |
| `BulkUpdateUsingPUT1` | `BulkUpdate` | covered | |
| `GetItemsHistoryUsingGET` | `GetHistory` | covered | |
| `GetTicketIdsUsingGET` | `BugTrackingScope.GetTicketIds` | covered | Spec-tagged here; wrapped on BugTrackingScope. |
| `GetTicketIdsForProjectUsingGET` | `BugTrackingScope.GetTicketIdsForProject` | covered | Spec-tagged here; wrapped on BugTrackingScope. |
| `GetTestItemsUsingGET1` | n/a | deferred | Provider-scoped listing variant; `GetTestItemsUsingGET2` (List) covers primary use case. |
| `GetTestItemsV2UsingGET` | n/a | deferred | V2 alternate listing endpoint; V1 is the stable contract. |
| `GetAttributeKeysForProjectUsingGET` | n/a | deferred | UI autocomplete; low automation value. |
| `GetAttributeKeysUsingGET1` | n/a | deferred | UI autocomplete; low automation value. |
| `GetAttributeKeysUsingGET2` | n/a | deferred | UI autocomplete; low automation value. |
| `GetAttributeValuesUsingGET1` | n/a | deferred | UI autocomplete; low automation value. |
| `GetAttributeValuesUsingGET2` | n/a | deferred | UI autocomplete; low automation value. |
| `GetSuggestedClusterItemsUsingGET` | n/a | deferred | ML suggestion feature; low automation demand. |
| `GetSuggestedItemsUsingGET` | n/a | deferred | ML suggestion feature; low automation demand. |
| `HandleSuggestChooseUsingPUT` | n/a | deferred | ML suggestion acceptance; low automation demand. |
| `LinkExternalIssuesUsingPUT` | n/a | deferred | BTS linking; can be added if automation use case arises. |
| `UnlinkExternalIssuesUsingPUT` | n/a | deferred | BTS unlinking; can be added if automation use case arises. |

## log-controller

Wrapper scope: `LogScope` (project-scoped via `Client.Project(name).Logs()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetLogUsingGET` | `GetByID` | covered | |
| `GetLogByUuidUsingGET` | `GetByUUID` | covered | |
| `GetLogsUsingGET` | `List` | covered | |
| `CreateLogEntryUsingPOST1` | `Create` | covered | |
| `DeleteLogUsingDELETE` | `Delete` | covered | |
| `CreateLogUsingPOST1` | `CreateBatch` | covered | Multipart batch creation. |
| `SearchLogsUsingPOST` | `Search` | covered | |
| `GetLogsUnderUsingPOST` | `GetLogsUnder` | covered | |
| `GetErrorPageUsingGET` | n/a | deferred | Error-page log navigation; UI pagination utility. |
| `GetNestedItemsUsingGET` | n/a | deferred | Nested item log aggregation; niche UI use case. |
| `GetPageNumberUsingGET` | n/a | deferred | UI pagination offset calculation; no automation value. |

## integration-controller

Wrapper scopes: `IntegrationsScope` (instance-scoped via `Client.Integrations()`)
and `ProjectIntegrationsScope` (project-scoped via `Client.Project(name).Integrations()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetGlobalIntegrationsUsingGET1` | `IntegrationsScope.ListGlobal` | covered | |
| `GetGlobalIntegrationsUsingGET` | `IntegrationsScope.ListGlobalByPlugin` | covered | |
| `GetGlobalIntegrationUsingGET` | `IntegrationsScope.GetGlobal` | covered | |
| `TestIntegrationConnectionUsingGET` | `IntegrationsScope.TestGlobalConnection` | covered | |
| `CreateGlobalIntegrationUsingPOST` | `IntegrationsScope.CreateGlobal` | covered | |
| `UpdateGlobalIntegrationUsingPUT` | `IntegrationsScope.UpdateGlobal` | covered | |
| `DeleteGlobalIntegrationUsingDELETE` | `IntegrationsScope.DeleteGlobal` | covered | |
| `GetProjectIntegrationsUsingGET1` | `ProjectIntegrationsScope.List` | covered | |
| `GetProjectIntegrationsUsingGET` | `ProjectIntegrationsScope.ListByPlugin` | covered | |
| `GetProjectIntegrationUsingGET` | `ProjectIntegrationsScope.Get` | covered | |
| `TestIntegrationConnectionUsingGET1` | `ProjectIntegrationsScope.TestConnection` | covered | |
| `CreateProjectIntegrationUsingPOST` | `ProjectIntegrationsScope.Create` | covered | |
| `UpdateProjectIntegrationUsingPUT` | `ProjectIntegrationsScope.Update` | covered | |
| `DeleteProjectIntegrationUsingDELETE` | `ProjectIntegrationsScope.Delete` | covered | |
| `ExecuteIntegrationCommandUsingPUT` | `ProjectIntegrationsScope.ExecuteCommand` | covered | |
| `DeleteAllIntegrationsUsingDELETE` | n/a | excluded | Destructive batch deletion of all global integrations; no safe automation use case. |
| `DeleteAllProjectIntegrationsUsingDELETE` | n/a | excluded | Destructive batch deletion of all project integrations; no safe automation use case. |

## bug-tracking-system-controller

Wrapper scope: `BugTrackingScope` (project-scoped via `Client.Project(name).BugTracking()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetSetOfIntegrationSystemFieldsUsingGET1` | `GetSystemFields` | covered | |
| `GetAllowableIssueTypesUsingGET1` | `GetAllowableIssueTypes` | covered | |
| `GetTicketUsingGET` | `GetTicket` | covered | |
| `CreateIssueUsingPOST` | `CreateIssue` | covered | |
| `GetSetOfIntegrationSystemFieldsUsingGET` | n/a | deferred | Deprecated variant; `GET1` is the current version. |
| `GetAllowableIssueTypesUsingGET` | n/a | deferred | Deprecated variant; `GET1` is the current version. |

## project-controller

Wrapper scope: `ProjectsScope` (instance-scoped via `Client.Projects()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetProjectUsingGET` | `Get` | covered | |
| `GetAllProjectNamesUsingGET` | `ListNames` | covered | |
| `GetAllProjectsInfoUsingGET` | `List` | covered | |
| `SearchProjectNamesUsingGET` | `SearchNames` | covered | |
| `GetProjectInfoUsingGET` | `GetInfo` | covered | |
| `CreateProjectUsingPOST` | `Create` | covered | |
| `DeleteProjectUsingDELETE` | `Delete` | covered | |
| `UpdateProjectUsingPUT` | `Update` | covered | |
| `AssignProjectUsersUsingPUT` | `AssignUsers` | covered | |
| `UnassignProjectUsersUsingPUT` | `UnassignUsers` | covered | |
| `GetProjectUsersUsingGET1` | `ListUsers` | covered | |
| `AddUserPreferenceUsingPUT` | n/a | deferred | UI user preferences; no automation value. |
| `DeleteProjectIndexUsingDELETE` | n/a | deferred | Analyzer index management; admin-only utility. |
| `DeleteProjectUsingDELETE1` | n/a | deferred | Soft-delete variant; standard `Delete` covers the primary use case. |
| `ExportProjectsUsingGET` | n/a | deferred | HTML export of project list; not automation-friendly. |
| `GetAnalyzerIndexingStatusUsingGET` | n/a | deferred | Analyzer index status; admin-only utility. |
| `GetProjectUsersUsingGET` | n/a | deferred | Deprecated variant; `GET1` is the current version. |
| `GetProjectWidgetUsingGET` | n/a | deferred | UI widget preview; no automation value. |
| `GetUserPreferenceUsingGET` | n/a | deferred | UI user preferences; no automation value. |
| `GetUsersForAssignUsingGET` | n/a | deferred | UI autocomplete for user assignment; low automation value. |
| `IndexProjectDataUsingPUT` | n/a | deferred | Analyzer re-indexing trigger; admin-only utility. |
| `RemoveUserPreferenceUsingDELETE` | n/a | deferred | UI user preferences; no automation value. |
| `SearchForUserUsingGET` | n/a | deferred | UI autocomplete for user search; low automation value. |
| `UpdateProjectNotificationConfigUsingPUT` | n/a | deferred | Notification config; overlaps with project-settings-controller. |

## project-settings-controller

Wrapper scope: `ProjectSettingsScope` (project-scoped via `Client.Project(name).Settings()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetProjectSettingsUsingGET` | `Get` | covered | |
| `GetNotificationsUsingGET` | `GetNotifications` | covered | |
| `CreateNotificationUsingPOST` | `CreateNotification` | covered | |
| `UpdateNotificationUsingPUT` | `UpdateNotification` | covered | |
| `DeleteNotificationUsingDELETE` | `DeleteNotification` | covered | |
| `CreatePatternTemplateUsingPOST` | `CreatePatternTemplate` | covered | |
| `UpdatePatternTemplateUsingPUT` | `UpdatePatternTemplate` | covered | |
| `DeletePatternTemplateUsingDELETE` | `DeletePatternTemplate` | covered | |
| `CreateProjectIssueSubTypeUsingPOST` | `CreateIssueSubType` | covered | |
| `UpdateProjectIssueSubTypeUsingPUT` | `UpdateIssueSubType` | covered | |
| `DeleteProjectIssueSubTypeUsingDELETE` | `DeleteIssueSubType` | covered | |

## dashboard-controller

Wrapper scope: `DashboardScope` (project-scoped via `Client.Project(name).Dashboards()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetAllDashboardsUsingGET` | `List` | covered | |
| `GetDashboardUsingGET` | `GetByID` | covered | |
| `CreateDashboardUsingPOST` | `Create` | covered | |
| `UpdateDashboardUsingPUT` | `Update` | covered | |
| `DeleteDashboardUsingDELETE` | `Delete` | covered | |
| `AddWidgetUsingPUT` | `AddWidget` | covered | |
| `RemoveWidgetUsingDELETE` | `RemoveWidget` | covered | |

## widget-controller

Wrapper scope: `WidgetScope` (project-scoped via `Client.Project(name).Widgets()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetWidgetUsingGET1` | `GetByID` | covered | |
| `CreateWidgetUsingPOST` | `Create` | covered | |
| `UpdateWidgetUsingPUT` | `Update` | covered | |
| `GetWidgetUsingGET` | `GetData` | covered | Fetches rendered widget data. |
| `GetWidgetNamesUsingGET` | `GetNames` | covered | |
| `GetWidgetPreviewUsingPOST` | n/a | deferred | UI preview rendering; no automation value. |

## user-filter-controller

Wrapper scope: `UserFilterScope` (project-scoped via `Client.Project(name).Filters()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetAllFiltersUsingGET` | `List` | covered | |
| `GetFilterUsingGET` | `GetByID` | covered | |
| `CreateFilterUsingPOST` | `Create` | covered | |
| `UpdateUserFilterUsingPUT` | `Update` | covered | |
| `DeleteFilterUsingDELETE` | `Delete` | covered | |
| `GetUserFiltersUsingGET` | `GetByIds` | covered | |
| `GetAllFiltersNamesUsingGET` | `GetNames` | covered | |
| `UpdateUserFiltersUsingPUT` | n/a | deferred | Batch filter update (orders/positions); low automation value. |

## activity-controller

Wrapper scope: `ActivityScope` (project-scoped via `Client.Project(name).Activities()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetActivityUsingGET` | `GetByID` | covered | |
| `GetTestItemActivitiesUsingGET` | `ListForItem` | covered | |

## activity-event-controller

Wrapper scope: `ActivityEventsScope` (instance-scoped via `Client.ActivityEvents()`).

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetActivitiesUsingPOST` | `Search` | covered | |
| `GetProjectSubjectNameUsingGET` | `GetProjectSubjectNames` | covered | |

## file-storage-controller

File download is exposed directly on `ProjectScope.DownloadFile`.

| Generated endpoint | Wrapper method | Status | Rationale |
|---|---|---|---|
| `GetFileUsingGET` | `ProjectScope.DownloadFile` | covered | Download attachments by data ID. |
| `DeletePhotoUsingDELETE` | n/a | excluded | User photo management; outside client scope. |
| `GetMyPhotoUsingGET` | n/a | excluded | User photo management; outside client scope. |
| `GetUserPhotoUsingGET` | n/a | excluded | User photo management; outside client scope. |
| `UploadPhotoUsingPOST` | n/a | excluded | User photo management; outside client scope. |
| `RemoveAttachmentsByCsvUsingPOST` | n/a | excluded | Destructive bulk attachment cleanup; no safe automation use case. |
