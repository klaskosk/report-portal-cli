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

// newGetLogCmd builds the "get log" subcommand for listing or retrieving log entries.
func newGetLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log [id-or-uuid]",
		Short: "List logs or get a specific log entry",
		Long: `List log entries in a project, or get a single log by numeric ID or UUID.

When listing, at least one of --item or --launch is required to scope the results.

Examples:
  rp get log --item 42                List logs for test item 42
  rp get log --launch 10              List logs for launch 10
  rp get log --item 42 --level ERROR  List error-level logs for item 42
  rp get log 100                      Get log by numeric ID
  rp get log abc-def-123              Get log by UUID`,
		Args: cobra.MaximumNArgs(1),
		RunE: withPrintedErrors(runGetLog),
	}

	cmd.Flags().StringP("project", "p", "", "project name (overrides profile default)")
	registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: []string{"logTime,desc"}})
	registerLogFilterFlags(cmd)

	return cmd
}

// runGetLog executes the get log command, dispatching to single-get or list mode.
func runGetLog(cmd *cobra.Command, args []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	project, err := resolveProject(cmd, conn)
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
		logger.Debug("getting single log", "project", project, "idOrUUID", args[0])

		return getSingleLog(cmd, clientInstance, project, args[0], printer)
	}

	logger.Debug("listing logs", "project", project)

	return listLogs(cmd, clientInstance, project, printer)
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

	if isNumericID(idOrUUID) {
		logEntry, err = scope.GetByID(ctx, idOrUUID)
	} else {
		logEntry, err = scope.GetByUUID(ctx, idOrUUID)
	}

	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting log %s: %w", idOrUUID, err)
	}

	headers, row := logRow(logEntry)

	return printSingleResource(printer, logEntry, headers, row, logRequiredJSONKeys, "log")
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

	headers := logHeaders()
	rows := make([][]string, 0, len(result.Content))

	for i := range result.Content {
		_, row := logRow(&result.Content[i])
		rows = append(rows, row)
	}

	return printResourceList(printer, result.Content, headers, rows, logRequiredJSONKeys, "log")
}

// logHeaders returns the table column headers for log output.
func logHeaders() []string {
	return []string{"ID", "LEVEL", "ITEM ID", "LAUNCH ID", "MESSAGE", "HAS ATTACHMENT"}
}

var logRequiredJSONKeys = []string{"id", "uuid", "level", "message", "itemId", "launchId"}

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
		level = string(*logEntry.Level)
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
