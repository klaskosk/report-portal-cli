# Prompt 50: Integration Tests for Plugins & Integrations

## Goal

Expand integration tests for the `PluginScope` and `IntegrationsScope` / `ProjectIntegrationsScope`. The existing `client_plugin_test.go` is minimal (36 lines, only `List` and `ListPublic`). The existing `client_integrations_test.go` (135 lines) covers global and project integration listing, get-by-ID, and global CRUD, but lacks coverage for project-level CRUD, connection testing, command execution, and plugin management operations.

## Context

This is PR 9 of the test coverage improvement plan (Phase 3). It runs in parallel with PRs 7, 8, and 10. Phase 2 must be merged first.

## Prerequisites

- Phases 1-2 (PRs 1-6) are merged.

## Scopes under test

### `PluginScope` — `Client.Plugins()`

| Method | Signature | Current coverage |
|--------|-----------|-----------------|
| `List` | `(ctx) ([]types.IntegrationTypeResource, error)` | Tested |
| `ListPublic` | `(ctx) ([]types.IntegrationTypeResource, error)` | Tested |
| `UpdateState` | `(ctx, pluginID int64, stateUpdate types.UpdatePluginStateRQ) (*types.OperationCompletionRS, error)` | **Not tested** |
| `Delete` | `(ctx, pluginID int64) (*types.OperationCompletionRS, error)` | **Not tested** |
| `Upload` | `(ctx, filename string, data io.Reader) (*types.EntryCreatedRS, error)` | **Not tested** |
| `ExecuteCommand` | `(ctx, projectName, pluginName, command string, body map[string]any) (map[string]any, error)` | **Not tested** |
| `Import` | `(ctx, projectName, pluginName string, filename string, data io.Reader, launchImportRQ *types.LaunchImportRQ) (map[string]any, error)` | **Not tested** |
| `ExecutePublicCommand` | `(ctx, pluginName, command string, body map[string]any) (map[string]any, error)` | **Not tested** |
| `GetPublicFile` | `(ctx, pluginName, fileName string) ([]byte, error)` | **Not tested** |

### `ProjectIntegrationsScope` — `Client.Project(name).Integrations()`

| Method | Signature | Current coverage |
|--------|-----------|-----------------|
| `List` | `(ctx) ([]types.IntegrationResource, error)` | Tested |
| `ListByPlugin` | `(ctx, pluginName string) ([]types.IntegrationResource, error)` | Tested |
| `Get` | `(ctx, integrationID int64) (*types.IntegrationResource, error)` | **Not tested** |
| `TestConnection` | `(ctx, integrationID int64) (bool, error)` | **Not tested** |
| `Create` | `(ctx, pluginName string, request types.CreateIntegrationRQ) (*types.EntryCreatedRS, error)` | **Not tested** |
| `Update` | `(ctx, integrationID int64, request types.UpdateIntegrationRQ) (*types.OperationCompletionRS, error)` | **Not tested** |
| `Delete` | `(ctx, integrationID int64) (*types.OperationCompletionRS, error)` | **Not tested** |
| `DeleteAllByType` | `(ctx, pluginName string) (*types.OperationCompletionRS, error)` | **Not tested** |
| `ExecuteCommand` | `(ctx, integrationID int64, command string, body map[string]any) (map[string]any, error)` | **Not tested** |

## Implementation plan

### 1) Expand `test/integration/client_plugin_test.go`

Add tests for plugin management operations. Note that some operations (Upload, Delete) are destructive and may affect the test environment, so approach with care:

```
Describe("Client: Plugins", Label("client", "plugins"))
  BeforeEach: newClient()

  [existing List/ListPublic contexts remain]

  Context("UpdateState")
    It("toggles a plugin's enabled state")
      By("listing plugins to find a known plugin")
      By("updating the plugin state")
      By("restoring the original state")

  Context("Plugin Commands")
    It("executes a public command on a known plugin")
      - Use a built-in plugin name (check available plugins first)
      - Call c.Plugins().ExecutePublicCommand(ctx, pluginName, command, body)
      - Assert no error (or specific error if the command is unsupported)
```

**Caution**: `Upload` and `Delete` operations for plugins are heavyweight and could disrupt the test environment. If feasible, skip these and document why. If the test environment is ephemeral (testcontainers), they may be safe to test.

### 2) Expand `test/integration/client_integrations_test.go`

Add project-level integration CRUD and advanced operations:

```
Describe("Client: Integrations", Label("client", "integrations"))
  [existing global/project listing contexts remain]

  Context("Project Integration CRUD")
    It("creates, gets, updates, and deletes a project integration")
      By("creating a test project")
      By("creating a project integration for a known plugin")
      By("getting the integration by ID")
      By("updating the integration")
      By("deleting the integration")

  Context("Connection Testing")
    It("tests connection for a known integration")
      - Find or create an integration
      - Call TestConnection(ctx, integrationID)
      - Assertion depends on whether the integration backend is reachable

  Context("Delete All By Type")
    It("deletes all project integrations of a given type")
      By("creating two project integrations of the same type")
      By("calling DeleteAllByType")
      By("verifying none remain")
```

### 3) Determine available plugins

Before implementing, run the test suite once and log which plugins are available:

```go
plugins, _ := c.Plugins().List(ctx)
for _, p := range plugins {
    fmt.Printf("Plugin: %s (ID: %d, Enabled: %v)\n", p.Name, p.TypeId, p.Enabled)
}
```

Common built-in plugins include `rally`, `jira`, `email`, `ldap`, `saml`. Use whichever is available for integration tests.

### 4) Read request types

Read `types.CreateIntegrationRQ`, `types.UpdateIntegrationRQ`, `types.UpdatePluginStateRQ` to understand their fields.

### 5) Follow existing test conventions

- `NodeTimeout` on setup/teardown
- `By("description")` for multi-step lifecycle tests
- Clean up created integrations in `AfterEach` or `DeferCleanup`
- Labels: `Label("client", "plugins")`, `Label("client", "integrations")`

### 6) Run tests

```bash
make test-integration
make lint
```

## Files touched

| File | Change |
|------|--------|
| `test/integration/client_plugin_test.go` | Expand with state management and command tests |
| `test/integration/client_integrations_test.go` | Expand with project integration CRUD, connection, and cleanup tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- Plugin Upload/Delete tested only if safe in the ephemeral test environment
