package filter

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewCommand builds the "filter" parent command that groups all filter subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Annotations: map[string]string{core.GroupAnnotationKey: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		Use:   "filter",
		Short: "Manage saved filters",
	}
	core.RegisterProjectFlag(cmd)
	cmd.AddCommand(newFilterListCmd())
	cmd.AddCommand(newFilterGetCmd())
	cmd.AddCommand(newFilterCreateCmd())
	cmd.AddCommand(newFilterUpdateCmd())
	cmd.AddCommand(newFilterDeleteCmd())

	return cmd
}

// newFilterListCmd builds the "filter list" subcommand.
func newFilterListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved filters in a project",
		Long: `List saved filters in a project.

Examples:
  rp filter list -p my_project               List saved filters
  rp filter list -o json -p my_project       List filters as JSON
  rp filter list --page-size 25 -p my_project  List with custom page size`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runFilterList),
	}

	core.RegisterListFlagsWithDefaults(cmd, core.ListFlagDefaults{Sort: []string{"name,ASC"}})

	return cmd
}

// newFilterGetCmd builds the "filter get" subcommand.
func newFilterGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a saved filter by numeric ID",
		Long: `Get a saved filter by numeric ID.

Examples:
  rp filter get 42 -p my_project             Get filter by ID
  rp filter get 42 -o json -p my_project     Get filter as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runFilterGet),
	}
}

// newFilterCreateCmd builds the "filter create" subcommand.
func newFilterCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a saved filter from a JSON file",
		Long: `Create a saved filter from a JSON file describing the filter definition.

Examples:
  rp filter create --from-file filter.json   Create a filter from file`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runFilterCreate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing UpdateUserFilterRQ (use - for stdin; flags override file values)",
	)
	cmd.Flags().String("name", "", "override filter name from file")
	cmd.Flags().String("description", "", "override filter description from file")

	return cmd
}

// newFilterUpdateCmd builds the "filter update" subcommand.
func newFilterUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a saved filter from a JSON file",
		Long: `Update a saved filter using a complete JSON definition from a file.

Examples:
  rp filter update 42 --from-file f.json     Update a filter from file`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runFilterUpdate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing UpdateUserFilterRQ (use - for stdin; flags override file values)",
	)
	cmd.Flags().String("name", "", "override filter name from file")
	cmd.Flags().String("description", "", "override filter description from file")

	return cmd
}

// newFilterDeleteCmd builds the "filter delete" subcommand.
func newFilterDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a saved filter by numeric ID",
		Long: `Delete a saved filter by numeric ID.

