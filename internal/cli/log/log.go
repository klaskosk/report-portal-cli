package log

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// logCreateScope is the narrow interface used to dispatch log creation.
// It is satisfied by *client.LogScope and can be faked in tests.
type logCreateScope interface {
	Create(ctx context.Context, req types.SaveLogRQ) (*types.EntryCreatedAsyncRS, error)
	CreateAsync(ctx context.Context, req types.SaveLogRQ) (*types.EntryCreatedAsyncRS, error)
	CreateBatch(
		ctx context.Context,
		entries ...client.BatchLogEntry,
	) (*types.BatchSaveOperatingRS, error)
	CreateBatchAsync(
		ctx context.Context,
		entries ...client.BatchLogEntry,
	) (*types.BatchSaveOperatingRS, error)
}

// dispatchSingleCreate calls Create or CreateAsync on scope depending on the async flag.
func dispatchSingleCreate(
	ctx context.Context,
	scope logCreateScope,
	req types.SaveLogRQ,
	async bool,
) (*types.EntryCreatedAsyncRS, error) {
	var (
		result *types.EntryCreatedAsyncRS
		err    error
	)

	if async {
		result, err = scope.CreateAsync(ctx, req)
	} else {
		result, err = scope.Create(ctx, req)
	}

	if err != nil {
		return nil, fmt.Errorf("create log: %w", err)
	}

	return result, nil
}

// dispatchBatchCreate calls CreateBatch or CreateBatchAsync on scope depending on the async flag.
func dispatchBatchCreate(
	ctx context.Context,
	scope logCreateScope,
	entries []client.BatchLogEntry,
	async bool,
) (*types.BatchSaveOperatingRS, error) {
	var (
		result *types.BatchSaveOperatingRS
		err    error
	)

	if async {
		result, err = scope.CreateBatchAsync(ctx, entries...)
	} else {
		result, err = scope.CreateBatch(ctx, entries...)
	}

	if err != nil {
		return nil, fmt.Errorf("create batch log: %w", err)
	}

	return result, nil
}

// NewCommand builds the "log" parent command with list, get, create, and delete subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Annotations: map[string]string{core.GroupAnnotationKey: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		Use:   "log",
		Short: "Manage log entries",
	}

	core.RegisterProjectFlag(cmd)
	cmd.AddCommand(newLogListCmd())
	cmd.AddCommand(newLogGetCmd())
	cmd.AddCommand(newLogCreateCmd())
	cmd.AddCommand(newLogDeleteCmd())

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
  rp log list --item 42 -p my_project  List logs for test item 42
  rp log list --launch 10 -o json -p my_project  List logs as JSON
  rp log list --item 42 --level ERROR -p my_project  List error-level logs for item 42`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runLogList),
	}

	core.RegisterListFlagsWithDefaults(cmd, core.ListFlagDefaults{Sort: []string{"logTime,desc"}})
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
  rp log get 100 -p my_project        Get log by numeric ID
  rp log get abc-def-123 -p my_project  Get log by UUID
  rp log get 100 -o json -p my_project  Get log as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runLogGet),
	}
}

// newLogCreateCmd builds the "log create" subcommand for creating log entries.
func newLogCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create one or more log entries from a JSON file",
		Long: `Create one or more log entries from a JSON file.

Log creation always requires a JSON file (--from-file) because the payload
includes structured fields such as timestamps and UUIDs that are cumbersome
to supply via individual flags.

Four creation variants are available, selected by flag combination:
  (default)          single sync   - calls Create
  --async            single async  - calls CreateAsync
  --batch            batch sync    - calls CreateBatch
  --batch --async    batch async   - calls CreateBatchAsync

Note: file attachments are not supported via the CLI. To attach files,
use the API directly.

