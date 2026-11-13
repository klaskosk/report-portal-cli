# report-portal-cli

A command-line tool and reusable Go client library for the
[Report Portal](https://reportportal.io/) REST API.

The CLI uses a noun-first command structure (`rp <resource> <verb>`) for managing
Report Portal launches, test items, logs, projects, users, dashboards, widgets,
filters, settings, and related resources from the terminal.

The API client is generated from the official OpenAPI specification with
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). A hand-written
wrapper in `pkg/client` provides a scope-based interface on top of the generated
HTTP client. The bundled specification targets Report Portal API **v5.15.0**.

## Installation

Requires **Go 1.27 or newer**.

```bash
go install github.com/klaskosk/report-portal-cli/cmd/rp@latest
```

Or build from source:

```bash
git clone https://github.com/klaskosk/report-portal-cli.git
cd report-portal-cli
make build
```

The resulting binary is named `rp`. Shell completions are available through
`rp completion <shell>`.

## Getting started

Create a configuration file with `rp init`:

```bash
# Interactive wizard
rp init

# Non-interactive setup for scripts and CI
rp init --url https://reportportal.example.com --token my-api-token

# Optional profile and default project
rp init --profile staging --default-project my_project
```

Then verify the connection:

```bash
rp whoami
```

The default configuration file is
`$XDG_CONFIG_HOME/rp/config.yaml` (or `$HOME/.config/rp/config.yaml`). Use
`--config` to select another file.

### Configuration

Multiple named profiles are supported under the `profiles` key. Select one with
`--profile`, or set `default-profile` in the config file.

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

Environment variables are also recognized:

```bash
export RP_URL="https://reportportal.example.com"
export RP_TOKEN="my-api-token"
export RP_PROJECT="my_project"
```

CLI flags take precedence over environment variables, which take precedence
over the active profile.

## Usage

Every command and subcommand supports `--help` with detailed usage, flags, and
examples:

```bash
rp --help
rp launch --help
rp launch get --help
```

Common workflows:

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
```

Many `create` and `update` commands accept a JSON request body with
`--from-file` (or `-f -` for standard input):

```bash
rp user create --from-file user.json
rp setting pattern create --from-file pattern.json -p my_project
```

Global flags control connection settings and output:

- `-p` selects the project for project-scoped commands.
- `-o` selects `table`, `json`, or `go-template=<template>` output.
- `--json` is shorthand for `-o json`.
- `-v` controls log verbosity (`0` off, `1` info, `2+` debug).
- `--config`, `--profile`, `--url`, `--token`, and
  `--insecure-skip-verify` override configuration values.

## Go client library

The `pkg/` module can be imported independently, so library consumers do not
pull in the CLI framework.

```bash
go get github.com/klaskosk/report-portal-cli/pkg@latest
```

Import `github.com/klaskosk/report-portal-cli/pkg/client` for the high-level
client and `github.com/klaskosk/report-portal-cli/pkg/types` for generated data
types. Wire scopes through `client.Client` and `client.ProjectScope`; avoid
using generated API internals directly in application code.

API errors are returned as `*client.APIError`. Use helpers such as
`client.IsNotFound`, `client.IsUnauthorized`, and `client.HasStatusCode`
rather than inspecting status fields manually.

See [`pkg/client/launch/example_launch_test.go`](pkg/client/launch/example_launch_test.go)
for a minimal usage example.

## Development

Run `make help` to list all targets. Common commands include:

```bash
make build              # build the CLI as ./rp
make test               # run unit tests
make test-integration   # run integration tests (requires Docker)
make verify             # lint, vet, and run all tests
make lint               # run golangci-lint
make generate           # regenerate the OpenAPI client and types
```

To import an upstream OpenAPI specification before regenerating:

```bash
make import-spec SPEC=path/to/spec.json
make generate
```

Integration tests use [testcontainers](https://golang.testcontainers.org/) and
the Docker Compose definition in
`test/integration/docker-compose.test.yml`.

### Project layout

```text
cmd/rp/              CLI entry point
internal/cli/        Commands, configuration, output, and logging
pkg/client/          High-level reusable client library
pkg/types/           Generated data types
pkg/internal/api/    Generated low-level HTTP client
pkg/internal/openapi/ OpenAPI specification and generation inputs
test/integration/    Docker-based integration tests
```

Do not hand-edit generated `*.gen.go` files. Regenerate them with
`make generate`.

## License

> [!IMPORTANT]
> This is not an official Red Hat product.

Apache License 2.0 — see [LICENSE](LICENSE). Copyright 2026 Red Hat, LLC.
