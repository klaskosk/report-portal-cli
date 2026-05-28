package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newSettingNotificationCmd builds the "setting notification" grouping command.
func newSettingNotificationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notification",
		Short: "Manage project notification rules",
	}
	cmd.AddCommand(newSettingNotificationListCmd())
	cmd.AddCommand(newSettingNotificationCreateCmd())
	cmd.AddCommand(newSettingNotificationUpdateCmd())
	cmd.AddCommand(newSettingNotificationDeleteCmd())

	return cmd
}

func newSettingNotificationListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List project notification rules",
		Long: `List notification rules configured for a project.

Examples:
  rp setting notification list -p my_project
  rp setting notification list -p my_project -o json`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runSettingNotificationList),
	}
}

func newSettingNotificationCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a notification rule from a JSON file",
		Long: `Create a notification rule from a JSON file describing SenderCaseDTO.

Examples:
  rp setting notification create --from-file rule.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runSettingNotificationCreate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing SenderCaseDTO (use - for stdin)",
	)
	_ = cmd.MarkFlagRequired("from-file")

	return cmd
}

func newSettingNotificationUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a notification rule from a JSON file",
		Long: `Update a notification rule from a JSON file describing SenderCaseDTO.

The JSON must include the notification rule ID.

Examples:
  rp setting notification update --from-file rule.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runSettingNotificationUpdate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing SenderCaseDTO (use - for stdin)",
	)
	_ = cmd.MarkFlagRequired("from-file")

	return cmd
}

func newSettingNotificationDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a notification rule by numeric ID",
		Long: `Delete a notification rule by numeric ID.

Examples:
  rp setting notification delete 42 -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runSettingNotificationDelete),
	}
}

func runSettingNotificationList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing notification rules", "project", project)

	notifications, err := deps.client.Project(project).Settings().GetNotifications(cmd.Context())
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("listing notification rules in project %s: %w", project, err)
	}

	headers := notificationHeaders()
	rows := make([][]string, 0, len(notifications))

	for i := range notifications {
		rows = append(rows, notificationRow(&notifications[i]))
	}

	return deps.printer.Print(notifications, headers, rows)
}

func runSettingNotificationCreate(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	path, err := requireFromFile(
		cmd,
		printer,
		"--from-file is required for notification rule creation",
	)
	if err != nil {
		return err
	}

	request, err := readNotificationRequestFromFile(path)
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

	deps.logger.Debug("creating notification rule", "project", project)

	result, err := deps.client.Project(project).
		Settings().
		CreateNotification(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating notification rule in project %s: %w", project, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{formatOptional(result.Id)}}

	return deps.printer.Print(result, headers, rows)
}

func runSettingNotificationUpdate(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	path, err := requireFromFile(
		cmd,
		printer,
		"--from-file is required for notification rule update",
	)
	if err != nil {
		return err
	}

	request, err := readNotificationRequestFromFile(path)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	if request.Id == nil {
		err := errors.New("notification rule ID must be set in --from-file payload")
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

	deps.logger.Debug("updating notification rule", "project", project, "id", request.Id)

	result, err := deps.client.Project(project).
		Settings().
		UpdateNotification(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating notification rule in project %s: %w", project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runSettingNotificationDelete(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	notificationID, err := parseNotificationID(args[0])
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

	deps.logger.Debug("deleting notification rule", "project", project, "id", notificationID)

	result, err := deps.client.Project(project).
		Settings().
		DeleteNotification(cmd.Context(), notificationID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf(
			"deleting notification rule %d in project %s: %w",
			notificationID,
			project,
			err,
		)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func readNotificationRequestFromFile(path string) (types.SenderCaseDTO, error) {
	result, err := readRequestFromFile[types.SenderCaseDTO](path)
	if err != nil {
		return result, wrapFileError(err, "notification rule", "notification rule JSON")
	}

	return result, nil
}

func notificationHeaders() []string {
	return []string{"ID", "RULE NAME", "SEND CASE", "RECIPIENTS", "ENABLED"}
}

func notificationRow(notification *types.SenderCaseDTO) []string {
	return []string{
		formatOptional(notification.Id),
		notification.RuleName,
		string(notification.SendCase),
		formatStringSlice(notification.Recipients),
		formatOptionalBool(notification.Enabled),
	}
}

func formatStringSlice(values *[]string) string {
	if values == nil || len(*values) == 0 {
		return ""
	}

	return strings.Join(*values, ", ")
}

func formatOptionalBool(value *bool) string {
	if value == nil {
		return ""
	}

	if *value {
		return "true"
	}

	return "false"
}

func parseNotificationID(raw string) (int64, error) {
	return parseIDArg(raw, "notification")
}
