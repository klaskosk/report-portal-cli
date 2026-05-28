package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newDashboardCmd builds the "dashboard" parent command that groups all dashboard subcommands.
func newDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Manage dashboards",
	}

	cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.AddCommand(newDashboardListCmd())
	cmd.AddCommand(newDashboardGetCmd())
	cmd.AddCommand(newDashboardCreateCmd())
	cmd.AddCommand(newDashboardUpdateCmd())
	cmd.AddCommand(newDashboardDeleteCmd())
	cmd.AddCommand(newDashboardConfigCmd())
	cmd.AddCommand(newDashboardPreconfiguredCmd())

	return cmd
}

// newDashboardConfigCmd builds the "dashboard config" grouping command.
func newDashboardConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Access dashboard widget configuration",
	}
	cmd.AddCommand(newDashboardConfigGetCmd())

	return cmd
}

// newDashboardConfigGetCmd builds the "dashboard config get <id>" subcommand.
func newDashboardConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get dashboard widget configuration by numeric ID",
		Long: `Get the widget layout configuration for a dashboard.

The table view lists one row per widget. Use -o json for the full DashboardConfigResource.

Examples:
  rp dashboard config get 42                 Get widget configuration
  rp dashboard config get 42 -o json         Get configuration as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runDashboardConfigGet),
	}
}

// newDashboardPreconfiguredCmd builds the "dashboard preconfigured" grouping command.
func newDashboardPreconfiguredCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preconfigured",
		Short: "Create dashboards from preconfigured templates",
	}
	cmd.AddCommand(newDashboardPreconfiguredCreateCmd())

	return cmd
}

// newDashboardPreconfiguredCreateCmd builds the "dashboard preconfigured create" subcommand.
func newDashboardPreconfiguredCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a dashboard from a preconfigured template file",
		Long: `Create a dashboard from a JSON file describing a preconfigured dashboard.

Examples:
  rp dashboard preconfigured create --from-file dash.json
  rp dashboard preconfigured create --from-file dash.json --name "My Dashboard"`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runDashboardPreconfiguredCreate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing DashboardPreconfiguredRq (use - for stdin)",
	)
	cmd.Flags().String("name", "", "override dashboard name from file")
	cmd.Flags().String("description", "", "override dashboard description from file")
	_ = cmd.MarkFlagRequired("from-file")

	return cmd
}

// newDashboardListCmd builds the "dashboard list" subcommand.
func newDashboardListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List dashboards in a project",
		Long: `List dashboards in a project.

Examples:
  rp dashboard list                          List dashboards
  rp dashboard list --page-size 25           List with custom page size`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runDashboardList),
	}

	registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: nil})

	return cmd
}

// newDashboardGetCmd builds the "dashboard get" subcommand.
func newDashboardGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a dashboard by numeric ID",
		Long: `Get a dashboard by numeric ID.

Examples:
  rp dashboard get 42                        Get dashboard by ID
  rp dashboard get 42 -o json                Get dashboard as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runDashboardGet),
	}
}

// newDashboardCreateCmd builds the "dashboard create" subcommand.
func newDashboardCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a dashboard",
		Long: `Create a dashboard.

Examples:
  rp dashboard create --name "My Dashboard"                          Create a dashboard
  rp dashboard create --name "My Dashboard" --description "desc"     Create with description`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runDashboardCreate),
	}

	cmd.Flags().String("name", "", "dashboard name (required)")
	cmd.Flags().String("description", "", "dashboard description")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

// newDashboardUpdateCmd builds the "dashboard update" subcommand.
func newDashboardUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a dashboard",
		Long: `Update a dashboard.

If --name is omitted, the current dashboard name is preserved automatically.

Examples:
  rp dashboard update 42 --name "New Name"            Rename a dashboard
  rp dashboard update 42 --description "New desc"     Update description`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runDashboardUpdate),
	}

	cmd.Flags().String("name", "", "updated name")
	cmd.Flags().String("description", "", "updated description")

	return cmd
}

// newDashboardDeleteCmd builds the "dashboard delete" subcommand.
func newDashboardDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a dashboard by numeric ID",
		Long: `Delete a dashboard by numeric ID.

Examples:
  rp dashboard delete 42                     Delete a dashboard`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runDashboardDelete),
	}
}

