# PR 25: Launch Analysis Commands

## Goal

Implement `analyze` and `compare` commands for launches. These are specialized read/compute operations that do not fit standard CRUD patterns — `analyze` triggers an analysis run and `compare` returns a statistical comparison of multiple launches.

## Context

Both operations use `LaunchScope` on the client (`client.Project(name).Launches()`). They are **project-scoped**, requiring `--project`, `RP_PROJECT`, or `default-project`.

**Prerequisite PRs and dependencies:**
- PR 2 (verb group scaffolding: `analyze` and `compare` parent commands)
- PR 3 (`commands.go` / `buildCommandTree` for command registration)
- PR 7 (shared helpers: `readRequestFromFile`, `parseResourceID`)

### Client surface

| Method | Parameters | Returns |
|--------|------------|---------|
| `Analyze` | `ctx`, `request types.AnalyzeLaunchRQ` | `(*types.OperationCompletionRS, error)` |
| `Compare` | `ctx`, `ids []int64` | `(map[string][]types.ChartStatisticsContent, error)` |

### Key types

```go
type AnalyzeLaunchRQ struct {
    AnalyzeItemsMode []AnalyzeLaunchRQAnalyzeItemsMode `json:"analyzeItemsMode"`
    AnalyzerMode     AnalyzeLaunchRQAnalyzerMode       `json:"analyzerMode"`
    AnalyzerTypeName AnalyzeLaunchRQAnalyzerTypeName   `json:"analyzerTypeName"`
    LaunchId         int64                             `json:"launchId"`
}

type ChartStatisticsContent struct {
    Id        *int64             `json:"id,omitempty"`
    Name      *string            `json:"name,omitempty"`
    Number    *int32             `json:"number,omitempty"`
    StartTime *time.Time         `json:"startTime,omitempty"`
    Values    *map[string]string `json:"values,omitempty"`
}
```

`AnalyzeLaunchRQAnalyzeItemsMode` values: `AUTO_ANALYZED`, `MANUALLY_ANALYZED`, `TO_INVESTIGATE`.
`AnalyzeLaunchRQAnalyzerMode` values: `ALL`, `CURRENT_AND_THE_SAME_NAME`, `CURRENT_LAUNCH`, `LAUNCH_NAME`, `PREVIOUS_LAUNCH`.
`AnalyzeLaunchRQAnalyzerTypeName` values: `autoAnalyzer`, `patternAnalyzer`.

## Scope

Create `internal/cli/analysis.go` and `internal/cli/analysis_test.go`. Register the new commands in `internal/cli/commands.go`.

## CLI shape

| Command | Description |
|---------|-------------|
| `rp analyze launch <id> --file <path>` | Analyze a launch |
| `rp compare launch <id1> <id2> [<id3>...]` | Compare two or more launches |

## Implementation

### 1. Create `internal/cli/analysis.go`

#### `newAnalyzeLaunchCmd()`

```go
func newAnalyzeLaunchCmd() *cobra.Command
```

- `Use`: `"launch <id>"`
- `Short`: `"Analyze a launch"`
- `Long`: Include example showing `--file` with a launch ID. Include sample JSON showing all required fields (`analyzeItemsMode`, `analyzerMode`, `analyzerTypeName`, `launchId`). Document valid enum values for each field. Note that `launchId` in the JSON is overridden by the positional argument.
- `Args`: `cobra.ExactArgs(1)` (the numeric launch ID)
- `RunE`: `withPrintedErrors(runAnalyzeLaunch)`
- Flags:
  - `--project` / `-p` (string): project name override
  - `--file` / `-f` (string, required): path to JSON file containing `AnalyzeLaunchRQ`

**Why `--file`?** `AnalyzeLaunchRQ` has an array field (`AnalyzeItemsMode`) and multiple enum fields, so a file-backed request is much clearer than inline flags. Per PR 17's frozen UX, `--file` is the right name when the file is the command's primary request-body input mode; reserve `--from-file` for cases where the file is only one option among multiple input modes. This command still takes a positional launch ID, but the request body itself always comes from the JSON file.

Sample JSON:

```json
{
  "analyzeItemsMode": ["TO_INVESTIGATE"],
  "analyzerMode": "CURRENT_LAUNCH",
  "analyzerTypeName": "autoAnalyzer",
  "launchId": 42
}
```

The run function:
1. Parse `args[0]` as int64 using `parseResourceID(args[0], "launch")`
2. Validate `--file` is non-empty
3. Read and unmarshal using `readRequestFromFile[types.AnalyzeLaunchRQ](path)`
4. **Override the `LaunchId` field** with the positional argument: `request.LaunchId = launchID`. This ensures the positional arg is the source of truth. This follows the same positional-arg-overrides-file-payload pattern used elsewhere in the specialized command wave, though there is no implementation dependency on those PRs.
5. Resolve connection, project, and client
6. Call `scope.Analyze(ctx, request)`
7. Print the `OperationCompletionRS` result using `messageResultHeaders()` and `operationResultRows(result)`

#### `newCompareLaunchCmd()`

```go
func newCompareLaunchCmd() *cobra.Command
```

- `Use`: `"launch <id1> <id2> [<id3>...]"`
- `Short`: `"Compare launches"`
- `Long`: Include examples showing comparing two and three launches. Note that at least two launch IDs are required.
- `Args`: `cobra.MinimumNArgs(2)` (at least two numeric launch IDs)
- `RunE`: `withPrintedErrors(runCompareLaunch)`
- Flags:
  - `--project` / `-p` (string): project name override

