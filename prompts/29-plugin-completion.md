# PR 12: Plugin Completion

## Goal

Complete the `PluginScope` in `pkg/client/plugin.go` by adding plugin upload, project-scoped plugin command execution, plugin import, and public plugin command execution. After this PR, every synchronous plugin endpoint in the OpenAPI spec will be accessible through the high-level client.

## Context

`PluginScope` already covers plugin listing, state management, deletion, and public file retrieval (5 methods). The generated client exposes four additional operations that have no high-level wrapper. Two of these are project-scoped (command execution and import), and two are instance-level (upload and public command execution).

### Already implemented (do not modify signatures)

| Method | Generated operation | Path |
|--------|-------------------|------|
| `List` | `GetPluginsWithResponse` | `GET /v1/plugin` |
| `ListPublic` | `GetPlugins1WithResponse` | `GET /v1/plugin/public` |
| `UpdateState` | `UpdatePluginStateWithResponse` | `PUT /v1/plugin/{pluginId}` |
| `Delete` | `DeletePluginWithResponse` | `DELETE /v1/plugin/{pluginId}` |
| `GetPublicFile` | `GetPublicFileWithResponse` | `GET /v1/plugin/public/{pluginName}/file/{name}` |

### Methods to add

| Proposed method | Scope | Generated operation | HTTP | Path | Notes |
|----------------|-------|-------------------|------|------|-------|
| `Upload` | `PluginScope` (instance-level) | `UploadPluginWithBodyWithResponse` | POST | `/v1/plugin` | Multipart `file` upload. Admin only. |
| `ExecuteCommand` | project-scoped | `ExecutePluginCommandWithResponse` | PUT | `/v1/plugin/{projectName}/{pluginName}/common/{command}` | JSON body is an arbitrary object (open schema). Project-scoped. |
| `Import` | project-scoped | `ExecuteImportPluginCommandWithBodyWithResponse` | POST | `/v1/plugin/{projectName}/{pluginName}/import` | Multipart upload with `file` (required) and optional `launchImportRq` JSON part. Project-scoped. |
| `ExecutePublicCommand` | `PluginScope` (instance-level) | `ExecutePublicPluginCommandWithResponse` | PUT | `/v1/plugin/public/{pluginName}/{command}` | JSON body is an arbitrary object. No auth required on some deployments. |

## Implementation plan

### 1) Discover response and request types

Before writing any code, inspect `pkg/internal/api/api.gen.go`:

- `UploadPluginWithBodyWithResponse` — multipart upload; check the response struct for `JSON200`/`JSONDefault` fields and what the success type is.
- `ExecutePluginCommandWithResponse` — takes `projectName`, `pluginName`, `command` strings and a JSON request body. The request body type is `ExecutePluginCommandJSONRequestBody` — check what this aliases to (likely an open `map[string]any` or similar).
- `ExecuteImportPluginCommandWithBodyWithResponse` — multipart with `file` and optional `launchImportRq`. Check `ExecuteImportPluginCommandMultipartBody` for the struct definition.
- `ExecutePublicPluginCommandWithResponse` — takes `pluginName`, `command` strings and a JSON body. Check request body type.

### 2) Decide scope placement for project-scoped methods

`ExecuteCommand` and `Import` are project-scoped (`/v1/plugin/{projectName}/...`). Two options:

**Option A (preferred)**: Add these methods directly to `PluginScope` but require `projectName` as a parameter, keeping all plugin operations in one place. This is pragmatic given there are only two project-scoped methods.

**Option B**: Create a small `ProjectPluginScope` accessible via `ProjectScope.Plugins()`, following the nested scope pattern. This is more architecturally consistent but adds a new scope accessor to `project.go`.

Choose whichever approach makes the API most intuitive after examining the method signatures. If Option B is chosen, add `Plugins() *ProjectPluginScope` to `ProjectScope` in `project.go`.

### 3) Add `Upload` to `PluginScope`

This is a multipart file upload. The high-level method should accept an `io.Reader` and filename:

```go
func (s *PluginScope) Upload(ctx context.Context, filename string, data io.Reader) (*types.SomeResponseType, error)
```

