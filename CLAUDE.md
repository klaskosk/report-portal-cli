# report-portal-cli

Reusable API client and CLI for Report Portal.

## Architecture

Two main packages, intentionally separated:

- `pkg/` — Standalone, reusable API client module. Public API, meant for consumption by other libraries.
- `internal/cli/` — CLI built on Cobra/Viper. Consumes `pkg/client`.

The API client wraps a generated OpenAPI client (`pkg/internal/api/`) with a scope-based API: `Client` → `ProjectScope` → `LaunchScope`. The generated client should never be exposed publically. However, the generated types are publically accessible in `pkg/types/`.

## Key patterns

- **API errors**: Use predicate functions (`IsNotFound`, `IsUnauthorized`, etc.) rather than type assertions. Follow Dave Cheney's philosophy.
- **Self-documenting Makefile**: The Makefile is self-documenting and should be used to understand the project. Common commands and checks are included in the Makefile.

## Miscellaneous notes

- Never mark tests as pending through Ginkgo. Integration tests must always pass or fail, never be skipped.
- Docker compose is not used in this project. Instead, testcontainers runs the docker compose file directly.
- Do not trust `./prompts/completed/` to be up to date. It shows evolution, not the final state.
