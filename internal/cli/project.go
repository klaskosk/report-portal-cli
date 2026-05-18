package cli

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newProjectCmd builds the "project" parent command with list and get subcommands.
func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectGetCmd())

	return cmd
}

// newProjectListCmd builds the "project list" subcommand for listing accessible projects.
func newProjectListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List accessible projects",
		Long: `List accessible projects.

This command does not require --project/-p since it operates on the instance level.

Examples:
  rp project list                   List accessible projects
  rp project list -o json           List accessible projects as JSON`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runProjectList),
	}
}

// newProjectGetCmd builds the "project get" subcommand for retrieving a specific project.
func newProjectGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Get details of a specific project by name",
		Long: `Get details of a specific project by name.

This command does not require --project/-p since it operates on the instance level.

Examples:
  rp project get my_project          Get project details by name
  rp project get my_project -o json  Get project details as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runProjectGet),
	}
}

// runProjectList executes the project list command.
func runProjectList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing projects")

	return listProjects(cmd, deps.client, deps.printer)
}

// runProjectGet executes the project get command.
func runProjectGet(cmd *cobra.Command, args []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("getting single project", "name", args[0])

	return getSingleProject(cmd, deps.client, args[0], deps.printer)
}

// getSingleProject retrieves and prints a single project by name.
func getSingleProject(
	cmd *cobra.Command,
	clientInstance *client.Client,
	name string,
	printer *Printer,
) error {
	project, err := clientInstance.Projects().Get(cmd.Context(), name)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting project %s: %w", name, err)
	}

	headers, row := projectRow(project)

	return printer.Print(project, headers, [][]string{row})
}

// listProjects retrieves and prints all projects accessible to the current user.
func listProjects(
	cmd *cobra.Command,
	clientInstance *client.Client,
	printer *Printer,
) error {
	ctx := cmd.Context()

	// Get current user to find their accessible projects.
	user, err := clientInstance.Users().GetMyself(ctx)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting current user: %w", err)
	}

	projects, err := clientInstance.Users().ListProjects(ctx, user.UserId)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing projects: %w", err)
	}

	names := make([]string, 0, len(projects))
	for name := range projects {
		names = append(names, name)
	}

	sort.Strings(names)

	if printer.isJSON() {
		return printer.PrintJSON(names)
	}

	if tmpl, ok := printer.goTemplateString(); ok {
		return printer.PrintTemplate(tmpl, names)
	}

	headers := []string{"PROJECT NAME"}
	rows := make([][]string, 0, len(names))

	for _, name := range names {
		rows = append(rows, []string{name})
	}

	return printer.PrintTable(headers, rows)
}

// projectRow formats a project resource as a table row, returning both headers and column values.
func projectRow(project *types.ProjectResource) ([]string, []string) {
	headers := []string{"ID", "NAME", "ENTRY TYPE", "ORGANIZATION", "CREATED"}

	organization := ""
	if project.Organization != nil {
		organization = *project.Organization
	}

	created := project.CreationDate.UTC().Format(time.DateTime)

	return headers, []string{
		strconv.FormatInt(project.ProjectId, 10),
		project.ProjectName,
		project.EntryType,
		organization,
		created,
	}
}
