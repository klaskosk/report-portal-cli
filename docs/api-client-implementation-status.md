# High-Level Summary

The API client is highly functional for day-to-day reporting and querying tasks. It has excellent coverage for core reporting entities (Launches, Test Items, Logs, Dashboards, Widgets).

However, it is missing significant portions of **administrative and management APIs** (User management, API key management, advanced Project configuration) and **advanced analytical features** (ML clusters, suggested items, statistics).

## Architectural Note: Sync vs. Async (v1 vs v2)

Report Portal provides `v1` (synchronous) and `v2` (asynchronous) endpoints for reporting data. **The client currently exclusively implements the `v1` synchronous controllers** (`launch-controller`, `test-item-controller`, `log-controller`). The `v2` async controllers (`launch-async-controller`, `test-item-async-controller`, `log-async-controller`) are **not implemented** at all.

---

## 🟢 Fully Implemented Controllers

These controllers have 100% of their OpenAPI operations implemented in the Go client:

- `**activity-controller`** (`ActivityScope`): Getting activities for test items and by ID.
- `**activity-event-controller**` (`ActivityEventsScope`): Searching activities and getting project subject names.
- `**bug-tracking-system-controller**` (`BugTrackingScope`): Fetching BTS fields, issue types, tickets, and creating issues.
- `**dashboard-controller**` (`DashboardScope`): Full CRUD for dashboards, plus adding/removing widgets.
- `**demo-data-controller**` (`DemoDataScope`): Generating demo data for a project.
- `**project-settings-controller**` (`ProjectSettingsScope`): Full CRUD for project notifications, pattern templates, and custom issue sub-types.
- `**onboarding-controller**` (`OnboardingScope`): Getting onboarding status.

---

## 🟡 Mostly Implemented Controllers

These controllers cover the primary use cases but are missing specific endpoints:

- `**launch-controller**` (`LaunchScope`):
  - **Implemented:** Start, Finish, ForceFinish, Merge, Analyze, Import, Compare, Get (ID/UUID), List, Delete, DeleteBulk, GetLatest, GetDebug, GetNames, GetReport, Update, BulkUpdate.
  - **Not Implemented:** Fetching unique attribute keys/values, fetching ML clusters (`/cluster/{launchId}`), creating clusters, fetching all unique owners, and fetching statuses.
- `**test-item-controller`** (`TestItemScope`):
  - **Implemented:** StartRoot, StartChild, Finish, Get (ID/UUID), GetByIDs, List, Update, BulkUpdate, Delete, DeleteBulk, GetHistory, DefineIssueType.
  - **Not Implemented:** Fetching unique attribute keys/values (for items and steps), fetching accumulated statistics (`/statistics`), ML suggested items (`/suggest/{itemId}` and `/suggest/cluster/{clusterId}`), handling user choice for suggested items, and linking/unlinking external issues.
- `**log-controller`** (`LogScope`):
  - **Implemented:** Create (single/batch), Get (ID/UUID), List, Delete, Search, GetLogsUnder.
  - **Not Implemented:** Getting next/previous log in test item (`/locations/{parentId}`), getting nested steps with logs (`/nested/{parentId}`), and getting page numbers for logs.
- `**integration-controller`** (`IntegrationsScope` / `ProjectIntegrationsScope`):
  - **Implemented:** Full CRUD and connection testing for both Global and Project-level integrations, plus executing commands.
  - **Not Implemented:** Bulk deleting all integrations by type (`/all/{type}`).
- `**project-controller`** (`ProjectsScope`):
  - **Implemented:** Create, Get, Update, Delete, List, SearchNames, ListNames, GetInfo, AssignUsers, UnassignUsers, ListUsers.
  - **Not Implemented:** Exporting projects, getting analyzer indexing status, managing user preferences (`/preference/{login}`), indexing project data in ML, and updating project notification configuration.
- `**user-filter-controller`** (`UserFilterScope`):
  - **Implemented:** Create, Get, List, Update, Delete, GetNames, GetByIds.
  - **Not Implemented:** Bulk updating a list of user filters.
- `**widget-controller`** (`WidgetScope`):
  - **Implemented:** Create, Get, Update, GetData (preview), GetNames.
  - **Not Implemented:** Getting multilevel widgets by ID (`/multilevel/{widgetId}`).
- `**plugin-controller`** & `**plugin-public-controller**` (`PluginScope`):
  - **Implemented:** Listing plugins (public/private), updating state, deleting, getting public files.
  - **Not Implemented:** Uploading new plugins via POST, and executing plugin commands.
- `**settings-controller`** (`SettingsScope`):
  - **Implemented:** Getting server settings, updating analytics.
  - **Not Implemented:** There is a duplicate PUT/POST for analytics in the spec; the client only implements one.

---

## 🔴 Barely Implemented / Missing Controllers

These controllers represent significant gaps in the API client's capabilities:

- `**user-controller`** (`UsersScope`):
  - **Implemented:** `GetMyself`, `GetByLogin`, and `ListProjects`.
  - **Not Implemented (19 operations):** All administrative user management (Create, Edit, Delete), password management (Change, Reset, Restore), **API Key management** (Create, List, Delete), exporting users, and registration/invitation workflows.
- `**file-storage-controller`** (`ProjectScope.DownloadFile`):
  - **Implemented:** `getFile` (`DownloadFile`).
  - **Not Implemented (5 operations):** Uploading/deleting user photos, getting user photos, and removing attachments via CSV.

## ⚫ Intentionally Not Implemented

- `**deprecated-user-controller`**: This entire controller is deprecated in the OpenAPI spec in favor of `user-controller`, so its absence in the Go client is correct and expected.

