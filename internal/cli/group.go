package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newGroupCmd builds the "group" parent command that groups all group subcommands.
func newGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage groups",
	}
	cmd.AddCommand(newGroupListCmd())
	cmd.AddCommand(newGroupGetCmd())
	cmd.AddCommand(newGroupCreateCmd())
	cmd.AddCommand(newGroupUpdateCmd())
	cmd.AddCommand(newGroupDeleteCmd())
	cmd.AddCommand(newGroupUserCmd())
	cmd.AddCommand(newGroupProjectCmd())

	return cmd
}

// newGroupListCmd builds the "group list" subcommand.
func newGroupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List groups",
		Long: `List groups.

Examples:
  rp group list                              List groups
  rp group list --page-size 25              List with custom page size`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runGroupList),
	}

	registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: []string{"name,ASC"}})

	return cmd
}

// newGroupGetCmd builds the "group get" subcommand.
func newGroupGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a group by numeric ID",
		Long: `Get a group by numeric ID.

Examples:
  rp group get 42                            Get group by ID
  rp group get 42 -o json                    Get group as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runGroupGet),
	}
}

// newGroupCreateCmd builds the "group create" subcommand.
func newGroupCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a group",
		Long: `Create a group.

The --slug flag accepts a URL-safe identifier: lowercase letters, numbers, and hyphens,
not starting or ending with a hyphen. If omitted, the API generates one from the name.

Examples:
  rp group create --name "My Group"          Create a group
  rp group create --name "My Group" --slug my-group  Create with explicit slug`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runGroupCreate),
	}

	cmd.Flags().String("name", "", "display name for the group (required)")
	cmd.Flags().
		String("slug", "", "URL-safe slug (lowercase, numbers, hyphens; no leading/trailing hyphens)")

	return cmd
}

// newGroupUpdateCmd builds the "group update" subcommand.
func newGroupUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a group",
		Long: `Update a group.

The --slug flag accepts a URL-safe identifier: lowercase letters, numbers, and hyphens,
not starting or ending with a hyphen.

If --name is omitted, the current group name is preserved automatically.

Examples:
  rp group update 42 --name "New Name"       Rename a group
  rp group update 42 --slug new-slug         Update slug (preserves current name)`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runGroupUpdate),
	}

	cmd.Flags().String("name", "", "display name for the group")
	cmd.Flags().
		String("slug", "", "URL-safe slug (lowercase, numbers, hyphens; no leading/trailing hyphens)")

	return cmd
}

// newGroupDeleteCmd builds the "group delete" subcommand.
func newGroupDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a group by numeric ID",
		Long: `Delete a group by numeric ID.

