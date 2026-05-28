package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newLaunchAnalyzeCmd builds the "launch analyze" subcommand.
func newLaunchAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze <id>",
		Short: "Trigger auto-analysis on a launch",
		Long: `Trigger auto-analysis on a launch by numeric ID.

With --from-file, the JSON file supplies the full AnalyzeLaunchRQ body.
The positional id is still required by the CLI but ignored.

Examples:
  rp launch analyze 42 --analyzer-mode ALL -p my_project
  rp launch analyze 42 --analyzer-mode CURRENT_LAUNCH --items-mode TO_INVESTIGATE -p my_project
  rp launch analyze 42 --from-file analyze.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runLaunchAnalyze),
	}

	cmd.Flags().String("analyzer-mode", "", "analyzer mode (required without --from-file)")
	cmd.Flags().
		StringArray("items-mode", nil, "items mode: TO_INVESTIGATE, AUTO_ANALYZED, MANUALLY_ANALYZED (repeatable)")
	cmd.Flags().String("analyzer-type", "autoAnalyzer", "analyzer type name")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing AnalyzeLaunchRQ (use - for stdin)")

	return cmd
}

// newLaunchCompareCmd builds the "launch compare" subcommand.
func newLaunchCompareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare statistics across launches",
		Long: `Compare launch statistics for two or more launches.

Examples:
  rp launch compare --id 41 --id 42 -p my_project
  rp launch compare --id 41 --id 42 --id 43 -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runLaunchCompare),
	}

	cmd.Flags().Int64Slice("id", nil, "launch ID to compare (repeatable, at least 2 required)")

	return cmd
}

func runLaunchAnalyze(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	launchID, err := parseIDArg(args[0], "launch")
	if err != nil {
		printer.PrintError(err)

		return err
	}

	request, err := buildAnalyzeLaunchRequest(cmd, launchID)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	result, err := deps.client.Project(project).Launches().Analyze(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("analyzing launch %d in project %s: %w", launchID, project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runLaunchCompare(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	ids, err := resolveCompareLaunchIDs(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	result, err := deps.client.Project(project).Launches().Compare(cmd.Context(), ids)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("comparing launches in project %s: %w", project, err)
	}

	headers, rows := compareLaunchTable(ids, result)

	return deps.printer.Print(result, headers, rows)
}

func buildAnalyzeLaunchRequest(cmd *cobra.Command, launchID int64) (types.AnalyzeLaunchRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		req, err := readRequestFromFile[types.AnalyzeLaunchRQ](path)
		if err != nil {
			return types.AnalyzeLaunchRQ{}, wrapFileError(
				err,
				"launch analyze file",
				"launch analyze JSON",
			)
		}

		return req, nil
	}

	analyzerModeRaw, _ := cmd.Flags().GetString("analyzer-mode")
	if analyzerModeRaw == "" {
		return types.AnalyzeLaunchRQ{}, errors.New(
			"--analyzer-mode is required when not using --from-file",
		)
	}

	analyzerMode, err := resolveAnalyzeLaunchAnalyzerMode(analyzerModeRaw)
	if err != nil {
		return types.AnalyzeLaunchRQ{}, err
	}

	itemsMode, err := resolveAnalyzeLaunchItemsMode(cmd)
	if err != nil {
		return types.AnalyzeLaunchRQ{}, err
	}

	analyzerTypeRaw, _ := cmd.Flags().GetString("analyzer-type")

	analyzerType, err := resolveAnalyzeLaunchAnalyzerType(analyzerTypeRaw)
	if err != nil {
		return types.AnalyzeLaunchRQ{}, err
	}

	return types.AnalyzeLaunchRQ{
		LaunchId:         launchID,
		AnalyzerMode:     analyzerMode,
		AnalyzeItemsMode: itemsMode,
		AnalyzerTypeName: analyzerType,
	}, nil
}

func resolveAnalyzeLaunchAnalyzerMode(raw string) (types.AnalyzeLaunchRQAnalyzerMode, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))

	switch normalized {
	case string(types.ALL),
		string(types.LAUNCHNAME),
		string(types.CURRENTLAUNCH),
		string(types.PREVIOUSLAUNCH),
		string(types.CURRENTANDTHESAMENAME):
		return types.AnalyzeLaunchRQAnalyzerMode(normalized), nil
	default:
		return "", fmt.Errorf(
			"invalid analyzer-mode %q: must be ALL, LAUNCH_NAME, CURRENT_LAUNCH, PREVIOUS_LAUNCH, or CURRENT_AND_THE_SAME_NAME",
			raw,
		)
	}
}

