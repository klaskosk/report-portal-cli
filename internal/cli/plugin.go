package cli

import (
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newPluginCmd builds the "plugin" parent command that groups all plugin subcommands.
// All plugin operations require admin privileges.
func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins (admin)",
		Long:  "Manage installed plugins. All operations require admin privileges.",
	}
	cmd.AddCommand(newPluginListCmd())
	cmd.AddCommand(newPluginUpdateCmd())
	cmd.AddCommand(newPluginDeleteCmd())

	return cmd
}

// newPluginListCmd builds the "plugin list" subcommand.
func newPluginListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		Long: `List all installed plugins.

Examples:
  rp plugin list                             List all installed plugins
  rp plugin list -o json                     List plugins as JSON`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runPluginList),
	}
}

// newPluginUpdateCmd builds the "plugin update" subcommand.
func newPluginUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Enable or disable a plugin",
		Long: `Enable or disable a plugin by numeric ID.

The --enabled flag is required and must be provided explicitly to
make the intent clear (--enabled=true to enable, --enabled=false to disable).

Examples:
  rp plugin update 4 --enabled=true          Enable a plugin
  rp plugin update 4 --enabled=false         Disable a plugin`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runPluginUpdate),
	}

	cmd.Flags().Bool("enabled", false, "set the plugin's enabled state (required)")
	_ = cmd.MarkFlagRequired("enabled")

	return cmd
}

// newPluginDeleteCmd builds the "plugin delete" subcommand.
func newPluginDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a plugin by numeric ID",
		Long: `Delete a plugin by numeric ID.

Examples:
  rp plugin delete 4                         Delete a plugin`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runPluginDelete),
	}
}

func runPluginList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing plugins")

	plugins, err := deps.client.Plugins().List(cmd.Context())
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("listing plugins: %w", err)
	}

	headers := pluginHeaders()
	rows := make([][]string, 0, len(plugins))

	for idx := range plugins {
		_, row := pluginRow(&plugins[idx])
		rows = append(rows, row)
	}

	return deps.printer.Print(plugins, headers, rows)
}

func runPluginUpdate(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	pluginID, err := parseIDArg(args[0], "plugin")
	if err != nil {
		printer.PrintError(err)

		return err
	}

	enabled, _ := cmd.Flags().GetBool("enabled")

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("updating plugin state", "id", pluginID, "enabled", enabled)

	request := types.UpdatePluginStateRQ{Enabled: &enabled}

	result, err := deps.client.Plugins().UpdateState(cmd.Context(), pluginID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating plugin %d: %w", pluginID, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runPluginDelete(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	pluginID, err := parseIDArg(args[0], "plugin")
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("deleting plugin", "id", pluginID)

	result, err := deps.client.Plugins().Delete(cmd.Context(), pluginID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("deleting plugin %d: %w", pluginID, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// pluginHeaders returns uppercase table headers for plugin rows.
func pluginHeaders() []string {
	return []string{"ID", "NAME", "GROUP", "ENABLED", "PLUGIN TYPE"}
}

// pluginRow formats a plugin resource as a table row, returning both headers and column values.
func pluginRow(plugin *types.IntegrationTypeResource) ([]string, []string) {
	return pluginHeaders(), []string{
		formatOptional(plugin.Type),
		formatOptional(plugin.Name),
		formatOptional(plugin.GroupType),
		formatOptional(plugin.Enabled),
		formatOptional(plugin.PluginType),
	}
}
