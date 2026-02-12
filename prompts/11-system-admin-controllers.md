# Prompt 11: System / Admin / Internal Controllers (Milestone 12)

## Goal

Round out client-only coverage for the remaining low-level, admin, and internal API controllers: server settings, plugins, onboarding, and demo data. These are the "long tail" endpoints that complete the Go client's API surface. No CLI commands are added — these are explicitly client-only (or highly selective `rp admin ...` at most).

## Context

With the preceding prompts (05–10) covering the high-value API areas, these controllers represent the remaining generated API surface:

- **Settings controller**: server-level configuration (analytics settings)
- **Plugin controller**: manage installed plugins (upload, enable/disable, delete, execute commands)
- **Plugin public controller**: public plugin endpoints (list plugins, get plugin files, execute public commands)
- **Onboarding controller**: first-run onboarding status
- **Demo data controller**: generate demo data for testing/evaluation

These are low-traffic, admin-oriented endpoints. Most users will never call them directly, but having them in the client library enables automation tools, admin scripts, and completeness for library consumers.

## Generated API endpoints available

### Settings controller (instance-scoped)

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetServerSettingsUsingGETWithResponse(ctx)` | Get server settings (analytics config) | -> `AnalyticsResource` or map |
| `SaveAnalyticsSettingsUsingPUTWithResponse(ctx, body)` | Update analytics settings (PUT) | JSON body -> `OperationCompletionRS` |
| `SaveAnalyticsSettingsUsingPOSTWithResponse(ctx, body)` | Save analytics settings (POST) | JSON body -> `OperationCompletionRS` |

### Plugin controller (instance-scoped, admin)

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetPluginsUsingGETWithResponse(ctx)` | List all installed plugins | -> plugin list |
| `UploadPluginUsingPOSTWithBody(ctx, contentType, body)` | Upload a new plugin (multipart) | multipart -> `EntryCreatedRS` or similar |
| `UpdatePluginStateUsingPUTWithResponse(ctx, pluginId int64, UpdatePluginStateRQ)` | Enable/disable a plugin | `UpdatePluginStateRQ` -> `OperationCompletionRS` |
| `DeletePluginUsingDELETEWithResponse(ctx, pluginId int64)` | Delete a plugin | -> `OperationCompletionRS` |
| `ExecutePluginCommandUsingPUTWithResponse(ctx, projectName, pluginName, command, body)` | Execute a plugin command in project context | JSON body -> response |

### Plugin public controller (instance-scoped, no auth required for some)

| Generated method | What it does | Request/Response |
|---|---|---|
| `GetPluginsUsingGET1WithResponse(ctx)` | List plugins (public endpoint) | -> plugin list |
| `GetPublicFileUsingGETWithResponse(ctx, pluginName, name)` | Get a public file from a plugin (icons, etc.) | -> binary content |
| `ExecutePublicPluginCommandUsingPUTWithResponse(ctx, pluginName, command, body)` | Execute a public plugin command | JSON body -> response |

### Onboarding controller (instance-scoped)

| Generated method | What it does | Request/Response |
|---|---|---|
| `OnBoardingUsingGETWithResponse(ctx, OnBoardingUsingGETParams)` | Get onboarding status/page | -> onboarding response |

`OnBoardingUsingGETParams` may have a `Page` parameter or similar.

### Demo data controller (project-scoped)

| Generated method | What it does | Request/Response |
|---|---|---|
| `GenerateUsingPOSTWithResponse(ctx, projectName, DemoDataRq)` | Generate demo data in a project | `DemoDataRq` -> `DemoDataRs` |

### Key types

```go
type AnalyticsResource struct {
    Type *string `json:"type,omitempty"`
}

type UpdatePluginStateRQ struct {
    Enabled *bool `json:"enabled,omitempty"`
}

type DemoDataRq struct {
    CreateDashboard *bool `json:"createDashboard,omitempty"`
}

type DemoDataRs struct {
    DashboardId *int64   `json:"dashboardId,omitempty"`
    LaunchIds   *[]int64 `json:"launchIds,omitempty"`
}
```

## Client requirements

### SettingsScope (instance-scoped)

Create `pkg/client/settings.go` with a `SettingsScope` struct:

- **Instance-scoped**: accessible via `client.Settings()`
- Add a `Settings()` method to `Client` in `pkg/client/client.go`
- Struct holds `client *Client`

Methods to implement:

1. **`GetServerSettings(ctx) (*types.AnalyticsResource, error)`** — get current server settings. Note: investigate the actual response type from the generated client — it may be `AnalyticsResource` or a more generic map/struct.
2. **`UpdateAnalyticsSettings(ctx, request) (*types.OperationCompletionRS, error)`** — update analytics settings (use the PUT variant). Check what the request body type is in the generated client.

### PluginScope (instance-scoped)

Create `pkg/client/plugin.go` with a `PluginScope` struct:

- **Instance-scoped**: accessible via `client.Plugins()`
- Add a `Plugins()` method to `Client` in `pkg/client/client.go`
- Struct holds `client *Client`

Methods to implement:

1. **`List(ctx) ([]types.IntegrationTypeResource, error)`** — list all installed plugins. Note: The return type may be `[]IntegrationTypeResource` or a similar plugin descriptor type — check the generated response struct for `GetPluginsUsingGET`.
2. **`UpdateState(ctx, pluginId int64, request UpdatePluginStateRQ) (*types.OperationCompletionRS, error)`** — enable or disable a plugin
3. **`Delete(ctx, pluginId int64) (*types.OperationCompletionRS, error)`** — delete a plugin

