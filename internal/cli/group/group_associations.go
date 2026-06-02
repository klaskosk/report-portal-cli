package group

import (
	"errors"
	"fmt"
	"time"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

const defaultGroupAssociationPageSize int32 = 10

// newGroupUserCmd builds the "group user" parent command for group-user associations.
func newGroupUserCmd() *cobra.Command {
	cmd := core.MarkGroupCommand(&cobra.Command{
		Use:   "user",
		Short: "Manage group user memberships",
	})
	cmd.AddCommand(newGroupUserListCmd())
	cmd.AddCommand(newGroupUserGetCmd())
	cmd.AddCommand(newGroupUserAddCmd())
	cmd.AddCommand(newGroupUserRemoveCmd())

	return cmd
}

// newGroupUserListCmd builds the "group user list" subcommand.
func newGroupUserListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <group-id>",
		Short: "List users in a group",
		Long: `List users assigned to a group.

Examples:
  rp group user list 42
  rp group user list 42 --page-size 25 --offset 50`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runGroupUserList),
	}

	registerGroupAssociationListFlags(cmd)

	return cmd
}

// newGroupUserGetCmd builds the "group user get" subcommand.
func newGroupUserGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <group-id> <user-id>",
		Short: "Get a user membership in a group",
		Long: `Get details of a user assigned to a group.

Examples:
  rp group user get 42 7
  rp group user get 42 7 -o json`,
		Args: cobra.ExactArgs(2),
		RunE: core.WithPrintedErrors(runGroupUserGet),
	}
}

// newGroupUserAddCmd builds the "group user add" subcommand.
func newGroupUserAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <group-id> <user-id>",
		Short: "Add a user to a group",
		Long: `Add a user to a group.

Examples:
  rp group user add 42 7`,
		Args: cobra.ExactArgs(2),
		RunE: core.WithPrintedErrors(runGroupUserAdd),
	}
}

// newGroupUserRemoveCmd builds the "group user remove" subcommand.
func newGroupUserRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <group-id> <user-id>",
		Short: "Remove a user from a group",
		Long: `Remove a user from a group.

Examples:
  rp group user remove 42 7`,
		Args: cobra.ExactArgs(2),
		RunE: core.WithPrintedErrors(runGroupUserRemove),
	}
}

// newGroupProjectCmd builds the "group project" parent command for group-project associations.
func newGroupProjectCmd() *cobra.Command {
	cmd := core.MarkGroupCommand(&cobra.Command{
		Use:   "project",
		Short: "Manage group project associations",
	})
	cmd.AddCommand(newGroupProjectListCmd())
	cmd.AddCommand(newGroupProjectGetCmd())
	cmd.AddCommand(newGroupProjectAddCmd())
	cmd.AddCommand(newGroupProjectRemoveCmd())

	return cmd
}

// newGroupProjectListCmd builds the "group project list" subcommand.
func newGroupProjectListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <group-id>",
		Short: "List projects in a group",
		Long: `List projects assigned to a group.

Examples:
  rp group project list 42
  rp group project list 42 --page-size 25 --offset 50`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runGroupProjectList),
	}

	registerGroupAssociationListFlags(cmd)

	return cmd
}

// newGroupProjectGetCmd builds the "group project get" subcommand.
func newGroupProjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <group-id> <project-id>",
		Short: "Get a project association in a group",
		Long: `Get details of a project assigned to a group.

Examples:
  rp group project get 42 3
  rp group project get 42 3 -o json`,
		Args: cobra.ExactArgs(2),
		RunE: core.WithPrintedErrors(runGroupProjectGet),
	}
}

// newGroupProjectAddCmd builds the "group project add" subcommand.
func newGroupProjectAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <group-id> <project-id>",
		Short: "Add a project to a group",
		Long: `Add a project to a group.

Examples:
  rp group project add 42 3
  rp group project add 42 3 --permission MEMBER`,
		Args: cobra.ExactArgs(2),
		RunE: core.WithPrintedErrors(runGroupProjectAdd),
	}

	cmd.Flags().
		String("permission", "", "project role (CUSTOMER, MEMBER, OPERATOR, PROJECT_MANAGER)")

	return cmd
}

