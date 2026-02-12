# Prompt 09: Integrations + Bug Tracking Systems Scopes (Milestone 9)

## Goal

Implement client scopes for managing integrations (e.g., Jira, Rally, email, LDAP, SAML) and bug tracking system (BTS) operations in `pkg/client`. These are the endpoints that connect Report Portal to external systems for defect tracking and notifications. Start with list/get/validate flows; configuration and mutation endpoints are secondary.

## Context

Report Portal supports two levels of integrations:

- **Global integrations**: configured at the instance level, shared across all projects (e.g., LDAP, SAML, email server)
- **Project integrations**: configured per-project, override or supplement global ones (e.g., per-project Jira connection)

The BTS (Bug Tracking System) controller provides operations specific to bug trackers: getting available fields, issue types, creating/linking issues, and retrieving tickets.

For the client library, both global and project-scoped integration access should be available. BTS operations are project-scoped since they involve project-specific integration configurations.

## Generated API endpoints available

### Integration controller

#### Instance-scoped (global integrations)

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetGlobalIntegrationsUsingGET1WithResponse(ctx)` | List all global integrations | -> integration list |
| `GetGlobalIntegrationsUsingGETWithResponse(ctx, pluginName)` | List global integrations by plugin name | -> integration list |
| `GetGlobalIntegrationUsingGETWithResponse(ctx, integrationId int64)` | Get a single global integration | -> `IntegrationResource` |
| `CreateGlobalIntegrationUsingPOSTWithResponse(ctx, pluginName, IntegrationRQ)` | Create a global integration | `IntegrationRQ` -> `EntryCreatedRS` |
| `UpdateGlobalIntegrationUsingPUTWithResponse(ctx, integrationId int64, IntegrationRQ)` | Update a global integration | `IntegrationRQ` -> `OperationCompletionRS` |
| `DeleteGlobalIntegrationUsingDELETEWithResponse(ctx, integrationId int64)` | Delete a global integration | -> `OperationCompletionRS` |
| `DeleteAllIntegrationsUsingDELETEWithResponse(ctx, pType)` | Delete all integrations of a type | -> `OperationCompletionRS` |
| `TestIntegrationConnectionUsingGETWithResponse(ctx, integrationId int64)` | Test connection for a global integration | -> boolean/success |

#### Project-scoped (project integrations)

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetProjectIntegrationsUsingGET1WithResponse(ctx, projectName)` | List all integrations for a project | -> integration list |
| `GetProjectIntegrationsUsingGETWithResponse(ctx, projectName, pluginName)` | List project integrations by plugin name | -> integration list |
| `GetProjectIntegrationUsingGETWithResponse(ctx, projectName, integrationId int64)` | Get a single project integration | -> `IntegrationResource` |
| `CreateProjectIntegrationUsingPOSTWithResponse(ctx, projectName, pluginName, IntegrationRQ)` | Create a project integration | `IntegrationRQ` -> `EntryCreatedRS` |
| `UpdateProjectIntegrationUsingPUTWithResponse(ctx, projectName, integrationId int64, IntegrationRQ)` | Update a project integration | `IntegrationRQ` -> `OperationCompletionRS` |
| `DeleteProjectIntegrationUsingDELETEWithResponse(ctx, projectName, integrationId int64)` | Delete a project integration | -> `OperationCompletionRS` |
| `DeleteAllProjectIntegrationsUsingDELETEWithResponse(ctx, projectName, pType)` | Delete all project integrations of a type | -> `OperationCompletionRS` |
| `TestIntegrationConnectionUsingGET1WithResponse(ctx, projectName, integrationId int64)` | Test connection for a project integration | -> boolean/success |
| `ExecuteIntegrationCommandUsingPUTWithResponse(ctx, projectName, integrationId int64, command, body)` | Execute a plugin command on a project integration | JSON body -> response |

