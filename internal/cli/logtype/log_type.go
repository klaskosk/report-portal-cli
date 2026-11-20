package logtype

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// NewCommand builds the "log-type" parent command that groups all log type subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Annotations: map[string]string{core.GroupAnnotationKey: "true"},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		Use:   "log-type",
		Short: "Manage log types",
	}
	core.RegisterProjectFlag(cmd)
	cmd.AddCommand(newLogTypeListCmd())
	cmd.AddCommand(newLogTypeCreateCmd())
	cmd.AddCommand(newLogTypeUpdateCmd())
	cmd.AddCommand(newLogTypeDeleteCmd())

	return cmd
}

// newLogTypeListCmd builds the "log-type list" subcommand.
func newLogTypeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all log types in a project",
		Long: `List all log types in a project.

Returns all log types for the project. No pagination is available.

Examples:
  rp log-type list -p my_project               List all log types
  rp log-type list -o json -p my_project       List log types as JSON`,
		Args: cobra.NoArgs,
		RunE: runLogTypeList,
	}
}

// newLogTypeCreateCmd builds the "log-type create" subcommand.
func newLogTypeCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a log type",
		Long: `Create a log type using flags or a JSON file.

Style configuration is only available via --from-file. Users who need custom
styles must provide a complete LogTypeRequest JSON file.

Examples:
  rp log-type create --name "TRACE" --level 5000 -p my_project
  rp log-type create --name "TRACE" --level 5000 --filterable -p my_project
  rp log-type create --from-file log_type.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: runLogTypeCreate,
	}

	cmd.Flags().String("name", "", "log type name (required without --from-file)")
	cmd.Flags().Int32("level", 0, "numeric log level, must be > 0 (required without --from-file)")
	cmd.Flags().Bool("filterable", false, "whether the log type is filterable")
	cmd.Flags().String(
		"from-file",
		"",
		"path to JSON file containing LogTypeRequest (use - for stdin)",
	)

	return cmd
}

// newLogTypeUpdateCmd builds the "log-type update" subcommand.
func newLogTypeUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a log type from a JSON file",
		Long: `Update a log type using a JSON definition from a file.

Name and level are required. is_filterable is optional and may be omitted.

Examples:
  rp log-type update 5 --from-file updated.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: runLogTypeUpdate,
	}

	cmd.Flags().
		String("from-file", "", "path to JSON file containing LogTypeRequest (use - for stdin)")

	return cmd
}

