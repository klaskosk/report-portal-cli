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
- **CLI UX**: This project attempts to achieve a CLI experience similar to `kubectl`. Use it as an example for planning and designing the CLI.

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

## Naming conventions

- **CLI nouns**: `launch`, `item` (alias `test-item`), `log`, `project`. These are the resource names used in CLI commands (e.g., `rp get launch`, `rp get item`).
- **Client scopes**: `LaunchScope`, `TestItemScope`, `LogScope` (project-scoped); `UsersScope`, `ProjectsScope` (instance-scoped). Access via `Client.Project(name).Launches()`, `.Items()`, `.Logs()` or `Client.Users()`, `Client.Projects()`.
- **JSON output is a machine contract**: Field names in JSON output come directly from the API types and must not be renamed. `omitempty` behavior is preserved, so optional fields may be omitted rather than emitted as `null`.
- **Table columns**: Use uppercase for column headers (e.g., `ID`, `UUID`, `NAME`, `STATUS`, `LEVEL`).
- **Error messages**: Use lowercase, no trailing punctuation. Wrap with `fmt.Errorf("doing thing: %w", err)`.

## Test driven development

This project uses test-driven development (TDD) to drive the development of new features. When planning and writing new code, begin by writing tests which cover the desired functionality. Then, implement the feature and continue iterating until the tests pass.
