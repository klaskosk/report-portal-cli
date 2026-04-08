# PR 11: File Storage Completion

## Goal

Complete the file storage API surface in `pkg/client` by adding wrappers for current-user photo management, project-scoped user photo retrieval, and CSV-driven attachment cleanup. After this PR, every synchronous `/v1/data/...` endpoint will be accessible through the high-level client.

## Context

File storage currently has a single method: `ProjectScope.DownloadFile` in `pkg/client/file.go`, which wraps `GET /v1/data/{projectName}/{dataId}` for downloading attachments by data store ID. The generated client already supports the remaining five operations, but they have no high-level wrappers.

### Already implemented (do not modify)

| Method | Scope | Generated operation | Path |
|--------|-------|-------------------|------|
| `DownloadFile` | `ProjectScope` | `GetFileWithResponse` | `GET /v1/data/{projectName}/{dataId}` |

### Methods to add

| Proposed method | Scope | Generated operation | HTTP | Path | Notes |
|----------------|-------|-------------------|------|------|-------|
| `GetMyPhoto` | instance-level | `GetMyPhotoWithResponse` | GET | `/v1/data/photo` | Returns raw image bytes. Optional `loadThumbnail` query param. Non-JSON response pattern. |
| `UploadPhoto` | instance-level | `UploadPhotoWithBodyWithResponse` | POST | `/v1/data/photo` | Multipart `file` upload. Returns `OperationCompletionRS`. |
| `DeletePhoto` | instance-level | `DeletePhotoWithResponse` | DELETE | `/v1/data/photo` | Deletes the current user's photo. Returns `OperationCompletionRS`. |
| `GetUserPhoto` | `ProjectScope` | `GetUserPhotoWithResponse` | GET | `/v1/data/{projectName}/userphoto` | Returns raw image bytes. Required `login` query param, optional `loadThumbnail`. Non-JSON response pattern. |
| `CleanAttachments` | instance-level | `RemoveAttachmentsByCsvWithBodyWithResponse` | POST | `/v1/data/clean` | Multipart CSV file upload. Returns `OperationCompletionRS`. |

## Implementation plan

### 1) Decide on scope placement

The current-user photo endpoints (`/v1/data/photo`) and the CSV cleanup endpoint (`/v1/data/clean`) are **not** project-scoped. They need an instance-level home. Two reasonable options:

**Option A (preferred)**: Create a new `FileScope` or `FileStorageScope` struct with `client *Client` for the three instance-level endpoints. Wire it as `Client.Files() *FileStorageScope` in `client.go`. Keep `GetUserPhoto` on `ProjectScope` since it's project-scoped (similar to how `DownloadFile` already lives there).

**Option B**: Add methods directly to `Client` without a dedicated scope. This is simpler but inconsistent with the scope-based API pattern used everywhere else.

Choose **Option A** unless there is a compelling reason not to.

### 2) Discover response types

Before writing code, inspect `pkg/internal/api/api.gen.go` for each operation:

- `GetMyPhotoResponse` / `GetMyPhotoParams` — likely has `loadThumbnail` bool.
- `UploadPhotoResponse` — check for `WithBody` variant; the generated API uses `UploadPhotoWithBodyWithResponse` (not a typed multipart struct).
- `DeletePhotoResponse` — simple DELETE, check JSON fields.
- `GetUserPhotoResponse` / `GetUserPhotoParams` — has `login` (required string) and `loadThumbnail` (optional bool).
- `RemoveAttachmentsByCsvResponse` — multipart upload with `WithBody` variant.

### 3) Handle non-JSON responses

`GetMyPhoto` and `GetUserPhoto` return raw image bytes, not JSON. Follow the same pattern as `DownloadFile` and `GetPublicFile`:

1. Check `response.JSONDefault` for API error body.
2. Check status code manually (`< 200 || >= 300`).
3. Return a `*FileContent` struct with `Data`, `ContentType`, and `ContentLength`.

Reuse the existing `FileContent` struct from `file.go`.

### 4) Handle multipart uploads

`UploadPhoto` and `CleanAttachments` require multipart form data. The generated client exposes `WithBody` variants that accept `contentType string` and `body io.Reader`. The high-level wrapper should accept an `io.Reader` and filename, build the multipart body internally (using `mime/multipart`), and pass it to the generated client.