func resolveAnalyzeLaunchItemsMode(
	cmd *cobra.Command,
) ([]types.AnalyzeLaunchRQAnalyzeItemsMode, error) {
	rawModes, _ := cmd.Flags().GetStringArray("items-mode")
	if len(rawModes) == 0 {
		return []types.AnalyzeLaunchRQAnalyzeItemsMode{types.TOINVESTIGATE}, nil
	}

	modes := make([]types.AnalyzeLaunchRQAnalyzeItemsMode, 0, len(rawModes))

	for _, raw := range rawModes {
		normalized := strings.ToUpper(strings.TrimSpace(raw))

		switch normalized {
		case string(types.TOINVESTIGATE),
			string(types.AUTOANALYZED),
			string(types.MANUALLYANALYZED):
			modes = append(modes, types.AnalyzeLaunchRQAnalyzeItemsMode(normalized))
		default:
			return nil, fmt.Errorf(
				"invalid items-mode %q: must be TO_INVESTIGATE, AUTO_ANALYZED, or MANUALLY_ANALYZED",
				raw,
			)
		}
	}

	return modes, nil
}

func resolveAnalyzeLaunchAnalyzerType(raw string) (types.AnalyzeLaunchRQAnalyzerTypeName, error) {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return types.AutoAnalyzer, nil
	}

	switch normalized {
	case string(types.AutoAnalyzer), string(types.PatternAnalyzer):
		return types.AnalyzeLaunchRQAnalyzerTypeName(normalized), nil
	default:
		return "", fmt.Errorf(
			"invalid analyzer-type %q: must be autoAnalyzer or patternAnalyzer",
			raw,
		)
	}
}

func resolveCompareLaunchIDs(cmd *cobra.Command) ([]int64, error) {
	ids, _ := cmd.Flags().GetInt64Slice("id")
	if len(ids) < 2 {
		return nil, fmt.Errorf(
			"at least two launch IDs are required for compare (got %d)",
			len(ids),
		)
	}

	return ids, nil
}

func compareLaunchTable(
	ids []int64,
	result map[string][]types.ChartStatisticsContent,
) ([]string, [][]string) {
	headers := compareLaunchHeaders(ids)

	groupKeys := make([]string, 0, len(result))
	for groupKey := range result {
		groupKeys = append(groupKeys, groupKey)
	}

	sort.Strings(groupKeys)

	//nolint:prealloc // row count is derived inside compareLaunchGroupRows
	var rows [][]string

	for _, groupKey := range groupKeys {
		rows = append(rows, compareLaunchGroupRows(groupKey, ids, result[groupKey])...)
	}

	return headers, rows
}

func compareLaunchHeaders(ids []int64) []string {
	headers := make([]string, 0, len(ids)+1)
	headers = append(headers, "METRIC")

	for _, id := range ids {
		headers = append(headers, fmt.Sprintf("LAUNCH %d", id))
	}

	return headers
}

func compareLaunchGroupRows(
	groupKey string,
	ids []int64,
	contents []types.ChartStatisticsContent,
) [][]string {
	byID := make(map[int64]map[string]string, len(contents))

	for _, content := range contents {
		if content.Id == nil || content.Values == nil {
			continue
		}

		byID[*content.Id] = *content.Values
	}

	valueKeys := collectCompareValueKeys(byID)

	sortedValueKeys := make([]string, 0, len(valueKeys))
	for key := range valueKeys {
		sortedValueKeys = append(sortedValueKeys, key)
	}

	sort.Strings(sortedValueKeys)

	rows := make([][]string, 0, len(sortedValueKeys))

	for _, valueKey := range sortedValueKeys {
		metricName := groupKey + "$" + valueKey
		row := make([]string, len(ids)+1)
		row[0] = metricName

		for i, id := range ids {
			if vals, ok := byID[id]; ok {
				row[i+1] = vals[valueKey]
			}
		}

		rows = append(rows, row)
	}

	return rows
}

func collectCompareValueKeys(byID map[int64]map[string]string) map[string]struct{} {
	valueKeys := make(map[string]struct{})

	for _, vals := range byID {
		for key := range vals {
			valueKeys[key] = struct{}{}
		}
	}

	return valueKeys
}
