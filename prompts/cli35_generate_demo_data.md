# PR 35: `generate demo-data`

## Goal

Implement the `generate demo-data` command for populating a project with sample data. This is an isolated, single-purpose command that creates demo launches and optionally a demo dashboard.

## Context

Demo data generation is managed through `DemoDataScope` on the client (`client.Project(name).DemoData()`). It is **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 2 (verb group scaffolding: `generate` parent command)
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `optionalBoolFlag`)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `Generate` | `ctx`, `demoDataRequest types.DemoDataRq` | `(*types.DemoDataRs, error)` |

### Key types

```go
type DemoDataRq struct {
    CreateDashboard *bool `json:"createDashboard,omitempty"`
}

type DemoDataRs struct {
    DashboardId *int64   `json:"dashboardId,omitempty"`
    LaunchIds   *[]int64 `json:"launchIds,omitempty"`
}
```

## Scope

Create `internal/cli/demo_data.go` and `internal/cli/demo_data_test.go`. Register the new command in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp generate demo-data [--create-dashboard]` | Generate demo data in a project |

## Implementation

### 1. Create `internal/cli/demo_data.go`

#### `newGenerateDemoDataCmd()`

```go
func newGenerateDemoDataCmd() *cobra.Command
```

- `Use`: `"demo-data"`
- `Short`: `"Generate demo data in a project"`
- `Long`: Include example. Note that this creates sample launches and optionally a demo dashboard. Mention the `--create-dashboard` flag.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runGenerateDemoData)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--create-dashboard` (bool): also create a demo dashboard. Registered as a bool flag but consumed via `optionalBoolFlag` — unset produces `nil` (server default), `--create-dashboard` produces `&true`, `--create-dashboard=false` produces `&false`.

The run function:
1. Read `--create-dashboard` flag
2. Build `types.DemoDataRq{}`. If `--create-dashboard` was explicitly set, include `CreateDashboard: &createDashboard`. If the flag was not changed, leave `CreateDashboard` as nil (the server decides the default behavior).

   **Handling the `*bool` field:** Use `optionalBoolFlag(cmd, "create-dashboard")` to get `*bool`:
   - If the user passed `--create-dashboard`: returns `&true`
   - If the user passed `--create-dashboard=false`: returns `&false`
   - If the user didn't pass the flag: returns `nil` (server default)

3. Resolve connection, project, and client
4. Call `scope.Generate(ctx, request)`
5. Print the `DemoDataRs` result

**Table formatting for `DemoDataRs`:**

```go
func demoDataHeaders() []string {
    return []string{"DASHBOARD ID", "LAUNCH IDS"}
}

func demoDataRows(result *types.DemoDataRs) [][]string {
    dashboardID := ""
    if result.DashboardId != nil {
        dashboardID = strconv.FormatInt(*result.DashboardId, 10)
    }

    launchIDs := ""
    if result.LaunchIds != nil {
        ids := make([]string, len(*result.LaunchIds))
        for i, id := range *result.LaunchIds {
            ids[i] = strconv.FormatInt(id, 10)
        }

        launchIDs = strings.Join(ids, ", ")
    }

    return [][]string{{dashboardID, launchIDs}}
}
```

### 2. Register commands in `commands.go`

```go
// generate
generate.AddCommand(newGenerateDemoDataCmd())
```

### 3. Create `internal/cli/demo_data_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewGenerateDemoDataCmd_Flags`** — Verify `--create-dashboard` and `--project` / `-p` flags are registered. Do **not** assert the flag "defaults to false" in a way that implies `&false` is sent — the implementation uses `optionalBoolFlag`, so an unset flag produces `nil` (server default), not `&false`. The test should verify the flag exists and that `cmd.Flags().Changed("create-dashboard")` is false before the flag is explicitly set.

2. **`TestNewGenerateDemoDataCmd_NoArgs`** — Verify `cobra.NoArgs`.

3. **`TestDemoDataRows_WithAllFields`** — Build a `DemoDataRs` with dashboard ID and launch IDs, verify row output.

4. **`TestDemoDataRows_NilFields`** — Verify nil `DashboardId` and nil `LaunchIds` produce empty strings.

5. **`TestDemoDataRows_EmptyLaunchIds`** — Verify empty `LaunchIds` slice produces empty string.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **Single-purpose command.** This is the simplest command in the specialized wave — one request type, one response type, one optional flag.
- **`--create-dashboard` uses `optionalBoolFlag`.** Because `DemoDataRq.CreateDashboard` is `*bool`, the CLI should distinguish between "user didn't pass the flag" (nil — server default) and "user explicitly passed false" (`&false`). The `optionalBoolFlag` helper handles this correctly.
- **Launch IDs joined with commas in table.** The launch IDs array is formatted as a comma-separated list for the table. For full structure, use `-o json`.
- **Reuse shared helpers.** Use `optionalBoolFlag`.

## What NOT to do

- **Do not add complex configuration flags.** The request type only has one field.
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp generate --help` shows `demo-data`
- `rp generate demo-data --help` shows `--create-dashboard` and `--project`

## Definition of done

- [ ] `internal/cli/demo_data.go` created
- [ ] `internal/cli/demo_data_test.go` created
- [ ] Command registered under `generate` parent
- [ ] Uses `--project` / `-p`
- [ ] `--create-dashboard` uses `optionalBoolFlag` for proper nil handling
- [ ] Table shows DASHBOARD ID and LAUNCH IDS
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