func runDashboardList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing dashboards", "project", project)

	return listDashboards(cmd, deps.client, project, deps.printer)
}

//nolint:dupl // structural similarity with listFilters is inherent to the project-scoped list pattern
func listDashboards(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	printer *Printer,
) error {
	flags, err := resolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid list flags: %w", err)
	}

	options := []client.ListDashboardsOption{
		client.WithDashboardPageNumber(flags.page),
		client.WithDashboardPageSize(flags.pageSize),
		client.WithDashboardSort(flags.sort...),
	}

	result, err := clientInstance.Project(project).Dashboards().List(cmd.Context(), options...)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing dashboards: %w", err)
	}

	var dashboards []types.DashboardResource
	if result.Content != nil {
		dashboards = *result.Content
	}

	headers := dashboardHeaders()
	rows := make([][]string, 0, len(dashboards))

	for i := range dashboards {
		_, row := dashboardRow(&dashboards[i])
		rows = append(rows, row)
	}

	return printer.Print(dashboards, headers, rows)
}

func runDashboardGet(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	dashboardID, err := parseDashboardID(args[0])
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

	deps.logger.Debug("getting dashboard", "project", project, "id", dashboardID)

	result, err := deps.client.Project(project).Dashboards().GetByID(cmd.Context(), dashboardID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("getting dashboard %d: %w", dashboardID, err)
	}

	headers, row := dashboardRow(result)

	return deps.printer.Print(result, headers, [][]string{row})
}

