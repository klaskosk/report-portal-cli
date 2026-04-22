---
name: code-review
description: Review local changes for correctness, simplicity, and conventions. Do not use for remote pull request reviews.
---

# Code Review

Review local changes on the current branch with a **correctness-first, simplicity-first** mindset. This is a modern Go codebase, priorizing idiomatic and effective Go. Simplicity and readability are paramount. The best solution frequently involves less code.

## Safety rails (critical)

- Do not use `gh`.
- Do not use `git worktree`.
- Prefer read-only inspection: `git status`, `git diff`, `git show`, `git log`.

## Scope (pick the smallest diff)

- Unstaged: `git diff`
- Staged: `git diff --staged`
- Latest commit: `git show --patch --stat HEAD`
- Range vs base: `git diff <base>...HEAD` (and `git log --oneline <base>..HEAD`)

State what you reviewed (unstaged/staged/commit(s)/range).

If a commit message references a prompt, read the corresponding file in `prompts/` (or `prompts/completed/`) and review against its definition of done.

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

1. If requested, run `make lint`, `make test`, and `make test-integration` to verify that PR checks will pass.
2. If a prompt is mentioned in the commit message, verify that the changes meet the definition of done for that prompt in the `prompts/` directory.
3. Thoroughly perform a code review of the changes. Review changes in the PR and ensure they are in line with the priorities and critical checks above.
4. Give a report of your review following the output format below.

## Output Format

**Summary**: 2-4 bullets on changes and risks.
**Validation**: Explicitly state what was run (`make lint`, `make test`, etc.) and results.

**Findings** (Enumerated and grouped by: Blocker, Major, Minor, Nit):

- **[Severity] Location**: File/Symbol
  - **Issue**: Why it matters (correctness, simplicity, convention).
  - **Fix**: Specific change or simpler alternative.

**Verdict**: Approve / Request changes.