### BTS controller

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetSetOfIntegrationSystemFieldsUsingGETWithResponse(ctx, integrationId int64, params)` | Get BTS system fields (global) | -> field list |
| `GetAllowableIssueTypesUsingGETWithResponse(ctx, integrationId int64)` | Get allowable issue types (global) | -> string list |
| `GetSetOfIntegrationSystemFieldsUsingGET1WithResponse(ctx, projectName, integrationId int64, params)` | Get BTS system fields (project) | -> field list |
| `GetAllowableIssueTypesUsingGET1WithResponse(ctx, projectName, integrationId int64)` | Get allowable issue types (project) | -> string list |
| `GetTicketUsingGETWithResponse(ctx, projectName, ticketId, params)` | Get a ticket by ID from BTS | -> `Ticket` |
| `CreateIssueUsingPOSTWithResponse(ctx, projectName, integrationId int64, PostTicketRQ)` | Create an issue in BTS | `PostTicketRQ` -> `Ticket` |

### Key types

```go
type IntegrationResource struct {
    CreationDate          *EpochMillis             `json:"creationDate,omitempty"`
    Creator               *string                  `json:"creator,omitempty"`
    Enabled               *bool                    `json:"enabled,omitempty"`
    Id                    *int64                   `json:"id,omitempty"`
    IntegrationParameters *map[string]interface{}  `json:"integrationParameters,omitempty"`
    IntegrationType       *IntegrationTypeResource `json:"integrationType,omitempty"`
    Name                  *string                  `json:"name,omitempty"`
    ProjectId             *int64                   `json:"projectId,omitempty"`
}

type IntegrationRQ struct {
    Enabled               *bool                  `json:"enabled,omitempty"`
    IntegrationParameters *map[string]interface{} `json:"integrationParameters,omitempty"`
    Name                  string                 `json:"name"`
}

type Ticket struct {
    Id      *string `json:"id,omitempty"`
    Summary *string `json:"summary,omitempty"`
    Status  *string `json:"status,omitempty"`
    Url     *string `json:"url,omitempty"`
}