func runDashboardCreate(cmd *cobra.Command, _ []string) error {
	name := mustGetStringFlag(cmd, "name")
	if name == "" {
		err := errors.New("--name must not be empty")
		resolvePrinter(cmd).PrintError(err)

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

	request := types.CreateDashboardRQ{
		Name:        name,
		Description: optionalStringFlag(cmd, "description"),
	}

	deps.logger.Debug("creating dashboard", "project", project, "name", request.Name)

	result, err := deps.client.Project(project).Dashboards().Create(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating dashboard in project %s: %w", project, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{formatOptional(result.Id)}}

	return deps.printer.Print(result, headers, rows)
}

func runDashboardUpdate(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	dashboardID, err := parseDashboardID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("description") {
		noChangeErr := errors.New("at least one flag (--name, --description) must be provided")
		printer.PrintError(noChangeErr)

		return noChangeErr
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	scope := deps.client.Project(project).Dashboards()

	name, err := resolveDashboardUpdateName(cmd, scope, dashboardID)
	if err != nil {
		deps.printer.PrintError(err)

		return err
	}

	request := types.UpdateDashboardRQ{
		Name:        name,
		Description: optionalStringFlag(cmd, "description"),
	}

	deps.logger.Debug("updating dashboard", "project", project, "id", dashboardID)

	result, err := scope.Update(cmd.Context(), dashboardID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating dashboard %d in project %s: %w", dashboardID, project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// resolveDashboardUpdateName returns the name to use for a dashboard update.
// If --name was explicitly set, it uses that value. Otherwise it fetches the current dashboard
// to preserve the existing name.
func resolveDashboardUpdateName(
	cmd *cobra.Command,
	scope *client.DashboardScope,
	dashboardID int64,
) (string, error) {
	if cmd.Flags().Changed("name") {
		name := mustGetStringFlag(cmd, "name")
		if name == "" {
			return "", errors.New("--name must not be empty")
		}

		return name, nil
	}

	current, err := scope.GetByID(cmd.Context(), dashboardID)
	if err != nil {
		return "", fmt.Errorf("fetching dashboard %d to preserve name: %w", dashboardID, err)
	}

	return current.Name, nil
}

func runDashboardDelete(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	dashboardID, err := parseDashboardID(args[0])
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

	deps.logger.Debug("deleting dashboard", "project", project, "id", dashboardID)

	result, err := deps.client.Project(project).Dashboards().Delete(cmd.Context(), dashboardID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("deleting dashboard %d in project %s: %w", dashboardID, project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// dashboardHeaders returns the table column headers for dashboard rows.
func dashboardHeaders() []string {
	return []string{"ID", "NAME", "OWNER", "DESCRIPTION"}
}

// dashboardRow formats a DashboardResource as a table row, returning both headers and column values.
func dashboardRow(dashboard *types.DashboardResource) ([]string, []string) {
	description := formatOptional(dashboard.Description)
	if len(description) > 60 {
		description = description[:60] + "..."
	}

	return dashboardHeaders(), []string{
		strconv.FormatInt(dashboard.Id, 10),
		dashboard.Name,
		formatOptional(dashboard.Owner),
		description,
	}
}

// parseDashboardID parses a dashboard ID from CLI input, which must be numeric.
func parseDashboardID(raw string) (int64, error) {
	return parseIDArg(raw, "dashboard")
}

func runDashboardConfigGet(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	dashboardID, err := parseDashboardID(args[0])
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

	deps.logger.Debug("getting dashboard config", "project", project, "id", dashboardID)

	config, err := deps.client.Project(project).Dashboards().GetConfig(cmd.Context(), dashboardID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("getting dashboard config %d: %w", dashboardID, err)
	}

	headers := dashboardConfigHeaders()
	rows := dashboardConfigRows(config)

	return deps.printer.Print(config, headers, rows)
}

func runDashboardPreconfiguredCreate(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	path, err := requireFromFile(
		cmd,
		printer,
		"--from-file is required for preconfigured dashboard creation",
	)
	if err != nil {
		return err
	}

	request, err := loadDashboardPreconfiguredRequest(cmd, printer, path)
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

	deps.logger.Debug("creating preconfigured dashboard", "project", project, "name", request.Name)

	result, err := deps.client.Project(project).
		Dashboards().
		CreatePreconfigured(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating preconfigured dashboard in project %s: %w", project, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{formatOptional(result.Id)}}

	return deps.printer.Print(result, headers, rows)
}

func loadDashboardPreconfiguredRequest(
	cmd *cobra.Command,
	printer *Printer,
	path string,
) (types.DashboardPreconfiguredRq, error) {
	request, err := readRequestFromFile[types.DashboardPreconfiguredRq](path)
	if err != nil {
		printer.PrintError(err)

		return types.DashboardPreconfiguredRq{}, wrapFileError(
			err,
			"preconfigured dashboard definition",
			"preconfigured dashboard JSON",
		)
	}

	applyDashboardPreconfiguredFlagOverrides(cmd, &request)

	if request.Name == "" {
		nameErr := errors.New(
			"dashboard name is required (provide via --from-file or --name flag)",
		)
		printer.PrintError(nameErr)

		return types.DashboardPreconfiguredRq{}, nameErr
	}

	return request, nil
}

func applyDashboardPreconfiguredFlagOverrides(
	cmd *cobra.Command,
	request *types.DashboardPreconfiguredRq,
) {
	if name := optionalStringFlag(cmd, "name"); name != nil {
		request.Name = *name
	}

	if description := optionalStringFlag(cmd, "description"); description != nil {
		request.Description = description
	}
}

func dashboardConfigHeaders() []string {
	return []string{"WIDGET ID", "WIDGET NAME", "WIDGET TYPE", "POSITION", "SIZE"}
}

func dashboardConfigRows(config *types.DashboardConfigResource) [][]string {
	if config.Widgets == nil || len(*config.Widgets) == 0 {
		return nil
	}

	rows := make([][]string, 0, len(*config.Widgets))

	for i := range *config.Widgets {
		widget := &(*config.Widgets)[i]
		rows = append(rows, dashboardConfigRow(widget))
	}

	return rows
}

func dashboardConfigRow(widget *types.WidgetConfigResource) []string {
	widgetName := formatOptional(widget.WidgetObject.WidgetName)
	if widgetName == "" {
		widgetName = widget.WidgetResource.Name
	}

	return []string{
		strconv.FormatInt(widget.WidgetObject.WidgetId, 10),
		widgetName,
		formatOptional(widget.WidgetObject.WidgetType),
		formatWidgetPosition(widget.WidgetObject.WidgetPosition),
		formatWidgetSize(widget.WidgetObject.WidgetSize),
	}
}

func formatWidgetPosition(position *types.Position) string {
	if position == nil {
		return ""
	}

	return strings.TrimSpace(fmt.Sprintf(
		"%s,%s",
		formatOptional(position.PositionX),
		formatOptional(position.PositionY),
	))
}

func formatWidgetSize(size *types.Size) string {
	if size == nil {
		return ""
	}

	return strings.TrimSpace(fmt.Sprintf(
		"%sx%s",
		formatOptional(size.Width),
		formatOptional(size.Height),
	))
}
