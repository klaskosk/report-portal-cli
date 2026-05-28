package cli

import (
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/spf13/cobra"
)

// newLaunchReportCmd builds the "launch report" grouping command.
func newLaunchReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Access launch reports",
	}
	cmd.AddCommand(newLaunchReportGetCmd())

	return cmd
}

// newLaunchReportGetCmd builds the "launch report get <id>" subcommand.
func newLaunchReportGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Download a launch report by numeric ID",
		Long: `Download the launch report for a launch using the server default view
(typically HTML). PDF and XLS views are not selectable yet; --view is not exposed.

If --output-file is specified, the report is written to that path and a
confirmation is printed to stderr. Otherwise, raw bytes are written to stdout
(pipe-friendly: rp launch report get 42 > report.html).

The -o/--output format flags do not affect the report payload.

Examples:
  rp launch report get 42 -p my_project > report.html
  rp launch report get 42 -p my_project --output-file report.html`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runLaunchReportGet),
	}

	cmd.Flags().String("output-file", "", "write to file instead of stdout")

	return cmd
}

func runLaunchReportGet(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	launchID, err := parseIDArg(args[0], "launch")
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

	deps.logger.Debug("getting launch report", "project", project, "id", launchID)

	report, err := deps.client.Project(project).Launches().GetReport(cmd.Context(), launchID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("getting launch report %d: %w", launchID, err)
	}

	return writeDownloadedContent(cmd, &client.FileContent{Data: report})
}
