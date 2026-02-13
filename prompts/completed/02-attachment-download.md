# Prompt 02: Attachment Download (Client + CLI)

## Goal

Add file/attachment download support to the Go client and a `rp download` CLI command. This is the second half of Milestone 4 (Logs + Attachments). This prompt depends on Prompt 01 (Log Read Surface) being completed first, since attachments are discovered through log entries.

## Context

In Report Portal, attachments (screenshots, files, etc.) are associated with log entries via the `binaryContent` field on `LogResource`. The `BinaryContent` schema has:

```
BinaryContent:
  contentType: string  (MIME type)
  id: string           (data store ID used to fetch the file)
  thumbnailId: string  (data store ID for thumbnail, if applicable)
```

The file retrieval endpoint is `GET /v1/data/{projectName}/{dataId}`, which returns raw binary content. The generated client method is `GetFileUsingGETWithResponse`.

The `GetFileUsingGETResponse` struct has a `Body []byte` field (the raw file content) and `JSONDefault *ErrorRS` for errors. Note: this is different from most other endpoints — the success response is raw bytes, not a JSON-typed field.

## Client requirements

Create `pkg/client/file.go` (or extend `log.go` if the surface is small enough — use judgment):

- Add a method on `ProjectScope` directly (not a separate scope, since file download is a single endpoint):
  - `DownloadFile(ctx, dataID string) (*FileContent, error)` — downloads file content by data store ID
- `FileContent` struct should contain:
  - `Data []byte` — raw file content
  - `ContentType string` — from the HTTP response `Content-Type` header
  - `ContentLength int64` — from the HTTP response
- The generated response for this endpoint returns raw bytes (`Body []byte`), not a JSON200 field. Handle this differently from the standard `handleResponse` pattern — check `JSONDefault` for errors, but on success extract `Body` and HTTP headers directly.

Optionally, add a convenience method on `LogScope` or as a standalone helper:
- `DownloadAttachment(ctx, log LogResource) (*FileContent, error)` — given a log entry with `BinaryContent`, download the attached file. Return a clear error if `BinaryContent` is nil.

## Client tests

Create or extend `pkg/client/file_test.go` (black-box, `package client_test`):

- `DownloadFile`: success (serve a known byte payload from httptest, verify content and content-type), 404 not found, 401 unauthorized
- If `DownloadAttachment` helper exists: success, nil-binaryContent error
- All tests must use `t.Parallel()`

## CLI command requirements

Add a `rp download` command in `internal/cli/download.go`:

- Register in `internal/cli/root.go` as a top-level command (not under `get`, since `download` is an action, not a query)
- Command shape: `rp download <data-id>`
  - Positional arg: the data store ID (from a log's `binaryContent.id`)
- Flags:
  - `--project` / `-p` — project name override
  - `--output` / `-o` — output file path (if omitted, write to stdout — useful for piping)
  - `--log` — optional: instead of a raw data ID, accept a log ID and automatically resolve its `binaryContent.id`
- Behavior:
  - Fetch the file content via `DownloadFile`
  - If `--output` is specified, write to that file path and print a confirmation message to stderr
  - If `--output` is not specified, write raw bytes to stdout (binary-safe)
  - If `--log` is specified, first fetch the log by ID to resolve the attachment data ID
- Error handling:
  - Clear error if the log has no attachment (`binaryContent` is nil)
  - Standard API error handling for 404, 401, etc.

## Integration tests

Add to `test/integration/`:

- `client_file_test.go` — client-level integration tests:
  - Seed a launch + item with an attachment (this may require creating a log entry with a file upload via the raw API during seeding)
  - `DownloadFile` for a known attachment
  - 404 for a non-existent data ID
- `cli_download_test.go` — CLI integration tests:
  - `rp download <data-id> --output /tmp/test-file` — verify file is written
  - `rp download <data-id>` — verify stdout output
  - Error cases: missing project, non-existent file

Note: Seeding attachments in integration tests is more complex than seeding launches/items. The Report Portal API accepts file uploads via `POST /v1/{projectName}/log` with multipart form data. You will likely need to add a raw HTTP helper for this seeding step, since the generated client may not handle multipart well. Look at how the existing helpers in `helpers_test.go` handle raw HTTP requests for guidance.

## Definition of done

- [ ] `pkg/client/file.go` with `DownloadFile` (and optionally `DownloadAttachment`)
- [ ] `pkg/client/file_test.go` with unit tests
- [ ] `internal/cli/download.go` with `rp download` command
- [ ] `test/integration/client_file_test.go` with real API tests
- [ ] `test/integration/cli_download_test.go` with CLI integration tests
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-integration` passes
