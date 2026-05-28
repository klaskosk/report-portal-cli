package cli

import (
	"errors"
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newItemIssueCmd builds the "item issue" grouping command.
func newItemIssueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Link or unlink external BTS issues on test items",
	}

	cmd.AddCommand(newItemIssueLinkCmd())
	cmd.AddCommand(newItemIssueUnlinkCmd())

	return cmd
}

// newItemIssueLinkCmd builds the "item issue link" subcommand.
func newItemIssueLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Link an external BTS issue to test items",
		Long: `Link an external bug tracking system issue to one or more test items.

Flags or --from-file may be provided. When --from-file is given, flags are ignored.

Examples:
  rp item issue link --item-id 100 --ticket-id PROJ-456 \
    --url https://jira.example.com/browse/PROJ-456 -p my_project
  rp item issue link --item-id 100 --item-id 101 --ticket-id PROJ-456 \
    --url https://jira.example.com/browse/PROJ-456 --bts-url https://jira.example.com \
    --bts-project PROJ -p my_project
  rp item issue link --from-file link_issue.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runItemIssueLink),
	}

	cmd.Flags().Int64Slice("item-id", nil, "test item ID to link (repeatable)")
	cmd.Flags().String("ticket-id", "", "external ticket ID (required without --from-file)")
	cmd.Flags().String("url", "", "URL of the external ticket (required without --from-file)")
	cmd.Flags().String("bts-url", "", "BTS instance URL")
	cmd.Flags().String("bts-project", "", "BTS project key")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing LinkExternalIssueRQ (use - for stdin)")

	return cmd
}

// newItemIssueUnlinkCmd builds the "item issue unlink" subcommand.
func newItemIssueUnlinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlink",
		Short: "Unlink external BTS issues from test items",
		Long: `Remove external bug tracking system issue links from test items.

Flags or --from-file may be provided. When --from-file is given, flags are ignored.

Examples:
  rp item issue unlink --item-id 100 --ticket-id PROJ-456 -p my_project
  rp item issue unlink --item-id 100 --ticket-id PROJ-456 --ticket-id PROJ-789 -p my_project
  rp item issue unlink --from-file unlink_issue.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runItemIssueUnlink),
	}

	cmd.Flags().Int64Slice("item-id", nil, "test item ID to unlink (repeatable)")
	cmd.Flags().StringSlice("ticket-id", nil, "ticket ID to unlink (repeatable)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing UnlinkExternalIssueRQ (use - for stdin)")

	return cmd
}

// runItemIssueLink executes the item issue link command.
func runItemIssueLink(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	request, err := buildLinkExternalIssueRequest(cmd)
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

	deps.logger.Debug("linking external issue to test items", "project", project)

	results, err := deps.client.Project(project).Items().LinkExternalIssues(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("linking external issue: %w", err)
	}

	return deps.printer.Print(results, operationResultHeaders(), operationResultsRows(results))
}

// runItemIssueUnlink executes the item issue unlink command.
func runItemIssueUnlink(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	request, err := buildUnlinkExternalIssueRequest(cmd)
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

	deps.logger.Debug("unlinking external issue from test items", "project", project)

	results, err := deps.client.Project(project).
		Items().
		UnlinkExternalIssues(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("unlinking external issue: %w", err)
	}

	return deps.printer.Print(results, operationResultHeaders(), operationResultsRows(results))
}

// buildLinkExternalIssueRequest builds a LinkExternalIssueRQ from flags or a file.
func buildLinkExternalIssueRequest(cmd *cobra.Command) (types.LinkExternalIssueRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		request, err := readRequestFromFile[types.LinkExternalIssueRQ](path)
		if err != nil {
			return types.LinkExternalIssueRQ{}, wrapFileError(
				err,
				"item issue link file",
				"item issue link JSON",
			)
		}

		return request, nil
	}

	return buildLinkExternalIssueRequestFromFlags(cmd)
}

// buildLinkExternalIssueRequestFromFlags builds a LinkExternalIssueRQ from individual flags.
func buildLinkExternalIssueRequestFromFlags(cmd *cobra.Command) (types.LinkExternalIssueRQ, error) {
	itemIDs, _ := cmd.Flags().GetInt64Slice("item-id")
	ticketID, _ := cmd.Flags().GetString("ticket-id")
	url, _ := cmd.Flags().GetString("url")

	if len(itemIDs) == 0 {
		return types.LinkExternalIssueRQ{}, errors.New(
			"--item-id is required when not using --from-file",
		)
	}

	if ticketID == "" {
		return types.LinkExternalIssueRQ{}, errors.New(
			"--ticket-id is required when not using --from-file",
		)
	}

	if url == "" {
		return types.LinkExternalIssueRQ{}, errors.New(
			"--url is required when not using --from-file",
		)
	}

	issue := types.ExternalSystemIssue{
		TicketId: ticketID,
		Url:      url,
	}

	if cmd.Flags().Changed("bts-url") {
		issue.BtsUrl = mustGetStringFlag(cmd, "bts-url")
	}

	if cmd.Flags().Changed("bts-project") {
		issue.BtsProject = mustGetStringFlag(cmd, "bts-project")
	}

	return types.LinkExternalIssueRQ{
		TestItemIds: itemIDs,
		Issues:      issue,
	}, nil
}

// buildUnlinkExternalIssueRequest builds an UnlinkExternalIssueRQ from flags or a file.
func buildUnlinkExternalIssueRequest(cmd *cobra.Command) (types.UnlinkExternalIssueRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		request, err := readRequestFromFile[types.UnlinkExternalIssueRQ](path)
		if err != nil {
			return types.UnlinkExternalIssueRQ{}, wrapFileError(
				err,
				"item issue unlink file",
				"item issue unlink JSON",
			)
		}

		return request, nil
	}

	return buildUnlinkExternalIssueRequestFromFlags(cmd)
}

// buildUnlinkExternalIssueRequestFromFlags builds an UnlinkExternalIssueRQ from individual flags.
func buildUnlinkExternalIssueRequestFromFlags(
	cmd *cobra.Command,
) (types.UnlinkExternalIssueRQ, error) {
	itemIDs, _ := cmd.Flags().GetInt64Slice("item-id")
	ticketIDs, _ := cmd.Flags().GetStringSlice("ticket-id")

	if len(itemIDs) == 0 {
		return types.UnlinkExternalIssueRQ{}, errors.New(
			"--item-id is required when not using --from-file",
		)
	}

	if len(ticketIDs) == 0 {
		return types.UnlinkExternalIssueRQ{}, errors.New(
			"--ticket-id is required when not using --from-file",
		)
	}

	return types.UnlinkExternalIssueRQ{
		TestItemIds: itemIDs,
		TicketIds:   ticketIDs,
	}, nil
}
