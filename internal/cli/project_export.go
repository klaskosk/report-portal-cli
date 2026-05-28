package cli

import (
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/spf13/cobra"
)

// newProjectExportCmd builds the "project export" subcommand.
func newProjectExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export projects as CSV",
		Long: `Export project information as CSV (admin only).

If --output-file is specified, the export is written to that path and a
confirmation is printed to stderr. Otherwise, raw bytes are written to stdout
(pipe-friendly: rp project export > projects.csv).

The -o/--output format flags do not affect the export payload.

Examples:
  rp project export > projects.csv
  rp project export --output-file projects.csv
  rp project export --name my_project --type INTERNAL`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runProjectExport),
	}

	cmd.Flags().String("output-file", "", "write to file instead of stdout")
	cmd.Flags().String("view", "", "export format view (e.g. csv)")
	cmd.Flags().String("name", "", "filter by exact project name")
	cmd.Flags().String("type", "", "filter by entry type (e.g. INTERNAL)")
	cmd.Flags().String("last-run", "", "filter by last run date (YYYY-MM-DD)")

	return cmd
}

func runProjectExport(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	options, err := resolveProjectExportOptions(cmd)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("invalid project export flags: %w", err)
	}

	deps.logger.Debug("exporting projects")

	data, err := deps.client.Projects().Export(cmd.Context(), options...)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("exporting projects: %w", err)
	}

	return writeDownloadedContent(cmd, &client.FileContent{Data: data})
}

func resolveProjectExportOptions(cmd *cobra.Command) ([]client.ExportProjectsOption, error) {
	var options []client.ExportProjectsOption

	view, _ := cmd.Flags().GetString("view")
	if view != "" {
		options = append(options, client.WithExportView(view))
	}

	name, _ := cmd.Flags().GetString("name")
	if name != "" {
		options = append(options, client.WithExportFilterName(name))
	}

	projectType, _ := cmd.Flags().GetString("type")
	if projectType != "" {
		options = append(options, client.WithExportFilterType(projectType))
	}

	lastRun, _ := cmd.Flags().GetString("last-run")
	if lastRun != "" {
		parsed, err := time.Parse(time.DateOnly, lastRun)
		if err != nil {
			return nil, fmt.Errorf("invalid --last-run %q: expected YYYY-MM-DD: %w", lastRun, err)
		}

		options = append(options, client.WithExportFilterLastRun(openapi_types.Date{Time: parsed}))
	}

	return options, nil
}