// newGroupProjectRemoveCmd builds the "group project remove" subcommand.
func newGroupProjectRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <group-id> <project-id>",
		Short: "Remove a project from a group",
		Long: `Remove a project from a group.

Examples:
  rp group project remove 42 3`,
		Args: cobra.ExactArgs(2),
		RunE: core.WithPrintedErrors(runGroupProjectRemove),
	}
}

func runGroupUserList(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	groupID, err := parseGroupID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	pageSize, offset, err := resolveGroupAssociationPagination(cmd)
	if err != nil {
		deps.Printer.PrintError(err)

		return err
	}

	deps.Logger.Debug(
		"listing group users",
		"groupId", groupID,
		"pageSize", pageSize,
		"offset", offset,
	)

	result, err := deps.Client.Groups().ListUsers(
		cmd.Context(),
		groupID,
		client.WithGroupUserPageSize(pageSize),
		client.WithGroupUserOffset(offset),
	)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("listing users in group %d: %w", groupID, err)
	}

	return printGroupUserList(result, deps.Printer)
}

func runGroupUserGet(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	groupID, userID, err := parseGroupUserIDs(args)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("getting group user", "groupId", groupID, "userId", userID)

	result, err := deps.Client.Groups().GetUser(cmd.Context(), groupID, userID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("getting user %d in group %d: %w", userID, groupID, err)
	}

	headers, row := groupUserRow(result)

	return deps.Printer.Print(result, headers, [][]string{row})
}

func runGroupUserAdd(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	groupID, userID, err := parseGroupUserIDs(args)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("adding user to group", "groupId", groupID, "userId", userID)

	result, err := deps.Client.Groups().AddUser(cmd.Context(), groupID, userID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("adding user %d to group %d: %w", userID, groupID, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.SuccessfulUpdateRows(result),
	)
}

func runGroupUserRemove(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	groupID, userID, err := parseGroupUserIDs(args)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("removing user from group", "groupId", groupID, "userId", userID)

	err = deps.Client.Groups().RemoveUser(cmd.Context(), groupID, userID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("removing user %d from group %d: %w", userID, groupID, err)
	}

	msg := fmt.Sprintf("User %d removed from group %d", userID, groupID)
	result, headers, rows := core.DeleteSuccessResult(msg)

	return deps.Printer.Print(result, headers, rows)
}

func runGroupProjectList(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	groupID, err := parseGroupID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	pageSize, offset, err := resolveGroupAssociationPagination(cmd)
	if err != nil {
		deps.Printer.PrintError(err)

		return err
	}

	deps.Logger.Debug(
		"listing group projects",
		"groupId", groupID,
		"pageSize", pageSize,
		"offset", offset,
	)

	result, err := deps.Client.Groups().ListProjects(
		cmd.Context(),
		groupID,
		client.WithGroupProjectPageSize(pageSize),
		client.WithGroupProjectOffset(offset),
	)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("listing projects in group %d: %w", groupID, err)
	}

	return printGroupProjectList(result, deps.Printer)
}

func runGroupProjectGet(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	groupID, projectID, err := parseGroupProjectIDs(args)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("getting group project", "groupId", groupID, "projectId", projectID)

	result, err := deps.Client.Groups().GetProject(cmd.Context(), groupID, projectID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("getting project %d in group %d: %w", projectID, groupID, err)
	}

	headers, row := groupProjectRow(result)

	return deps.Printer.Print(result, headers, [][]string{row})
}

