# Prompt 55: Final Verification & Coverage Report

## Goal

Run a full verification pass across the entire repository. Generate final coverage reports. Address any minor remaining gaps to ensure integration coverage exceeds 80% and unit test coverage exceeds 95%. Verify there is no test pollution between parallel integration tests.

## Context

This is PR 14 of the test coverage improvement plan (Phase 5). This is the final fan-in step — all previous PRs (1-13) must be merged first.

## Prerequisites

- All PRs from Phases 1-4 (prompts 42-54) are merged.

## Implementation plan

### 1) Run full verification

```bash
make lint
make test
make test-integration
```

All three commands must pass cleanly.

### 2) Generate unit test coverage report

```bash
cd pkg && go test -coverprofile=coverage.out -covermode=atomic ./client/...
go tool cover -func=coverage.out | tail -1
```

Target: > 95% unit test coverage for `pkg/client`.

If coverage is below 95%, identify the uncovered lines:

```bash
go tool cover -html=coverage.out -o coverage.html
```

Open the HTML report and find the uncovered code paths. Add targeted tests for any significant gaps.

### 3) Generate integration test coverage report

```bash
make test-integration
```

The integration test suite collects coverage via `GOCOVERDIR`. After the run completes, merge the coverage data:

```bash
go tool covdata textfmt -i=<coverage_dir> -o integration_coverage.out
go tool cover -func=integration_coverage.out | grep 'pkg/client'
```

Target: > 80% integration test coverage for `pkg/client`.

If coverage is below 80%, identify the gaps and determine if they can be addressed with additional integration tests or if they represent code paths that are genuinely untestable in integration (e.g., error handling for network failures).

### 4) Check for test pollution

Run the integration test suite multiple times to verify no test pollution exists:

```bash
make test-integration
make test-integration
```

Both runs should pass. If the second run fails, it indicates test pollution — leftover state from the first run is interfering.

Also run with randomized test order (Ginkgo supports `--randomize-all`):

```bash
cd test/integration && go test -tags integration -v --randomize-all ./...
```

If tests fail with randomized ordering, there are implicit dependencies between tests that need to be resolved.

### 5) Verify parallel test isolation

Check that new integration tests create unique project names (with timestamps or random suffixes) and clean up after themselves:

```bash
rg 'fmt.Sprintf.*test_' test/integration/
```

Every project name should include a unique identifier (e.g., `time.Now().UnixNano()`) to prevent collisions when tests run in parallel.

### 6) Address remaining gaps

If either coverage target is not met:

1. List the uncovered functions/lines from the coverage report
2. Prioritize by impact — focus on public API methods first
3. Add targeted tests — either unit tests (for error handling paths) or integration tests (for happy paths)
4. Re-run the coverage report to verify improvement

### 7) Document final coverage

Update or create a coverage summary. The output of `go tool cover -func` for both unit and integration tests should be captured.

## Verification checklist

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] `make test-integration` passes
- [ ] Unit test coverage > 95% for `pkg/client`
- [ ] Integration test coverage > 80% for `pkg/client`
- [ ] No test pollution (repeated runs pass)
- [ ] Randomized test ordering passes
- [ ] All new test projects use unique names
- [ ] All new tests clean up after themselves

## Files touched

| File | Change |
|------|--------|
| Various `pkg/client/*_test.go` | Minor gap-filling tests if needed |
| Various `test/integration/*_test.go` | Minor gap-filling tests if needed |
| `TEST_COVERAGE_PLAN.md` | Remove or mark as completed |

## What this PR does NOT change

- No `pkg/client` production code modified
- No architectural changes
- No new public API
