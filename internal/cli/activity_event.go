package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newActivityEventCmd builds the "activity-event" parent command.
func newActivityEventCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activity-event",
		Short: "Search instance-level activity audit events",
	}
	cmd.AddCommand(newActivityEventListCmd())

	return cmd
}

// newActivityEventListCmd builds the "activity-event list" subcommand.
func newActivityEventListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activity events with optional filters",
		Long: `List activity audit events across all projects.

This command operates at the instance level and does not require --project/-p.

Examples:
  rp activity-event list
  rp activity-event list -o json
  rp activity-event list --user admin
  rp activity-event list --object-type launch --event-name startLaunch
  rp activity-event list --project-name my_project --page 2 --page-size 25`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runActivityEventList),
	}

	cmd.Flags().String("user", "", "filter by subject (user) name")
	cmd.Flags().String("object-type", "", "filter by object type")
	cmd.Flags().String("event-name", "", "filter by event name")
	cmd.Flags().String("project-name", "", "filter by project name")
	registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: []string{"createdAt,desc"}})

	return cmd
}

func runActivityEventList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing activity events")

	return listActivityEvents(cmd, deps.client, deps.printer)
}

func listActivityEvents(cmd *cobra.Command, clientInstance *client.Client, printer *Printer) error {
	flags, err := resolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid list flags: %w", err)
	}

	criteria := buildActivityEventSearchCriteria(cmd)
	sortField, sortOrder := parseActivityEventSort(flags.sort)

	options := []client.SearchActivityEventsOption{
		client.WithEventLimit(flags.pageSize),
		client.WithEventOffset(flags.page * flags.pageSize),
		client.WithEventSort(sortField),
		client.WithEventOrder(client.EventSortOrder(sortOrder)),
	}

	result, err := clientInstance.ActivityEvents().Search(cmd.Context(), criteria, options...)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing activity events: %w", err)
	}

	headers := activityEventHeaders()
	rows := make([][]string, 0, len(result.Items))

	for i := range result.Items {
		rows = append(rows, activityEventRow(&result.Items[i]))
	}

	return printer.Print(result.Items, headers, rows)
}

func buildActivityEventSearchCriteria(cmd *cobra.Command) types.SearchCriteriaRQ {
	var filters []types.SearchCriteria

	if user, _ := cmd.Flags().GetString("user"); user != "" {
		filters = append(filters, client.EventFilterByUser(user))
	}

	if objectType, _ := cmd.Flags().GetString("object-type"); objectType != "" {
		filters = append(filters, client.EventFilterByObjectType(objectType))
	}

	if eventName, _ := cmd.Flags().GetString("event-name"); eventName != "" {
		filters = append(filters, client.EventFilterByEventName(eventName))
	}

	if projectName, _ := cmd.Flags().GetString("project-name"); projectName != "" {
		filters = append(filters, client.EventFilterByProjectName(projectName))
	}

	return client.NewEventSearchCriteria(filters...)
}

// parseActivityEventSort maps the first --sort value to the activity-event API's
// single field and order; additional --sort values are ignored.
func parseActivityEventSort(sort []string) (string, string) {
	const (
		defaultField = "createdAt"
		defaultOrder = "DESC"
	)

	if len(sort) == 0 {
		return defaultField, defaultOrder
	}

	parts := strings.SplitN(sort[0], ",", 2)

	sortField := strings.TrimSpace(parts[0])
	if sortField == "" {
		return defaultField, defaultOrder
	}

	if len(parts) == 2 {
		if dir := strings.ToUpper(strings.TrimSpace(parts[1])); dir != "" {
			return sortField, dir
		}

		return sortField, defaultOrder
	}

	return sortField, defaultOrder
}

func activityEventHeaders() []string {
	return []string{
		"ID",
		"EVENT",
		"OBJECT TYPE",
		"OBJECT NAME",
		"PROJECT",
		"USER",
		"CREATED AT",
	}
}

func activityEventRow(event *types.ActivityEventResource) []string {
	return []string{
		strconv.FormatInt(event.Id, 10),
		event.EventName,
		event.ObjectType,
		event.ObjectName,
		event.ProjectName,
		event.SubjectName,
		event.CreatedAt.UTC().Format(time.DateTime),
	}
}
