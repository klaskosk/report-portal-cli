package setting

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newSettingNotificationCmd builds the "setting notification" grouping command.
func newSettingNotificationCmd() *cobra.Command {
	cmd := core.MarkGroupCommand(&cobra.Command{
		Use:   "notification",
		Short: "Manage project notification rules",
	})
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
		RunE: core.WithPrintedErrors(runSettingNotificationList),
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
		RunE: core.WithPrintedErrors(runSettingNotificationCreate),
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
		RunE: core.WithPrintedErrors(runSettingNotificationUpdate),
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
		RunE: core.WithPrintedErrors(runSettingNotificationDelete),
	}
}

func runSettingNotificationList(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("listing notification rules", "project", project)

	notifications, err := deps.Client.Project(project).Settings().GetNotifications(cmd.Context())
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("listing notification rules in project %s: %w", project, err)
	}

	headers := notificationHeaders()
	rows := make([][]string, 0, len(notifications))

	for i := range notifications {
		rows = append(rows, notificationRow(&notifications[i]))
	}

	return deps.Printer.Print(notifications, headers, rows)
}

func runSettingNotificationCreate(cmd *cobra.Command, _ []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	path, err := core.RequireFromFile(
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

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("creating notification rule", "project", project)

	result, err := deps.Client.Project(project).
		Settings().
		CreateNotification(cmd.Context(), request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("creating notification rule in project %s: %w", project, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{core.FormatOptional(result.Id)}}

	return deps.Printer.Print(result, headers, rows)
}

func runSettingNotificationUpdate(cmd *cobra.Command, _ []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	path, err := core.RequireFromFile(
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

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("updating notification rule", "project", project, "id", request.Id)

	result, err := deps.Client.Project(project).
		Settings().
		UpdateNotification(cmd.Context(), request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("updating notification rule in project %s: %w", project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func runSettingNotificationDelete(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	notificationID, err := parseNotificationID(args[0])
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

	deps.Logger.Debug("deleting notification rule", "project", project, "id", notificationID)

	result, err := deps.Client.Project(project).
		Settings().
		DeleteNotification(cmd.Context(), notificationID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf(
			"deleting notification rule %d in project %s: %w",
			notificationID,
			project,
			err,
		)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func readNotificationRequestFromFile(path string) (types.SenderCaseDTO, error) {
	result, err := core.ReadRequestFromFile[types.SenderCaseDTO](path)
	if err != nil {
		return result, core.WrapFileError(err, "notification rule", "notification rule JSON")
	}

	return result, nil
}

func notificationHeaders() []string {
	return []string{"ID", "RULE NAME", "SEND CASE", "RECIPIENTS", "ENABLED"}
}

func notificationRow(notification *types.SenderCaseDTO) []string {
	return []string{
		core.FormatOptional(notification.Id),
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
	return core.ParseIDArg(raw, "notification")
}
