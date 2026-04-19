# Prompt 41: Execute Cleanup

## Goal

Remove the now-dead `handleResponse`, `handleVoidResponse`, and `apiResponse` code from `pkg/client/response.go` and their associated test fixtures. This is a pure deletion PR — no behavioral changes.

## Prerequisites

- Prompts 35-40 are all merged: no remaining callers of `handleResponse` or `handleVoidResponse` outside of `response.go` and `response_test.go`.

## Context

This is PR 9 of 9 in the codegen boilerplate reduction series. See prompt 33 for the full series overview.

## Pre-flight check

Before making any changes, verify there are no remaining callers:

```bash
rg 'handleResponse\(' pkg/client/ --glob '!response.go' --glob '!response_test.go'
rg 'handleVoidResponse\(' pkg/client/ --glob '!response.go' --glob '!response_test.go'
```

Both commands should return zero results. If any callers remain, do NOT proceed — go back and migrate them first.

## Implementation plan

### 1) Remove from `pkg/client/response.go`

Delete the following:

- The `apiResponse` interface constraint (replaced by `typedResponse` and `voidResponse`)
- The `handleResponse` function
- The `handleVoidResponse` function

Keep the following (still in use):

- `handleV2Response` — still used by 3 v2 call sites (`launch.go` GetByUUID/Merge, `item.go` GetByUUID)
- `legacyPayloadPresent` — used by `handleV2Response`
- `operationLogger` — used by all project-scoped methods
- `typedResponse[T]` interface — used by `execute`
- `voidResponse` interface — used by `executeVoid`
- `execute` function
- `executeVoid` function

### 2) Remove from `pkg/client/response_test.go`

Delete the following test fixtures and tests:

- `fakeResponse` struct and its `StatusCode()` method
- `TestHandleResponse_Success`
- `TestHandleResponse_Error`
- `TestHandleResponse_NilSuccess`
- `TestHandleResponse_NonJSONError`
- `TestHandleResponse_NonJSONErrorPreservesBody`
- `TestHandleResponse_2xxWithNilJSON`

Keep the following (still testing live code):

- `fakeV2Response` struct — used by `TestHandleV2Response`
- `TestHandleV2Response` — tests `handleV2Response` which is still in use
- `ptr` helper — used by `TestHandleV2Response`
- All `execute` / `executeVoid` tests added in prompt 34
- The generic `fakeTypedResponse[T]` and `fakeVoidResponse` test fakes from prompt 34
- Any compile-time assertions added in prompt 34

### 3) Check for `apiResponse` references

The `apiResponse` interface was only used as a type constraint in the old `handleResponse` and `handleVoidResponse` signatures. Verify no other code references it:

```bash
rg 'apiResponse' pkg/client/
```

Should return zero results after the deletions.

## Verification

- `make lint` — clean
- `make test` — all unit tests pass
- `make test-integration` — all integration tests pass (final end-to-end validation of the entire series)
- `go build ./...` — compiles cleanly

## Files touched

| File | Change |
|---|---|
| `pkg/client/response.go` | Remove `apiResponse`, `handleResponse`, `handleVoidResponse` |
| `pkg/client/response_test.go` | Remove `fakeResponse`, 6 `TestHandleResponse_*` tests |