Build the multipart body internally (using `mime/multipart`), then call `s.client.raw.UploadPluginWithBodyWithResponse(ctx, writer.FormDataContentType(), &buf)`.

Follow the multipart pattern from `LogScope.CreateBatch` in `log.go`.

### 4) Add `ExecuteCommand`

This takes an arbitrary JSON body. Use `map[string]any` (or whatever the generated request body type aliases to) as the request parameter:

```go
func (s *PluginScope) ExecuteCommand(
    ctx context.Context,
    projectName, pluginName, command string,
    body map[string]any,
) (*types.SomeResponseType, error)
```

Or if using a project-scoped approach:

```go
func (s *ProjectPluginScope) ExecuteCommand(
    ctx context.Context,
    pluginName, command string,
    body map[string]any,
) (*types.SomeResponseType, error)
```

### 5) Add `Import`

This is a multipart upload with a `file` part and an optional `launchImportRq` JSON part. The method signature should be user-friendly:

```go
func (s *PluginScope) Import(
    ctx context.Context,
    projectName, pluginName string,
    filename string,
    data io.Reader,
    launchImportRQ *types.LaunchImportRQ, // nil if not needed
) (*types.SomeResponseType, error)
```

Build the multipart body with the `file` field. If `launchImportRQ` is non-nil, add a JSON part named `launchImportRq`.

### 6) Add `ExecutePublicCommand`

Similar to `ExecuteCommand` but instance-level and potentially unauthenticated:

```go
func (s *PluginScope) ExecutePublicCommand(
    ctx context.Context,
    pluginName, command string,
    body map[string]any,
) (*types.SomeResponseType, error)
```

### 7) Wire scope changes (if needed)

If Option B is chosen for project-scoped methods, add to `project.go`:

```go
func (p *ProjectScope) Plugins() *ProjectPluginScope {
    return &ProjectPluginScope{project: p}
}
```

`PluginScope` is already wired via `Client.Plugins()` in `client.go`, so no change needed there for instance-level additions.

## Test requirements

### Unit tests

Add tests in `pkg/client/plugin_test.go` using the black-box pattern (`package client_test`). Every test must call `t.Parallel()`.

For each new method:

1. **Success test**: Mock the expected HTTP interaction. Assert return values, HTTP method, path, and (for multipart) Content-Type and form field structure.
2. **Error test**: At least one error case per method (unauthorized, not found, etc.).

#### Upload test

Verify multipart Content-Type with boundary, presence of `file` form field with correct filename.

#### ExecuteCommand / ExecutePublicCommand tests

Verify the JSON body is sent correctly. Parse the request body in the mock server and assert key fields.

#### Import test

Verify multipart body includes the `file` part. If `launchImportRQ` is provided, verify it appears as a JSON part in the multipart body.

### Test helpers

Use `newTestClient`, `assertBearerToken`, `assertMethod`, `assertPath`, `jsonResponse`. Access the scope via `c.Plugins()`.

## Files to modify

| File | Change |
|------|--------|
| `pkg/client/plugin.go` | Add `Upload`, `ExecuteCommand`, `Import`, and `ExecutePublicCommand` methods |
| `pkg/client/plugin_test.go` | Add corresponding unit tests |
| `pkg/client/project.go` | Add `Plugins()` accessor **only if** using a project-scoped `ProjectPluginScope` (Option B) |

## Files NOT to modify

- `pkg/client/client.go` — `Plugins()` already exists for instance-level access
- `pkg/internal/api/` — generated code, do not hand-edit
- `pkg/types/` — generated types, do not hand-edit

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] `Upload` method added to `PluginScope` with multipart file handling
- [ ] `ExecuteCommand` method added (instance-level or project-scoped, as decided)
- [ ] `Import` method added with multipart file + optional JSON part
- [ ] `ExecutePublicCommand` method added to `PluginScope`
- [ ] Scope placement for project-scoped methods is documented in code comments
- [ ] Unit tests cover success and error cases for every new method
- [ ] Multipart tests verify Content-Type, form fields, and file data
- [ ] JSON command tests verify request body serialization
- [ ] All error messages are lowercase with no trailing punctuation
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
