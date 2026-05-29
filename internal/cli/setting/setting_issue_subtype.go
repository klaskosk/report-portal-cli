package setting

import (
	"errors"
	"fmt"
	"sort"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newSettingIssueSubtypeCmd builds the "setting issue-subtype" grouping command.
func newSettingIssueSubtypeCmd() *cobra.Command {
	cmd := core.MarkGroupCommand(&cobra.Command{
		Use:   "issue-subtype",
		Short: "Manage project issue sub-types",
	})
	cmd.AddCommand(newSettingIssueSubtypeListCmd())
	cmd.AddCommand(newSettingIssueSubtypeCreateCmd())
	cmd.AddCommand(newSettingIssueSubtypeUpdateCmd())
	cmd.AddCommand(newSettingIssueSubtypeDeleteCmd())

	return cmd
}

func newSettingIssueSubtypeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List project issue sub-types",
		Long: `List issue sub-types configured for a project.

Examples:
  rp setting issue-subtype list -p my_project
  rp setting issue-subtype list -p my_project -o json`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runSettingIssueSubtypeList),
	}
}

func newSettingIssueSubtypeCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom issue sub-type",
		Long: `Create a custom issue sub-type using flags or a JSON file.

Examples:
  rp setting issue-subtype create --long-name "Custom Bug" --short-name CB \
    --color "#ff0000" --type-ref PRODUCT_BUG -p my_project
  rp setting issue-subtype create --from-file subtype.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runSettingIssueSubtypeCreate),
	}

	cmd.Flags().String("long-name", "", "long display name (required without --from-file)")
	cmd.Flags().String("short-name", "", "short display name (required without --from-file)")
	cmd.Flags().String("color", "", "hex color, e.g. #ff0000 (required without --from-file)")
	cmd.Flags().String("type-ref", "", "parent issue type reference (required without --from-file)")
	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing CreateIssueSubTypeRQ (use - for stdin)",
	)

	return cmd
}

func newSettingIssueSubtypeUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an issue sub-type from a JSON file",
		Long: `Update an issue sub-type from a JSON file describing UpdateOneIssueSubTypeRQ.

Examples:
  rp setting issue-subtype update --from-file subtype.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: core.WithPrintedErrors(runSettingIssueSubtypeUpdate),
	}

	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing UpdateOneIssueSubTypeRQ (use - for stdin)",
	)
	_ = cmd.MarkFlagRequired("from-file")

	return cmd
}

func newSettingIssueSubtypeDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an issue sub-type by numeric ID",
		Long: `Delete a custom issue sub-type by numeric ID.

Examples:
  rp setting issue-subtype delete 42 -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: core.WithPrintedErrors(runSettingIssueSubtypeDelete),
	}
}

func runSettingIssueSubtypeList(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("listing issue sub-types", "project", project)

	settings, err := deps.Client.Project(project).Settings().Get(cmd.Context())
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("listing issue sub-types in project %s: %w", project, err)
	}

	subTypes := flattenIssueSubTypes(settings.SubTypes)
	headers := issueSubTypeHeaders()
	rows := make([][]string, 0, len(subTypes))

	for i := range subTypes {
		rows = append(rows, issueSubTypeRow(&subTypes[i]))
	}

	return deps.Printer.Print(subTypes, headers, rows)
}

func runSettingIssueSubtypeCreate(cmd *cobra.Command, _ []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	request, err := resolveIssueSubTypeCreateRequest(cmd, printer)
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

	deps.Logger.Debug("creating issue sub-type", "project", project, "longName", request.LongName)

	result, err := deps.Client.Project(project).
		Settings().
		CreateIssueSubType(cmd.Context(), request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("creating issue sub-type in project %s: %w", project, err)
	}

	headers := []string{"ID", "LOCATOR"}
	rows := [][]string{{core.FormatOptional(result.Id), core.FormatOptional(result.Locator)}}

	return deps.Printer.Print(result, headers, rows)
}

func runSettingIssueSubtypeUpdate(cmd *cobra.Command, _ []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	path, err := core.RequireFromFile(
		cmd,
		printer,
		"--from-file is required for issue sub-type update",
	)
	if err != nil {
		return err
	}

	request, err := readIssueSubTypeUpdateRequestFromFile(path)
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

	deps.Logger.Debug("updating issue sub-type", "project", project, "locator", request.Locator)

	result, err := deps.Client.Project(project).
		Settings().
		UpdateIssueSubType(cmd.Context(), request)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("updating issue sub-type in project %s: %w", project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func runSettingIssueSubtypeDelete(cmd *cobra.Command, args []string) error {
	printer := core.NewPrinterFromCmd(cmd)

	subTypeID, err := parseIssueSubTypeID(args[0])
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

	deps.Logger.Debug("deleting issue sub-type", "project", project, "id", subTypeID)

	result, err := deps.Client.Project(project).
		Settings().
		DeleteIssueSubType(cmd.Context(), subTypeID)
	if err != nil {
		deps.Printer.PrintError(err)

		return fmt.Errorf("deleting issue sub-type %d in project %s: %w", subTypeID, project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func resolveIssueSubTypeCreateRequest(
	cmd *cobra.Command,
	printer *core.Printer,
) (types.CreateIssueSubTypeRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		request, err := core.ReadRequestFromFile[types.CreateIssueSubTypeRQ](path)
		if err != nil {
			printer.PrintError(err)

			return types.CreateIssueSubTypeRQ{}, core.WrapFileError(
				err,
				"issue sub-type",
				"issue sub-type JSON",
			)
		}

		return request, validateIssueSubTypeCreateRequest(request, printer)
	}

	longName, _ := cmd.Flags().GetString("long-name")
	shortName, _ := cmd.Flags().GetString("short-name")
	color, _ := cmd.Flags().GetString("color")
	typeRef, _ := cmd.Flags().GetString("type-ref")

	if !cmd.Flags().Changed("long-name") &&
		!cmd.Flags().Changed("short-name") &&
		!cmd.Flags().Changed("color") &&
		!cmd.Flags().Changed("type-ref") {
		err := errors.New(
			"no input provided: use --long-name, --short-name, --color, and --type-ref, or --from-file",
		)
		printer.PrintError(err)

		return types.CreateIssueSubTypeRQ{}, err
	}

	request := types.CreateIssueSubTypeRQ{
		LongName:  longName,
		ShortName: shortName,
		Color:     color,
		TypeRef:   typeRef,
	}

	return request, validateIssueSubTypeCreateRequest(request, printer)
}

func validateIssueSubTypeCreateRequest(
	request types.CreateIssueSubTypeRQ,
	printer *core.Printer,
) error {
	switch {
	case request.LongName == "":
		err := errors.New("--long-name is required")
		printer.PrintError(err)

		return err
	case request.ShortName == "":
		err := errors.New("--short-name is required")
		printer.PrintError(err)

		return err
	case request.Color == "":
		err := errors.New("--color is required")
		printer.PrintError(err)

		return err
	case request.TypeRef == "":
		err := errors.New("--type-ref is required")
		printer.PrintError(err)

		return err
	default:
		return nil
	}
}

func readIssueSubTypeUpdateRequestFromFile(path string) (types.UpdateOneIssueSubTypeRQ, error) {
	result, err := core.ReadRequestFromFile[types.UpdateOneIssueSubTypeRQ](path)
	if err != nil {
		return result, core.WrapFileError(err, "issue sub-type", "issue sub-type JSON")
	}

	return result, nil
}

func flattenIssueSubTypes(
	subTypes map[string][]types.IssueSubTypeResource,
) []types.IssueSubTypeResource {
	groups := make([]string, 0, len(subTypes))
	for group := range subTypes {
		groups = append(groups, group)
	}

	sort.Strings(groups)

	flat := make([]types.IssueSubTypeResource, 0)
	for _, group := range groups {
		flat = append(flat, subTypes[group]...)
	}

	return flat
}

func issueSubTypeHeaders() []string {
	return []string{"ID", "LOCATOR", "LONG NAME", "SHORT NAME", "COLOR"}
}

func issueSubTypeRow(subType *types.IssueSubTypeResource) []string {
	return []string{
		core.FormatOptional(subType.Id),
		core.FormatOptional(subType.Locator),
		core.FormatOptional(subType.LongName),
		core.FormatOptional(subType.ShortName),
		core.FormatOptional(subType.Color),
	}
}

func parseIssueSubTypeID(raw string) (int64, error) {
	return core.ParseIDArg(raw, "issue-subtype")
}
