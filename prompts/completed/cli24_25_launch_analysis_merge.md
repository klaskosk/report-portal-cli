# PR 24+25: Launch Analysis and Merge Commands

## Goal

Add `rp launch analyze`, `rp launch compare`, and `rp launch merge` in sub-files (`launch_analysis.go`, `launch_merge.go`).

## Context

Phase 7. The launch command already has list/get/update/delete/report/start/finish (once PR 17 lands, but this PR is independent). The `LaunchScope` client already has `Analyze`, `Compare`, and `Merge`/`MergeAsync` methods.

These PRs are combined because they both add non-CRUD operations to the launch command and would conflict on the parent registration.

### Client API surface

**LaunchScope** (via `client.Project(name).Launches()`):

| Method | Signature | Returns |
|--------|-----------|---------|
| `Analyze` | `Analyze(ctx, types.AnalyzeLaunchRQ)` | `*types.OperationCompletionRS` |
| `Compare` | `Compare(ctx, ids []int64)` | `map[string][]types.ChartStatisticsContent` |
| `Merge` | `Merge(ctx, types.MergeLaunchesRQ)` | `*types.LaunchResource` |

### Relevant types

```go
type AnalyzeLaunchRQ struct {
    AnalyzeItemsMode []string `json:"analyzeItemsMode"`
    AnalyzerMode     string   `json:"analyzerMode"`
    AnalyzerTypeName string   `json:"analyzerTypeName"`
    LaunchId         int64    `json:"launchId"`
}
// analyzerMode: "ALL", "LAUNCH_NAME", "CURRENT_LAUNCH", "PREVIOUS_LAUNCH", "CURRENT_AND_THE_SAME_NAME"
// analyzeItemsMode: ["TO_INVESTIGATE"], ["AUTO_ANALYZED"], ["MANUALLY_ANALYZED"], or combinations

type ChartStatisticsContent struct {
    Values map[string]string `json:"values,omitempty"`
}

type MergeLaunchesRQ struct {
    Attributes       *[]ItemAttributeResource     `json:"attributes,omitempty"`
    Description      string                       `json:"description"`
    EndTime          *time.Time                   `json:"endTime,omitempty"`
    ExtendSuitesDescription bool                  `json:"extendSuitesDescription"`
    Launches         []int64                      `json:"launches"`
    MergeType        MergeLaunchesRQMergeType     `json:"mergeType"`
    Mode             *MergeLaunchesRQMode         `json:"mode,omitempty"`
    Name             string                       `json:"name"`
    StartTime        *time.Time                   `json:"startTime,omitempty"`
    Tags             *[]string                    `json:"tags,omitempty"`
}
// MergeType: "BASIC", "DEEP"
// Mode: "DEFAULT", "DEBUG"
```

Look up exact types in `pkg/types/`.

## Implementation plan

### 1. Create `internal/cli/launch_analysis.go`

#### `rp launch analyze <id>`

- One positional arg: numeric launch ID
- Required flag: `--analyzer-mode` (string) — one of ALL, LAUNCH_NAME, CURRENT_LAUNCH, PREVIOUS_LAUNCH, CURRENT_AND_THE_SAME_NAME
- Optional flags:
  - `--items-mode` (string slice) — TO_INVESTIGATE, AUTO_ANALYZED, MANUALLY_ANALYZED (defaults to ["TO_INVESTIGATE"])
  - `--analyzer-type` (string) — analyzer type name (defaults to "autoAnalyzer" or whatever the server expects)
  - `--from-file` (string) — full AnalyzeLaunchRQ from JSON
- Call `scope.Analyze(ctx, request)`
- Output: `OperationCompletionRS` message

#### `rp launch compare`

- No positional args
- Required flag: `--id` (int64 slice, repeatable) — launch IDs to compare (at least 2)
- Call `scope.Compare(ctx, ids)`
- Table output: Metric name + values across launches (format as readable table)
- JSON output: raw map response

### 2. Create `internal/cli/launch_merge.go`

#### `rp launch merge`

- No positional args
- Required flags:
  - `--launch` (int64 slice, repeatable) — launch IDs to merge (at least 2)
  - `--name` (string) — name for the merged launch
- Optional flags:
  - `--description` (string)
  - `--merge-type` (string) — "BASIC" or "DEEP" (default: "BASIC")
  - `--mode` (string) — "DEFAULT" or "DEBUG"
  - `--extend-suites` (bool) — extend suites description
  - `--attribute` (string slice) — `key:value` format
  - `--from-file` (string) — full MergeLaunchesRQ from JSON
- Call `scope.Merge(ctx, request)`
- Table output: Standard launch row (ID, NAME, NUMBER, STATUS, etc.) from returned `LaunchResource`
- JSON output: raw `LaunchResource`

### 3. Register in launch.go

Add to `newLaunchCmd()`:
```go
cmd.AddCommand(newLaunchAnalyzeCmd())
cmd.AddCommand(newLaunchCompareCmd())
cmd.AddCommand(newLaunchMergeCmd())
```

## Test requirements

### Unit tests

- **`internal/cli/launch_analysis_test.go`:**
  - Analyze: requires positional ID, `--analyzer-mode` required
  - Analyze: `--analyzer-mode` validation (valid modes accepted, invalid rejected)
  - Compare: `--id` required, at least 2 IDs needed
  - Command wiring

- **`internal/cli/launch_merge_test.go`:**
  - Merge: `--launch` and `--name` required
  - Merge: at least 2 launch IDs needed
  - Merge: `--merge-type` validation (BASIC/DEEP)
  - Merge: `--mode` validation (DEFAULT/DEBUG)
  - `--from-file` overrides flags

## Edge cases

- **Project-scoped:** Inherits `-p`/`--project` from launch parent
- **Compare needs 2+ IDs:** Validate minimum count before API call
- **Merge needs 2+ launches:** Same validation
- **Merge returns LaunchResource:** Reuse existing `launchRow` table formatter from `launch.go` if it's exported or accessible
- **`--from-file` takes precedence** on both analyze and merge
- **Default analyzer items mode:** If `--items-mode` not provided, default to `["TO_INVESTIGATE"]`

## Example commands

```bash
# Analyze a launch
$ rp launch analyze 42 --analyzer-mode ALL -p my_project
MESSAGE
Launch analysis has been started.

# Compare launches
$ rp launch compare --id 41 --id 42 --id 43 -p my_project
METRIC                    LAUNCH 41  LAUNCH 42  LAUNCH 43
statistics$executions$total  100       120        115
statistics$executions$passed  95       110        108

# Merge launches
$ rp launch merge --launch 41 --launch 42 --name "Merged Nightly" --merge-type DEEP -p my_project
ID    NAME            NUMBER  STATUS
44    Merged Nightly  1       IN_PROGRESS
```

## Validation

```
make lint
make test
make test-integration
```

## Definition of done

- [ ] `internal/cli/launch_analysis.go` with analyze and compare commands
- [ ] `internal/cli/launch_merge.go` with merge command
- [ ] Commands registered in `newLaunchCmd()`
- [ ] Analyzer mode and merge type validation
- [ ] Compare requires 2+ IDs, merge requires 2+ launches
- [ ] Unit tests cover validation, wiring, and from-file
- [ ] Existing launch subcommands unchanged
- [ ] `make lint` and `make test` pass
- [ ] This file moved to `prompts/completed/`