Follow the existing multipart pattern from `LogScope.CreateBatch` in `log.go`:

```go
var buf bytes.Buffer
writer := multipart.NewWriter(&buf)
part, err := writer.CreateFormFile("file", filename)
// ... copy data to part ...
writer.Close()
// call raw.UploadPhotoWithBodyWithResponse(ctx, writer.FormDataContentType(), &buf)
```

The method signatures should be user-friendly:

```go
func (s *FileStorageScope) UploadPhoto(ctx context.Context, filename string, data io.Reader) (*types.OperationCompletionRS, error)
func (s *FileStorageScope) CleanAttachments(ctx context.Context, filename string, csvData io.Reader) (*types.OperationCompletionRS, error)
```

### 5) Add `GetUserPhoto` to `ProjectScope`

Since `GetUserPhoto` is project-scoped (`/v1/data/{projectName}/userphoto`), add it to `file.go` alongside the existing `DownloadFile`. It takes a login string and returns `*FileContent`.

```go
func (p *ProjectScope) GetUserPhoto(ctx context.Context, login string, options ...GetUserPhotoOption) (*FileContent, error)
```

Define a `GetUserPhotoOption` for the optional `loadThumbnail` parameter.

### 6) Wire the new scope into `client.go`

```go
// Files returns a FileStorageScope for instance-level file storage operations.
func (c *Client) Files() *FileStorageScope {
    return &FileStorageScope{client: c}
}
```

## Test requirements

### Unit tests

Add tests in `pkg/client/file_test.go` using the black-box pattern (`package client_test`). Every test must call `t.Parallel()`.

For each new method:

1. **Success test**: Mock server returning appropriate response (raw bytes for photo endpoints, JSON for mutating endpoints). Assert return values, HTTP method, and path.
2. **Error test**: At least one error case per method.

#### Photo download tests

For `GetMyPhoto` and `GetUserPhoto`, return mock image bytes and verify the returned `FileContent` has correct `Data`, `ContentType`, and `ContentLength`.

#### Multipart upload tests

For `UploadPhoto` and `CleanAttachments`, verify that:
- The request Content-Type is `multipart/form-data` with a boundary.
- The `file` form field exists with the expected filename.
- The response body is parsed correctly.

Follow the pattern from `TestLogScope_CreateBatch_WithAttachment` in `log_test.go`.

#### GetUserPhoto

Verify the `login` query parameter is sent. If a `loadThumbnail` option is provided, verify it appears in the query string.

### Test helpers

Use `newTestClient`, `assertBearerToken`, `assertMethod`, `assertPath`, `jsonResponse`. For the new instance-level scope, access via `c.Files()`. For `GetUserPhoto`, access via `c.Project("proj").GetUserPhoto(...)`.

## Files to create

| File | Content |
|------|---------|
| (none — use existing `file.go` and `file_test.go`) | |

## Files to modify

| File | Change |
|------|--------|
| `pkg/client/file.go` | Add `FileStorageScope` struct, its 3 methods (`GetMyPhoto`, `UploadPhoto`, `DeletePhoto`), `CleanAttachments` method, and `GetUserPhoto` on `ProjectScope` |
| `pkg/client/file_test.go` | Add corresponding unit tests |
| `pkg/client/client.go` | Add `Files() *FileStorageScope` factory method |

## Files NOT to modify

- `pkg/internal/api/` — generated code, do not hand-edit
- `pkg/types/` — generated types, do not hand-edit
- `pkg/client/log.go` — the multipart pattern here is a reference, not a modification target

## Validation commands

Run before finalizing:

```shell
make lint
make test
```

## Definition of done

- [ ] `FileStorageScope` struct created with `GetMyPhoto`, `UploadPhoto`, `DeletePhoto`, and `CleanAttachments` methods
- [ ] `ProjectScope.GetUserPhoto` method added for project-scoped user photo retrieval
- [ ] `Client.Files()` factory method added to `client.go`
- [ ] Photo download methods return `*FileContent` using the non-JSON response pattern
- [ ] Multipart upload methods accept `io.Reader` and build the form body internally
- [ ] Functional options defined where needed (e.g., `loadThumbnail`)
- [ ] Unit tests cover success and error cases for every method
- [ ] Multipart tests verify Content-Type and form field structure
- [ ] All error messages are lowercase with no trailing punctuation
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
