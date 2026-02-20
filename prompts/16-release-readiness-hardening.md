# Prompt 16: Release Readiness Hardening (Post-Gate Stability)

## Goal

Harden release readiness for automation users by expanding compatibility guidance and adding integration assertions for output stability across high-value workflows.

This PR should reduce upgrade/regression risk immediately before larger CLI expansion.

## Context

After coverage closure and pre-CLI contract lock, the next risk is release quality: behavior drift across Report Portal versions, output regressions for scripts, and gaps in release verification. This slice focuses on hardening, not large feature additions.

## Scope and exclusions

### In scope

- Compatibility notes for known version-sensitive behaviors.
- Additional integration assertions for stable JSON/error contracts.
- Release checklist improvements for command/output contract verification.
- Documentation updates that help operators upgrade safely.

### Out of scope

- Broad new client API coverage work.
- Broad new CLI command families (belongs to Prompt 17+).

## Required deliverables

### 1) Compatibility notes

Create/update a compatibility document (for example `docs/compatibility.md`) that includes:

- supported baseline target (`24.1` / API `v5.11`)
- known behavior differences observed in practice (where relevant)
- fallback/handling guidance when an endpoint behaves differently

Keep notes practical and tied to tested workflows.

### 2) Output stability integration assertions

Extend integration tests for automation-facing commands and client flows to assert:

- required JSON keys remain present and unchanged in name
- machine-readable error semantics remain stable
- key workflow outputs remain script-friendly

Focus on commands already in the CLI baseline.

### 3) Release verification checklist

Add or update a release checklist document (for example `docs/release-checklist.md`) covering:

- unit + integration test pass criteria
- lint pass criteria
- contract/compatibility docs review steps
- command/docs/shell-completion verification (if supported in repo)

## Testing requirements

### Integration assertions to strengthen

Prioritize these existing command families:

- launch/item/log/project reads
- attachment download flow
- identity/discovery flow (`whoami`, project lookup)

For each family, assert structure-level stability rather than brittle full-payload snapshots.

### Client-level assertions

Add targeted integration or unit checks for:

- representative error predicate behavior
- stable response handling where multiple server behaviors are possible

## Design considerations

- Keep compatibility notes evidence-based and concise; avoid speculative entries.
- Avoid overfitting assertions to seed data values; assert shapes and invariants.
- Make checklist items binary and repeatable so release readiness can be audited quickly.
- If compatibility uncertainty remains, document residual risk explicitly.

## Validation commands

Run before finalizing:

- `go test ./...`
- `golangci-lint-v2 run --fix`
- `go test -tags=integration ./test/integration/...` (or project-standard integration target)

## Definition of done

- [ ] Compatibility notes are added/updated with actionable version guidance
- [ ] Integration assertions are expanded for output/error stability in core workflows
- [ ] Release checklist is added/updated with explicit verification steps
- [ ] Documentation reflects current supported behavior and caveats
- [ ] Lint and tests pass locally
- [ ] `code-verifier` subagent says your changes are good to go
- [ ] This file moved to the `completed` directory