func runGroupProjectAdd(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	groupID, projectID, err := parseGroupProjectIDs(args)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	request := types.AddProjectToGroupByIdRequest{}

	if permission := core.OptionalStringFlag(cmd, "permission"); permission != nil {
		perm := types.AddProjectToGroupByIdRequestPermission(*permission)
		request.Permission = &perm
	}

	deps.Logger.Debug(
		"adding project to group",
		"groupId", groupID,
		"projectId", projectID,
		"permission", request.Permission,
	)

	result, err := deps.Client.Groups().AddProject(cmd.Context(), groupID, projectID, request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("adding project %d to group %d: %w", projectID, groupID, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.SuccessfulUpdateRows(result),
	)
}

func runGroupProjectRemove(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	groupID, projectID, err := parseGroupProjectIDs(args)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("removing project from group", "groupId", groupID, "projectId", projectID)

	err = deps.Client.Groups().RemoveProject(cmd.Context(), groupID, projectID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("removing project %d from group %d: %w", projectID, groupID, err)
	}

	msg := fmt.Sprintf("Project %d removed from group %d", projectID, groupID)
	result, headers, rows := core.DeleteSuccessResult(msg)

	return deps.Printer.Print(result, headers, rows)
}

func registerGroupAssociationListFlags(cmd *cobra.Command) {
	cmd.Flags().Int32("page-size", defaultGroupAssociationPageSize, "number of results per page")
	cmd.Flags().Int32("offset", 0, "offset for pagination")
}

func resolveGroupAssociationPagination(cmd *cobra.Command) (int32, int32, error) {
	pageSize, err := cmd.Flags().GetInt32("page-size")
	if err != nil {
		return 0, 0, fmt.Errorf("invalid page-size flag: %w", err)
	}

	if pageSize <= 0 {
		return 0, 0, errors.New("page-size must be greater than 0")
	}

	offset, err := cmd.Flags().GetInt32("offset")
	if err != nil {
		return 0, 0, fmt.Errorf("invalid offset flag: %w", err)
	}

	if offset < 0 {
		return 0, 0, errors.New("offset must be greater than or equal to 0")
	}

	return pageSize, offset, nil
}

func printGroupUserList(result *types.GroupUsersPage, printer *core.Printer) error {
	var users []types.GroupUserInfo
	if result.Items != nil {
		users = *result.Items
	}

	headers := groupUserHeaders()
	rows := make([][]string, 0, len(users))

	for i := range users {
		_, row := groupUserRow(&users[i])
		rows = append(rows, row)
	}

	return printer.Print(result, headers, rows)
}

func printGroupProjectList(result *types.GroupProjectsPage, printer *core.Printer) error {
	var projects []types.GroupProjectInfo
	if result.Items != nil {
		projects = *result.Items
	}

	headers := groupProjectHeaders()
	rows := make([][]string, 0, len(projects))

	for i := range projects {
		_, row := groupProjectRow(&projects[i])
		rows = append(rows, row)
	}

	return printer.Print(result, headers, rows)
}

func groupUserHeaders() []string {
	return []string{"ID", "FULL NAME", "EMAIL", "INSTANCE ROLE", "ACTIVE", "ADDED AT"}
}

func groupUserRow(user *types.GroupUserInfo) ([]string, []string) {
	var addedAt string
	if user.AddedAt != nil {
		addedAt = user.AddedAt.UTC().Format(time.DateTime)
	}

	return groupUserHeaders(), []string{
		core.FormatOptional(user.Id),
		core.FormatOptional(user.FullName),
		core.FormatOptional(user.Email),
		core.FormatOptional(user.InstanceRole),
		core.FormatOptional(user.Active),
		addedAt,
	}
}

func groupProjectHeaders() []string {
	return []string{"ID", "NAME", "PERMISSION", "ADDED AT", "UPDATED AT"}
}

func groupProjectRow(project *types.GroupProjectInfo) ([]string, []string) {
	var addedAt, updatedAt string
	if project.AddedAt != nil {
		addedAt = project.AddedAt.UTC().Format(time.DateTime)
	}

	if project.UpdatedAt != nil {
		updatedAt = project.UpdatedAt.UTC().Format(time.DateTime)
	}

	return groupProjectHeaders(), []string{
		core.FormatOptional(project.Id),
		core.FormatOptional(project.Name),
		core.FormatOptional(project.Permission),
		addedAt,
		updatedAt,
	}
}

func parseGroupUserIDs(args []string) (int64, int64, error) {
	groupID, err := parseGroupID(args[0])
	if err != nil {
		return 0, 0, err
	}

	userID, err := parseUserID(args[1])
	if err != nil {
		return 0, 0, err
	}

	return groupID, userID, nil
}

func parseGroupProjectIDs(args []string) (int64, int64, error) {
	groupID, err := parseGroupID(args[0])
	if err != nil {
		return 0, 0, err
	}

	projectID, err := parseProjectID(args[1])
	if err != nil {
		return 0, 0, err
	}

	return groupID, projectID, nil
}

func parseUserID(raw string) (int64, error) {
	return core.ParseIDArg(raw, "user")
}

func parseProjectID(raw string) (int64, error) {
	return core.ParseIDArg(raw, "project")
}