The run function:
1. Parse all positional arguments as int64:
   ```go
   ids := make([]int64, len(args))
   for i, arg := range args {
       id, err := parseResourceID(arg, "launch")
       if err != nil {
           return err
       }
       ids[i] = id
   }
   ```
2. Resolve connection, project, and client
3. Call `scope.Compare(ctx, ids)` — returns `map[string][]types.ChartStatisticsContent`
4. Print the result

**Result formatting for compare:** The compare response is a `map[string][]ChartStatisticsContent` where keys are category names (e.g., `"statistics$defects$automation_bug$total"`) and values are arrays of chart data points per launch.

For table output, flatten into rows:

```go
func compareHeaders() []string {
    return []string{"CATEGORY", "LAUNCH ID", "LAUNCH NAME", "NUMBER", "VALUES"}
}

func compareRows(result map[string][]types.ChartStatisticsContent) [][]string {
    categories := make([]string, 0, len(result))
    for k := range result {
        categories = append(categories, k)
    }

    sort.Strings(categories)

    var rows [][]string
    for _, category := range categories {
        for _, entry := range result[category] {
            values := ""
            if entry.Values != nil {
                // Format as key=value pairs
                pairs := make([]string, 0, len(*entry.Values))
                for k, v := range *entry.Values {
                    pairs = append(pairs, k+"="+v)
                }

                sort.Strings(pairs)
                values = strings.Join(pairs, ", ")
            }

            rows = append(rows, []string{
                category,
                formatInt64Ptr(entry.Id),
                formatStringPtr(entry.Name),
                formatInt32Ptr(entry.Number),
                values,
            })
        }
    }

    return rows
}
```

**Output mode behavior:**
- **Table mode:** Uses the flattened `compareRows` above — one row per category/launch data point.
- **JSON / Go-template mode:** Passes the raw `map[string][]ChartStatisticsContent` through unchanged. The nested map is the natural structure for programmatic consumption.

Print via `printer.Print(result, compareHeaders(), compareRows(result))`.

### 2. Register commands in `commands.go`

```go
// analyze
analyze.AddCommand(newAnalyzeLaunchCmd())

// compare
compare.AddCommand(newCompareLaunchCmd())
```

### 3. Create `internal/cli/analysis_test.go`

```go
package cli //nolint:testpackage
```

#### Tests to write

1. **`TestNewAnalyzeLaunchCmd_Flags`** — Verify `--file` / `-f` and `--project` / `-p` flags. Verify 1 arg.

2. **`TestNewCompareLaunchCmd_Args`** — Verify at least 2 args required (`cobra.MinimumNArgs(2)`). Verify `--project` / `-p`.

3. **`TestCompareRows_WithData`** — Build a sample `map[string][]ChartStatisticsContent` with two categories and verify row output. Verify categories are sorted.

4. **`TestCompareRows_EmptyMap`** — Verify empty map produces empty rows.

5. **`TestCompareRows_NilValues`** — Verify entries with nil `Values` produce empty strings in the VALUES column.

All tests should use `t.Parallel()` and the `testify` assertion libraries.

## Design notes

- **`--file` for analyze.** The request has three enum-typed fields and an array. A file-backed request avoids error-prone enum entry on the command line, and PR 17's naming rule says commands whose file input is primary or only should use `--file`.
- **LaunchId override from positional arg.** The positional arg is the source of truth, even if the JSON file also contains `launchId`. This keeps the target explicit at the call site.
- **Compare takes variadic positional args.** At least two launch IDs are required. This is more ergonomic than `--ids 1,2,3`.
- **Compare table is a flattened view.** The nested `map[string][]ChartStatisticsContent` is best consumed via `-o json` or `-o go-template`. The table provides a readable summary with one row per category×launch data point.
- **Categories are sorted for deterministic output.** Go maps have random iteration order; sorting ensures consistent output.
- **Route all output through `printer.Print`.** Analyze uses `printer.Print(result, messageResultHeaders(), operationResultRows(result))`. Compare uses `printer.Print(result, compareHeaders(), compareRows(result))`. This ensures all output modes work.
- **Reuse shared helpers.** Use `parseResourceID`, `readRequestFromFile[types.AnalyzeLaunchRQ]`, `messageResultHeaders`/`operationResultRows`, `formatStringPtr`, `formatInt64Ptr`, `formatInt32Ptr`, `withPrintedErrors`.
- **Command registration follows PR 3's centralized command-tree layout.** Shared mutation/result helpers assume PR 7's naming conventions.
- **No implementation dependency on other specialized prompts.** The positional-arg-overrides-file-payload pattern can be implemented locally here without coupling this PR to another specialized command family.

## What NOT to do

- **Do not validate enum values locally.** Let the API validate.
- **Do not modify existing launch commands.**
- **Do not add integration tests.**

## Validation

```bash
make build
make test
make lint
```

Verify:
- `rp analyze --help` shows `launch` as a subcommand
- `rp compare --help` shows `launch` as a subcommand
- `rp analyze launch --help` shows `--file` and `--project`
- `rp compare launch --help` shows it requires at least 2 args

## Definition of done

- [ ] `internal/cli/analysis.go` created
- [ ] `internal/cli/analysis_test.go` created
- [ ] Commands registered under `analyze` and `compare` parents
- [ ] Both commands use `--project` / `-p`
- [ ] Analyze uses `--file` with positional ID override
- [ ] Compare accepts variadic launch IDs (minimum 2)
- [ ] Compare table flattens the nested map with sorted categories
- [ ] All existing tests pass
- [ ] Lint passes
- [ ] This file moved to the `completed` directory
