package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newGetItemCmd builds the "get item" subcommand for listing or retrieving test items.
func newGetItemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "item [id-or-uuid]",
		Aliases: []string{"test-item"},
		Short:   "List test items or get a specific test item",
		Long: `List test items in a project, or get a single test item by numeric ID or UUID.

Examples:
  rp get item                                      List recent test items
  rp get item 42                                   Get test item by numeric ID
  rp get item 550e8400-e29b-41d4-a716-446655440000 Get test item by UUID
  rp get item --launch 101                         List test items for launch 101
  rp get item --launch 101 --suites                List suite items for launch 101
  rp get item --parent 2001                        List child items under suite item 2001`,
		Args: cobra.MaximumNArgs(1),
		RunE: withPrintedErrors(runGetItem),
	}

	cmd.Flags().StringP("project", "p", "", "project name (overrides profile default)")
	registerListFlags(cmd)
	registerItemFilterFlags(cmd)

	return cmd
}

// runGetItem executes the get item command, dispatching to single-get or list mode.
func runGetItem(cmd *cobra.Command, args []string) error {
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
		logger.Debug("getting single test item", "project", project, "idOrUUID", args[0])

		return getSingleItem(cmd, clientInstance, project, args[0], printer)
	}

	logger.Debug("listing test items", "project", project)

	return listItems(cmd, clientInstance, project, printer)
}

// getSingleItem retrieves and prints a single test item by numeric ID or UUID.
func getSingleItem(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project, idOrUUID string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Items()

	var (
		item *types.TestItemResource
		err  error
	)

	if isNumericID(idOrUUID) {
		item, err = scope.GetByID(ctx, idOrUUID)
	} else {
		item, err = scope.GetByUUID(ctx, idOrUUID)
	}

	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting item %s: %w", idOrUUID, err)
	}

	headers, row := itemRow(item)

	return printer.Print(item, headers, [][]string{row})
}

// listItems retrieves and prints a paginated, filtered list of test items.
func listItems(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Items()

	flags, err := resolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid list flags: %w", err)
	}

	filterFlags, err := resolveItemFilterFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid filter flags: %w", err)
	}

	options := make([]client.ListTestItemsOption, 0, 3)
	options = append(
		options,
		client.WithItemPageSize(flags.pageSize),
		client.WithItemPageNumber(flags.page),
		client.WithItemSort(flags.sort...),
	)
	options = append(options, applyItemFilterOptions(filterFlags)...)

	result, err := scope.List(ctx, options...)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing items: %w", err)
	}

	headers := itemHeaders()
	rows := make([][]string, 0, len(result.Content))

	for i := range result.Content {
		_, row := itemRow(&result.Content[i])
		rows = append(rows, row)
	}

	return printer.Print(result.Content, headers, rows)
}

// itemHeaders returns the table column headers for test item output.
func itemHeaders() []string {
	return []string{"ID", "UUID", "NAME", "TYPE", "STATUS", "PARENT", "LAUNCH"}
}

// itemRow formats a test item as a table row, returning both headers and column values.
func itemRow(item *types.TestItemResource) ([]string, []string) {
	return itemHeaders(), []string{
		formatInt64Ptr(item.Id),
		formatStringPtr(item.Uuid),
		formatStringPtr(item.Name),
		formatStringPtr(item.Type),
		formatStringPtr(item.Status),
		formatInt64Ptr(item.Parent),
		formatInt64Ptr(item.LaunchId),
	}
}

// formatStringPtr returns the dereferenced string or an empty string if nil.
func formatStringPtr(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

// formatInt64Ptr returns the formatted int64 or an empty string if nil.
func formatInt64Ptr(value *int64) string {
	if value == nil {
		return ""
	}

	return strconv.FormatInt(*value, 10)
}

// itemFilterFlags holds resolved filter flags for test item queries.
type itemFilterFlags struct {
	launchID       int64
	hasLaunch      bool
	parentID       int64
	hasParent      bool
	itemType       string
	name           string
	status         string
	hasChildren    bool
	hasChildrenSet bool
}

// registerItemFilterFlags adds item-specific filter flags to cmd.
func registerItemFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("launch", "", "filter by launch ID")
	cmd.Flags().String("parent", "", "filter by parent item ID")
	cmd.Flags().String("type", "", "filter by item type (e.g. SUITE, TEST, STEP)")
	cmd.Flags().Bool("suites", false, "show only suite items (shortcut for --type SUITE)")
	cmd.Flags().String("name", "", "filter by item name")
	cmd.Flags().String("status", "", "filter by item status")
	cmd.Flags().Bool("has-children", false, "filter by whether item has children")
}

// resolveItemFilterFlags reads test item filter flags from cmd and returns them.
func resolveItemFilterFlags(cmd *cobra.Command) (itemFilterFlags, error) {
	launchRaw, _ := cmd.Flags().GetString("launch")
	parentRaw, _ := cmd.Flags().GetString("parent")
	itemType, _ := cmd.Flags().GetString("type")
	suites, _ := cmd.Flags().GetBool("suites")
	name, _ := cmd.Flags().GetString("name")
	status, _ := cmd.Flags().GetString("status")
	hasChildren, _ := cmd.Flags().GetBool("has-children")
	hasChildrenSet := cmd.Flags().Changed("has-children")

	launchID, hasLaunch, err := parseOptionalInt64Flag("launch", launchRaw)
	if err != nil {
		return itemFilterFlags{}, err
	}

	parentID, hasParent, err := parseOptionalInt64Flag("parent", parentRaw)
	if err != nil {
		return itemFilterFlags{}, err
	}

	itemType = strings.ToUpper(strings.TrimSpace(itemType))
	if suites {
		if itemType != "" && itemType != "SUITE" {
			return itemFilterFlags{}, fmt.Errorf("cannot use --suites with --type %q", itemType)
		}

		itemType = "SUITE"
	}

	return itemFilterFlags{
		launchID:       launchID,
		hasLaunch:      hasLaunch,
		parentID:       parentID,
		hasParent:      hasParent,
		itemType:       itemType,
		name:           name,
		status:         status,
		hasChildren:    hasChildren,
		hasChildrenSet: hasChildrenSet,
	}, nil
}

// parseOptionalInt64Flag parses value as an int64 if non-empty, returning (0, false, nil) when empty.
func parseOptionalInt64Flag(name, value string) (int64, bool, error) {
	if value == "" {
		return 0, false, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("invalid --%s value %q: must be a numeric ID", name, value)
	}

	return parsed, true, nil
}

// applyItemFilterOptions converts itemFilterFlags to client options.
func applyItemFilterOptions(filters itemFilterFlags) []client.ListTestItemsOption {
	var options []client.ListTestItemsOption

	if filters.hasLaunch {
		options = append(options, client.WithItemLaunchID(filters.launchID))
	}

	if filters.hasParent {
		options = append(options, client.WithItemParentID(filters.parentID))
	}

	if filters.itemType != "" {
		options = append(options, client.WithItemType(filters.itemType))
	}

	if filters.name != "" {
		options = append(options, client.WithItemName(filters.name))
	}

	if filters.status != "" {
		options = append(options, client.WithItemStatus(filters.status))
	}

	if filters.hasChildrenSet {
		options = append(options, client.WithItemHasChildren(filters.hasChildren))
	}

	return options
}
