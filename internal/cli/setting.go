package cli

import (
	"errors"
	"fmt"
	"sort"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newSettingCmd builds the "setting" parent command for server and project settings.
func newSettingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setting",
		Short: "Manage server and project settings",
	}

	cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.AddCommand(newSettingServerCmd())
	cmd.AddCommand(newSettingNotificationCmd())
	cmd.AddCommand(newSettingPatternCmd())
	cmd.AddCommand(newSettingIssueSubtypeCmd())

	return cmd
}

// newSettingServerCmd builds the "setting server" grouping command.
func newSettingServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Instance-level server settings (admin)",
	}
	cmd.AddCommand(newSettingServerGetCmd())
	cmd.AddCommand(newSettingServerUpdateCmd())

	return cmd
}

// newSettingServerGetCmd builds the "setting server get" subcommand.
func newSettingServerGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get instance-level server settings",
		Long: `Get instance-level server settings.

This command is instance-scoped and requires admin privileges. The --project flag is ignored.

Examples:
  rp setting server get
  rp setting server get -o json`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runSettingServerGet),
	}
}

// newSettingServerUpdateCmd builds the "setting server update" subcommand.
func newSettingServerUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an instance-level server setting",
		Long: `Update an instance-level server setting by key and value, or from a JSON file.

Examples:
  rp setting server update --key analytics.enabled --value true
  rp setting server update --from-file setting.json`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runSettingServerUpdate),
	}

	cmd.Flags().String("key", "", "setting key (required without --from-file)")
	cmd.Flags().String("value", "", "setting value (required without --from-file)")
	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing UpdateSettingsRq (use - for stdin)",
	)

	return cmd
}

func runSettingServerGet(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("getting server settings")

	settings, err := deps.client.Settings().GetServerSettings(cmd.Context())
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("getting server settings: %w", err)
	}

	headers := serverSettingHeaders()
	rows := serverSettingRows(settings)

	return deps.printer.Print(settings, headers, rows)
}

func runSettingServerUpdate(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	request, err := resolveServerSettingUpdateRequest(cmd, printer)
	if err != nil {
		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("updating server setting", "key", request.Key)

	result, err := deps.client.Settings().UpdateServerSettings(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating server setting %q: %w", request.Key, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func resolveServerSettingUpdateRequest(
	cmd *cobra.Command,
	printer *Printer,
) (types.UpdateSettingsRq, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		request, err := readRequestFromFile[types.UpdateSettingsRq](path)
		if err != nil {
			printer.PrintError(err)

			return types.UpdateSettingsRq{}, wrapFileError(
				err,
				"server setting",
				"server setting JSON",
			)
		}

		if request.Key == "" {
			err = errors.New("setting key must not be empty in --from-file payload")
			printer.PrintError(err)

			return types.UpdateSettingsRq{}, err
		}

		return request, nil
	}

	key, _ := cmd.Flags().GetString("key")
	value, _ := cmd.Flags().GetString("value")

	if key == "" || !cmd.Flags().Changed("key") {
		err := errors.New("--key is required when not using --from-file")
		printer.PrintError(err)

		return types.UpdateSettingsRq{}, err
	}

	if !cmd.Flags().Changed("value") {
		err := errors.New("--value is required when not using --from-file")
		printer.PrintError(err)

		return types.UpdateSettingsRq{}, err
	}

	return types.UpdateSettingsRq{Key: key, Value: value}, nil
}

func serverSettingHeaders() []string {
	return []string{"KEY", "VALUE"}
}

func serverSettingRows(settings map[string]string) [][]string {
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, settings[key]})
	}

	return rows
}
