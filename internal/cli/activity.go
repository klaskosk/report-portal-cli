package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newActivityCmd builds the "activity" parent command for project-scoped item activity logs.
func newActivityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activity",
		Short: "View project-scoped item activity logs",
	}
	cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.AddCommand(newActivityGetCmd())
	cmd.AddCommand(newActivityListCmd())

	return cmd
}

// newActivityGetCmd builds the "activity get" subcommand.
func newActivityGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an activity record by numeric ID",
		Long: `Get a single project-scoped activity record by numeric ID.

Examples:
  rp activity get 123 -p my_project               Get activity by ID
  rp activity get 123 -p my_project -o json       Get activity as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runActivityGet),
	}
}

// newActivityListCmd builds the "activity list" subcommand.
func newActivityListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activity logs for a test item",
		Long: `List activity logs for a specific test item in a project.

Examples:
  rp activity list --item 456 -p my_project
  rp activity list --item 456 -o json -p my_project  List activities as JSON
  rp activity list --item 456 -p my_project --page 2 --page-size 25`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runActivityList),
	}

	cmd.Flags().Int64("item", 0, "test item ID to list activities for")
	_ = cmd.MarkFlagRequired("item")
	registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: []string{"lastModified,desc"}})

	return cmd
}

func runActivityGet(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	activityID, err := parseActivityID(args[0])
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

	deps.logger.Debug("getting activity", "project", project, "id", activityID)

	return getActivity(cmd, deps.client, project, activityID, deps.printer)
}

func runActivityList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	itemID, _ := cmd.Flags().GetInt64("item")

	deps.logger.Debug("listing activities", "project", project, "item", itemID)

	return listActivities(cmd, deps.client, project, itemID, deps.printer)
}

func getActivity(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	activityID int64,
	printer *Printer,
) error {
	activity, err := clientInstance.Project(project).Activities().GetByID(cmd.Context(), activityID)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting activity %d: %w", activityID, err)
	}

	headers := activityHeaders()
	row := activityRow(activity)

	return printer.Print(activity, headers, [][]string{row})
}

func listActivities(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	itemID int64,
	printer *Printer,
) error {
	flags, err := resolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid list flags: %w", err)
	}

	options := []client.ListActivitiesOption{
		client.WithActivityPageNumber(flags.page),
		client.WithActivityPageSize(flags.pageSize),
		client.WithActivitySort(flags.sort...),
	}

	result, err := clientInstance.Project(project).
		Activities().
		ListForItem(cmd.Context(), itemID, options...)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing activities for item %d: %w", itemID, err)
	}

	headers := activityHeaders()
	rows := make([][]string, 0, len(result.Content))

	for i := range result.Content {
		rows = append(rows, activityListRow(&result.Content[i]))
	}

	return printer.Print(result, headers, rows)
}

func activityHeaders() []string {
	return []string{
		"ID",
		"USER",
		"OBJECT TYPE",
		"ACTION TYPE",
		"CREATED AT",
	}
}

func activityRow(activity *types.ActivityResource) []string {
	return []string{
		strconv.FormatInt(activity.Id, 10),
		activity.User,
		activity.ObjectType,
		activity.ActionType,
		activity.LastModified.UTC().Format(time.DateTime),
	}
}

func activityListRow(event *types.ActivityEventResource) []string {
	return []string{
		strconv.FormatInt(event.Id, 10),
		event.SubjectName,
		event.ObjectType,
		event.EventName,
		event.CreatedAt.UTC().Format(time.DateTime),
	}
}

func parseActivityID(raw string) (int64, error) {
	return parseIDArg(raw, "activity")
}
