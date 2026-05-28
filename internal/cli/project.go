package cli

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newProjectCmd builds the "project" parent command with list, get, create, update, and delete subcommands.
func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddCommand(newProjectListCmd())
	cmd.AddCommand(newProjectGetCmd())
	cmd.AddCommand(newProjectCreateCmd())
	cmd.AddCommand(newProjectUpdateCmd())
	cmd.AddCommand(newProjectDeleteCmd())

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

// newProjectCreateCmd builds the "project create" subcommand.
func newProjectCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Long: `Create a new project.

This command does not require --project/-p since it operates on the instance level.

Examples:
  rp project create --name "new_project"                Create a project
  rp project create --name "new_project" --entry-type INTERNAL  Create with explicit entry type`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runProjectCreate),
	}

	cmd.Flags().String("name", "", "project name (required)")
	cmd.Flags().String("entry-type", "INTERNAL", "project entry type")

	return cmd
}

// newProjectUpdateCmd builds the "project update" subcommand.
func newProjectUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update a project from a JSON file",
		Long: `Update a project using a JSON payload from a file.

The JSON file may contain "configuration" (project attributes) and/or "users"
(user-to-role assignments). Both fields are optional.

This command does not require --project/-p since it operates on the instance level.

Examples:
  rp project update my_project --from-file config.json  Update project configuration`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runProjectUpdate),
	}

	cmd.Flags().
		String("from-file", "", "path to JSON file containing UpdateProjectRQ payload (use - for stdin)")

	return cmd
}

// newProjectDeleteCmd builds the "project delete" subcommand.
func newProjectDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a project by name",
		Long: `Delete a project by name.

The project name is resolved to a numeric ID before deletion, since the API
deletes by ID.

This command does not require --project/-p since it operates on the instance level.

Examples:
  rp project delete old_project   Delete a project by name`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runProjectDelete),
	}
}

func runProjectCreate(cmd *cobra.Command, _ []string) error {
	name := mustGetStringFlag(cmd, "name")
	if name == "" {
		err := errors.New("--name is required")
		resolvePrinter(cmd).PrintError(err)

		return err
	}

	entryType := mustGetStringFlag(cmd, "entry-type")

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("creating project", "name", name, "entryType", entryType)

	request := types.CreateProjectRQ{
		ProjectName: name,
		EntryType:   types.CreateProjectRQEntryType(entryType),
	}

	result, err := deps.client.Projects().Create(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating project: %w", err)
	}

	headers := []string{"ID"}
	rows := [][]string{{formatOptional(result.Id)}}

	return deps.printer.Print(result, headers, rows)
}

func runProjectUpdate(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	if !cmd.Flags().Changed("from-file") {
		err := errors.New(
			"--from-file is required for project update (the payload contains structured configuration and user fields)",
		)
		printer.PrintError(err)

		return err
	}

	path := mustGetStringFlag(cmd, "from-file")

	request, err := readProjectUpdateFromFile(path)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	projectName := args[0]

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("updating project", "name", projectName)

	result, err := deps.client.Projects().Update(cmd.Context(), projectName, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating project %s: %w", projectName, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("resolving project ID for deletion", "name", projectName)

	return deleteProject(cmd, deps.client, projectName, deps.printer)
}

func deleteProject(
	cmd *cobra.Command,
	clientInstance *client.Client,
	name string,
	printer *Printer,
) error {
	project, err := clientInstance.Projects().Get(cmd.Context(), name)
	if err != nil {
		var wrappedErr error
		if client.IsNotFound(err) {
			wrappedErr = fmt.Errorf("project %q not found: cannot delete: %w", name, err)
		} else {
			wrappedErr = fmt.Errorf("resolving project %q for deletion: %w", name, err)
		}

		printer.PrintError(wrappedErr)

		return wrappedErr
	}

	result, err := clientInstance.Projects().Delete(cmd.Context(), project.ProjectId)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("deleting project %s: %w", name, err)
	}

	headers, rows := formatDeleteBulkResult(result, fmt.Sprintf("Project '%s'", name))

	return printer.Print(result, headers, rows)
}

// readProjectUpdateFromFile reads a JSON file into an UpdateProjectRQ.
func readProjectUpdateFromFile(path string) (types.UpdateProjectRQ, error) {
	result, err := readRequestFromFile[types.UpdateProjectRQ](path)
	if err != nil {
		return result, wrapFileError(err, "project update payload", "project update JSON")
	}

	return result, nil
}