Examples:
  rp log create --from-file entry.json -p my_project
  rp log create --from-file entry.json --async -p my_project
  rp log create --batch --from-file batch.json -p my_project
  rp log create --batch --async --from-file batch.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runLogCreate),
	}

	cmd.Flags().String("from-file", "", "path to JSON file containing SaveLogRQ (use - for stdin)")
	cmd.Flags().Bool("async", false, "use the asynchronous API variant")
	cmd.Flags().
		Bool("batch", false, "interpret file as an array of batch log entries (attachments not supported)")

	return cmd
}

// newLogDeleteCmd builds the "log delete <id>" subcommand for deleting a log entry.
func newLogDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a log entry by numeric ID",
		Long: `Delete a log entry by numeric ID.

Examples:
  rp log delete 42 -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runLogDelete),
	}
}

// runLogList executes the log list command.
func runLogList(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("listing logs", "project", project)

	return listLogs(cmd, deps.Client, project, deps.Printer)
}

// runLogGet executes the log get command.
func runLogGet(cmd *cobra.Command, args []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("getting single log", "project", project, "idOrUUID", args[0])

	return getSingleLog(cmd, deps.Client, project, args[0], deps.Printer)
}

// runLogCreate executes the log create command, delegating to single or batch helpers.
func runLogCreate(cmd *cobra.Command, _ []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	if !cmd.Flags().Changed("from-file") {
		err := errors.New("--from-file is required for log creation")
		printer.PrintError(err)

		return err
	}

	path, _ := cmd.Flags().GetString("from-file")
	async, _ := cmd.Flags().GetBool("async")
	batch, _ := cmd.Flags().GetBool("batch")

	if batch {
		return runLogCreateBatch(cmd, printer, path, async)
	}

	return runLogCreateSingle(cmd, printer, path, async)
}

// runLogCreateSingle handles single-entry log creation (sync or async).
func runLogCreateSingle(cmd *cobra.Command, printer *core.Printer, path string, async bool) error {
	request, err := readSingleLogEntry(path)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	err = validateSingleLogEntry(request)
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

	deps.Logger.Debug("creating log entry", "project", project, "async", async)

	ctx := cmd.Context()
	scope := deps.Client.Project(project).Logs()

	result, err := dispatchSingleCreate(ctx, scope, request, async)
	if err != nil {
		deps.Printer.PrintError(err)

		return err
	}

	headers := []string{"ID"}
	rows := [][]string{{core.FormatOptional(result.Id)}}

	return deps.Printer.Print(result, headers, rows)
}

// runLogCreateBatch handles batch log creation (sync or async).
func runLogCreateBatch(cmd *cobra.Command, printer *core.Printer, path string, async bool) error {
	jsonEntries, err := readBatchLogEntries(path)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	err = validateBatchLogEntries(jsonEntries)
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

	entries := make([]client.BatchLogEntry, len(jsonEntries))
	for i, e := range jsonEntries {
		entries[i] = toBatchLogEntry(e)
	}

	deps.Logger.Debug(
		"creating batch log entries",
		"project",
		project,
		"count",
		len(entries),
		"async",
		async,
	)

	ctx := cmd.Context()
	scope := deps.Client.Project(project).Logs()

	result, err := dispatchBatchCreate(ctx, scope, entries, async)
	if err != nil {
		deps.Printer.PrintError(err)

		return err
	}

	return deps.Printer.Print(result, batchLogHeaders(), batchLogRows(result))
}

// runLogDelete executes the log delete command.
func runLogDelete(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	logID, err := core.ParseIDArg(args[0], "log")
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

	deps.Logger.Debug("deleting log entry", "project", project, "id", logID)

	result, err := deps.Client.Project(project).Logs().Delete(cmd.Context(), logID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("deleting log %d: %w", logID, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

// getSingleLog retrieves and prints a single log entry by numeric ID or UUID.
func getSingleLog(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project, idOrUUID string,
	printer *core.Printer,
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
	printer *core.Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Logs()

	flags, err := core.ResolveListFlags(cmd)
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
		client.WithLogPageSize(flags.PageSize),
		client.WithLogPageNumber(flags.Page),
		client.WithLogSort(flags.Sort...),
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
	message := core.FormatOptional(logEntry.Message)
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
		core.FormatOptional(logEntry.ItemId),
		core.FormatOptional(logEntry.LaunchId),
		message,
		hasAttachment,
	}
}

// batchLogEntryJSON is the JSON-serializable representation of a log entry for --from-file
// in batch mode. AttachmentName and AttachmentData are not supported via the CLI.
type batchLogEntryJSON struct {
	LaunchUUID string    `json:"launchUuid"`
	ItemUUID   string    `json:"itemUuid,omitempty"`
	Level      string    `json:"level,omitempty"`
	Message    string    `json:"message,omitempty"`
	Time       time.Time `json:"time"`
}

// toBatchLogEntry converts a batchLogEntryJSON to a client.BatchLogEntry.
// Attachment fields are intentionally left empty; file attachments are not
// supported via the CLI.
func toBatchLogEntry(entry batchLogEntryJSON) client.BatchLogEntry {
	return client.BatchLogEntry{
		LaunchUUID: entry.LaunchUUID,
		ItemUUID:   entry.ItemUUID,
		Level:      entry.Level,
		Message:    entry.Message,
		Time:       entry.Time,
	}
}

// batchLogHeaders returns the table column headers for batch log creation output.
func batchLogHeaders() []string {
	return []string{"ID", "MESSAGE"}
}

// batchLogRows formats a BatchSaveOperatingRS as table rows (one per response element).
func batchLogRows(result *types.BatchSaveOperatingRS) [][]string {
	if result.Responses == nil {
		return nil
	}

	rows := make([][]string, 0, len(*result.Responses))
	for _, r := range *result.Responses {
		rows = append(rows, []string{
			core.FormatOptional(r.Id),
			core.FormatOptional(r.Message),
		})
	}

	return rows
}

// readSingleLogEntry reads a SaveLogRQ from the given file path.
func readSingleLogEntry(path string) (types.SaveLogRQ, error) {
	result, err := core.ReadRequestFromFile[types.SaveLogRQ](path)
	if err != nil {
		return result, core.WrapFileError(err, "log entry file", "log entry JSON")
	}

	return result, nil
}

// validateSingleLogEntry checks that required fields of a SaveLogRQ are populated.
func validateSingleLogEntry(req types.SaveLogRQ) error {
	if req.LaunchUuid == "" {
		return errors.New("log entry must have a non-empty launchUuid")
	}

	if req.Time.IsZero() {
		return errors.New("log entry must have a non-zero time")
	}

	return nil
}

// readBatchLogEntries reads a JSON array of batch log entries from the given file path.
func readBatchLogEntries(path string) ([]batchLogEntryJSON, error) {
	result, err := core.ReadRequestFromFile[[]batchLogEntryJSON](path)
	if err != nil {
		return nil, core.WrapFileError(err, "batch log entries file", "batch log entries JSON")
	}

	return result, nil
}

// validateBatchLogEntries checks that the batch is non-empty and each entry is valid.
func validateBatchLogEntries(entries []batchLogEntryJSON) error {
	if len(entries) == 0 {
		return errors.New("batch log entries must contain at least one entry")
	}

	for i, e := range entries {
		if e.LaunchUUID == "" {
			return fmt.Errorf("batch entry %d must have a non-empty launchUuid", i)
		}

		if e.Time.IsZero() {
			return fmt.Errorf("batch entry %d must have a non-zero time", i)
		}
	}

	return nil
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

	itemID, hasItem, err := core.ParseOptionalInt64Flag("item", itemRaw)
	if err != nil {
		return logFilterFlags{}, err
	}

	launchID, hasLaunch, err := core.ParseOptionalInt64Flag("launch", launchRaw)
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
