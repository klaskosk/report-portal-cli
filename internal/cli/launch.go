package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newLaunchCmd builds the "launch" parent command with list, get, update, and delete subcommands.
func newLaunchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Manage launches",
	}

	cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.AddCommand(newLaunchListCmd())
	cmd.AddCommand(newLaunchGetCmd())
	cmd.AddCommand(newLaunchUpdateCmd())
	cmd.AddCommand(newLaunchDeleteCmd())

	return cmd
}

// newLaunchListCmd builds the "launch list" subcommand for listing launches.
func newLaunchListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List launches in a project",
		Long: `List recent launches in a project.

Examples:
  rp launch list                     List the 10 most recent launches
  rp launch list -p other_project    List launches in a different project
  rp launch list --user john         List launches by user john
  rp launch list --mode DEBUG        List debug-mode launches
  rp launch list --latest            List latest launches
  rp launch list --debug             List debug launches (shorthand)`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runLaunchList),
	}

	registerListFlags(cmd)
	registerLaunchFilterFlags(cmd)

	return cmd
}

// newLaunchGetCmd builds the "launch get" subcommand for retrieving a single launch.
func newLaunchGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-uuid>",
		Short: "Get a specific launch by numeric ID or UUID",
		Long: `Get a single launch by numeric ID or UUID.

Examples:
  rp launch get 42                  Get launch by numeric ID
  rp launch get abc-def-123         Get launch by UUID
  rp launch get 42 -p other_project Get launch in a different project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runLaunchGet),
	}
}

// runLaunchList executes the launch list command.
func runLaunchList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing launches", "project", project)

	return listLaunches(cmd, deps.client, project, deps.printer)
}

// runLaunchGet executes the launch get command.
func runLaunchGet(cmd *cobra.Command, args []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("getting single launch", "project", project, "idOrUUID", args[0])

	return getSingleLaunch(cmd, deps.client, project, args[0], deps.printer)
}

// newLaunchUpdateCmd builds the "launch update" subcommand.
func newLaunchUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a launch by numeric ID",
		Long: `Update a launch by numeric ID.

Examples:
  rp launch update 42 --description "Updated description" -p my_project
  rp launch update 42 --mode DEBUG -p my_project
  rp launch update 42 --attribute "env:staging" -p my_project
  rp launch update 42 --from-file launch_update.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runLaunchUpdate),
	}

	cmd.Flags().String("description", "", "updated description")
	cmd.Flags().String("mode", "", "updated mode (DEBUG or DEFAULT)")
	cmd.Flags().StringArray("attribute", []string{}, "attribute in key:value format (repeatable)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing UpdateLaunchRQ (use - for stdin)")

	return cmd
}

