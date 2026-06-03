# report-portal-cli

> komandlinia interfaco por interagi kun raportportalo

A command-line tool and reusable Go client library for the
[Report Portal](https://reportportal.io/) REST API. The CLI uses a noun-first
command structure (`rp <resource> <verb>`) for managing Report Portal resources
from the terminal.

The API client is generated from the official OpenAPI specification using
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). A hand-crafted
wrapper in `pkg/client` provides a scope-based interface (`Client` →
`ProjectScope` → domain scopes such as launches and items) on top of the
generated client. The bundled spec targets **Report Portal API v5.15.0**.

## Installation

Requires **Go 1.26+**.

```bash
go install github.com/klaskosk/report-portal-cli/cmd/rp@latest
```

Or build from source:

```bash
git clone https://github.com/klaskosk/report-portal-cli.git
cd report-portal-cli
make build        # produces ./rp
```

Shell completions are available via `rp completion <shell>`.

## Usage

```bash
# Launches and test items
rp launch list -p my_project
rp launch get 42 -p my_project
rp item list --launch 42 -p my_project

# Projects, users, and identity
rp project list
rp project get my_project
rp user list
rp whoami

# Dashboards, widgets, and filters
rp dashboard list -p my_project
rp widget list -p my_project
rp filter list -p my_project

# Logs and settings
rp log list --item 123 -p my_project
rp setting pattern list -p my_project

# JSON output (--json is shorthand for -o json)
rp launch list -p my_project --json
```

Every command and subcommand supports `--help` with detailed usage, flags, and
examples:

```bash
rp --help
rp launch --help
rp launch get --help
```

### Configuration

Configuration is loaded from `$XDG_CONFIG_HOME/rp/config.yaml` (or
`$HOME/.config/rp/config.yaml`). Override with `--config`.

Multiple named profiles are supported under the `profiles` key; select one with
`--profile` or set `default-profile` in the config file.

```yaml
default-profile: work

profiles:
  work:
    url: https://reportportal.example.com
    token: my-api-token
    default-project: my_project

  staging:
    url: https://rp-staging.internal
    token: staging-token
    default-project: staging_project
    insecure-skip-verify: true
```

Environment variables `RP_URL`, `RP_TOKEN`, `RP_PROJECT`, and
`RP_INSECURE_SKIP_VERIFY` are also recognized. CLI flags take the highest
precedence, then environment variables, then the active profile.

### Request bodies

Many `create` and `update` commands accept a request body via `--from-file`
(JSON; stdin with `-f -`). When flags and `--from-file` are both supported,
either can supply the payload depending on the command.

```bash
rp user create --from-file user.json
rp setting pattern create --from-file pattern.json -p my_project
```

### Output

Global flags control connection, logging, and output format:

- `-p` / project from config — project scope (required on most project-scoped commands)
- `-o` / `--json` — `table`, `json`, or `go-template=<tmpl>`
- `-v` — verbosity (0=off, 1=info, 2+=debug)
- `--url`, `--token`, `--profile`, `--insecure-skip-verify`

## Client library

The `pkg/` module can be imported independently — consumers pull in only the
client and its dependencies, not the CLI framework.

```bash
go get github.com/klaskosk/report-portal-cli/pkg@latest
```

Import `github.com/klaskosk/report-portal-cli/pkg/client` for the API and
`github.com/klaskosk/report-portal-cli/pkg/types` for DTOs. Wire scopes through
[`Client`](pkg/client/client.go) and [`ProjectScope`](pkg/client/project.go);
do not use generated API types from application code.

API errors are returned as `*client.APIError` and should be handled with
predicate helpers such as `client.IsNotFound`, `client.IsUnauthorized`, and
`client.HasStatusCode` rather than type assertions.

See [`pkg/client/launch/example_launch_test.go`](pkg/client/launch/example_launch_test.go)
for a minimal usage example.

## Development

Run `make help` to see all available targets. Key commands:

```bash
make build              # build the CLI (./rp)
make test               # unit tests
make test-integration   # integration tests (requires Docker)
make verify             # lint, vet, and all tests
make lint               # golangci-lint
make generate           # regenerate OpenAPI client and types
```

To import an upstream OpenAPI spec before regenerating:

```bash
make import-spec SPEC=path/to/spec.json
make generate
```

Integration tests use [testcontainers](https://golang.testcontainers.org/) to
run the Report Portal stack from `test/integration/docker-compose.test.yml`.

### Code generation

Two `go:generate` directives in `pkg/internal/openapi/generate.go` produce:

1. `pkg/types/types.gen.go` — data models
2. `pkg/internal/api/api.gen.go` — HTTP client

Regenerate with `make generate`. Do not hand-edit `*.gen.go` files.

### Project layout

```text
cmd/rp/              CLI entry point
internal/cli/        Commands, config, output formatting, logging
pkg/
  client/            High-level client library (importable)
  types/             Generated types and helpers
  internal/
    api/             Generated low-level API client (not importable)
    openapi/         OpenAPI specs, overlays, and generation config
test/integration/    Docker-based integration tests
```

## License

[Apache License 2.0](LICENSE)

Copyright 2026 Red Hat, LLC.

This is not an official Red Hat product.
