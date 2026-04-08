# Prompt 34: Execute Wrappers

## Goal

Define `typedResponse[T]` and `voidResponse` interfaces in `pkg/client` and implement `execute[T]` and `executeVoid` helper functions that consume them. Write thorough unit tests. This PR is purely additive — no existing wrapper methods change.

## Prerequisites

- Prompt 33 is merged: generated response structs have `GetBody()`, `GetJSONDefault()`, and (where applicable) `GetSuccess()` methods.

## Context

This is PR 2 of 9 in the codegen boilerplate reduction series. See prompt 33 for the full series overview.

## Implementation plan

### 1) Define interfaces in `pkg/client/response.go`

Add alongside the existing `apiResponse` constraint (which remains for now):

```go
// typedResponse is implemented by generated response structs that have a
// single 2xx success field. The codegen template generates these methods.
type typedResponse[T any] interface {
    StatusCode() int
    GetBody() []byte
    GetJSONDefault() *types.ErrorRS
    GetSuccess() *T
}

// voidResponse is implemented by generated response structs for endpoints
// that return no success body (e.g. DELETE with 204 No Content).
type voidResponse interface {
    StatusCode() int
    GetBody() []byte
    GetJSONDefault() *types.ErrorRS
}
```

Both interfaces are unexported — they are internal implementation details.

### 2) Implement `execute` in `pkg/client/response.go`

```go
func execute[T any](
    ctx context.Context,
    logger *slog.Logger,
    opName string,
    resp typedResponse[T],
    err error,
) (*T, error) {
    if err != nil {
        return nil, fmt.Errorf("%s: %w", opName, err)
    }

    logger.DebugContext(ctx, "API response", "status", resp.StatusCode())

    if jsonDefault := resp.GetJSONDefault(); jsonDefault != nil {
        return nil, fmt.Errorf("%s: %w", opName, newAPIError(resp.StatusCode(), jsonDefault))
    }

    code := resp.StatusCode()
    if code < 200 || code >= 300 {
        return nil, fmt.Errorf("%s: %w", opName, newUnexpectedStatusError(code, resp.GetBody()))
    }

    success := resp.GetSuccess()
    if success == nil {
        return nil, fmt.Errorf(
            "%s: API returned success status (%d) but no valid response body was parsed",
            opName, code,
        )
    }

    return success, nil
}
```

### 3) Implement `executeVoid` in `pkg/client/response.go`

```go
func executeVoid(
    ctx context.Context,
    logger *slog.Logger,
    opName string,
    resp voidResponse,
    err error,
) error {
    if err != nil {
        return fmt.Errorf("%s: %w", opName, err)
    }

    logger.DebugContext(ctx, "API response", "status", resp.StatusCode())

    if jsonDefault := resp.GetJSONDefault(); jsonDefault != nil {
        return fmt.Errorf("%s: %w", opName, newAPIError(resp.StatusCode(), jsonDefault))
    }

    code := resp.StatusCode()
    if code < 200 || code >= 300 {
        return fmt.Errorf("%s: %w", opName, newUnexpectedStatusError(code, resp.GetBody()))
    }

    return nil
}
```

### 4) Design decision: logging semantics

Do **not** move `logger.InfoContext(ctx, "API request")` into `execute` or `executeVoid`.

Those helpers receive the already-evaluated `(response, err)` pair from the generated client call. If they emitted the request log internally, the `"API request"` line would move from *before* the transport call to *after* it returns, which is a semantic change. Preserve current observability by keeping the request log at each call site and letting `execute` / `executeVoid` own only:

- operation-level error wrapping
- response-status debug logging
- structured/default error handling
- success-body validation

### 5) Design decision: error wrapping

`execute` wraps all error returns with the `opName` prefix (`fmt.Errorf("%s: %w", opName, ...)`). This replaces the pattern where callers wrote:

```go
result, err := handleResponse(...)
if err != nil {
    return nil, fmt.Errorf("some op: %w", err)
}
```