type PostTicketRQ struct {
    BackLinks  *map[string][]IssueDefinition `json:"backLinks,omitempty"`
    Fields     *[]PostFormField               `json:"fields,omitempty"`
    IsIncludeAttachments *bool                `json:"isIncludeAttachments,omitempty"`
    IsIncludeComments    *bool                `json:"isIncludeComments,omitempty"`
    IsIncludeLogs        *bool                `json:"isIncludeLogs,omitempty"`
    IsIncludeScreenshots *bool                `json:"isIncludeScreenshots,omitempty"`
    LogQuantity          *int32               `json:"logQuantity,omitempty"`
    NumberOfLogs         *int32               `json:"numberOfLogs,omitempty"`
    TestItemId           *int64               `json:"testItemId,omitempty"`
}
```

## Client requirements

### IntegrationsScope (instance-scoped global operations)

Create `pkg/client/integrations.go` with an `IntegrationsScope` struct:

- **Instance-scoped**: accessible via `client.Integrations()`
- Add an `Integrations()` method to `Client` in `pkg/client/client.go`
- Struct holds `client *Client`

Methods to implement (prioritized — implement read/validate first):

1. **`ListGlobal(ctx) ([]types.IntegrationResource, error)`** — list all global integrations
2. **`ListGlobalByPlugin(ctx, pluginName string) ([]types.IntegrationResource, error)`** — list global integrations for a specific plugin
3. **`GetGlobal(ctx, id int64) (*types.IntegrationResource, error)`** — get a single global integration
4. **`TestGlobalConnection(ctx, id int64) (bool, error)`** — test connectivity for a global integration

Optional (lower priority):
5. `CreateGlobal(ctx, pluginName string, request IntegrationRQ) (*types.EntryCreatedRS, error)`
6. `UpdateGlobal(ctx, id int64, request IntegrationRQ) (*types.OperationCompletionRS, error)`
7. `DeleteGlobal(ctx, id int64) (*types.OperationCompletionRS, error)`

### Project-scoped integration methods

Add methods to a new `ProjectIntegrationsScope` or extend the existing integration scoping. Recommended approach: create a project-scoped accessor.

Create `pkg/client/project_integrations.go` or add to `integrations.go`:

- **Project-scoped**: accessible via `client.Project(name).Integrations()`
- Add an `Integrations()` method to `ProjectScope` in `pkg/client/project.go`

Methods to implement:

1. **`List(ctx) ([]types.IntegrationResource, error)`** — list all integrations for the project
2. **`ListByPlugin(ctx, pluginName string) ([]types.IntegrationResource, error)`** — list project integrations by plugin
3. **`Get(ctx, id int64) (*types.IntegrationResource, error)`** — get a single project integration
4. **`TestConnection(ctx, id int64) (bool, error)`** — test connectivity for a project integration

Optional (lower priority):
5. `Create(ctx, pluginName string, request IntegrationRQ) (*types.EntryCreatedRS, error)`
6. `Update(ctx, id int64, request IntegrationRQ) (*types.OperationCompletionRS, error)`
7. `Delete(ctx, id int64) (*types.OperationCompletionRS, error)`

### BugTrackingScope (project-scoped)

Create `pkg/client/bts.go` with a `BugTrackingScope` struct:

- **Project-scoped**: accessible via `client.Project(name).BugTracking()`
- Add a `BugTracking()` method to `ProjectScope` in `pkg/client/project.go`

Methods to implement:

1. **`GetSystemFields(ctx, integrationId int64) ([]types.PostFormField, error)`** — get available fields for the BTS (use the project-scoped endpoint)
2. **`GetAllowableIssueTypes(ctx, integrationId int64) ([]string, error)`** — get allowable issue types
3. **`GetTicket(ctx, ticketId string) (*types.Ticket, error)`** — get a ticket from the BTS (note: needs `btsUrl` and `btsProject` as params — check `GetTicketUsingGETParams`)

Optional (lower priority):
4. `CreateIssue(ctx, integrationId int64, request PostTicketRQ) (*types.Ticket, error)` — create an issue in the BTS

### Naming note: scope accessor method conflict

`ProjectScope` already has `Integrations()` returning a different type wouldn't work if there's a name collision. If the instance-scoped `IntegrationsScope` method on `Client` is named `Integrations()`, then the project-scoped one can also be named `Integrations()` since they're on different receiver types (`*Client` vs `*ProjectScope`). This is the recommended approach.

## Client tests

### `pkg/client/integrations_test.go` (new, `package client_test`)

- **ListGlobal**: success (200 with list), unauthorized (401)
- **ListGlobalByPlugin**: success (200), empty list for unknown plugin
- **GetGlobal**: success (200), not found (404)
- **TestGlobalConnection**: success (true), not found (404 — invalid integration ID)
- Project-scoped equivalents: **List**, **Get**, **TestConnection** — success and error cases

### `pkg/client/bts_test.go` (new, `package client_test`)

- **GetSystemFields**: success (200), not found (404 — integration not found)
- **GetAllowableIssueTypes**: success (200), not found (404)
- **GetTicket**: success (200), not found (404 — ticket not found)

### `export_test.go`

Add test-only constructors for `IntegrationsScope`, project-scoped integration scope, and `BugTrackingScope`.

## Integration tests

### `test/integration/client_integrations_test.go` (new)

- Use `Label("client", "integrations")`
- **ListGlobal**: verify returns without error (may be empty on a fresh instance)
- **ListProjectIntegrations**: verify returns without error for the test project

Note: Integration tests for this area are limited because configuring real external integrations (Jira, LDAP, etc.) in a test environment is complex and fragile. Focus on verifying that endpoints respond correctly, even if the results are empty.

### BTS integration tests

BTS tests are even more environment-dependent (require a configured BTS integration). Consider marking these tests with an additional label (e.g., `Label("bts")`) and documenting that they only pass when a BTS integration is pre-configured. For the basic PR, verifying that the API returns sensible errors (e.g., "integration not found" when no BTS is configured) is sufficient.

## Design considerations

- **Two scoping levels**: Integrations exist at both global (instance) and project level. The client should reflect both. The `IntegrationsScope` on `Client` handles global; a project-scoped equivalent on `ProjectScope` handles project-level.
- **BTS is separate from integrations**: Although BTS depends on integrations, it has its own controller with distinct operations (fields, issue types, tickets). Keep it as a separate scope.
- **Test connection returns**: The `TestIntegrationConnectionUsingGET` endpoint may return a `YesNoRS` (`Is *bool`) or an `OperationCompletionRS`. Check the generated response struct and adapt the wrapper to return a `bool`.
- **Plugin commands**: The `ExecuteIntegrationCommandUsingPUT` endpoint is a generic "execute command" endpoint that different plugins handle differently. Defer this to a follow-up unless there's a clear, commonly-used command.
- **Limited test coverage**: Since real BTS integrations can't easily be provisioned in the test container stack, integration tests for this area will be lighter than for launches/items. This is acceptable.

## Definition of done

- [ ] `pkg/client/integrations.go` with instance-scoped `IntegrationsScope` (ListGlobal, ListGlobalByPlugin, GetGlobal, TestGlobalConnection)
- [ ] Project-scoped integration methods (List, ListByPlugin, Get, TestConnection)
- [ ] `pkg/client/bts.go` with `BugTrackingScope` (GetSystemFields, GetAllowableIssueTypes, GetTicket)
- [ ] Scope accessor methods on `Client` and `ProjectScope`
- [ ] Unit tests for all new scopes
- [ ] Integration tests (list/get flows that work on a fresh instance)
- [ ] `code-verifier` subagent says your changes are good to go
