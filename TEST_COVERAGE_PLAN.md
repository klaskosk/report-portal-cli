# Test Coverage Improvement Plan: `pkg/client`

## Executive Summary
This document outlines the strategy to bridge the testing gap in the `pkg/client` module. Currently, unit test coverage is excellent (88.5%), while integration test coverage lags significantly (32.9%).

To maximize the efficiency of AI agents executing this work, the plan is structured into **Phases** designed for high parallelism (Fanning Out). Tasks within the same phase touch disjoint sets of files, allowing multiple agents to work concurrently without generating merge conflicts. The phases build upon each other, ensuring that foundational helpers and technical debt are addressed before tackling the most complex data hierarchies.

---

## Phase 1: Foundation & Quick Wins
*Objective: Close the remaining unit test gaps and clean up technical debt in the integration test suite to pave the way for faster test authoring.*

This phase can be fanned out immediately to 3 concurrent agents.

*   **PR 1: Unit Test Functional Options (Items & Launches)**
    *   **Scope:** `pkg/client/item_test.go`, `pkg/client/launch_test.go`
    *   **Task:** Add table-driven tests to verify that `With...` functional options (e.g., `WithHistoryFilterID`, `WithLaunchSort`) correctly mutate the underlying HTTP request.
*   **PR 2: Unit Test Functional Options (Projects & Users)**
    *   **Scope:** `pkg/client/projects_test.go`, `pkg/client/users_test.go`, `pkg/client/dashboard_test.go`, `pkg/client/filter_test.go`
    *   **Task:** Add table-driven tests to verify functional options for these remaining core resources. This should push unit test coverage > 95%.
*   **PR 3: Refactor Raw HTTP Calls in Integration Tests**
    *   **Scope:** `test/integration/helpers_test.go`, `test/integration/client_widget_test.go`, `test/integration/client_launch_test.go`
    *   **Task:** Replace raw `http.NewRequest` and `httpClient.Do` calls with their respective `pkg/client` wrapper methods.
        *   Update `createProject` to use `client.Projects().Create()`.
        *   Update `createLog` / `createMultipartLog` to use `client.Project(name).Logs().CreateBatch()`.
        *   Update `client_widget_test.go` to use `widgets.GetNames()`.
        *   *Note: Evaluate `client_launch_test.go`'s `Compare` test to see if it can use the wrapper or if it strictly requires raw JSON validation.*

---

## Phase 2: "Zero-to-One" Integration Coverage
*Objective: Establish baseline integration tests for files that currently have 0% coverage. These are largely independent systems and can be heavily parallelized.*

Once Phase 1 is merged (Fan-in), fan out to 3 concurrent agents.

*   **PR 4: Integration Tests for Groups**
    *   **Scope:** `test/integration/client_groups_test.go`, `test/integration/client_project_groups_test.go`
    *   **Task:** Implement CRUD and listing tests for user groups and project groups.
*   **PR 5: Integration Tests for Bug Tracking Systems (BTS)**
    *   **Scope:** `test/integration/client_instance_bts_test.go`, `test/integration/client_bts_test.go`
    *   **Task:** Implement tests for instance-level and project-level bug tracking system integrations.
*   **PR 6: Integration Tests for Onboarding & Log Types**
    *   **Scope:** `test/integration/client_onboarding_test.go`, `test/integration/client_log_types_test.go`
    *   **Task:** Implement the missing integration tests for these smaller, isolated endpoints.

---

## Phase 3: Core Resource Expansion
*Objective: Expand integration coverage for core resources that currently have very low coverage (< 30%). These resources are mostly independent of the heavy Launch/Item hierarchy.*

Fan out to 4 concurrent agents.

*   **PR 7: Integration Tests for Users Expansion**
    *   **Scope:** `test/integration/client_users_test.go`
    *   **Task:** Expand beyond basic fetching. Add tests for creating, editing, deleting users, and managing API keys.
*   **PR 8: Integration Tests for Projects Expansion**
    *   **Scope:** `test/integration/client_projects_test.go`
    *   **Task:** Expand beyond basic fetching. Add tests for project preferences, indexing, and analyzer status.
*   **PR 9: Integration Tests for Plugins & Integrations**
    *   **Scope:** `test/integration/client_plugin_test.go`, `test/integration/client_integrations_test.go`
    *   **Task:** Implement tests for plugin uploading/management and global/project integrations.
*   **PR 10: Integration Tests for Files**
    *   **Scope:** `test/integration/client_file_test.go`
    *   **Task:** Implement tests for user photo uploads and project file attachments.

---

## Phase 4: The Complex Hierarchies
*Objective: Tackle the most complex and heavily used endpoints (Launches, Items, Logs). These require careful test data setup (Project -> Launch -> Item -> Log).*

Because Phase 1 refactored the helpers, setting up this data will be much easier. Fan out to 3 concurrent agents.

*   **PR 11: Integration Tests for Launches (Advanced)**
    *   **Scope:** `test/integration/client_launch_test.go`
    *   **Task:** Implement tests for advanced filtering, bulk force finishing, merging, and comparing launches.
*   **PR 12: Integration Tests for Test Items (Advanced)**
    *   **Scope:** `test/integration/client_item_test.go`
    *   **Task:** Implement tests for linking/unlinking external issues, bulk operations, and advanced querying/history.
*   **PR 13: Integration Tests for Logs (Advanced)**
    *   **Scope:** `test/integration/client_log_test.go`
    *   **Task:** Implement tests for log searching, nested log retrieval, and edge cases in batch creation.

---

## Phase 5: Fan-In & Final Polish
*Objective: Consolidate the work, resolve any final edge cases, and verify the coverage metrics.*

*   **PR 14: Final Verification & Coverage Report**
    *   **Scope:** Entire repository.
    *   **Task:** Run a full `make verify`. Generate the final coverage report. Address any minor remaining gaps to ensure integration coverage is > 80% and unit test coverage is > 95%. Ensure no test pollution exists between the newly added parallel integration tests.
