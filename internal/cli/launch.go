package cli

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newGetLaunchCmd builds the "get launch" subcommand for listing or retrieving launches.
func newGetLaunchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch [id-or-uuid]",
		Short: "List launches or get a specific launch",
		Long: `List recent launches in a project, or get a single launch by numeric ID or UUID.

Examples:
  rp get launch                     List the 10 most recent launches
  rp get launch 42                  Get launch by numeric ID
  rp get launch abc-def-123         Get launch by UUID
  rp get launch -p other_project    List launches in a different project
  rp get launch --user john         List launches by user john
  rp get launch --mode DEBUG        List debug-mode launches
  rp get launch --latest            List latest launches
  rp get launch --debug             List debug launches (shorthand)`,
		Args: cobra.MaximumNArgs(1),
		RunE: withPrintedErrors(runGetLaunch),
	}

	cmd.Flags().StringP("project", "p", "", "project name (overrides profile default)")
	registerListFlags(cmd)
	registerLaunchFilterFlags(cmd)

	return cmd
}

// runGetLaunch executes the get launch command, dispatching to single-get or list mode.
func runGetLaunch(cmd *cobra.Command, args []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	project, err := resolveProject(cmd, conn)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	if len(args) == 1 {
		logger.Debug("getting single launch", "project", project, "idOrUUID", args[0])

		return getSingleLaunch(cmd, clientInstance, project, args[0], printer)
	}

	logger.Debug("listing launches", "project", project)

	return listLaunches(cmd, clientInstance, project, printer)
}

// getSingleLaunch retrieves and prints a single launch by numeric ID or UUID.
func getSingleLaunch(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project, idOrUUID string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Launches()

	var (
		launch *types.LaunchResource
		err    error
	)

	numericID, parseErr := strconv.ParseInt(idOrUUID, 10, 64)
	if parseErr == nil {
		launch, err = scope.GetByID(ctx, numericID)
	} else {
		launch, err = scope.GetByUUID(ctx, idOrUUID)
	}

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

	var result *types.IterableLaunchResource

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

	headers := launchHeaders()
	rows := make([][]string, 0, len(result.Content))

	for i := range result.Content {
		_, row := launchRow(&result.Content[i])
		rows = append(rows, row)
	}

	return printer.Print(result.Content, headers, rows)
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
	startTime := time.Time(launch.StartTime).UTC().Format(time.DateTime)

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
