---
name: code-review
description: Review pull requests, branches, or local diffs for correctness, simplicity, and conventions.
---

# Code Review

Review changes with a **correctness-first, simplicity-first** mindset. This is a modern Go codebase, priorizing idiomatic and effective Go. Simplicity and readability are paramount. The best solution frequently involves less code.

## Priorities

1. **Correctness**: Does it actually work? Are edge cases handled?
2. **Simplicity**: Is this the simplest way to achieve the goal?
3. **Conventions**: Does it follow `CLAUDE.md` and Go idioms?
4. **Verification**: Do tests and linters pass?

## Critical Checks

- **Architecture**:
  - `pkg/` (reusable client) vs `internal/cli/` (Cobra/Viper). CLI must consume `pkg/client`, not reimplement HTTP.
  - No direct exposure of `pkg/internal/api/`. Use `pkg/types/` for shared API shapes.
  - Multi-module layout: Root, `pkg/`, and `test/integration/` have separate `go.mod`s. Keep them consistent.
- **Client API**:
  - Use predicate helpers (`IsNotFound`, `HasStatusCode`) over type assertions.
  - Use functional options (`WithLaunchName`) for lists and optional parameters.
  - Central response handling: HTTP calls should flow through shared helpers (`execute` / `executeVoid`).
- **CLI**:
  - Nouns match: `launch`, `item`, `log`, `project`. Follow `kubectl`-like UX.
  - JSON output fields match API types exactly. Changing JSON shape is a breaking contract.
  - Table vs JSON: Pass the same value to `Print` for JSON/template and typed table rows for table mode.
  - Table headers are uppercase (`ID`, `STATUS`).
  - Errors: lowercase, no trailing punctuation, wrapped with `%w`.
  - Cobra root uses `SilenceUsage: true` and `SilenceErrors: true`. Handlers should use `withPrintedErrors`.
- **Testing**:
  - TDD: Tests exist and cover behavior.
  - Unit: `t.Parallel()`, `httptest` via shared helpers, black-box where possible (`package client_test`).
  - Integration: `//go:build integration`, Ginkgo/Gomega. Testcontainers drives compose.
  - Table-driven: `DescribeTable` (Ginkgo) or `[]struct` (std lib).
  - Never mark tests as pending through Ginkgo. Integration tests must always pass or fail, never be skipped.

## Verification Steps

### For local changes

1. If requested, run `make lint`, `make test`, and `make test-integration` to verify that PR checks will pass.
2. If a prompt is mentioned in the commit message, verify that the changes meet the definition of done for that prompt in the `prompts/` directory.
3. Thoroughly perform a code review of the changes. Review changes in the PR and ensure they are in line with the priorities and critical checks above.
4. Give a report of your review following the output format below.

### For pull requests

1. Use the `gh` CLI to check out the pull request branch. Add the `--detach` flag to avoid checking out the pull request branch. This helps to avoid conflicts with worktrees.
2. Follow the verification steps for local changes.
3. Submit your review using the `gh` CLI and add the appropriate label to the PR based on your verdict.

## Output Format

**Summary**: 2-4 bullets on changes and risks.
**Validation**: Explicitly state what was run (`make lint`, `make test`, etc.) and results.

**Findings** (Enumerated and grouped by: Blocker, Major, Minor, Nit):

- **[Severity] Location**: File/Symbol
  - **Issue**: Why it matters (correctness, simplicity, convention).
  - **Fix**: Specific change or simpler alternative.

**Verdict**: Approve / Approve with nits / Request changes.

## Submitting the PR review

If you are reviewing a pull request, not local changes, please submit your review using the `gh` CLI. You won't be able to approve or request changes, so comment and make clear your verdict in the comment itself.

Additionally, label the PR based on your verdict (and remove any existing `ai/` labels).

- `ai/approved`: The PR is approved.
- `ai/nits`: The PR is approved with nits or minor issues.
- `ai/rejected`: The PR is rejected and changes are requested.

Thank you for your help!
