# PR 26: Launch Merge Commands

## Goal

Implement the `merge launch` command for combining multiple launches into one. Merging uses `--file` because the `MergeLaunchesRQ` JSON document is the command's primary input and carries multiple required fields, arrays, and timestamps.

## Context

Merge is managed through `LaunchScope` on the client (`client.Project(name).Launches()`). It is **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 2 (verb group scaffolding: `merge` parent command)
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `readRequestFromFile`, `mustGetStringFlag`)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `Merge` | `ctx`, `request types.MergeLaunchesRQ` | `(*types.LaunchResource, error)` |
| `MergeAsync` | `ctx`, `request types.MergeLaunchesRQ` | `(*types.LaunchResource, error)` |

Both methods return the merged `LaunchResource`. The client defaults empty `MergeType` to `BASIC`.

### Key types

```go
type MergeLaunchesRQ struct {
    Attributes              *[]ItemAttributeResource `json:"attributes,omitempty"`
    Description             *string                  `json:"description,omitempty"`
    EndTime                 *time.Time               `json:"endTime,omitempty"`
    ExtendSuitesDescription bool                     `json:"extendSuitesDescription"`
    Launches                []int64                  `json:"launches"`
    MergeType               MergeLaunchesRQMergeType `json:"mergeType"`
    Mode                    *MergeLaunchesRQMode     `json:"mode,omitempty"`
    Name                    string                   `json:"name"`
    StartTime               *time.Time               `json:"startTime,omitempty"`
}
```

`MergeLaunchesRQMergeType` values: `BASIC`, `DEEP`.

## Scope

Create `internal/cli/merge.go` and `internal/cli/merge_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp merge launch --file <path> [--async]` | Merge multiple launches |

## Implementation

### 1. Create `internal/cli/merge.go`

#### `newMergeLaunchCmd()`

```go
func newMergeLaunchCmd() *cobra.Command
```

- `Use`: `"launch"`
- `Short`: `"Merge multiple launches into one"`
- `Long`: Include example showing `--file` and `--async`. Include sample JSON demonstrating required fields (`name`, `launches`, `mergeType`) and optional fields.
- `Args`: `cobra.NoArgs`
- `RunE`: `withPrintedErrors(runMergeLaunch)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `MergeLaunchesRQ`
  - `--async` (bool, default `false`): use the asynchronous merge endpoint

Sample JSON:

```json
{
  "name": "Merged Launch",
  "launches": [1, 2, 3],
  "mergeType": "BASIC",
  "extendSuitesDescription": false
}
```

The run function:
1. Validate `--file` is non-empty
2. Read and unmarshal using `readRequestFromFile[types.MergeLaunchesRQ](path)`
3. Resolve connection, project, and client
4. If `--async`, call `scope.MergeAsync(ctx, request)`. Otherwise call `scope.Merge(ctx, request)`.
5. Print the resulting `LaunchResource`

**Result formatting:** Reuse the existing `launchHeaders` and `launchRow` from `launch.go` since the merge returns a standard `LaunchResource`. If those functions are not exported or accessible (they are in the same package), call them directly.

If `launchRow` is not suitable (e.g., it takes a different pointer type), define a local formatter that returns `([]string, []string)` — matching the single-row convention used by all other formatters in the CLI:

```go
func mergedLaunchHeaders() []string {
    return []string{"ID", "NAME", "NUMBER", "STATUS", "UUID"}
}

func mergedLaunchRow(launch *types.LaunchResource) ([]string, []string) {
    return mergedLaunchHeaders(), []string{
        strconv.FormatInt(launch.Id, 10),
        launch.Name,
        strconv.FormatInt(launch.Number, 10),
        launch.Status,
        launch.Uuid,
    }
}
```

**Important:** Do not return `[][]string` from this formatter. All single-resource formatters in this CLI return `(headers []string, row []string)` and are wrapped at the `printer.Print` call site. The `[][]string` pattern is only used for explicit multi-row results (e.g., `demoDataRows`).

### 2. Register commands in `commands.go`

```go
// merge
merge.AddCommand(newMergeLaunchCmd())
```

### 3. Create `internal/cli/merge_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewMergeLaunchCmd_Flags`** — Verify `--file` / `-f`, `--async`, and `--project` / `-p` flags. Verify `cobra.NoArgs`.

2. **`TestMergedLaunchRow_Formatting`** — Verify row formatting for a sample `LaunchResource`.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **`--file` is required.** Per PR 17's file-flag rule, merge uses `--file` because the request JSON is the primary input mode. `MergeLaunchesRQ` carries launch IDs, timestamps, attributes, and merge metadata that do not fit ergonomic inline flags.
- **`--async` selects the async variant.** Same pattern as lifecycle commands (PR 19).
- **Returns a `LaunchResource`.** The merged launch is returned as a full resource. The table shows key fields; `-o json` shows the complete resource.
- **Launch IDs are in the JSON, not positional args.** Unlike `compare launch` (PR 25) which takes IDs as args, merge requires many additional fields alongside the IDs. Putting everything in the JSON file keeps the interface clean.
- **Reuse shared helpers.** Use `readRequestFromFile[types.MergeLaunchesRQ]`, `mustGetStringFlag`.

## What NOT to do

- **Do not modify existing launch commands.**
- **Do not validate `MergeType` values locally.**
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp merge --help` shows `launch` as a subcommand
- `rp merge launch --help` shows `--file`, `--async`, `--project`

## Definition of done

- [ ] `internal/cli/merge.go` created
- [ ] `internal/cli/merge_test.go` created
- [ ] Command registered under `merge` parent
- [ ] Uses `--project` / `-p` and `--file` / `-f`
- [ ] `--async` selects async merge endpoint
- [ ] Result shows merged `LaunchResource` details
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
