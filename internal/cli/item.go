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

// newItemCmd builds the "item" parent command with list, get, update, and delete subcommands.
func newItemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "item",
		Aliases: []string{"test-item"},
		Short:   "Manage test items",
	}

	cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.AddCommand(newItemListCmd())
	cmd.AddCommand(newItemGetCmd())
	cmd.AddCommand(newItemUpdateCmd())
	cmd.AddCommand(newItemDeleteCmd())
	cmd.AddCommand(newItemStartCmd())
	cmd.AddCommand(newItemFinishCmd())

	return cmd
}

// newItemListCmd builds the "item list" subcommand for listing test items.
func newItemListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List test items in a project",
		Long: `List test items in a project.

Examples:
  rp item list                                      List recent test items
  rp item list --launch 101                         List test items for launch 101
  rp item list --launch 101 --suites                List suite items for launch 101
  rp item list --parent 2001                        List child items under suite item 2001`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runItemList),
	}

	registerListFlags(cmd)
	registerItemFilterFlags(cmd)

	return cmd
}

// newItemGetCmd builds the "item get" subcommand for retrieving a single test item.
func newItemGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-uuid>",
		Short: "Get a specific test item by numeric ID or UUID",
		Long: `Get a single test item by numeric ID or UUID.

Examples:
  rp item get 42                                   Get test item by numeric ID
  rp item get 550e8400-e29b-41d4-a716-446655440000 Get test item by UUID`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runItemGet),
	}
}

// runItemList executes the item list command.
func runItemList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing test items", "project", project)

	return listItems(cmd, deps.client, project, deps.printer)
}

// runItemGet executes the item get command.
func runItemGet(cmd *cobra.Command, args []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("getting single test item", "project", project, "idOrUUID", args[0])

	return getSingleItem(cmd, deps.client, project, args[0], deps.printer)
}

// newItemUpdateCmd builds the "item update" subcommand.
func newItemUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a test item by numeric ID",
		Long: `Update a test item's description, status, or attributes.

Flags or --from-file may be provided. When --from-file is given, flags are ignored.
At least one flag must be set when not using --from-file.

Examples:
  rp item update 100 --description "Updated description" -p my_project
  rp item update 100 --status PASSED -p my_project
  rp item update 100 --attribute "env:prod" --attribute "feature:login" -p my_project
  rp item update 100 --from-file item_update.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runItemUpdate),
	}

	cmd.Flags().String("description", "", "updated description")
	cmd.Flags().String("status", "", "updated status (e.g. PASSED, FAILED, SKIPPED)")
	cmd.Flags().StringArray("attribute", []string{}, "attribute in key:value format (repeatable)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing UpdateTestItemRQ (use - for stdin)")

	return cmd
}

// newItemDeleteCmd builds the "item delete" subcommand.
func newItemDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a test item by numeric ID",
		Long: `Delete a test item by its numeric ID.

Examples:
  rp item delete 100 -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runItemDelete),
	}
}

// runItemUpdate executes the item update command.
func runItemUpdate(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	itemID, err := parseIDArg(args[0], "item")
	if err != nil {
		printer.PrintError(err)

		return err
	}

	request, err := buildItemUpdateRequest(cmd)
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

	deps.logger.Debug("updating test item", "project", project, "itemID", itemID)

	result, err := deps.client.Project(project).Items().Update(cmd.Context(), itemID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating item %d: %w", itemID, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// buildItemUpdateRequest builds an UpdateTestItemRQ from flags or a file.
func buildItemUpdateRequest(cmd *cobra.Command) (types.UpdateTestItemRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		request, err := readRequestFromFile[types.UpdateTestItemRQ](path)
		if err != nil {
			return types.UpdateTestItemRQ{}, wrapFileError(
				err,
				"item update file",
				"item update JSON",
			)
		}

		return request, nil
	}

	return buildItemUpdateRequestFromFlags(cmd)
}

// buildItemUpdateRequestFromFlags builds an UpdateTestItemRQ from individual flags.
func buildItemUpdateRequestFromFlags(cmd *cobra.Command) (types.UpdateTestItemRQ, error) {
	attributes, _ := cmd.Flags().GetStringArray("attribute")

	if !cmd.Flags().Changed("description") && !cmd.Flags().Changed("status") &&
		len(attributes) == 0 {
		return types.UpdateTestItemRQ{}, errors.New(
			"at least one of --description, --status, or --attribute must be provided",
		)
	}

	request := types.UpdateTestItemRQ{
		Description: optionalStringFlag(cmd, "description"),
		Status:      resolveItemStatus(cmd),
	}

	if len(attributes) > 0 {
		attrResources, err := buildAttributeResources(attributes)
		if err != nil {
			return types.UpdateTestItemRQ{}, err
		}

		request.Attributes = &attrResources
	}

	return request, nil
}

// runItemDelete executes the item delete command.
func runItemDelete(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	itemID, err := parseIDArg(args[0], "item")
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

	deps.logger.Debug("deleting test item", "project", project, "itemID", itemID)

	result, err := deps.client.Project(project).Items().Delete(cmd.Context(), itemID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("deleting item %d: %w", itemID, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// resolveItemStatus returns the normalized (uppercase) status from --status, or nil when unset.
func resolveItemStatus(cmd *cobra.Command) *string {
	if !cmd.Flags().Changed("status") {
		return nil
	}

	raw, _ := cmd.Flags().GetString("status")
	normalized := strings.ToUpper(strings.TrimSpace(raw))

	return &normalized
}

// getSingleItem retrieves and prints a single test item by numeric ID or UUID.
//
//nolint:dupl // structural similarity with getSingleLaunch is inherent to the get-single-resource pattern
func getSingleItem(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project, idOrUUID string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Items()

	numericID, parseErr := strconv.ParseInt(idOrUUID, 10, 64)
	if parseErr == nil {
		item, err := scope.GetByID(ctx, numericID)
		if err != nil {
			printer.PrintError(err)

			return fmt.Errorf("getting item %s: %w", idOrUUID, err)
		}

		headers, row := itemRow(item)

		return printer.Print(item, headers, [][]string{row})
	}

	item, err := scope.GetByUUID(ctx, idOrUUID)
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

	var items []types.TestItemResource
	if result.Content != nil {
		items = *result.Content
	}

	headers := itemHeaders()
	rows := make([][]string, 0, len(items))

	for i := range items {
		_, row := itemRow(&items[i])
		rows = append(rows, row)
	}

	return printer.Print(items, headers, rows)
}

// itemHeaders returns the table column headers for test item output.
func itemHeaders() []string {
	return []string{"ID", "UUID", "NAME", "TYPE", "STATUS", "PARENT", "LAUNCH"}
}

// itemRow formats a test item as a table row, returning both headers and column values.
func itemRow(item *types.TestItemResource) ([]string, []string) {
	return itemHeaders(), []string{
		formatOptional(item.Id),
		formatOptional(item.Uuid),
		formatOptional(item.Name),
		formatOptional(item.Type),
		formatOptional(item.Status),
		formatOptional(item.Parent),
		formatOptional(item.LaunchId),
	}
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
