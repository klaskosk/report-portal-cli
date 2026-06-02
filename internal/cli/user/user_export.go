package user

import (
	"fmt"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/spf13/cobra"
)

// newUserExportCmd builds the "user export" subcommand.
func newUserExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export users as CSV",
		Long: `Export user information as CSV (admin only).

If --output-file is specified, the export is written to that path and a
confirmation is printed to stderr. Otherwise, raw bytes are written to stdout
(pipe-friendly: rp user export > users.csv).

The -o/--output format flags do not affect the export payload.

Examples:
  rp user export > users.csv
  rp user export --output-file users.csv
  rp user export --role ADMINISTRATOR --active`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runUserExport),
	}

	cmd.Flags().String("output-file", "", "write to file instead of stdout")
	registerUserListFilterFlags(cmd)

	return cmd
}

func runUserExport(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	options := resolveUserExportOptions(cmd)

	deps.Logger.Debug("exporting users")

	data, err := deps.Client.Users().Export(cmd.Context(), options...)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("exporting users: %w", err)
	}

	return core.WriteDownloadedContent(cmd, &client.FileContent{Data: data})
}

func resolveUserExportOptions(cmd *cobra.Command) []client.ListUsersOption {
	return userFilterClientOptions(resolveUserListFilterFlags(cmd))
}
