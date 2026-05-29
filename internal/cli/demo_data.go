package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newDemoDataCmd builds the "demo-data" parent command.
func newDemoDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo-data",
		Short: "Generate demo data",
	}
	cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.AddCommand(newDemoDataGenerateCmd())

	return cmd
}

// newDemoDataGenerateCmd builds the "demo-data generate" subcommand.
func newDemoDataGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate demo data in a project",
		Long: `Generate demo data (launches and optionally a dashboard) in a project.

If --from-file is omitted, an empty request is sent and the server applies defaults.

Examples:
  rp demo-data generate -p my_project
  rp demo-data generate --from-file demo_config.json -p my_project
  rp demo-data generate -p my_project -o json`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runDemoDataGenerate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing DemoDataRq payload (use - for stdin)",
	)

	return cmd
}

func runDemoDataGenerate(cmd *cobra.Command, _ []string) error {
	request, err := loadDemoDataRequest(cmd)
	if err != nil {
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

	deps.logger.Debug("generating demo data", "project", project)

	result, err := deps.client.Project(project).DemoData().Generate(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("generating demo data in project %s: %w", project, err)
	}

	headers, rows := demoDataGenerateTable(result)

	return deps.printer.Print(result, headers, rows)
}

func loadDemoDataRequest(cmd *cobra.Command) (types.DemoDataRq, error) {
	if !cmd.Flags().Changed("from-file") {
		return types.DemoDataRq{}, nil
	}

	path, _ := cmd.Flags().GetString("from-file")

	request, err := readRequestFromFile[types.DemoDataRq](path)
	if err != nil {
		printer := resolvePrinter(cmd)
		printer.PrintError(err)

		return types.DemoDataRq{}, wrapFileError(err, "demo data request", "demo data JSON")
	}

	return request, nil
}

func demoDataGenerateTable(result *types.DemoDataRs) ([]string, [][]string) {
	headers := []string{"RESOURCE", "IDS"}
	rows := make([][]string, 0, 2)

	if result.LaunchIds != nil {
		rows = append(rows, []string{"Launches", formatInt64Slice(result.LaunchIds)})
	}

	if result.DashboardId != nil {
		rows = append(rows, []string{"Dashboard", formatOptional(result.DashboardId)})
	}

	if len(rows) == 0 {
		return headers, [][]string{{"STATUS", "Demo data generated successfully"}}
	}

	return headers, rows
}

func formatInt64Slice(ids *[]int64) string {
	if ids == nil {
		return ""
	}

	if len(*ids) == 0 {
		return "[]"
	}

	parts := make([]string, len(*ids))
	for i, id := range *ids {
		parts[i] = strconv.FormatInt(id, 10)
	}

	return "[" + strings.Join(parts, ", ") + "]"
}
