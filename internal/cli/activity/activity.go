package activity

import (
	"fmt"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewCommand builds the "activity" parent command for project-scoped item activity logs.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Annotations: map[string]string{core.GroupAnnotationKey: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		Use:   "activity",
		Short: "View project-scoped item activity logs",
	}
	core.RegisterProjectFlag(cmd)
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
		RunE: core.WithPrintedErrors(runActivityGet),
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
		RunE: core.WithPrintedErrors(runActivityList),
	}

	cmd.Flags().Int64("item", 0, "test item ID to list activities for")
	_ = cmd.MarkFlagRequired("item")
	core.RegisterListFlagsWithDefaults(
		cmd,
		core.ListFlagDefaults{Sort: []string{"lastModified,desc"}},
	)

	return cmd
}

func runActivityGet(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	activityID, err := parseActivityID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("getting activity", "project", project, "id", activityID)

	return getActivity(cmd, deps.Client, project, activityID, deps.Printer)
}

func runActivityList(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	itemID, _ := cmd.Flags().GetInt64("item")

	deps.Logger.Debug("listing activities", "project", project, "item", itemID)

	return listActivities(cmd, deps.Client, project, itemID, deps.Printer)
}

func getActivity(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	activityID int64,
	printer *core.Printer,
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
	printer *core.Printer,
) error {
	flags, err := core.ResolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid list flags: %w", err)
	}

	options := []client.ListActivitiesOption{
		client.WithActivityPageNumber(flags.Page),
		client.WithActivityPageSize(flags.PageSize),
		client.WithActivitySort(flags.Sort...),
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
	return core.ParseIDArg(raw, "activity")
}