Examples:
  rp filter delete 42                        Delete a filter`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runFilterDelete),
	}
}

func runFilterList(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("listing filters", "project", project)

	return listFilters(cmd, deps.Client, project, deps.Printer)
}

func runFilterGet(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	filterID, err := parseFilterID(args[0])
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

	deps.Logger.Debug("getting filter", "project", project, "id", filterID)

	return getSingleFilter(cmd, deps.Client, project, filterID, deps.Printer)
}

func runFilterCreate(cmd *cobra.Command, _ []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	path, err := core.RequireFromFile(
		cmd,
		printer,
		"--from-file is required for filter creation (filter definitions contain structured conditions and orders)",
	)
	if err != nil {
		return err
	}

	request, err := loadFilterMutationRequest(cmd, printer, path)
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

	result, err := deps.Client.Project(project).Filters().Create(cmd.Context(), request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("creating filter in project %s: %w", project, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{core.FormatOptional(result.Id)}}

	return deps.Printer.Print(result, headers, rows)
}

func runFilterUpdate(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	filterID, err := parseFilterID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	path, err := core.RequireFromFile(
		cmd,
		printer,
		"--from-file is required for filter update (the API expects the complete filter definition)",
	)
	if err != nil {
		return err
	}

	request, err := loadFilterMutationRequest(cmd, printer, path)
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
		Filters().
		Update(cmd.Context(), filterID, request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("updating filter %d in project %s: %w", filterID, project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func runFilterDelete(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	filterID, err := parseFilterID(args[0])
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

	result, err := deps.Client.Project(project).Filters().Delete(cmd.Context(), filterID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("deleting filter %d in project %s: %w", filterID, project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

// loadFilterMutationRequest reads the JSON file, applies CLI overrides, and validates name.
func loadFilterMutationRequest(
	cmd *cobra.Command,
	printer *core.Printer,
	path string,
) (types.UpdateUserFilterRQ, error) {
	request, err := readFilterRequestFromFile(path)
	if err != nil {
		printer.PrintError(err)

		return types.UpdateUserFilterRQ{}, err
	}

	applyFilterRequestFlagOverrides(cmd, &request)

	if request.Name == "" {
		nameErr := errors.New("filter name is required (provide via --from-file or --name flag)")
		printer.PrintError(nameErr)

		return types.UpdateUserFilterRQ{}, nameErr
	}

	return request, nil
}

func listFilters(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	printer *core.Printer,
) error {
	flags, err := core.ResolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid list flags: %w", err)
	}

	options := []client.ListFiltersOption{
		client.WithFilterPageNumber(flags.Page),
		client.WithFilterPageSize(flags.PageSize),
		client.WithFilterSort(flags.Sort...),
	}

	result, err := clientInstance.Project(project).Filters().List(cmd.Context(), options...)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing filters: %w", err)
	}

	var filters []types.UserFilterResource
	if result.Content != nil {
		filters = *result.Content
	}

	headers := filterHeaders()
	rows := make([][]string, 0, len(filters))

	for i := range filters {
		_, row := filterRow(&filters[i])
		rows = append(rows, row)
	}

	return printer.Print(filters, headers, rows)
}

func getSingleFilter(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	filterID int64,
	printer *core.Printer,
) error {
	filterResource, err := clientInstance.Project(project).
		Filters().
		GetByID(cmd.Context(), filterID)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting filter %d: %w", filterID, err)
	}

	headers, row := filterRow(filterResource)

	return printer.Print(filterResource, headers, [][]string{row})
}

// filterHeaders returns uppercase table headers for saved filter rows.
func filterHeaders() []string {
	return []string{"ID", "NAME", "TYPE", "OWNER", "DESCRIPTION", "CONDITIONS"}
}

// filterRow formats a user filter resource as a table row, returning both headers and column values.
func filterRow(filterResource *types.UserFilterResource) ([]string, []string) {
	description := core.FormatOptional(filterResource.Description)
	if len(description) > 50 {
		description = description[:50] + "..."
	}

	return filterHeaders(), []string{
		strconv.FormatInt(filterResource.Id, 10),
		filterResource.Name,
		filterResource.Type,
		filterResource.Owner,
		description,
		strconv.Itoa(len(filterResource.Conditions)),
	}
}

// parseFilterID parses a filter ID from CLI input, which must be numeric.
func parseFilterID(raw string) (int64, error) {
	return core.ParseIDArg(raw, "filter")
}

// applyFilterRequestFlagOverrides applies optional --name and --description overrides from cmd onto request.
func applyFilterRequestFlagOverrides(cmd *cobra.Command, request *types.UpdateUserFilterRQ) {
	if name := core.OptionalStringFlag(cmd, "name"); name != nil {
		request.Name = *name
	}

	if description := core.OptionalStringFlag(cmd, "description"); description != nil {
		request.Description = description
	}
}

// readFilterRequestFromFile reads JSON from path (or stdin when path is "-") into an UpdateUserFilterRQ.
func readFilterRequestFromFile(path string) (types.UpdateUserFilterRQ, error) {
	result, err := core.ReadRequestFromFile[types.UpdateUserFilterRQ](path)
	if err != nil {
		return result, core.WrapFileError(err, "filter definition", "filter JSON")
	}

	return result, nil
}
