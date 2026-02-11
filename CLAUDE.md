# report-portal-cli

CLI for [Report Portal](https://reportportal.io/) (v24.1/5.11). Go 1.25, multi-module workspace (`go.work`).

## Architecture

Two main packages, intentionally separated:

- `pkg/` — Standalone, reusable API client module. Public API, meant for consumption by other libraries.
- `internal/cli/` — CLI built on Cobra/Viper. Consumes `pkg/client`.

The API client wraps a generated OpenAPI client (`pkg/internal/api/`) with a scope-based API: `Client` → `ProjectScope` → `LaunchScope`. New API surface area goes through this layering — don't extend or expose the generated client directly.

## Key patterns

- **API errors**: Use predicate functions (`IsNotFound`, `IsUnauthorized`, etc.) rather than type assertions. Follow Dave Cheney's philosophy.
- **List options**: Functional options pattern (`WithName`, `WithStatus`, `WithPageSize`, ...).
- **Response handling**: Always check `response.JSONDefault` (error body) before using success fields.
- **Timestamps**: Use `types.EpochMillis`, not raw ints or `time.Time`.
- **Test style**: Black-box (`package client_test`), `t.Parallel()` everywhere, httptest for unit tests. `export_test.go` exposes test-only constructors.
- **Integration tests**: Ginkgo/Gomega, `//go:build integration`, testcontainers-compose for a real Report Portal stack.
- **Table testing**: In Ginkgo, use `DescribeTable` with `Entry` to test table-driven scenarios. In unit tests, use a `[]struct{}{}` with `t.Run` to test table-driven scenarios.

## Commands

```shell
make lint          # golangci-lint-v2 run --fix (also formats)
make test          # unit tests (root + pkg)
make test-integration  # integration tests (needs Docker)
make generate      # regenerate from OpenAPI specs
make tidy          # go mod tidy across all three modules
```

## Code generation

OpenAPI specs live in `pkg/internal/openapi/`. Overlays add error responses, typed iterables, and `EpochMillis` mappings. Regenerate with `make generate`. Don't hand-edit `*.gen.go` files.

## Test driven development

This project uses test-driven development (TDD) to drive the development of new features. When planning and writing new code, begin by writing tests which cover the desired functionality. Then, implement the feature and continue iterating until the tests pass.