The error chain is identical because `handleResponse` returns bare `*APIError` / `*UnexpectedStatusError` without operation wrapping — the caller's `fmt.Errorf` was the only wrapping layer. With `execute`, that wrapping moves inside. Error predicates (`IsNotFound`, `IsUnauthorized`, etc.) continue to work unchanged through `errors.As`.

### 6) Write unit tests in `pkg/client/response_test.go`

Create test fakes implementing the new interfaces. Cover the same cases as the existing `handleResponse` / `handleVoidResponse` tests:

#### Test fakes

```go
type fakeTypedResponse[T any] struct {
    statusCode  int
    body        []byte
    jsonDefault *types.ErrorRS
    success     *T
}

func (f fakeTypedResponse[T]) StatusCode() int                { return f.statusCode }
func (f fakeTypedResponse[T]) GetBody() []byte                { return f.body }
func (f fakeTypedResponse[T]) GetJSONDefault() *types.ErrorRS { return f.jsonDefault }
func (f fakeTypedResponse[T]) GetSuccess() *T                 { return f.success }

type fakeVoidResponse struct {
    statusCode  int
    body        []byte
    jsonDefault *types.ErrorRS
}

func (f fakeVoidResponse) StatusCode() int                { return f.statusCode }
func (f fakeVoidResponse) GetBody() []byte                { return f.body }
func (f fakeVoidResponse) GetJSONDefault() *types.ErrorRS { return f.jsonDefault }
```

#### Test cases for `execute`

| Test name | Scenario |
|---|---|
| `TestExecute_Success` | 200 with valid success pointer |
| `TestExecute_TransportError` | Non-nil err argument (network failure) |
| `TestExecute_APIError` | JSONDefault populated (structured error) |
| `TestExecute_NonJSONError` | Non-2xx without JSONDefault (proxy/HTML) |
| `TestExecute_NonJSONErrorPreservesBody` | Body accessible via `BodyOf` |
| `TestExecute_NilSuccess` | 200 with nil success (parse failure) |
| `TestExecute_ErrorWrapsOpName` | Error message starts with opName |
| `TestExecute_ErrorPredicates` | `IsNotFound`, `IsUnauthorized` work through wrapping |

#### Test cases for `executeVoid`

| Test name | Scenario |
|---|---|
| `TestExecuteVoid_Success` | 204 No Content |
| `TestExecuteVoid_TransportError` | Non-nil err argument |
| `TestExecuteVoid_APIError` | JSONDefault populated |
| `TestExecuteVoid_NonJSONError` | Non-2xx without JSONDefault |

These tests use `t.Parallel()`, table-driven style with `t.Run` where appropriate, and follow the existing `response_test.go` conventions.

### 7) Add compile-time spot checks

Add a few compile-time interface assertions near the tests or in a test-only file so the new codegen contract is pinned down with real generated types:

```go
var _ typedResponse[types.GroupPage] = api.GetGroupsResponse{}
var _ typedResponse[types.StartLaunchRS] = api.StartLaunch1Response{}
var _ typedResponse[[]string] = api.GetAllLaunchNamesResponse{}
var _ voidResponse = api.DeleteGroupResponse{}
```

The `[]string` assertion pins the contract for slice/map response types, where `GetSuccess()` returns `*[]string` rather than `*SomeStruct`. Keep this list short. The goal is to catch drift between the custom template and `execute`, not to exhaustively enumerate every generated response type.

### 8) Verify

- `make lint` — clean
- `make test` — all new tests pass, all existing tests still pass

## What this PR does NOT change

- No existing wrapper methods are modified
- `handleResponse`, `handleVoidResponse`, `handleV2Response` remain unchanged
- The `apiResponse` constraint remains (removed in prompt 41)
- No public API changes

## Files touched

| File | Change |
|---|---|
| `pkg/client/response.go` | Add interfaces + `execute` / `executeVoid` |
| `pkg/client/response_test.go` | Add tests for `execute` / `executeVoid` and a few compile-time assertions |
