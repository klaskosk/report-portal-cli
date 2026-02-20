# Prompt 18: Parallel-Safe Integration Test Isolation

## Goal

Make `test/integration` safe to run concurrently on the same Docker host without cross-run interference, stack teardown races, or output collisions.

This should provide namespace-like isolation semantics similar to Kubernetes namespaces, but for the local Testcontainers + Docker Compose workflow.

## Context

The current integration suite uses a fixed compose stack identifier (`rptest`) and shared discovery/output assumptions. When two integration runs execute at the same time (for example, two local terminals or multiple CI jobs on the same runner), they can target overlapping Docker resources and flaky failures can occur.

The suite already isolates most test data with per-test project names; this prompt focuses on infrastructure-level isolation between separate test processes.

## Scope

### In scope

- Isolate each integration test process with a unique compose stack identity.
- Ensure service discovery/routing cannot mix containers across parallel stacks.
- Remove shared filesystem output collisions for integration coverage artifacts.
- Add explicit verification that concurrent suites can both complete successfully.

### Out of scope

- General Report Portal performance tuning.
- Redesigning test data seeding patterns that are already process-local.

## Required implementation changes

### 1) Unique stack identifier per integration run

Update `test/integration/suite_test.go` so `compose.StackIdentifier(...)` is derived from a unique run ID rather than a constant.

Requirements:

- Deterministic format, safe for Docker resource names.
- Sufficient uniqueness for concurrent runs on one host (for example: timestamp + PID + short random suffix, or UUID-derived token).
- Log the generated stack identifier to test output for debugging.

### 2) Traefik discovery isolation between parallel stacks

Update `test/integration/docker-compose.test.yml` and suite startup wiring so Traefik only discovers containers belonging to the current stack/run, not every container with a generic label.

Possible approach (recommended):

- Replace the static `traefik.expose=true` convention with a stack-specific label value (for example `traefik.expose=${RP_TEST_STACK_ID}`).
- Set the gateway Docker provider constraint to match that exact value.
- Ensure both `api` and `uat` labels use the same run-specific value.

Implementation may use compose env interpolation; wire env values from Go setup accordingly.

### 3) Coverage artifact isolation

Avoid writing to a single fixed `integration.coverprofile` filename when multiple suites run concurrently.

Requirements:

- Use a unique output filename per suite run (for example suffixing with run ID), or write into a unique temp directory and surface the path in logs.
- Preserve existing behavior of generating a usable Go coverage profile.

### 4) Cleanup safety

Keep `stack.Down(...)` teardown behavior, but ensure each run only tears down its own stack resources.

## Testing requirements

### Unit-level

Add targeted tests for any new helper(s) used to generate stack/run IDs (format, allowed characters, and uniqueness expectations).

### Integration-level

Add or update integration harness tests/assertions where practical to validate:

- run-specific stack identifier is used
- stack-scoped discovery configuration is applied

### Concurrency validation (required)

Document and run a reproducible check that starts two integration test invocations at the same time on one machine and verifies both complete successfully.

Example verification pattern:

- start run A in background
- start run B in background
- assert both exit 0 and neither reports compose resource conflicts

## Design considerations

- Keep changes minimal and focused in the integration harness; avoid broad test rewrites.
- Favor explicit run-scoped configuration over implicit Docker defaults.
- Improve failure diagnostics by logging run ID, stack ID, mapped gateway URL, and coverage output path.

## Validation commands

Run before finalizing:

- `go test ./...`
- `golangci-lint-v2 run --fix`
- `go test -tags=integration ./test/integration/...`
- parallel integration run check (two simultaneous invocations on same host)

## Definition of done

- [ ] Compose stack identifier is unique per integration process
- [ ] Traefik discovery is restricted to the current stack/run
- [ ] Coverage profile output no longer collides across concurrent runs
- [ ] Two concurrent integration runs on the same host pass reliably
- [ ] Logging clearly identifies run/stack context for debugging
- [ ] Lint and tests pass
