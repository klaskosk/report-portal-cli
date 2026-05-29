package widget

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewCommand builds the "widget" parent command that groups all widget subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Annotations: map[string]string{core.GroupAnnotationKey: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		Use:   "widget",
		Short: "Manage widgets",
	}
	core.RegisterProjectFlag(cmd)
	cmd.AddCommand(newWidgetGetCmd())
	cmd.AddCommand(newWidgetCreateCmd())
	cmd.AddCommand(newWidgetUpdateCmd())
	cmd.AddCommand(newWidgetDataCmd())

	return cmd
}

// newWidgetDataCmd builds the "widget data" grouping command for widget data sub-resources.
func newWidgetDataCmd() *cobra.Command {
	cmd := core.MarkGroupCommand(&cobra.Command{
		Use:   "data",
		Short: "Access computed widget data",
	})
	cmd.AddCommand(newWidgetDataGetCmd())

	return cmd
}

// newWidgetGetCmd builds the "widget get <id>" subcommand.
func newWidgetGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a widget definition by numeric ID",
		Long: `Get a widget definition by numeric ID.

Examples:
  rp widget get 42                           Get widget definition
  rp widget get 42 -o json                   Get widget as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runWidgetGet),
	}
}

// newWidgetCreateCmd builds the "widget create" subcommand.
func newWidgetCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a widget from a JSON file",
		Long: `Create a widget from a JSON file describing the widget definition.

Examples:
  rp widget create --from-file widget.json   Create a widget from file`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runWidgetCreate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing WidgetRQ (use - for stdin; flags override file values)",
	)
	cmd.Flags().String("name", "", "override widget name from file")
	cmd.Flags().String("description", "", "override widget description from file")

	return cmd
}

// newWidgetUpdateCmd builds the "widget update <id>" subcommand.
func newWidgetUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a widget from a JSON file",
		Long: `Update a widget using a complete JSON definition from a file.

Examples:
  rp widget update 42 --from-file w.json     Update a widget from file`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runWidgetUpdate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing WidgetRQ (use - for stdin; flags override file values)",
	)
	cmd.Flags().String("name", "", "override widget name from file")
	cmd.Flags().String("description", "", "override widget description from file")

	return cmd
}

// newWidgetDataGetCmd builds the "widget data get <id>" subcommand.
func newWidgetDataGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get computed widget data by numeric ID",
		Long: `Get computed widget data by numeric ID.

The table view shows only a summary of the content keys. Use -o json / --json
to inspect the full computed content.

