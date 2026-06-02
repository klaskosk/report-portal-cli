package launch

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newLaunchMergeCmd builds the "launch merge" subcommand.
func newLaunchMergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge multiple launches into one",
		Long: `Merge two or more launches into a single launch.

Examples:
  rp launch merge --launch 41 --launch 42 --name "Merged Nightly" -p my_project
  rp launch merge --launch 41 --launch 42 --name "Merged" --merge-type DEEP -p my_project
  rp launch merge --from-file merge.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runLaunchMerge),
	}

	cmd.Flags().Int64Slice("launch", nil, "launch ID to merge (repeatable, at least 2 required)")
	cmd.Flags().String("name", "", "name for the merged launch (required without --from-file)")
	cmd.Flags().String("description", "", "description for the merged launch")
	cmd.Flags().String("merge-type", "BASIC", "merge type: BASIC or DEEP")
	cmd.Flags().String("mode", "", "launch mode: DEFAULT or DEBUG")
	cmd.Flags().Bool("extend-suites", false, "extend suites description")
	cmd.Flags().StringArray("attribute", nil, "attribute in key:value format (repeatable)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing MergeLaunchesRQ (use - for stdin)")

	return cmd
}

func runLaunchMerge(cmd *cobra.Command, _ []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	request, err := buildMergeLaunchesRequest(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	result, err := deps.Client.Project(project).Launches().Merge(cmd.Context(), request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("merging launches in project %s: %w", project, err)
	}

	headers, row := launchRow(result)

	return deps.Printer.Print(result, headers, [][]string{row})
}

func buildMergeLaunchesRequest(cmd *cobra.Command) (types.MergeLaunchesRQ, error) {
	if cmd.Flags().Changed("from-file") {
		return readMergeLaunchesRequestFromFile(cmd)
	}

	return buildMergeLaunchesRequestFromFlags(cmd)
}

func readMergeLaunchesRequestFromFile(cmd *cobra.Command) (types.MergeLaunchesRQ, error) {
	path, _ := cmd.Flags().GetString("from-file")

	req, err := core.ReadRequestFromFile[types.MergeLaunchesRQ](path)
	if err != nil {
		return types.MergeLaunchesRQ{}, core.WrapFileError(
			err,
			"launch merge file",
			"launch merge JSON",
		)
	}

	return req, nil
}

func buildMergeLaunchesRequestFromFlags(cmd *cobra.Command) (types.MergeLaunchesRQ, error) {
	launches, err := resolveMergeLaunchIDs(cmd)
	if err != nil {
		return types.MergeLaunchesRQ{}, err
	}

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return types.MergeLaunchesRQ{}, errors.New("--name is required when not using --from-file")
	}

	mergeTypeRaw, _ := cmd.Flags().GetString("merge-type")

	mergeType, err := resolveMergeLaunchesMergeType(mergeTypeRaw)
	if err != nil {
		return types.MergeLaunchesRQ{}, err
	}

	var mode *types.MergeLaunchesRQMode

	if cmd.Flags().Changed("mode") {
		modeRaw, _ := cmd.Flags().GetString("mode")

		m, modeErr := resolveMergeLaunchesMode(modeRaw)
		if modeErr != nil {
			return types.MergeLaunchesRQ{}, modeErr
		}

		mode = &m
	}

	extendSuites, _ := cmd.Flags().GetBool("extend-suites")

	var attrs *[]types.ItemAttributeResource

	if cmd.Flags().Changed("attribute") {
		rawAttrs, _ := cmd.Flags().GetStringArray("attribute")

		resources, buildErr := core.BuildAttributeResources(rawAttrs)
		if buildErr != nil {
			return types.MergeLaunchesRQ{}, buildErr
		}

		attrs = &resources
	}

	return types.MergeLaunchesRQ{
		Launches:                launches,
		Name:                    name,
		Description:             core.OptionalStringFlag(cmd, "description"),
		MergeType:               mergeType,
		Mode:                    mode,
		ExtendSuitesDescription: extendSuites,
		Attributes:              attrs,
	}, nil
}

func resolveMergeLaunchIDs(cmd *cobra.Command) ([]int64, error) {
	launches, _ := cmd.Flags().GetInt64Slice("launch")
	if len(launches) < 2 {
		return nil, fmt.Errorf(
			"at least two launch IDs are required for merge (got %d)",
			len(launches),
		)
	}

	return launches, nil
}

func resolveMergeLaunchesMergeType(raw string) (types.MergeLaunchesRQMergeType, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))

	switch normalized {
	case string(types.MergeLaunchesRQMergeTypeBASIC), string(types.MergeLaunchesRQMergeTypeDEEP):
		return types.MergeLaunchesRQMergeType(normalized), nil
	default:
		return "", fmt.Errorf("invalid merge-type %q: must be BASIC or DEEP", raw)
	}
}

func resolveMergeLaunchesMode(raw string) (types.MergeLaunchesRQMode, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))

	switch normalized {
	case string(types.MergeLaunchesRQModeDEBUG), string(types.MergeLaunchesRQModeDEFAULT):
		return types.MergeLaunchesRQMode(normalized), nil
	default:
		return "", fmt.Errorf("invalid mode %q: must be DEBUG or DEFAULT", raw)
	}
}