Optional (lower priority — these are complex and rarely needed):
4. `Upload(ctx, pluginData io.Reader) error` — upload a new plugin (multipart). This requires multipart encoding and is complex; defer unless straightforward.
5. `ExecuteCommand(ctx, projectName, pluginName, command string, body any) (map[string]any, error)` — execute a plugin command. The response is untyped; return as a generic map.

### PluginPublicScope (instance-scoped)

Create or add to `pkg/client/plugin.go`:

- Accessible via `client.PluginsPublic()` or merge into `PluginScope` with separate method names
- Recommended approach: add public methods to `PluginScope` with `Public` prefix, or create a separate `PluginPublicScope`

Methods to implement:

1. **`ListPublic(ctx) ([]types.IntegrationTypeResource, error)`** — list plugins via the public endpoint (no auth needed for some deployments)
2. **`GetPublicFile(ctx, pluginName, fileName string) ([]byte, error)`** — download a public file from a plugin (e.g., plugin icon). Returns raw bytes, similar to `DownloadFile`.

### OnboardingScope (instance-scoped)

This is a very small surface — consider adding it directly to `Client` or as a minimal scope.

Create `pkg/client/onboarding.go` or add to `settings.go`:

1. **`GetOnboardingStatus(ctx) (any, error)`** — get onboarding status. The response type is likely untyped or a simple struct. Investigate the generated response and use the appropriate type. If it's a generic JSON response, return `map[string]any`.

### DemoDataScope (project-scoped)

Create `pkg/client/demo_data.go` or add to an existing project-scoped file:

- **Project-scoped**: accessible via `client.Project(name).DemoData()`
- Add a `DemoData()` method to `ProjectScope` in `pkg/client/project.go`

Methods to implement:

1. **`Generate(ctx, request DemoDataRq) (*types.DemoDataRs, error)`** — generate demo data in the project. Returns IDs of created launches and dashboard.

## Client tests

### `pkg/client/settings_test.go` (new, `package client_test`)

- **GetServerSettings**: success (200), unauthorized (401)
- **UpdateAnalyticsSettings**: success (200), unauthorized (401)

### `pkg/client/plugin_test.go` (new, `package client_test`)

- **List**: success (200 with plugin list), unauthorized (401)
- **ListPublic**: success (200)
- **UpdateState**: success (200), not found (404 — invalid plugin ID)
- **Delete**: success (200), not found (404)
- **GetPublicFile**: success (200 with binary content), not found (404)

### `pkg/client/demo_data_test.go` (new, `package client_test`)

- **Generate**: success (200 with dashboard and launch IDs), unauthorized (401)

### `export_test.go`

Add test-only constructors for all new scopes.

## Integration tests

### `test/integration/client_settings_test.go` (new)

- Use `Label("client", "settings")`
- **GetServerSettings**: verify returns without error, inspect basic structure

### `test/integration/client_plugin_test.go` (new)

- Use `Label("client", "plugins")`
- **List**: verify returns without error (default Report Portal ships with built-in plugins like `emailServer`, `rally`, `jira`, etc.)
- **ListPublic**: same verification
- Note: Don't test upload/delete of plugins in integration tests — it could destabilize the test instance

### `test/integration/client_demo_data_test.go` (new)

- Use `Label("client", "demo-data")`
- **Generate**: create a dedicated test project, generate demo data, verify the response contains launch IDs. Clean up the project afterward.
- Note: Demo data generation creates real launches/items/dashboards, so use an isolated project to avoid polluting the main test project.

### Seeding notes

- These tests mostly don't need seeding — they operate at the instance/system level
- Demo data tests should create and tear down their own project
- Plugin tests should be read-only against the existing plugin set

## Design considerations

- **Admin-only endpoints**: Many of these endpoints require admin privileges. The test stack uses a superadmin user, so integration tests should work. Document in method godoc that admin privileges are required.
- **Untyped responses**: Several endpoints (plugin commands, onboarding) return loosely-typed JSON. Use `map[string]any` or `any` for return types where the generated client doesn't provide a typed response. Consider wrapping in a more specific type if the structure is stable.
- **Plugin upload is multipart**: The plugin upload endpoint accepts multipart form data. This is complex to wrap cleanly. Defer to a follow-up unless there's a clear need.
- **Deprecated controllers**: The roadmap mentions deprecated user endpoints. These are already partially covered by `UsersScope` (Prompt 04). If there are additional deprecated endpoints not yet covered, add them with `// Deprecated:` godoc comments.
- **No CLI for this slice**: These endpoints are explicitly client-only. If any CLI exposure is desired in the future, it would go under `rp admin ...` commands, which is a separate design decision.
- **Clearly mark deprecated endpoints**: If wrapping any deprecated API endpoints, use the `// Deprecated:` Go convention in the method documentation so tools and consumers can plan for removal.

## Definition of done

- [ ] `pkg/client/settings.go` with `SettingsScope` (GetServerSettings, UpdateAnalyticsSettings)
- [ ] `pkg/client/plugin.go` with `PluginScope` (List, ListPublic, UpdateState, Delete, GetPublicFile)
- [ ] `pkg/client/onboarding.go` or equivalent (GetOnboardingStatus)
- [ ] `pkg/client/demo_data.go` with project-scoped `DemoDataScope` (Generate)
- [ ] Scope accessor methods on `Client` and `ProjectScope`
- [ ] Unit tests for all new scopes
- [ ] Integration tests for read-safe operations
- [ ] `code-verifier` subagent says your changes are good to go
