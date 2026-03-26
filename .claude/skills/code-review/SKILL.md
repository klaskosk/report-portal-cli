---
name: code-review
description: Review pull requests, branches, or local diffs for correctness, simplicity, and conventions.
---

# Code Review

Review changes with a **correctness-first, simplicity-first** mindset. This is a modern Go codebase, priorizing idiomatic and effective Go. Simplicity and readability are paramount.

## Priorities

1. **Correctness**: Does it actually work? Are edge cases handled?
2. **Simplicity**: Is this the simplest way to achieve the goal?
3. **Conventions**: Does it follow `CLAUDE.md` and Go idioms?
4. **Verification**: Do tests and linters pass?

## Critical Checks

- **Architecture**: `pkg/` (reusable client) vs `internal/cli/` (Cobra/Viper). No direct exposure of `pkg/internal/api/`.
- **Client API**:
  - Use predicate helpers (`IsNotFound`) over type assertions.
  - Use functional options (`WithLaunchName`) for lists.
  - Check `response.JSONDefault` before success fields.
  - Use `types.EpochMillis` for time.
- **CLI**:
  - Nouns match: `launch`, `item`, `log`, `project`.
  - JSON output fields match API types exactly.
  - Table headers are uppercase (`ID`, `STATUS`).
  - Errors: lowercase, no trailing punctuation, wrapped with `%w`.
- **Testing**:
  - TDD: Tests exist and cover behavior.
  - Unit: `t.Parallel()`, `httptest`, black-box where possible.
  - Integration: `//go:build integration`, Ginkgo/Gomega.
  - Table-driven: `DescribeTable` (Ginkgo) or `[]struct` (std lib).

## Verification Steps

1. Use the `gh` CLI to check out the pull request branch.
2. `make lint` (fixes formatting too)
3. `make test`
4. `make test-integration` (skip if Docker unavailable, but note it)

## Output Format

**Summary**: 2-4 bullets on changes and risks.
**Validation**: Explicitly state what was run (`make lint`, `make test`, etc.) and results.

**Findings** (Enumerated and grouped by: Blocker, Major, Minor, Nit):

- **[Severity] Location**: File/Symbol
  - **Issue**: Why it matters (correctness, simplicity, convention).
  - **Fix**: Specific change or simpler alternative.

**Verdict**: Approve / Approve with nits / Request changes.
