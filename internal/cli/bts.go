package cli

import (
	"fmt"
	"strconv"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newBtsCmd builds the "bts" parent command for bug tracking system operations.
func newBtsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bts",
		Short: "Bug tracking system operations",
	}

	cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.AddCommand(newBtsTicketCmd())
	cmd.AddCommand(newBtsFieldCmd())
	cmd.AddCommand(newBtsIssueCmd())

	return cmd
}

// newBtsTicketCmd builds the "bts ticket" grouping command.
func newBtsTicketCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ticket",
		Short: "Retrieve BTS tickets",
	}
	cmd.AddCommand(newBtsTicketGetCmd())

	return cmd
}

// newBtsTicketGetCmd builds the "bts ticket get" subcommand.
func newBtsTicketGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <ticket-id>",
		Short: "Get a BTS ticket by ID",
		Long: `Get a ticket from the configured bug tracking system.

Examples:
  rp bts ticket get PROJ-123 --bts-url https://jira.example.com \
    --bts-project PROJ -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runBtsTicketGet),
	}

	cmd.Flags().String("bts-url", "", "BTS instance URL (required)")
	cmd.Flags().String("bts-project", "", "BTS project key (required)")
	_ = cmd.MarkFlagRequired("bts-url")
	_ = cmd.MarkFlagRequired("bts-project")

	return cmd
}

// newBtsFieldCmd builds the "bts field" grouping command.
func newBtsFieldCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "field",
		Short: "List BTS form fields",
	}
	cmd.AddCommand(newBtsFieldListCmd())

	return cmd
}

// newBtsFieldListCmd builds the "bts field list" subcommand.
func newBtsFieldListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List BTS system fields for an integration",
		Long: `List the form fields available when creating issues in a BTS integration.

Examples:
  rp bts field list --integration 1 -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runBtsFieldList),
	}

	cmd.Flags().Int64("integration", 0, "integration ID (required)")
	_ = cmd.MarkFlagRequired("integration")

	return cmd
}

// newBtsIssueCmd builds the "bts issue" grouping command.
func newBtsIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Create BTS issues",
	}
	cmd.AddCommand(newBtsIssueCreateCmd())

	return cmd
}

// newBtsIssueCreateCmd builds the "bts issue create" subcommand.
func newBtsIssueCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a BTS issue from a JSON request file",
		Long: `Create a new issue in the bug tracking system.

Examples:
  rp bts issue create --integration 1 --from-file issue.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runBtsIssueCreate),
	}

	cmd.Flags().Int64("integration", 0, "integration ID (required)")
	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing PostTicketRQ (use - for stdin)",
	)
	_ = cmd.MarkFlagRequired("integration")
	_ = cmd.MarkFlagRequired("from-file")

	return cmd
}

func runBtsTicketGet(cmd *cobra.Command, args []string) error {
	ticketID := args[0]
	btsURL := mustGetStringFlag(cmd, "bts-url")
	btsProject := mustGetStringFlag(cmd, "bts-project")

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug(
		"getting bts ticket",
		"project", project,
		"ticketId", ticketID,
		"btsUrl", btsURL,
		"btsProject", btsProject,
	)

	ticket, err := deps.client.Project(project).BugTracking().GetTicket(
		cmd.Context(), ticketID, btsURL, btsProject,
	)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("getting bts ticket %s: %w", ticketID, err)
	}

	headers, row := btsTicketRow(ticket)

	return deps.printer.Print(ticket, headers, [][]string{row})
}

func runBtsFieldList(cmd *cobra.Command, _ []string) error {
	integrationID, err := cmd.Flags().GetInt64("integration")
	if err != nil {
		return fmt.Errorf("reading --integration flag: %w", err)
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing bts fields", "project", project, "integrationId", integrationID)

	fields, err := deps.client.Project(project).
		BugTracking().
		GetSystemFields(cmd.Context(), integrationID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("listing bts fields for integration %d: %w", integrationID, err)
	}

	headers := btsFieldHeaders()
	rows := make([][]string, 0, len(fields))

	for i := range fields {
		rows = append(rows, btsFieldRow(&fields[i]))
	}

	return deps.printer.Print(fields, headers, rows)
}

func runBtsIssueCreate(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	path, err := requireFromFile(
		cmd,
		printer,
		"--from-file is required for bts issue creation",
	)
	if err != nil {
		return err
	}

	request, err := readRequestFromFile[types.PostTicketRQ](path)
	if err != nil {
		printer.PrintError(err)

		return wrapFileError(err, "bts issue request", "bts issue JSON")
	}

	integrationID, err := cmd.Flags().GetInt64("integration")
	if err != nil {
		return fmt.Errorf("reading --integration flag: %w", err)
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("creating bts issue", "project", project, "integrationId", integrationID)

	ticket, err := deps.client.Project(project).BugTracking().CreateIssue(
		cmd.Context(), integrationID, request,
	)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating bts issue for integration %d: %w", integrationID, err)
	}

	headers, row := btsTicketRow(ticket)

	return deps.printer.Print(ticket, headers, [][]string{row})
}

func btsTicketHeaders() []string {
	return []string{"ID", "SUMMARY", "STATUS", "URL", "PLUGIN"}
}

func btsTicketRow(ticket *types.Ticket) ([]string, []string) {
	return btsTicketHeaders(), []string{
		formatOptional(ticket.Id),
		formatOptional(ticket.Summary),
		formatOptional(ticket.Status),
		formatOptional(ticket.Url),
		formatOptional(ticket.PluginName),
	}
}

func btsFieldHeaders() []string {
	return []string{"ID", "NAME", "TYPE", "REQUIRED", "DESCRIPTION"}
}

func btsFieldRow(field *types.PostFormField) []string {
	return []string{
		field.Id,
		formatOptional(field.FieldName),
		formatOptional(field.FieldType),
		strconv.FormatBool(field.Required),
		formatOptional(field.Description),
	}
}