Examples:
  rp group delete 42                         Delete a group`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runGroupDelete),
	}
}

func runGroupList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	return listGroups(cmd, deps.client, deps.printer)
}

func listGroups(cmd *cobra.Command, clientInstance *client.Client, printer *Printer) error {
	logger := loggerFromCmd(cmd)

	flags, err := resolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid list flags: %w", err)
	}

	offset := flags.page * flags.pageSize
	limit := flags.pageSize

	if len(flags.sort) > 1 {
		logger.Debug(
			"groups API accepts a single sort field; ignoring extra sort values",
			"ignored",
			flags.sort[1:],
		)
	}

	sortField, sortOrder := parseGroupSort(flags.sort)

	logger.Debug(
		"listing groups",
		"offset",
		offset,
		"limit",
		limit,
		"sort",
		sortField,
		"order",
		sortOrder,
	)

	options := []client.ListGroupsOption{
		client.WithGroupPageSize(limit),
		client.WithGroupOffset(offset),
		client.WithGroupSort(sortField),
		client.WithGroupOrder(sortOrder),
	}

	result, err := clientInstance.Groups().List(cmd.Context(), options...)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing groups: %w", err)
	}

	return printGroupList(result, printer)
}

func printGroupList(result *types.GroupPage, printer *Printer) error {
	var groups []types.GroupInfo
	if result.Items != nil {
		groups = *result.Items
	}

	headers := groupHeaders()
	rows := make([][]string, 0, len(groups))

	for i := range groups {
		_, row := groupRow(&groups[i])
		rows = append(rows, row)
	}

	return printer.Print(groups, headers, rows)
}

func runGroupGet(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	groupID, err := parseGroupID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("getting group", "id", groupID)

	result, err := deps.client.Groups().Get(cmd.Context(), groupID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("getting group %d: %w", groupID, err)
	}

	headers, row := groupRow(result)

	return deps.printer.Print(result, headers, [][]string{row})
}

func runGroupCreate(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	name := mustGetStringFlag(cmd, "name")
	if name == "" {
		err := errors.New("--name is required")
		printer.PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	slugPtr := optionalStringFlag(cmd, "slug")

	deps.logger.Debug("creating group", "name", name)

	request := types.CreateGroupRequest{
		Name: name,
		Slug: slugPtr,
	}

	result, err := deps.client.Groups().Create(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating group: %w", err)
	}

	headers, row := groupRow(result)

	return deps.printer.Print(result, headers, [][]string{row})
}

func runGroupUpdate(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	groupID, err := parseGroupID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	name, err := resolveGroupUpdateName(cmd, deps.client.Groups(), groupID)
	if err != nil {
		deps.printer.PrintError(err)

		return err
	}

	slugPtr := optionalStringFlag(cmd, "slug")

	deps.logger.Debug("updating group", "id", groupID, "name", name)

	request := types.UpdateGroupRequest{
		Name: name,
		Slug: slugPtr,
	}

	result, err := deps.client.Groups().Update(cmd.Context(), groupID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating group %d: %w", groupID, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), successfulUpdateRows(result))
}

// resolveGroupUpdateName returns the name to use for a group update.
// If --name was explicitly set, it uses that value. Otherwise it fetches the current group
// to preserve the existing name.
func resolveGroupUpdateName(
	cmd *cobra.Command,
	scope *client.GroupsScope,
	groupID int64,
) (string, error) {
	if cmd.Flags().Changed("name") {
		name := mustGetStringFlag(cmd, "name")
		if name == "" {
			return "", errors.New("--name must not be empty")
		}

		return name, nil
	}

	current, err := scope.Get(cmd.Context(), groupID)
	if err != nil {
		return "", fmt.Errorf("fetching group %d to preserve name: %w", groupID, err)
	}

	if current.Name == nil {
		return "", errors.New("current group has no name; specify --name explicitly")
	}

	return *current.Name, nil
}

func runGroupDelete(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	groupID, err := parseGroupID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("deleting group", "id", groupID)

	err = deps.client.Groups().Delete(cmd.Context(), groupID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("deleting group %d: %w", groupID, err)
	}

	msg := fmt.Sprintf("Group %d deleted", groupID)
	result, headers, rows := deleteSuccessResult(msg)

	return deps.printer.Print(result, headers, rows)
}

// groupHeaders returns the table column headers for group rows.
func groupHeaders() []string {
	return []string{"ID", "NAME", "SLUG", "USERS", "PROJECTS", "CREATED"}
}

// groupRow formats a GroupInfo as a table row, returning both headers and column values.
func groupRow(group *types.GroupInfo) ([]string, []string) {
	var users, projects string
	if group.Stats != nil {
		users = formatOptional(group.Stats.UsersCount)
		projects = formatOptional(group.Stats.ProjectsCount)
	}

	var created string
	if group.CreatedAt != nil {
		created = group.CreatedAt.UTC().Format(time.DateTime)
	}

	return groupHeaders(), []string{
		formatOptional(group.Id),
		formatOptional(group.Name),
		formatOptional(group.Slug),
		users,
		projects,
		created,
	}
}

// parseGroupID parses a group ID from CLI input, which must be numeric.
func parseGroupID(raw string) (int64, error) {
	return parseIDArg(raw, "group")
}

// parseGroupSort parses a sort slice into a single field and order for the groups API.
// If multiple sort values are provided, only the first is used.
// Values are expected in "field,dir" format; if no comma, the entire value is treated as the field.
// Default direction is ASC. An empty sort slice or an empty first field (e.g. --sort "") falls
// back to the groups list default: name,ASC.
func parseGroupSort(sort []string) (string, string) {
	const (
		defaultField = "name"
		defaultOrder = "ASC"
	)

	if len(sort) == 0 {
		return defaultField, defaultOrder
	}

	parts := strings.SplitN(sort[0], ",", 2)

	sortField := strings.TrimSpace(parts[0])
	if sortField == "" {
		return defaultField, defaultOrder
	}

	if len(parts) == 2 {
		if dir := strings.ToUpper(strings.TrimSpace(parts[1])); dir != "" {
			return sortField, dir
		}

		return sortField, defaultOrder
	}

	return sortField, "ASC"
}
