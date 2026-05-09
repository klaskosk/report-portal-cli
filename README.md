# report-portal-cli

> komandlinia interfaco por interagi kun raportportalo

A Go project with two complementary goals: a command-line tool (`rp`) for
interacting with [Report Portal](https://reportportal.io/) from the terminal,
and a reusable Go client library (`pkg/client`) that can be imported into other
programs. Both target **Report Portal API v5.11** (Report Portal **24.1**).

The API client is generated from the official OpenAPI specification using
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen), and a
hand-crafted high-level wrapper in `pkg/client` provides an ergonomic,
scope-based interface on top of it.

## Install

### From source

Requires **Go 1.25+**.

```bash
go install github.com/klaskosk/report-portal-cli/cmd/rp@latest
```

### Build locally

```bash
git clone https://github.com/klaskosk/report-portal-cli.git
cd report-portal-cli
make build        # produces ./rp
```

## CLI usage

### Global flags

| Flag | Short | Description |
| --- | --- | --- |
| `--config` | | Path to config file (default `$XDG_CONFIG_HOME/rp/config.yaml`) |
| `--profile` | | Config profile to use (overrides `default-profile`) |
| `--url` | | Report Portal base URL (overrides profile) |
| `--token` | | API bearer token (overrides profile) |
| `--insecure-skip-verify` | | Skip TLS certificate verification |
| `--output` | `-o` | Output format: `table`, `json`, `go-template=<tmpl>` |
| `--json` | | Shorthand for `-o json` |
| `--verbosity` | `-v` | Log verbosity (0=off, 1=info, 2+=debug) |

### Examples

```bash
# List the 10 most recent launches (table output)
rp launch list -p my_project

# Get a single launch by numeric ID
rp launch get 42 -p my_project

# Get a launch by UUID
rp launch get abc-def-123 -p my_project

# JSON output (--json is shorthand for -o json)
rp launch list -p my_project --json

# Custom Go template
rp launch get 42 -p my_project -o 'go-template={{.Name}} — {{.Status}}'

# List users, create, update, delete
rp user list
rp user get superadmin
rp user create --login jane --email jane@example.com --full-name "Jane Doe"
rp user update jane --email jane.doe@example.com
rp user delete jane

# List or get projects
rp project list
rp project get my_project

# Use env vars instead of flags
export RP_URL=https://reportportal.example.com
export RP_TOKEN=my-api-token
export RP_PROJECT=my_project
rp launch list

# Increase verbosity for debugging
rp launch list -p my_project -vv
```

## Configuration

The CLI reads a YAML config file from `$XDG_CONFIG_HOME/rp/config.yaml`
(typically `~/.config/rp/config.yaml`). The config supports multiple named
profiles so you can manage connections to different Report Portal instances.

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

### Resolution order

Connection parameters are resolved with the following precedence (highest wins):

1. CLI flags (`--url`, `--token`, `--insecure-skip-verify`)
2. Environment variables (`RP_URL`, `RP_TOKEN`, `RP_PROJECT`, `RP_INSECURE_SKIP_VERIFY`)
3. Active profile from config file

## Client library

The `pkg/client` package can be imported independently. It is a separate Go
module, so consumers pull in only the client and its dependencies — not the CLI
framework.

```bash
go get github.com/klaskosk/report-portal-cli/pkg@latest
```

### Example

```go
package main

import (
        "context"
        "fmt"
        "log"

        "github.com/klaskosk/report-portal-cli/pkg/client"
)

func main() {
        c, err := client.New("https://reportportal.example.com", "my-api-token")
        if err != nil {
                log.Fatal(err)
        }

        // Get a launch by numeric ID
        launch, err := c.Project("my_project").Launches().GetByID(context.Background(), 42)
        if err != nil {
                if client.IsNotFound(err) {
                        log.Fatal("launch not found")
                }
                log.Fatal(err)
        }

        fmt.Println(launch.Name, launch.Status)

        // List launches with filtering
        result, err := c.Project("my_project").Launches().List(
                context.Background(),
                client.WithLaunchPageSize(5),
                client.WithLaunchSort("startTime,desc"),
                client.WithLaunchStatus("PASSED"),
        )
        if err != nil {
                log.Fatal(err)
        }

        for _, l := range result.Content {
                fmt.Printf("%d  %s  %s\n", l.Id, l.Name, l.Status)
        }
}
```

### Client options

```go
// Custom HTTP client (e.g. for TLS configuration)
c, err := client.New(url, token,
        client.WithHTTPClient(myHTTPClient),
        client.WithLogger(slog.Default()),
)
```

### Error handling

API errors are returned as `*client.APIError` and can be inspected with
predicate functions:

```go
if client.IsNotFound(err) { /* 404 */ }
if client.IsUnauthorized(err) { /* 401 */ }
if client.IsForbidden(err) { /* 403 */ }
if client.IsBadRequest(err) { /* 400 */ }
if client.IsConflict(err) { /* 409 */ }

// General-purpose predicates
if client.HasStatusCode(err, 429) { /* rate limited */ }
if client.HasErrorCode(err, 4004) { /* application error code */ }
```

## Makefile

| Target | Description |
| --- | --- |
| `make all` | Run tidy, generate, lint, and build |
| `make build` | Build the `rp` binary |
| `make vet` | Run `go vet` on all packages |
| `make generate` | Run `go generate` for all packages (regenerates OpenAPI client and types) |
| `make tidy` | Run `go mod tidy` for all modules (root + `pkg`) |
| `make lint` | Run `golangci-lint-v2 run --fix` |
| `make clean` | Remove the built binary |
| `make help` | Show available targets |

## Project structure

```text
cmd/rp/              CLI entry point
internal/cli/        Command definitions, config, output formatting, logging
pkg/
  client/            High-level client library (importable)
  types/             Generated Report Portal types + custom helpers
  internal/
    api/             Generated low-level API client (not importable)
    openapi/         OpenAPI specs, overlays, and generation config
```

## License

[Apache License 2.0](LICENSE)

Copyright 2026 Red Hat, LLC.

This is not an official Red Hat product.