// newLogTypeDeleteCmd builds the "log-type delete" subcommand.
func newLogTypeDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a log type by numeric ID",
		Long: `Delete a log type by numeric ID.

Examples:
  rp log-type delete 5 -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: runLogTypeDelete,
	}
}

func runLogTypeList(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("listing log types", "project", project)

	result, err := deps.Client.Project(project).LogTypes().List(cmd.Context())
	if err != nil {
		return fmt.Errorf("listing log types in project %s: %w", project, err)
	}

	headers := logTypeHeaders()
	rows := make([][]string, 0, len(result))

	for i := range result {
		_, row := logTypeRow(&result[i])
		rows = append(rows, row)
	}

	return deps.Printer.Print(result, headers, rows)
}

func runLogTypeCreate(cmd *cobra.Command, _ []string) error {
	request, err := resolveLogTypeCreateRequest(cmd)
	if err != nil {
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

	deps.Logger.Debug("creating log type", "project", project, "name", request.Name)

	result, err := deps.Client.Project(project).LogTypes().Create(cmd.Context(), request)
	if err != nil {
		return fmt.Errorf("creating log type in project %s: %w", project, err)
	}

	headers, row := logTypeRow(result)

	return deps.Printer.Print(result, headers, [][]string{row})
}

func runLogTypeUpdate(cmd *cobra.Command, args []string) error {
	logTypeID, err := parseLogTypeID(args[0])
	if err != nil {
		return err
	}

	path, err := core.RequireFromFile(
		cmd,
		"--from-file is required for log type update (the API expects the complete log type definition)",
	)
	if err != nil {
		return err
	}

	request, err := loadAndValidateLogTypeRequest(path)
	if err != nil {
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

	deps.Logger.Debug("updating log type", "project", project, "id", logTypeID)

	result, err := deps.Client.Project(project).LogTypes().Update(cmd.Context(), logTypeID, request)
	if err != nil {
		return fmt.Errorf("updating log type %d in project %s: %w", logTypeID, project, err)
	}

	return deps.Printer.Print(
		result,
		core.MessageHeaders(),
		core.MessageRows(core.FormatOptional(result.Message)),
	)
}

func runLogTypeDelete(cmd *cobra.Command, args []string) error {
	logTypeID, err := parseLogTypeID(args[0])
	if err != nil {
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

	deps.Logger.Debug("deleting log type", "project", project, "id", logTypeID)

	err = deps.Client.Project(project).LogTypes().Delete(cmd.Context(), logTypeID)
	if err != nil {
		return fmt.Errorf("deleting log type %d in project %s: %w", logTypeID, project, err)
	}

	const successMsg = "Log type deleted successfully"

	result, headers, rows := core.DeleteSuccessResult(successMsg)

	return deps.Printer.Print(result, headers, rows)
}

// resolveLogTypeCreateRequest determines the LogTypeRequest from either --from-file or flags.
func resolveLogTypeCreateRequest(cmd *cobra.Command) (types.LogTypeRequest, error) {
	request, err := core.RequestInput(cmd, "log type", buildLogTypeCreateRequestFromFlags)
	if err != nil {
		return types.LogTypeRequest{}, err
	}

	return validateLogTypeInput(request)
}

// buildLogTypeCreateRequestFromFlags builds the request from individual flags.
func buildLogTypeCreateRequestFromFlags(cmd *cobra.Command) (types.LogTypeRequest, error) {
	if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("level") {
		return types.LogTypeRequest{}, errors.New(
			"no input provided: use --name and --level flags, or --from-file to read from a JSON file",
		)
	}

	name, _ := cmd.Flags().GetString("name")
	level, _ := cmd.Flags().GetInt32("level")
	filterable, _ := cmd.Flags().GetBool("filterable")

	if name == "" {
		return types.LogTypeRequest{}, errors.New("--name is required when not using --from-file")
	}

	return types.LogTypeRequest{
		Name:         name,
		Level:        level,
		IsFilterable: &filterable,
	}, nil
}

// loadAndValidateLogTypeRequest reads a LogTypeRequest from a file and validates it.
func loadAndValidateLogTypeRequest(path string) (types.LogTypeRequest, error) {
	request, err := core.ReadRequestFromFile[types.LogTypeRequest](path, "log type")
	if err != nil {
		return types.LogTypeRequest{}, err
	}

	return validateLogTypeInput(request)
}

// validateLogTypeInput returns the request if its required fields are valid.
func validateLogTypeInput(request types.LogTypeRequest) (types.LogTypeRequest, error) {
	if request.Name == "" {
		return types.LogTypeRequest{}, errors.New("log type name must not be empty")
	}

	if request.Level <= 0 {
		return types.LogTypeRequest{}, errors.New("log type level must be greater than 0")
	}

	return request, nil
}

// logTypeHeaders returns the table column headers for log type rows.
func logTypeHeaders() []string {
	return []string{"ID", "NAME", "LEVEL", "FILTERABLE", "SYSTEM"}
}

// logTypeRow formats a LogTypeResponse as a table row, returning both headers and column values.
func logTypeRow(logType *types.LogTypeResponse) ([]string, []string) {
	filterable := ""
	if logType.IsFilterable != nil {
		filterable = strconv.FormatBool(*logType.IsFilterable)
	}

	return logTypeHeaders(), []string{
		core.FormatOptional(logType.Id),
		logType.Name,
		strconv.FormatInt(int64(logType.Level), 10),
		filterable,
		core.FormatOptional(logType.IsSystem),
	}
}

// parseLogTypeID parses a log type ID from CLI input, which must be numeric.
func parseLogTypeID(raw string) (int64, error) {
	return core.ParseIDArg(raw, "log-type")
}
