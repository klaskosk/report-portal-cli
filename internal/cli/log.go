package cli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newLogCmd builds the "log" parent command with list and get subcommands.
func newLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Manage log entries",
	}

	cmd.PersistentFlags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.AddCommand(newLogListCmd())
	cmd.AddCommand(newLogGetCmd())

	return cmd
}

// newLogListCmd builds the "log list" subcommand for listing log entries.
func newLogListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List log entries in a project",
		Long: `List log entries in a project.

At least one of --item or --launch is required to scope the results.

Examples:
  rp log list --item 42                List logs for test item 42
  rp log list --launch 10              List logs for launch 10
  rp log list --item 42 --level ERROR  List error-level logs for item 42`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runLogList),
	}

	registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: []string{"logTime,desc"}})
	registerLogFilterFlags(cmd)

	return cmd
}

// newLogGetCmd builds the "log get" subcommand for retrieving a single log entry.
func newLogGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-uuid>",
		Short: "Get a specific log entry by numeric ID or UUID",
		Long: `Get a single log entry by numeric ID or UUID.

Examples:
  rp log get 100                      Get log by numeric ID
  rp log get abc-def-123              Get log by UUID`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runLogGet),
	}
}

// runLogList executes the log list command.
func runLogList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing logs", "project", project)

	return listLogs(cmd, deps.client, project, deps.printer)
}

// runLogGet executes the log get command.
func runLogGet(cmd *cobra.Command, args []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("getting single log", "project", project, "idOrUUID", args[0])

	return getSingleLog(cmd, deps.client, project, args[0], deps.printer)
}

// getSingleLog retrieves and prints a single log entry by numeric ID or UUID.
func getSingleLog(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project, idOrUUID string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Logs()

	var (
		logEntry *types.LogResource
		err      error
	)

	numericID, parseErr := strconv.ParseInt(idOrUUID, 10, 64)
	if parseErr == nil {
		logEntry, err = scope.GetByID(ctx, numericID)
	} else {
		logEntry, err = scope.GetByUUID(ctx, idOrUUID)
	}

	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting log %s: %w", idOrUUID, err)
	}

	headers, row := logRow(logEntry)

	return printer.Print(logEntry, headers, [][]string{row})
}

// listLogs retrieves and prints a paginated, filtered list of log entries.
func listLogs(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Logs()

	flags, err := resolveListFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid list flags: %w", err)
	}

	filterFlags, err := resolveLogFilterFlags(cmd)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("invalid filter flags: %w", err)
	}

	if !filterFlags.hasItem && !filterFlags.hasLaunch {
		validationErr := errors.New("listing logs requires --item or --launch to scope results")
		printer.PrintError(validationErr)

		return validationErr
	}

	options := make([]client.ListLogsOption, 0, 6)
	options = append(
		options,
		client.WithLogPageSize(flags.pageSize),
		client.WithLogPageNumber(flags.page),
		client.WithLogSort(flags.sort...),
	)
	options = append(options, applyLogFilterOptions(filterFlags)...)

	result, err := scope.List(ctx, options...)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing logs: %w", err)
	}

	var logs []types.LogResource
	if result.Content != nil {
		logs = *result.Content
	}

	headers := logHeaders()
	rows := make([][]string, 0, len(logs))

	for i := range logs {
		_, row := logRow(&logs[i])
		rows = append(rows, row)
	}

	return printer.Print(logs, headers, rows)
}

// logHeaders returns the table column headers for log output.
func logHeaders() []string {
	return []string{"ID", "LEVEL", "ITEM ID", "LAUNCH ID", "MESSAGE", "HAS ATTACHMENT"}
}

// logRow formats a log entry as a table row, returning both headers and column values. The message is truncated to 80
// characters for readability.
func logRow(logEntry *types.LogResource) ([]string, []string) {
	message := formatStringPtr(logEntry.Message)
	if len(message) > 80 {
		message = message[:80] + "..."
	}

	hasAttachment := "false"
	if logEntry.BinaryContent != nil {
		hasAttachment = "true"
	}

	level := ""
	if logEntry.Level != nil {
		level = *logEntry.Level
	}

	return logHeaders(), []string{
		strconv.FormatInt(logEntry.Id, 10),
		level,
		formatInt64Ptr(logEntry.ItemId),
		formatInt64Ptr(logEntry.LaunchId),
		message,
		hasAttachment,
	}
}

// logFilterFlags holds resolved filter flags for log queries.
type logFilterFlags struct {
	itemID    int64
	hasItem   bool
	launchID  int64
	hasLaunch bool
	level     string
}

// registerLogFilterFlags adds log-specific filter flags to cmd.
func registerLogFilterFlags(cmd *cobra.Command) {
	cmd.Flags().String("item", "", "filter by test item ID")
	cmd.Flags().String("launch", "", "filter by launch ID")
	cmd.Flags().String("level", "", "filter by log level (error, warn, info, debug, fatal)")
}

// validLogLevels is the set of log levels accepted by the --level flag.
var validLogLevels = map[string]bool{
	"ERROR": true, "WARN": true, "INFO": true,
	"DEBUG": true, "FATAL": true,
}

// resolveLogFilterFlags reads the log filter flags from cmd and returns them.
func resolveLogFilterFlags(cmd *cobra.Command) (logFilterFlags, error) {
	itemRaw, _ := cmd.Flags().GetString("item")
	launchRaw, _ := cmd.Flags().GetString("launch")
	level, _ := cmd.Flags().GetString("level")

	itemID, hasItem, err := parseOptionalInt64Flag("item", itemRaw)
	if err != nil {
		return logFilterFlags{}, err
	}

	launchID, hasLaunch, err := parseOptionalInt64Flag("launch", launchRaw)
	if err != nil {
		return logFilterFlags{}, err
	}

	level = strings.ToUpper(strings.TrimSpace(level))

	if level != "" && !validLogLevels[level] {
		return logFilterFlags{}, fmt.Errorf(
			"invalid log level %q: must be one of error, warn, info, debug, fatal", level)
	}

	return logFilterFlags{
		itemID:    itemID,
		hasItem:   hasItem,
		launchID:  launchID,
		hasLaunch: hasLaunch,
		level:     level,
	}, nil
}

// applyLogFilterOptions converts logFilterFlags to client list options.
func applyLogFilterOptions(filters logFilterFlags) []client.ListLogsOption {
	var options []client.ListLogsOption

	if filters.hasItem {
		options = append(options, client.WithLogItemID(filters.itemID))
	}

	if filters.hasLaunch {
		options = append(options, client.WithLogLaunchID(filters.launchID))
	}

	if filters.level != "" {
		options = append(options, client.WithLogLevel(filters.level))
	}

	return options
}
