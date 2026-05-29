package cli

import (
	"errors"
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newDashboardWidgetCmd builds the "dashboard widget" grouping command.
func newDashboardWidgetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "widget",
		Short: "Manage widgets on dashboards",
	}
	cmd.AddCommand(newDashboardWidgetAddCmd())
	cmd.AddCommand(newDashboardWidgetRemoveCmd())

	return cmd
}

// newDashboardWidgetAddCmd builds the "dashboard widget add" subcommand.
func newDashboardWidgetAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <dashboard-id>",
		Short: "Add a widget to a dashboard",
		Long: `Add a widget to a dashboard.

Flags or --from-file may be provided. When --from-file is given, flags are ignored.

Examples:
  rp dashboard widget add 5 --widget-id 10 --widget-name "Pass Rate" --widget-type statisticTrend -p my_project
  rp dashboard widget add 5 --widget-id 10 --widget-name "Pass Rate" --widget-type statisticTrend \
    --width 6 --height 4 --position-x 0 --position-y 0 -p my_project
  rp dashboard widget add 5 --from-file widget.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runDashboardWidgetAdd),
	}

	cmd.Flags().Int64("widget-id", 0, "widget ID to add (required without --from-file)")
	cmd.Flags().String("widget-name", "", "widget display name (required without --from-file)")
	cmd.Flags().String("widget-type", "", "widget type (required without --from-file)")
	cmd.Flags().Int32("width", 0, "widget width")
	cmd.Flags().Int32("height", 0, "widget height")
	cmd.Flags().Int32("position-x", 0, "widget X position")
	cmd.Flags().Int32("position-y", 0, "widget Y position")
	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing AddWidgetRq (use - for stdin)",
	)

	return cmd
}

// newDashboardWidgetRemoveCmd builds the "dashboard widget remove" subcommand.
func newDashboardWidgetRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <dashboard-id> <widget-id>",
		Short: "Remove a widget from a dashboard",
		Long: `Remove a widget from a dashboard.

Examples:
  rp dashboard widget remove 5 10 -p my_project`,
		Args: cobra.ExactArgs(2),
		RunE: withPrintedErrors(runDashboardWidgetRemove),
	}
}

func runDashboardWidgetAdd(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	dashboardID, err := parseDashboardID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	request, err := resolveAddWidgetRequest(cmd, printer)
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

	deps.logger.Debug(
		"adding widget to dashboard",
		"project", project,
		"dashboardId", dashboardID,
		"widgetId", request.AddWidget.WidgetId,
	)

	result, err := deps.client.Project(project).
		Dashboards().
		AddWidget(cmd.Context(), dashboardID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf(
			"adding widget %d to dashboard %d in project %s: %w",
			request.AddWidget.WidgetId,
			dashboardID,
			project,
			err,
		)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runDashboardWidgetRemove(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	dashboardID, err := parseDashboardID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	widgetID, err := parseIDArg(args[1], "widget")
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

	deps.logger.Debug(
		"removing widget from dashboard",
		"project", project,
		"dashboardId", dashboardID,
		"widgetId", widgetID,
	)

	result, err := deps.client.Project(project).
		Dashboards().
		RemoveWidget(cmd.Context(), dashboardID, widgetID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf(
			"removing widget %d from dashboard %d in project %s: %w",
			widgetID,
			dashboardID,
			project,
			err,
		)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// resolveAddWidgetRequest determines the AddWidgetRq from either --from-file or flags.
func resolveAddWidgetRequest(
	cmd *cobra.Command,
	printer *Printer,
) (types.AddWidgetRq, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		return loadAddWidgetRequest(path, printer)
	}

	return buildAddWidgetRequestFromFlags(cmd, printer)
}

func buildAddWidgetRequestFromFlags(
	cmd *cobra.Command,
	printer *Printer,
) (types.AddWidgetRq, error) {
	if !cmd.Flags().Changed("widget-id") &&
		!cmd.Flags().Changed("widget-name") &&
		!cmd.Flags().Changed("widget-type") {
		err := errors.New(
			"no input provided: use --widget-id, --widget-name, and --widget-type flags, " +
				"or --from-file to read from a JSON file",
		)
		printer.PrintError(err)

		return types.AddWidgetRq{}, err
	}

	widgetID, _ := cmd.Flags().GetInt64("widget-id")
	widgetName, _ := cmd.Flags().GetString("widget-name")
	widgetType, _ := cmd.Flags().GetString("widget-type")

	err := validateAddWidgetFlags(cmd, widgetID, widgetName, widgetType, printer)
	if err != nil {
		return types.AddWidgetRq{}, err
	}

	model := types.WidgetObjectModel{
		WidgetId:       widgetID,
		WidgetName:     &widgetName,
		WidgetType:     &widgetType,
		WidgetSize:     optionalWidgetSize(cmd),
		WidgetPosition: optionalWidgetPosition(cmd),
	}

	return types.AddWidgetRq{AddWidget: model}, nil
}

func validateAddWidgetFlags(
	cmd *cobra.Command,
	widgetID int64,
	widgetName, widgetType string,
	printer *Printer,
) error {
	if !cmd.Flags().Changed("widget-id") || widgetID == 0 {
		err := errors.New("--widget-id is required when not using --from-file")
		printer.PrintError(err)

		return err
	}

	if widgetName == "" {
		err := errors.New("--widget-name is required when not using --from-file")
		printer.PrintError(err)

		return err
	}

	if widgetType == "" {
		err := errors.New("--widget-type is required when not using --from-file")
		printer.PrintError(err)

		return err
	}

	return nil
}

func optionalWidgetSize(cmd *cobra.Command) *types.Size {
	if !cmd.Flags().Changed("width") && !cmd.Flags().Changed("height") {
		return nil
	}

	size := &types.Size{}

	if cmd.Flags().Changed("width") {
		width, _ := cmd.Flags().GetInt32("width")
		size.Width = &width
	}

	if cmd.Flags().Changed("height") {
		height, _ := cmd.Flags().GetInt32("height")
		size.Height = &height
	}

	return size
}

func optionalWidgetPosition(cmd *cobra.Command) *types.Position {
	if !cmd.Flags().Changed("position-x") && !cmd.Flags().Changed("position-y") {
		return nil
	}

	position := &types.Position{}

	if cmd.Flags().Changed("position-x") {
		positionX, _ := cmd.Flags().GetInt32("position-x")
		position.PositionX = &positionX
	}

	if cmd.Flags().Changed("position-y") {
		positionY, _ := cmd.Flags().GetInt32("position-y")
		position.PositionY = &positionY
	}

	return position
}

func loadAddWidgetRequest(path string, printer *Printer) (types.AddWidgetRq, error) {
	request, err := readRequestFromFile[types.AddWidgetRq](path)
	if err != nil {
		printer.PrintError(err)

		return types.AddWidgetRq{}, wrapFileError(
			err,
			"add widget definition",
			"add widget JSON",
		)
	}

	return request, nil
}
