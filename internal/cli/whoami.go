package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newWhoamiCmd builds the "whoami" subcommand for displaying the current user.
func newWhoamiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Display the current authenticated user",
		Long: `Show information about the currently authenticated user and their accessible projects.

This command does not require a project to be specified.

Examples:
  rp whoami                Show current user info as table
  rp whoami -o json        Show current user info as JSON`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runWhoami),
	}

	return cmd
}

// runWhoami executes the whoami command, fetching the current user and their projects.
func runWhoami(cmd *cobra.Command, _ []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	ctx := cmd.Context()

	user, err := clientInstance.Users().GetMyself(ctx)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting current user: %w", err)
	}

	// Fetch projects for the user.
	projects, err := clientInstance.Users().ListProjects(ctx, user.UserId)
	if err != nil {
		logger.Debug("could not list projects for user", "error", err)
		printer.PrintWarning(fmt.Sprintf("could not list projects: %s", err))
	}

	projectNames := sortedProjectNames(projects)

	return printer.Print(user, whoamiHeaders(), whoamiRows(user, projectNames))
}

// whoamiHeaders returns the table column headers for whoami output.
func whoamiHeaders() []string {
	return []string{"FIELD", "VALUE"}
}

// whoamiRows formats a user resource and their project names as key-value table rows.
func whoamiRows(user *types.UserResource, projectNames []string) [][]string {
	projectList := strings.Join(projectNames, ", ")
	if projectList == "" {
		projectList = "(none)"
	}

	return [][]string{
		{"Login", user.UserId},
		{"Full Name", formatStringPtr(user.FullName)},
		{"Email", user.Email},
		{"Role", formatStringPtr(user.UserRole)},
		{"Projects", projectList},
	}
}

// sortedProjectNames extracts and returns project names from the map in alphabetical order.
func sortedProjectNames(projects map[string]types.AssignedProject) []string {
	names := make([]string, 0, len(projects))
	for name := range projects {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
