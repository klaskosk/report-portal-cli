# Prompt 45: Integration Tests for Groups

## Goal

Implement integration tests for user groups and project groups, covering the full CRUD lifecycle and listing operations. These are "zero-to-one" tests — currently no integration test files exist for groups.

## Context

This is PR 4 of the test coverage improvement plan (Phase 2). It runs in parallel with PRs 5 and 6. Phase 1 must be merged first.

## Prerequisites

- Phase 1 (PRs 1-3) is merged, particularly PR 3 which refactors the integration test helpers.

## Scopes under test

### `GroupsScope` — `Client.Groups()`

| Method | Signature |
|--------|-----------|
| `List` | `(ctx, options ...ListGroupsOption) (*types.GroupPage, error)` |
| `Get` | `(ctx, groupID int64) (*types.GroupInfo, error)` |
| `Create` | `(ctx, request types.CreateGroupRequest) (*types.GroupInfo, error)` |
| `Update` | `(ctx, groupID int64, request types.UpdateGroupRequest) (*types.SuccessfulUpdate, error)` |
| `Delete` | `(ctx, groupID int64) error` |
| `ListProjects` | `(ctx, groupID int64, options ...ListGroupProjectsOption) (*types.GroupProjectsPage, error)` |
| `GetProject` | `(ctx, groupID, projectID int64) (*types.GroupProjectInfo, error)` |
| `AddProject` | `(ctx, groupID, projectID int64, request types.AddProjectToGroupByIdRequest) (*types.SuccessfulUpdate, error)` |
| `RemoveProject` | `(ctx, groupID, projectID int64) error` |
| `ListUsers` | `(ctx, groupID int64, options ...ListGroupUsersOption) (*types.GroupUsersPage, error)` |
| `GetUser` | `(ctx, groupID, userID int64) (*types.GroupUserInfo, error)` |
| `AddUser` | `(ctx, groupID, userID int64) (*types.SuccessfulUpdate, error)` |
| `RemoveUser` | `(ctx, groupID, userID int64) error` |

### `ProjectGroupsScope` — `Client.Project(name).Groups()`

| Method | Signature |
|--------|-----------|
| `List` | `(ctx, options ...ListProjectGroupsOption) (*types.ProjectGroupsPage, error)` |
| `Get` | `(ctx, groupID int64) (*types.ProjectGroupInfo, error)` |
| `AddOrUpdate` | `(ctx, groupID int64, request types.AddProjectToGroupByIdRequest) (*types.SuccessfulUpdate, error)` |
| `Delete` | `(ctx, groupID int64) error` |

## Implementation plan

### 1) Create `test/integration/client_groups_test.go`

```go
//go:build integration

package integration_test

import (
    "fmt"
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "github.com/klaskosk/report-portal-cli/pkg/client"
    "github.com/klaskosk/report-portal-cli/pkg/types"
)
```

#### Describe block structure

```
Describe("Client: Groups", Label("client", "groups"))
  BeforeEach: newClient(), create a test group via c.Groups().Create()
  AfterEach: delete the test group via c.Groups().Delete()

  Context("CRUD Lifecycle")
    It("creates a group and retrieves it by ID")
    It("updates a group name")
    It("deletes a group")
    It("returns not found for a deleted group")

  Context("List")
    It("returns at least one group")
    It("supports pagination options")

  Context("User Management")
    It("adds and retrieves a user in a group")
    It("lists users in a group")
    It("removes a user from a group")

  Context("Project Management")
    It("adds a project to a group")
    It("retrieves a project in a group")
    It("lists projects in a group")
    It("removes a project from a group")
```

### 2) Test data setup

- **Group creation**: Use `c.Groups().Create(ctx, types.CreateGroupRequest{Name: "test_group_<timestamp>"})` to create an isolated test group.
- **User for user-management tests**: The admin user (from `c.Users().GetMyself()`) is always available and can be added to a group.
- **Project for project-management tests**: Create a test project via the refactored `createProject` helper, get its ID from `c.Projects().Get(ctx, projectName)`.

### 3) Test patterns

Follow existing integration test conventions:

- `NodeTimeout(60*time.Second)` on `BeforeEach`, `NodeTimeout(15*time.Second)` on `AfterEach`
- `By("description")` for documentation of multi-step tests
- `Expect(err).NotTo(HaveOccurred(), "contextual message")`
- `Expect(err).To(HaveOccurred())` + `Expect(client.IsNotFound(err)).To(BeTrue())` for not-found assertions

### 4) Create `test/integration/client_project_groups_test.go`

```
Describe("Client: Project Groups", Label("client", "project-groups"))
  BeforeEach: newClient(), create test project and test group
  AfterEach: delete test project and test group

  Context("CRUD Lifecycle")
    It("adds a group to a project and retrieves it")
    It("updates a group association on a project")
    It("deletes a group association from a project")

  Context("List")
    It("returns groups associated with a project")
```

### 5) Run integration tests

```bash
make test-integration
```

Focus on the new tests during development:

```bash
go test -tags integration -v -run 'Client:_Groups|Client:_Project_Groups' ./test/integration/
```

### 6) Run linter

```bash
make lint
```

## Edge cases to consider

- Group names must be unique — use timestamps or random suffixes to avoid collisions with parallel test runs
- Deleting a group that has associated projects/users should succeed (verify this or handle the teardown order)
- The admin user may already be in the default group — adding them to a test group should still succeed

## Files to create

| File | Action |
|------|--------|
| `test/integration/client_groups_test.go` | Create — GroupsScope integration tests |
| `test/integration/client_project_groups_test.go` | Create — ProjectGroupsScope integration tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No existing integration tests modified