// newLaunchDeleteCmd builds the "launch delete" subcommand.
func newLaunchDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a launch by numeric ID",
		Long: `Delete a launch by numeric ID.

Examples:
  rp launch delete 42 -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runLaunchDelete),
	}
}

// runLaunchUpdate executes the launch update command.
func runLaunchUpdate(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	launchID, err := parseIDArg(args[0], "launch")
	if err != nil {
		printer.PrintError(err)

		return err
	}

	request, err := buildLaunchUpdateRequest(cmd)
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

	result, err := deps.client.Project(project).Launches().Update(cmd.Context(), launchID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating launch %d in project %s: %w", launchID, project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// runLaunchDelete executes the launch delete command.
func runLaunchDelete(cmd *cobra.Command, args []string) error {
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

	result, err := deps.client.Project(project).Launches().Delete(cmd.Context(), launchID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("deleting launch %d in project %s: %w", launchID, project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// buildLaunchUpdateRequest builds the UpdateLaunchRQ for the update command. When --from-file is
// set it takes precedence; otherwise the request is assembled from individual flags.
func buildLaunchUpdateRequest(cmd *cobra.Command) (types.UpdateLaunchRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		req, err := readRequestFromFile[types.UpdateLaunchRQ](path)
		if err != nil {
			return types.UpdateLaunchRQ{}, wrapFileError(
				err,
				"launch update file",
				"launch update JSON",
			)
		}

		return req, nil
	}

	return buildUpdateLaunchRequestFromFlags(cmd)
}

// buildUpdateLaunchRequestFromFlags builds an UpdateLaunchRQ from individual flags.
// Returns an error if no flags are provided.
func buildUpdateLaunchRequestFromFlags(cmd *cobra.Command) (types.UpdateLaunchRQ, error) {
	descriptionChanged := cmd.Flags().Changed("description")
	modeChanged := cmd.Flags().Changed("mode")
	attributeChanged := cmd.Flags().Changed("attribute")

	if !descriptionChanged && !modeChanged && !attributeChanged {
		return types.UpdateLaunchRQ{}, errors.New(
			"at least one of --description, --mode, --attribute, or --from-file must be provided",
		)
	}

	description := optionalStringFlag(cmd, "description")

	var mode *types.UpdateLaunchRQMode

	if modeChanged {
		raw, _ := cmd.Flags().GetString("mode")

		m, err := resolveUpdateLaunchMode(raw)
		if err != nil {
			return types.UpdateLaunchRQ{}, err
		}

		mode = &m
	}

	var attrs *[]types.ItemAttributeResource

	if attributeChanged {
		rawAttrs, _ := cmd.Flags().GetStringArray("attribute")

		resources, buildErr := buildAttributeResources(rawAttrs)
		if buildErr != nil {
			return types.UpdateLaunchRQ{}, buildErr
		}

		attrs = &resources
	}

	return types.UpdateLaunchRQ{
		Description: description,
		Mode:        mode,
		Attributes:  attrs,
	}, nil
}

// resolveUpdateLaunchMode validates and normalizes a mode string for launch update.
// Accepts case-insensitive input and normalizes to uppercase. The caller must only
// invoke this when the --mode flag was explicitly set.
func resolveUpdateLaunchMode(raw string) (types.UpdateLaunchRQMode, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))

	switch normalized {
	case "DEBUG", "DEFAULT":
		return types.UpdateLaunchRQMode(normalized), nil
	default:
		return "", fmt.Errorf("invalid mode %q: must be DEBUG or DEFAULT", raw)
	}
}

// getSingleLaunch retrieves and prints a single launch by numeric ID or UUID.
//
//nolint:dupl // structural similarity with getSingleItem is inherent to the get-single-resource pattern
func getSingleLaunch(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project, idOrUUID string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Launches()

	numericID, parseErr := strconv.ParseInt(idOrUUID, 10, 64)
	if parseErr == nil {
		launch, err := scope.GetByID(ctx, numericID)
		if err != nil {
			printer.PrintError(err)

			return fmt.Errorf("getting launch %s: %w", idOrUUID, err)
		}

		headers, row := launchRow(launch)

		return printer.Print(launch, headers, [][]string{row})
	}

	launch, err := scope.GetByUUID(ctx, idOrUUID)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting launch %s: %w", idOrUUID, err)
	}

	headers, row := launchRow(launch)

	return printer.Print(launch, headers, [][]string{row})
}

// listLaunches retrieves and prints a paginated, filtered list of launches.
func listLaunches(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Launches()

	options, filterFlags, err := resolveLaunchOptions(cmd, printer)
	if err != nil {
		return err
	}

	var result *types.PageLaunchResource

	switch {
	case filterFlags.latest:
		result, err = scope.GetLatest(ctx, options...)
	case filterFlags.debug:
		result, err = scope.GetDebug(ctx, options...)
	default:
		result, err = scope.List(ctx, options...)
	}

	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing launches: %w", err)
	}

	var launches []types.LaunchResource
	if result.Content != nil {
		launches = *result.Content
	}

	headers := launchHeaders()
	rows := make([][]string, 0, len(launches))

	for i := range launches {
		_, row := launchRow(&launches[i])
		rows = append(rows, row)
	}

	return printer.Print(launches, headers, rows)
}

// resolveLaunchOptions builds the full set of list options from pagination and filter flags.
func resolveLaunchOptions(
	cmd *cobra.Command,
	printer *Printer,
) ([]client.ListLaunchesOption, launchFilterFlags, error) {
	flags, err := resolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return nil, launchFilterFlags{}, fmt.Errorf("invalid list flags: %w", err)
	}

	filterFlags, err := resolveLaunchFilterFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return nil, launchFilterFlags{}, fmt.Errorf("invalid filter flags: %w", err)
	}

	filterOptions, err := applyLaunchFilterOptions(filterFlags)
	if err != nil {
		printer.PrintError(err)

		return nil, launchFilterFlags{}, fmt.Errorf("invalid filter flags: %w", err)
	}

	options := make([]client.ListLaunchesOption, 0, 3+len(filterOptions))
	options = append(options,
		client.WithLaunchPageSize(flags.pageSize),
		client.WithLaunchPageNumber(flags.page),
		client.WithLaunchSort(flags.sort...),
	)
	options = append(options, filterOptions...)

	return options, filterFlags, nil
}

// launchHeaders returns the table column headers for launch output.
func launchHeaders() []string {
	return []string{"ID", "UUID", "NAME", "STATUS", "START TIME", "NUMBER"}
}

// launchRow formats a launch resource as a table row, returning both headers and column values.
func launchRow(launch *types.LaunchResource) ([]string, []string) {
	startTime := launch.StartTime.UTC().Format(time.DateTime)

	return launchHeaders(), []string{
		strconv.FormatInt(launch.Id, 10),
		launch.Uuid,
		launch.Name,
		launch.Status,
		startTime,
		strconv.FormatInt(launch.Number, 10),
	}
}

// launchFilterFlags holds the resolved filter flags for launch queries.
type launchFilterFlags struct {
	user        string
	mode        string
	attributes  []string
	description string
	hasRetries  bool
	latest      bool
	debug       bool
}

// registerLaunchFilterFlags adds launch-specific filter flags to cmd.
func registerLaunchFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("user", "", "filter by username")
	cmd.Flags().String("mode", "", "filter by mode (DEFAULT, DEBUG)")
	cmd.Flags().StringArray("attribute", []string{}, "filter by attribute key:value (repeatable)")
	cmd.Flags().String("description", "", "filter by description")
	cmd.Flags().Bool("has-retries", false, "filter launches with retries")
	cmd.Flags().Bool("latest", false, "show latest launches (uses /launch/latest endpoint)")
	cmd.Flags().Bool("debug", false, "show debug-mode launches (uses /launch/mode endpoint)")
}

// resolveLaunchFilterFlags reads the launch filter flags from cmd and returns them.
func resolveLaunchFilterFlags(cmd *cobra.Command) (launchFilterFlags, error) {
	user, _ := cmd.Flags().GetString("user")
	mode, _ := cmd.Flags().GetString("mode")
	attributes, _ := cmd.Flags().GetStringArray("attribute")
	description, _ := cmd.Flags().GetString("description")
	hasRetries, _ := cmd.Flags().GetBool("has-retries")
	latest, _ := cmd.Flags().GetBool("latest")
	debug, _ := cmd.Flags().GetBool("debug")

	if latest && debug {
		return launchFilterFlags{}, errors.New("cannot use both --latest and --debug flags")
	}

	return launchFilterFlags{
		user:        user,
		mode:        mode,
		attributes:  attributes,
		description: description,
		hasRetries:  hasRetries,
		latest:      latest,
		debug:       debug,
	}, nil
}

// applyLaunchFilterOptions converts launchFilterFlags to client options.
func applyLaunchFilterOptions(filters launchFilterFlags) ([]client.ListLaunchesOption, error) {
	options := make([]client.ListLaunchesOption, 0, 4+len(filters.attributes))

	if filters.user != "" {
		options = append(options, client.WithLaunchUser(filters.user))
	}

	if filters.mode != "" {
		options = append(options, client.WithLaunchMode(filters.mode))
	}

	for _, attr := range filters.attributes {
		key, value, found := parseAttribute(attr)
		if !found {
			return nil, fmt.Errorf("invalid attribute format %q (expected key:value)", attr)
		}

		options = append(options, client.WithLaunchAttribute(key, value))
	}

	if filters.description != "" {
		options = append(options, client.WithLaunchDescription(filters.description))
	}

	if filters.hasRetries {
		options = append(options, client.WithLaunchHasRetries(true))
	}

	return options, nil
}

// parseAttribute parses an attribute string in "key:value" format.
// The key must be non-empty; the value may be empty.
func parseAttribute(attr string) (string, string, bool) {
	for i := range len(attr) {
		if attr[i] == ':' {
			if i == 0 {
				return "", "", false
			}

			return attr[:i], attr[i+1:], true
		}
	}

	return "", "", false
}
