package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newProjectUserCmd builds the "project user" grouping command for user assignment subcommands.
func newProjectUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage project user assignments",
	}

	cmd.AddCommand(newProjectUserAssignCmd())
	cmd.AddCommand(newProjectUserUnassignCmd())

	return cmd
}

// newProjectUserAssignCmd builds the "project user assign" subcommand.
func newProjectUserAssignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assign <name>",
		Short: "Assign users to a project with roles",
		Long: `Assign users to a project with specified roles.

Provide user assignments via repeatable --user flags (username:role format) or
a JSON file via --from-file. When --from-file is set, --user flags are ignored.

Valid roles: MEMBER, PROJECT_MANAGER, CUSTOMER, OPERATOR (case-insensitive).

This command does not require --project/-p since it operates on the instance level.

Examples:
  rp project user assign --user john:MEMBER --user jane:PROJECT_MANAGER my_project
  rp project user assign --from-file users.json my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runProjectUserAssign),
	}

	cmd.Flags().StringArray("user", nil, "user assignment in username:role format (repeatable)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing AssignUsersRQ (use - for stdin)")

	return cmd
}

// newProjectUserUnassignCmd builds the "project user unassign" subcommand.
func newProjectUserUnassignCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unassign <name>",
		Short: "Remove users from a project",
		Long: `Remove users from a project.

Provide usernames via repeatable --user flags or a JSON file via --from-file.
When --from-file is set, --user flags are ignored.

This command does not require --project/-p since it operates on the instance level.

Examples:
  rp project user unassign --user john --user jane my_project
  rp project user unassign --from-file users.json my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runProjectUserUnassign),
	}

	cmd.Flags().StringArray("user", nil, "username to remove from the project (repeatable)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing UnassignUsersRQ (use - for stdin)")

	return cmd
}

func runProjectUserAssign(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	request, err := resolveAssignUsersRequest(cmd, printer)
	if err != nil {
		return err
	}

	projectName := args[0]

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("assigning users to project", "name", projectName)

	result, err := deps.client.Projects().AssignUsers(cmd.Context(), projectName, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("assigning users to project %s: %w", projectName, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runProjectUserUnassign(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	request, err := resolveUnassignUsersRequest(cmd, printer)
	if err != nil {
		return err
	}

	projectName := args[0]

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("unassigning users from project", "name", projectName)

	result, err := deps.client.Projects().UnassignUsers(cmd.Context(), projectName, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("unassigning users from project %s: %w", projectName, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// resolveAssignUsersRequest builds AssignUsersRQ from --from-file or --user flags.
func resolveAssignUsersRequest(cmd *cobra.Command, printer *Printer) (types.AssignUsersRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		request, err := readAssignUsersFromFile(path)
		if err != nil {
			printer.PrintError(err)

			return types.AssignUsersRQ{}, err
		}

		return request, nil
	}

	users, _ := cmd.Flags().GetStringArray("user")

	userNames, err := buildAssignUsersMap(users)
	if err != nil {
		printer.PrintError(err)

		return types.AssignUsersRQ{}, err
	}

	return types.AssignUsersRQ{UserNames: userNames}, nil
}

// resolveUnassignUsersRequest builds UnassignUsersRQ from --from-file or --user flags.
func resolveUnassignUsersRequest(
	cmd *cobra.Command,
	printer *Printer,
) (types.UnassignUsersRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		request, err := readUnassignUsersFromFile(path)
		if err != nil {
			printer.PrintError(err)

			return types.UnassignUsersRQ{}, err
		}

		return request, nil
	}

	users, _ := cmd.Flags().GetStringArray("user")

	userNames, err := buildUnassignUsersList(users)
	if err != nil {
		printer.PrintError(err)

		return types.UnassignUsersRQ{}, err
	}

	return types.UnassignUsersRQ{UserNames: userNames}, nil
}

// buildAssignUsersMap parses --user flag values into a username-to-role map.
func buildAssignUsersMap(users []string) (map[string]string, error) {
	if len(users) == 0 {
		return nil, errors.New("--user is required when not using --from-file")
	}

	result := make(map[string]string, len(users))

	for _, user := range users {
		username, role, ok := parseAttribute(user)
		if !ok {
			return nil, fmt.Errorf("invalid --user format %q (expected username:role)", user)
		}

		if _, exists := result[username]; exists {
			return nil, fmt.Errorf("duplicate --user entry for %q", username)
		}

		normalizedRole, err := normalizeProjectRole(role)
		if err != nil {
			return nil, err
		}

		result[username] = normalizedRole
	}

	return result, nil
}

// buildUnassignUsersList validates and returns usernames from --user flags.
func buildUnassignUsersList(users []string) ([]string, error) {
	if len(users) == 0 {
		return nil, errors.New("--user is required when not using --from-file")
	}

	return users, nil
}

// normalizeProjectRole validates and uppercases a project role string.
func normalizeProjectRole(raw string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))

	switch normalized {
	case "MEMBER", "PROJECT_MANAGER", "CUSTOMER", "OPERATOR":
		return normalized, nil
	default:
		return "", fmt.Errorf(
			"invalid role %q: must be MEMBER, PROJECT_MANAGER, CUSTOMER, or OPERATOR",
			raw,
		)
	}
}

// readAssignUsersFromFile reads a JSON file into an AssignUsersRQ.
func readAssignUsersFromFile(path string) (types.AssignUsersRQ, error) {
	result, err := readRequestFromFile[types.AssignUsersRQ](path)
	if err != nil {
		return result, wrapFileError(err, "project user assign payload", "project user assign JSON")
	}

	return result, nil
}

// readUnassignUsersFromFile reads a JSON file into an UnassignUsersRQ.
func readUnassignUsersFromFile(path string) (types.UnassignUsersRQ, error) {
	result, err := readRequestFromFile[types.UnassignUsersRQ](path)
	if err != nil {
		return result, wrapFileError(
			err,
			"project user unassign payload",
			"project user unassign JSON",
		)
	}

	return result, nil
}
