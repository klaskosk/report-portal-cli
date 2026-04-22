# Prompt 51: Integration Tests for Files

## Goal

Expand integration tests for file-related operations: user photo uploads via `FileStorageScope` and project file attachments via `ProjectScope`. The existing `client_file_test.go` (65 lines) only covers `DownloadFile` (success and not-found). This PR adds tests for user photo management and validates the full file upload/download lifecycle.

## Context

This is PR 10 of the test coverage improvement plan (Phase 3). It runs in parallel with PRs 7, 8, and 9. Phase 2 must be merged first.

## Prerequisites

- Phases 1-2 (PRs 1-6) are merged.

## Scopes under test

### `FileStorageScope` — `Client.Files()`

| Method | Signature | Current coverage |
|--------|-----------|-----------------|
| `GetMyPhoto` | `(ctx, options ...GetMyPhotoOption) (*FileContent, error)` | **Not tested** |
| `UploadPhoto` | `(ctx, filename string, data io.Reader) (*types.OperationCompletionRS, error)` | **Not tested** |
| `DeletePhoto` | `(ctx) (*types.OperationCompletionRS, error)` | **Not tested** |
| `CleanAttachments` | `(ctx, filename string, csvData io.Reader) (*types.OperationCompletionRS, error)` | **Not tested** |

### `ProjectScope` file methods

| Method | Signature | Current coverage |
|--------|-----------|-----------------|
| `GetUserPhoto` | `(ctx, login string, options ...GetUserPhotoOption) (*FileContent, error)` | **Not tested** |
| `DownloadFile` | `(ctx, dataID int64) (*FileContent, error)` | Tested |

### Options

| Option | Applies to |
|--------|-----------|
| `WithPhotoThumbnail(load bool)` | `GetMyPhoto` |
| `WithUserPhotoThumbnail(load bool)` | `GetUserPhoto` |

## Implementation plan

### 1) Expand `test/integration/client_file_test.go`

Keep the existing `DownloadFile` tests and add new contexts:

```
Describe("Client: Files", Label("client", "files"))
  [existing DownloadFile context remains unchanged]

  Context("User Photo Lifecycle")
    It("uploads, retrieves, and deletes a user photo")
      By("uploading a photo")
      - Create a small PNG image in memory (e.g., 1x1 pixel)
      - Call c.Files().UploadPhoto(ctx, "test.png", imageReader)
      - Assert no error

      By("retrieving the photo")
      - Call c.Files().GetMyPhoto(ctx)
      - Assert no error
      - Assert returned FileContent is non-nil
      - Assert content type is an image type
      - Assert body is non-empty

      By("retrieving the photo thumbnail")
      - Call c.Files().GetMyPhoto(ctx, client.WithPhotoThumbnail(true))
      - Assert no error

      By("deleting the photo")
      - Call c.Files().DeletePhoto(ctx)
      - Assert no error

  Context("Get User Photo")
    It("retrieves a user photo by login")
      By("uploading a photo for the current user first")
      By("retrieving via GetUserPhoto")
      - Call c.Project(projectName).GetUserPhoto(ctx, "superadmin")
      - Assert no error or handle gracefully if no photo exists

    It("handles missing user photo gracefully")
      - Call c.Project(projectName).GetUserPhoto(ctx, "nonexistent_user_xyz")
      - Assert appropriate error
```

### 2) Create a minimal test image

For the upload test, create a minimal valid PNG in memory:

```go
import (
    "bytes"
    "image"
    "image/color"
    "image/png"
)

func createTestPNG() *bytes.Reader {
    img := image.NewRGBA(image.Rect(0, 0, 1, 1))
    img.Set(0, 0, color.RGBA{R: 255, A: 255})
    var buf bytes.Buffer
    _ = png.Encode(&buf, img)
    return bytes.NewReader(buf.Bytes())
}
```

### 3) Read `FileContent` type

Before implementing, read the `FileContent` type definition in `pkg/client/file.go` to understand its fields (likely `ContentType string`, `Body []byte` or `Body io.ReadCloser`). Adjust assertions accordingly.

### 4) Consider `CleanAttachments`

The `CleanAttachments` method takes a CSV file listing attachment IDs to clean. This is an admin-level housekeeping operation. Test it if feasible:

- Upload an attachment via log creation (use the seeded test data)
- Create a CSV with the attachment's data ID
- Call `CleanAttachments`
- Verify the attachment is no longer downloadable

If this is too complex or disruptive for the test environment, skip it and document why.

### 5) Follow existing test conventions

- The existing `client_file_test.go` uses `BeforeEach` with project creation and test data seeding. Follow the same pattern for new tests that need a project context.
- `NodeTimeout` on setup/teardown
- Labels: `Label("client", "files")`
- Use `DeferCleanup` for photo deletion to avoid leaking test photos

### 6) Run tests

```bash
make test-integration
make lint
```

## Files touched

| File | Change |
|------|--------|
| `test/integration/client_file_test.go` | Expand with photo upload/download/delete lifecycle and user photo tests |

## What this PR does NOT change

- No `pkg/client` production code modified
- No unit tests modified
- No other integration test files modified