Examples:
  rp widget data get 42                      Get computed widget data
  rp widget data get 42 --json               Full computed data as JSON (recommended)
  rp widget data get 42 --attribute key1     Get data with attribute filter`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runWidgetDataGet),
	}

	cmd.Flags().StringArray("attribute", nil, "attribute key to pass to the API (repeatable)")

	return cmd
}

func runWidgetGet(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	widgetID, err := parseWidgetID(args[0])
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

	deps.Logger.Debug("getting widget", "project", project, "id", widgetID)

	result, err := deps.Client.Project(project).Widgets().GetByID(cmd.Context(), widgetID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("getting widget %d in project %s: %w", widgetID, project, err)
	}

	headers, row := widgetRow(result)

	return deps.Printer.Print(result, headers, [][]string{row})
}

func runWidgetCreate(cmd *cobra.Command, _ []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	path, err := requireWidgetFromFile(
		cmd,
		printer,
		"--from-file is required for widget creation (widget definitions contain complex content parameters)",
	)
	if err != nil {
		return err
	}

	request, err := loadWidgetMutationRequest(cmd, printer, path)
	if err != nil {
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

	result, err := deps.Client.Project(project).Widgets().Create(cmd.Context(), request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("creating widget in project %s: %w", project, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{core.FormatOptional(result.Id)}}

	return deps.Printer.Print(result, headers, rows)
}

func runWidgetUpdate(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	widgetID, err := parseWidgetID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	path, err := requireWidgetFromFile(
		cmd,
		printer,
		"--from-file is required for widget update (the API expects the complete widget definition)",
	)
	if err != nil {
		return err
	}

	request, err := loadWidgetMutationRequest(cmd, printer, path)
	if err != nil {
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

	result, err := deps.Client.Project(project).
		Widgets().
		Update(cmd.Context(), widgetID, request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("updating widget %d in project %s: %w", widgetID, project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func runWidgetDataGet(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	widgetID, err := parseWidgetID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	attrs, _ := cmd.Flags().GetStringArray("attribute")

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug(
		"getting widget data",
		"project",
		project,
		"id",
		widgetID,
		"attributes",
		attrs,
	)

	var options []client.GetWidgetDataOption
	if len(attrs) > 0 {
		options = append(options, client.WithWidgetAttributes(attrs...))
	}

	result, err := deps.Client.Project(project).
		Widgets().
		GetData(cmd.Context(), widgetID, options...)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("getting widget data %d in project %s: %w", widgetID, project, err)
	}

	headers, row := widgetDataRow(result)

	return deps.Printer.Print(result, headers, [][]string{row})
}

// requireWidgetFromFile returns the --from-file path after validating the flag was set.
func requireWidgetFromFile(
	cmd *cobra.Command,
	printer *core.Printer,
	missingMessage string,
) (string, error) {
	if !cmd.Flags().Changed("from-file") {
		err := errors.New(missingMessage)
		printer.PrintError(err)

		return "", err
	}

	path, _ := cmd.Flags().GetString("from-file")

	return path, nil
}

// loadWidgetMutationRequest reads the JSON file, applies CLI overrides, and validates name.
func loadWidgetMutationRequest(
	cmd *cobra.Command,
	printer *core.Printer,
	path string,
) (types.WidgetRQ, error) {
	request, err := readWidgetRequestFromFile(path)
	if err != nil {
		printer.PrintError(err)

		return types.WidgetRQ{}, err
	}

	applyWidgetRequestFlagOverrides(cmd, &request)

	if request.Name == "" {
		nameErr := errors.New(
			"widget name is required (provide via --from-file or --name flag)",
		)
		printer.PrintError(nameErr)

		return types.WidgetRQ{}, nameErr
	}

	return request, nil
}

// applyWidgetRequestFlagOverrides applies optional --name and --description overrides from cmd onto request.
func applyWidgetRequestFlagOverrides(cmd *cobra.Command, request *types.WidgetRQ) {
	if name := core.OptionalStringFlag(cmd, "name"); name != nil {
		request.Name = *name
	}

	if description := core.OptionalStringFlag(cmd, "description"); description != nil {
		request.Description = description
	}
}

// readWidgetRequestFromFile reads JSON from path (or stdin when path is "-") into a WidgetRQ.
func readWidgetRequestFromFile(path string) (types.WidgetRQ, error) {
	result, err := core.ReadRequestFromFile[types.WidgetRQ](path)
	if err != nil {
		return result, core.WrapFileError(err, "widget definition", "widget JSON")
	}

	return result, nil
}

// widgetHeaders returns the table column headers for widget rows.
func widgetHeaders() []string {
	return []string{"ID", "NAME", "TYPE", "OWNER", "DESCRIPTION", "FILTERS"}
}

// widgetRow formats a WidgetResource as a table row, returning both headers and column values.
func widgetRow(widget *types.WidgetResource) ([]string, []string) {
	description := core.FormatOptional(widget.Description)
	if len(description) > 50 {
		description = description[:50] + "..."
	}

	filterCount := 0
	if widget.AppliedFilters != nil {
		filterCount = len(*widget.AppliedFilters)
	}

	return widgetHeaders(), []string{
		strconv.FormatInt(widget.Id, 10),
		widget.Name,
		string(widget.WidgetType),
		core.FormatOptional(widget.Owner),
		description,
		strconv.Itoa(filterCount),
	}
}

// widgetDataHeaders returns the table column headers for widget data rows.
func widgetDataHeaders() []string {
	return []string{"ID", "NAME", "TYPE", "CONTENT KEYS"}
}

// widgetDataRow formats a WidgetResource (with data) as a table row, returning both headers and column values.
func widgetDataRow(widget *types.WidgetResource) ([]string, []string) {
	contentKeys := formatWidgetContentKeys(widget.Content)

	return widgetDataHeaders(), []string{
		strconv.FormatInt(widget.Id, 10),
		widget.Name,
		string(widget.WidgetType),
		contentKeys,
	}
}

// formatWidgetContentKeys returns a sorted, comma-joined list of top-level keys from the content map.
// Nil or empty content returns "(none)".
func formatWidgetContentKeys(content *map[string]any) string {
	if content == nil || len(*content) == 0 {
		return "(none)"
	}

	keys := make([]string, 0, len(*content))
	for k := range *content {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return strings.Join(keys, ", ")
}

// parseWidgetID parses a widget ID from CLI input, which must be numeric.
func parseWidgetID(raw string) (int64, error) {
	return core.ParseIDArg(raw, "widget")
}
