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

// newGetProjectCmd builds the "get project" subcommand for listing or retrieving projects.
func newGetProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project [name]",
		Short: "List projects or get a specific project",
		Long: `List accessible projects, or get details of a specific project by name.

This command does not require --project/-p since it operates on the instance level.

Examples:
  rp get project                   List accessible projects
  rp get project my_project        Get project details by name
  rp get project my_project -o json  Get project details as JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: withPrintedErrors(runGetProject),
	}

	return cmd
}

// runGetProject executes the get project command, dispatching to single-get or list mode.
func runGetProject(cmd *cobra.Command, args []string) error {
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

	if len(args) == 1 {
		logger.Debug("getting single project", "name", args[0])

		return getSingleProject(cmd, clientInstance, args[0], printer)
	}

	logger.Debug("listing projects")

	return listProjects(cmd, clientInstance, printer)
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

	created := ""
	if project.CreationDate != nil {
		created = time.Time(*project.CreationDate).UTC().Format(time.DateTime)
	}

	return headers, []string{
		strconv.FormatInt(project.ProjectId, 10),
		project.ProjectName,
		project.EntryType,
		organization,
		created,
	}
}
