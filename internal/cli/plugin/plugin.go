package plugin

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewCommand builds the "plugin" parent command that groups all plugin subcommands.
// All plugin operations require admin privileges.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Annotations: map[string]string{core.GroupAnnotationKey: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		Use:   "plugin",
		Short: "Manage plugins (admin)",
		Long:  "Manage installed plugins. All operations require admin privileges.",
	}
	cmd.AddCommand(newPluginListCmd())
	cmd.AddCommand(newPluginUpdateCmd())
	cmd.AddCommand(newPluginDeleteCmd())
	cmd.AddCommand(newPluginExecCmd())
	cmd.AddCommand(newPluginCreateCmd())

	return cmd
}

// newPluginExecCmd builds the "plugin exec" subcommand.
func newPluginExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <plugin-name> <command>",
		Short: "Execute a plugin command in a project",
		Long: `Execute a named command on a plugin within a project context.

If --from-file is omitted, an empty JSON object ({}) is sent as the request body.

Examples:
  rp plugin exec my-plugin run -p my_project
  rp plugin exec my-plugin run -p my_project --from-file body.json
  rp plugin exec my-plugin run -p my_project -o json`,
		Args: cobra.ExactArgs(2),
		RunE: core.WithPrintedErrors(runPluginExec),
	}

	cmd.Flags().StringP("project", "p", "", "project name (required)")
	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing the command request body (use - for stdin)",
	)
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

// newPluginCreateCmd builds the "plugin create" subcommand.
func newPluginCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Upload and install a plugin archive",
		Long: `Upload a plugin archive file to install a new plugin.

Examples:
  rp plugin create --file my-plugin.jar`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runPluginCreate),
	}

	cmd.Flags().String("file", "", "path to the plugin archive file (required)")
	_ = cmd.MarkFlagRequired("file")

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
		RunE: core.WithPrintedErrors(runPluginList),
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
		RunE: core.WithPrintedErrors(runPluginUpdate),
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
		RunE: core.WithPrintedErrors(runPluginDelete),
	}
}

func runPluginList(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("listing plugins")

	plugins, err := deps.Client.Plugins().List(cmd.Context())
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("listing plugins: %w", err)
	}

	headers := pluginHeaders()
	rows := make([][]string, 0, len(plugins))

	for idx := range plugins {
		_, row := pluginRow(&plugins[idx])
		rows = append(rows, row)
	}

	return deps.Printer.Print(plugins, headers, rows)
}

func runPluginUpdate(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	pluginID, err := core.ParseIDArg(args[0], "plugin")
	if err != nil {
		printer.PrintError(err)

		return err
	}

	enabled, _ := cmd.Flags().GetBool("enabled")

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("updating plugin state", "id", pluginID, "enabled", enabled)

	request := types.UpdatePluginStateRQ{Enabled: &enabled}

	result, err := deps.Client.Plugins().UpdateState(cmd.Context(), pluginID, request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("updating plugin %d: %w", pluginID, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func runPluginDelete(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	pluginID, err := core.ParseIDArg(args[0], "plugin")
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("deleting plugin", "id", pluginID)

	result, err := deps.Client.Plugins().Delete(cmd.Context(), pluginID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("deleting plugin %d: %w", pluginID, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func runPluginExec(cmd *cobra.Command, args []string) error {
	pluginName := args[0]
	commandName := args[1]

	body, err := loadPluginExecBody(cmd)
	if err != nil {
		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project := core.MustGetStringFlag(cmd, "project")
	if project == "" {
		projectErr := errors.New("--project is required for plugin exec")
		deps.Printer.PrintError(projectErr)

		return projectErr
	}

	deps.Logger.Debug(
		"executing plugin command",
		"project", project,
		"plugin", pluginName,
		"command", commandName,
	)

	result, err := deps.Client.Plugins().ExecuteCommand(
		cmd.Context(), project, pluginName, commandName, body,
	)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf(
			"executing plugin command %s/%s in project %s: %w",
			pluginName, commandName, project, err,
		)
	}

	headers, rows := pluginExecTable(result)

	return deps.Printer.Print(result, headers, rows)
}

// pluginExecTable formats a plugin command response for table output.
// An empty response shows a single success status row; otherwise keys are listed.
func pluginExecTable(result map[string]any) ([]string, [][]string) {
	if len(result) == 0 {
		return []string{"STATUS"}, [][]string{{"Command executed successfully"}}
	}

	keys := make([]string, 0, len(result))
	for key := range result {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, fmt.Sprintf("%v", result[key])})
	}

	return []string{"KEY", "VALUE"}, rows
}

func loadPluginExecBody(cmd *cobra.Command) (map[string]any, error) {
	if !cmd.Flags().Changed("from-file") {
		return map[string]any{}, nil
	}

	path, _ := cmd.Flags().GetString("from-file")

	body, err := core.ReadRequestFromFile[map[string]any](path)
	if err != nil {
		printer := core.NewPrinterFromCmd(cmd)
		printer.PrintError(err)

		return nil, core.WrapFileError(err, "plugin command body", "plugin command JSON")
	}

	if body == nil {
		return map[string]any{}, nil
	}

	return body, nil
}

func runPluginCreate(cmd *cobra.Command, _ []string) error {
	filePath := core.MustGetStringFlag(cmd, "file")

	//nolint:gosec // path is user-provided --file argument
	data, err := os.ReadFile(filePath)
	if err != nil {
		readErr := fmt.Errorf("reading plugin file %s: %w", filePath, err)
		core.NewPrinterFromCmd(cmd).PrintError(readErr)

		return readErr
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	filename := filepath.Base(filePath)

	deps.Logger.Debug("uploading plugin", "file", filename)

	result, err := deps.Client.Plugins().Upload(
		cmd.Context(), filename, bytes.NewReader(data),
	)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("uploading plugin %s: %w", filename, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{core.FormatOptional(result.Id)}}

	return deps.Printer.Print(result, headers, rows)
}

// pluginHeaders returns uppercase table headers for plugin rows.
func pluginHeaders() []string {
	return []string{"ID", "NAME", "GROUP", "ENABLED", "PLUGIN TYPE"}
}

// pluginRow formats a plugin resource as a table row, returning both headers and column values.
func pluginRow(plugin *types.IntegrationTypeResource) ([]string, []string) {
	return pluginHeaders(), []string{
		core.FormatOptional(plugin.Type),
		core.FormatOptional(plugin.Name),
		core.FormatOptional(plugin.GroupType),
		core.FormatOptional(plugin.Enabled),
		core.FormatOptional(plugin.PluginType),
	}
}
