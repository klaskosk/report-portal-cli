# PR 34: Demo-Data Generation

## Goal

Add `rp demo-data generate` for generating demo data in a project.

## Context

Phase 7. The `DemoDataScope` client already has a `Generate` method. This is a simple project-scoped command.

### Client API surface

**DemoDataScope** (via `client.Project(name).DemoData()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `Generate` | `Generate(ctx, types.DemoDataRq)` | `*types.DemoDataRs` |

### Relevant types

Look up in `pkg/types/`:
- `DemoDataRq` — request body for demo data generation (may have fields like DashboardsCount, LaunchesCount, etc., or may be empty — check the actual type)
- `DemoDataRs` — response containing IDs of created resources (DashboardIds, FilterIds, LaunchIds, etc.)

## Implementation plan

### 1. Create `internal/cli/demo_data.go`

#### Parent command

```go
func newDemoDataCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "demo-data",
        Short: "Generate demo data",
    }
    cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
    cmd.AddCommand(newDemoDataGenerateCmd())
    return cmd
}
```

#### `rp demo-data generate`

- No positional args
- Project-scoped (needs `-p`/`--project`)
- Optional: `--from-file` (string) — reads full `DemoDataRq` from JSON
- If no `--from-file`, send an empty/default `DemoDataRq{}`
- Call `scope.DemoData().Generate(ctx, request)` — check if the method path is `client.Project(name).DemoData().Generate()` or similar. Look at how `DemoDataScope` is accessed from the project scope.
- Table output: Show created resource IDs (LAUNCHES, DASHBOARDS, FILTERS, etc.)
- JSON output: raw `DemoDataRs`

### 2. Register in root.go

Add `cmd.AddCommand(newDemoDataCmd())` in `newRootCmd()`.

### 3. Table formatting

The `DemoDataRs` likely contains slices of IDs. Format as:
```
RESOURCE     IDS
Launches     [1, 2, 3]
Dashboards   [4, 5]
Filters      [6, 7]
```

Or if the response is simpler, adapt accordingly after checking the actual type.

## Test requirements

### Unit tests (`internal/cli/demo_data_test.go`)

- **Generate:** No required flags (just project)
- **From-file:** Valid JSON loads correctly
- **Command wiring:** verify `newDemoDataCmd()` has `generate` subcommand
- **Project flag:** verify `-p`/`--project` is available

## Edge cases

- **Project-scoped:** Must have a project context
- **DemoDataRq may be empty:** If the type has no required fields, sending `{}` should be fine
- **Check DemoDataScope access path:** Look at how other scopes are accessed from the project (e.g., `project.DemoData()` or similar). If no such method exists, you may need to check how the scope is instantiated.
- **Response structure:** Check `DemoDataRs` actual fields — it may have `DashboardIds *[]int64`, `FilterIds *[]int64`, `LaunchIds *[]int64` etc.

## Example commands

```bash
# Generate demo data with defaults
$ rp demo-data generate -p my_project
RESOURCE     IDS
Launches     [1, 2, 3, 4, 5]
Dashboards   [1, 2]
Filters      [1, 2, 3]

# Generate with custom config from file
$ rp demo-data generate --from-file demo_config.json -p my_project
RESOURCE     IDS
Launches     [6, 7, 8]
Dashboards   [3]
```

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [ ] `internal/cli/demo_data.go` with generate command
- [ ] Registered in `root.go`
- [ ] `--from-file` support
- [ ] Table and JSON output
- [ ] Unit tests cover command wiring and from-file loading
- [ ] `make lint` and `make test` pass
- [ ] This file moved to `prompts/completed/`
