package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newSettingPatternCmd builds the "setting pattern" grouping command.
func newSettingPatternCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pattern",
		Short: "Manage project pattern templates",
	}
	cmd.AddCommand(newSettingPatternListCmd())
	cmd.AddCommand(newSettingPatternCreateCmd())
	cmd.AddCommand(newSettingPatternUpdateCmd())
	cmd.AddCommand(newSettingPatternDeleteCmd())

	return cmd
}

func newSettingPatternListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List project pattern templates",
		Long: `List pattern templates configured for a project.

Pattern templates are read from the project configuration because the settings
API does not expose a dedicated list endpoint.

Examples:
  rp setting pattern list -p my_project
  rp setting pattern list -p my_project -o json`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runSettingPatternList),
	}
}

func newSettingPatternCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pattern template",
		Long: `Create a pattern template using flags or a JSON file.

Examples:
  rp setting pattern create --name timeout --type STRING --value "timeout" -p my_project
  rp setting pattern create --from-file pattern.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runSettingPatternCreate),
	}

	cmd.Flags().String("name", "", "pattern template name (required without --from-file)")
	cmd.Flags().String("type", "", "pattern type, e.g. STRING (required without --from-file)")
	cmd.Flags().String("value", "", "pattern value (required without --from-file)")
	cmd.Flags().Bool("enabled", true, "whether the pattern template is enabled")
	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing CreatePatternTemplateRQ (use - for stdin)",
	)

	return cmd
}

func newSettingPatternUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a pattern template",
		Long: `Update a pattern template using flags or a JSON file.

When using flags, at least one of --name or --enabled must be provided. Omitted
fields are loaded from the current project configuration.

Examples:
  rp setting pattern update 42 --name updated-name -p my_project
  rp setting pattern update 42 --from-file pattern.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runSettingPatternUpdate),
	}

	cmd.Flags().String("name", "", "updated pattern template name")
	cmd.Flags().Bool("enabled", true, "whether the pattern template is enabled")
	cmd.Flags().String(
		"from-file", "",
		"path to JSON file containing UpdatePatternTemplateRQ (use - for stdin)",
	)

	return cmd
}

func newSettingPatternDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a pattern template by numeric ID",
		Long: `Delete a pattern template by numeric ID.

Examples:
  rp setting pattern delete 42 -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runSettingPatternDelete),
	}
}

func runSettingPatternList(cmd *cobra.Command, _ []string) error {
	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.resolveProject(cmd)
	if err != nil {
		return err
	}

	deps.logger.Debug("listing pattern templates", "project", project)

	projectResource, err := deps.client.Projects().Get(cmd.Context(), project)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("listing pattern templates in project %s: %w", project, err)
	}

	patterns := projectResource.Configuration.Patterns
	if patterns == nil {
		empty := []types.PatternTemplateResource{}

		return deps.printer.Print(empty, patternHeaders(), nil)
	}

	headers := patternHeaders()
	rows := make([][]string, 0, len(*patterns))

	for i := range *patterns {
		rows = append(rows, patternRow(&(*patterns)[i]))
	}

	return deps.printer.Print(*patterns, headers, rows)
}

func runSettingPatternCreate(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	request, err := resolvePatternCreateRequest(cmd, printer)
	if err != nil {
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

	deps.logger.Debug("creating pattern template", "project", project, "name", request.Name)

	result, err := deps.client.Project(project).
		Settings().
		CreatePatternTemplate(cmd.Context(), request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating pattern template in project %s: %w", project, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{formatOptional(result.Id)}}

	return deps.printer.Print(result, headers, rows)
}

func runSettingPatternUpdate(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	patternID, err := parsePatternID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	if !cmd.Flags().Changed("from-file") &&
		!cmd.Flags().Changed("name") &&
		!cmd.Flags().Changed("enabled") {
		err = errors.New(
			"no input provided: use --name or --enabled, or --from-file to read from a JSON file",
		)
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

	request, err := resolvePatternUpdateRequest(cmd, printer, deps, project, patternID)
	if err != nil {
		return err
	}

	deps.logger.Debug("updating pattern template", "project", project, "id", patternID)

	result, err := deps.client.Project(project).
		Settings().
		UpdatePatternTemplate(cmd.Context(), patternID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating pattern template %d in project %s: %w", patternID, project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runSettingPatternDelete(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	patternID, err := parsePatternID(args[0])
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

	deps.logger.Debug("deleting pattern template", "project", project, "id", patternID)

	result, err := deps.client.Project(project).
		Settings().
		DeletePatternTemplate(cmd.Context(), patternID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("deleting pattern template %d in project %s: %w", patternID, project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func resolvePatternCreateRequest(
	cmd *cobra.Command,
	printer *Printer,
) (types.CreatePatternTemplateRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		request, err := readRequestFromFile[types.CreatePatternTemplateRQ](path)
		if err != nil {
			printer.PrintError(err)

			return types.CreatePatternTemplateRQ{}, wrapFileError(
				err,
				"pattern template",
				"pattern template JSON",
			)
		}

		if request.Name == "" {
			err = errors.New("pattern template name must not be empty in --from-file payload")
			printer.PrintError(err)

			return types.CreatePatternTemplateRQ{}, err
		}

		return request, nil
	}

	name, _ := cmd.Flags().GetString("name")
	patternType, _ := cmd.Flags().GetString("type")
	value, _ := cmd.Flags().GetString("value")
	enabled, _ := cmd.Flags().GetBool("enabled")

	if name == "" || !cmd.Flags().Changed("name") {
		err := errors.New("--name is required when not using --from-file")
		printer.PrintError(err)

		return types.CreatePatternTemplateRQ{}, err
	}

	if patternType == "" || !cmd.Flags().Changed("type") {
		err := errors.New("--type is required when not using --from-file")
		printer.PrintError(err)

		return types.CreatePatternTemplateRQ{}, err
	}

	if value == "" || !cmd.Flags().Changed("value") {
		err := errors.New("--value is required when not using --from-file")
		printer.PrintError(err)

		return types.CreatePatternTemplateRQ{}, err
	}

	return types.CreatePatternTemplateRQ{
		Name:    name,
		Type:    patternType,
		Value:   value,
		Enabled: enabled,
	}, nil
}

func resolvePatternUpdateRequest(
	cmd *cobra.Command,
	printer *Printer,
	deps *cmdDeps,
	project string,
	patternID int64,
) (types.UpdatePatternTemplateRQ, error) {
	if cmd.Flags().Changed("from-file") {
		return readPatternUpdateRequestFromFile(cmd, printer)
	}

	return buildPatternUpdateRequestFromFlags(cmd, printer, deps, project, patternID)
}

func readPatternUpdateRequestFromFile(
	cmd *cobra.Command,
	printer *Printer,
) (types.UpdatePatternTemplateRQ, error) {
	path, _ := cmd.Flags().GetString("from-file")

	request, err := readRequestFromFile[types.UpdatePatternTemplateRQ](path)
	if err != nil {
		printer.PrintError(err)

		return types.UpdatePatternTemplateRQ{}, wrapFileError(
			err,
			"pattern template",
			"pattern template JSON",
		)
	}

	if request.Name == "" {
		err = errors.New("pattern template name must not be empty in --from-file payload")
		printer.PrintError(err)

		return types.UpdatePatternTemplateRQ{}, err
	}

	return request, nil
}

func buildPatternUpdateRequestFromFlags(
	cmd *cobra.Command,
	printer *Printer,
	deps *cmdDeps,
	project string,
	patternID int64,
) (types.UpdatePatternTemplateRQ, error) {
	current, err := findPatternTemplate(deps, cmd.Context(), project, patternID)
	if err != nil {
		printer.PrintError(err)

		return types.UpdatePatternTemplateRQ{}, err
	}

	request := types.UpdatePatternTemplateRQ{
		Name:    formatOptional(current.Name),
		Enabled: current.Enabled != nil && *current.Enabled,
	}

	if name := optionalStringFlag(cmd, "name"); name != nil {
		request.Name = *name
	}

	if enabled := optionalBoolFlag(cmd, "enabled"); enabled != nil {
		request.Enabled = *enabled
	}

	if request.Name == "" {
		err = errors.New("pattern template name must not be empty")
		printer.PrintError(err)

		return types.UpdatePatternTemplateRQ{}, err
	}

	return request, nil
}

func findPatternTemplate(
	deps *cmdDeps,
	ctx context.Context,
	project string,
	patternID int64,
) (*types.PatternTemplateResource, error) {
	projectResource, err := deps.client.Projects().Get(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("loading project configuration: %w", err)
	}

	if projectResource.Configuration.Patterns == nil {
		return nil, fmt.Errorf("pattern template %d not found in project %s", patternID, project)
	}

	for i := range *projectResource.Configuration.Patterns {
		pattern := &(*projectResource.Configuration.Patterns)[i]
		if pattern.Id != nil && *pattern.Id == patternID {
			return pattern, nil
		}
	}

	return nil, fmt.Errorf("pattern template %d not found in project %s", patternID, project)
}

func patternHeaders() []string {
	return []string{"ID", "NAME", "TYPE", "VALUE", "ENABLED"}
}

func patternRow(pattern *types.PatternTemplateResource) []string {
	return []string{
		formatOptional(pattern.Id),
		formatOptional(pattern.Name),
		formatOptional(pattern.Type),
		formatOptional(pattern.Value),
		formatOptionalBool(pattern.Enabled),
	}
}

func parsePatternID(raw string) (int64, error) {
	return parseIDArg(raw, "pattern")
}
