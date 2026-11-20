package setting

import (
	"context"
	"errors"
	"fmt"

	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// newSettingPatternCmd builds the "setting pattern" grouping command.
func newSettingPatternCmd() *cobra.Command {
	cmd := core.MarkGroupCommand(&cobra.Command{
		Use:   "pattern",
		Short: "Manage project pattern templates",
	})
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
		RunE: runSettingPatternList,
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
		RunE: runSettingPatternCreate,
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
		RunE: runSettingPatternUpdate,
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
		RunE: runSettingPatternDelete,
	}
}

func runSettingPatternList(cmd *cobra.Command, _ []string) error {
	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	deps.Logger.Debug("listing pattern templates", "project", project)

	projectResource, err := deps.Client.Projects().Get(cmd.Context(), project)
	if err != nil {
		return fmt.Errorf("listing pattern templates in project %s: %w", project, err)
	}

	patterns := projectResource.Configuration.Patterns
	if patterns == nil {
		empty := []types.PatternTemplateResource{}

		return deps.Printer.Print(empty, patternHeaders(), nil)
	}

	headers := patternHeaders()
	rows := make([][]string, 0, len(*patterns))

	for i := range *patterns {
		rows = append(rows, patternRow(&(*patterns)[i]))
	}

	return deps.Printer.Print(*patterns, headers, rows)
}

func runSettingPatternCreate(cmd *cobra.Command, _ []string) error {
	request, err := resolvePatternCreateRequest(cmd)
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

	deps.Logger.Debug("creating pattern template", "project", project, "name", request.Name)

	result, err := deps.Client.Project(project).
		Settings().
		CreatePatternTemplate(cmd.Context(), request)
	if err != nil {
		return fmt.Errorf("creating pattern template in project %s: %w", project, err)
	}

	headers := []string{"ID"}
	rows := [][]string{{core.FormatOptional(result.Id)}}

	return deps.Printer.Print(result, headers, rows)
}

func runSettingPatternUpdate(cmd *cobra.Command, args []string) error {
	patternID, err := parsePatternID(args[0])
	if err != nil {
		return err
	}

	if !cmd.Flags().Changed("from-file") &&
		!cmd.Flags().Changed("name") &&
		!cmd.Flags().Changed("enabled") {
		return errors.New(
			"no input provided: use --name or --enabled, or --from-file to read from a JSON file",
		)
	}

	deps, err := core.ResolveCmd(cmd)
	if err != nil {
		return err
	}

	project, err := deps.ResolveProject(cmd)
	if err != nil {
		return err
	}

	request, err := resolvePatternUpdateRequest(cmd, deps.Client, project, patternID)
	if err != nil {
		return err
	}

	deps.Logger.Debug("updating pattern template", "project", project, "id", patternID)

	result, err := deps.Client.Project(project).
		Settings().
		UpdatePatternTemplate(cmd.Context(), patternID, request)
	if err != nil {
		return fmt.Errorf("updating pattern template %d in project %s: %w", patternID, project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

func runSettingPatternDelete(cmd *cobra.Command, args []string) error {
	patternID, err := parsePatternID(args[0])
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

	deps.Logger.Debug("deleting pattern template", "project", project, "id", patternID)

	result, err := deps.Client.Project(project).
		Settings().
		DeletePatternTemplate(cmd.Context(), patternID)
	if err != nil {
		return fmt.Errorf("deleting pattern template %d in project %s: %w", patternID, project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

// resolvePatternCreateRequest builds and validates the request from flags or a file.
func resolvePatternCreateRequest(cmd *cobra.Command) (types.CreatePatternTemplateRQ, error) {
	request, err := core.RequestInput(cmd, "pattern template", buildPatternCreateRequestFromFlags)
	if err != nil {
		return types.CreatePatternTemplateRQ{}, err
	}

	if request.Name == "" {
		return types.CreatePatternTemplateRQ{}, errors.New(
			"pattern template name must not be empty in --from-file payload",
		)
	}

	return request, nil
}

// buildPatternCreateRequestFromFlags builds the request from individual flags.
func buildPatternCreateRequestFromFlags(cmd *cobra.Command) (types.CreatePatternTemplateRQ, error) {
	name, _ := cmd.Flags().GetString("name")
	patternType, _ := cmd.Flags().GetString("type")
	value, _ := cmd.Flags().GetString("value")
	enabled, _ := cmd.Flags().GetBool("enabled")

	if name == "" || !cmd.Flags().Changed("name") {
		return types.CreatePatternTemplateRQ{}, errors.New(
			"--name is required when not using --from-file",
		)
	}

	if patternType == "" || !cmd.Flags().Changed("type") {
		return types.CreatePatternTemplateRQ{}, errors.New(
			"--type is required when not using --from-file",
		)
	}

	if value == "" || !cmd.Flags().Changed("value") {
		return types.CreatePatternTemplateRQ{}, errors.New(
			"--value is required when not using --from-file",
		)
	}

	return types.CreatePatternTemplateRQ{
		Name:    name,
		Type:    patternType,
		Value:   value,
		Enabled: enabled,
	}, nil
}

// resolvePatternUpdateRequest builds the update from flags or a file.
func resolvePatternUpdateRequest(
	cmd *cobra.Command,
	apiClient *client.Client,
	project string,
	patternID int64,
) (types.UpdatePatternTemplateRQ, error) {
	request, err := core.RequestInput(
		cmd,
		"pattern template",
		func(cmd *cobra.Command) (types.UpdatePatternTemplateRQ, error) {
			return buildPatternUpdateRequestFromFlags(cmd, apiClient, project, patternID)
		},
	)
	if err != nil {
		return types.UpdatePatternTemplateRQ{}, err
	}

	if request.Name == "" {
		return types.UpdatePatternTemplateRQ{}, errors.New(
			"pattern template name must not be empty in --from-file payload",
		)
	}

	return request, nil
}

// buildPatternUpdateRequestFromFlags applies flag values to the current pattern.
func buildPatternUpdateRequestFromFlags(
	cmd *cobra.Command,
	apiClient *client.Client,
	project string,
	patternID int64,
) (types.UpdatePatternTemplateRQ, error) {
	current, err := findPatternTemplate(apiClient, cmd.Context(), project, patternID)
	if err != nil {
		return types.UpdatePatternTemplateRQ{}, err
	}

	request := types.UpdatePatternTemplateRQ{
		Name:    core.FormatOptional(current.Name),
		Enabled: current.Enabled != nil && *current.Enabled,
	}

	if name := core.OptionalStringFlag(cmd, "name"); name != nil {
		request.Name = *name
	}

	if enabled := core.OptionalBoolFlag(cmd, "enabled"); enabled != nil {
		request.Enabled = *enabled
	}

	if request.Name == "" {
		return types.UpdatePatternTemplateRQ{}, errors.New(
			"pattern template name must not be empty",
		)
	}

	return request, nil
}

// findPatternTemplate looks up the pattern in the project configuration.
func findPatternTemplate(
	apiClient *client.Client,
	ctx context.Context,
	project string,
	patternID int64,
) (*types.PatternTemplateResource, error) {
	projectResource, err := apiClient.Projects().Get(ctx, project)
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
		core.FormatOptional(pattern.Id),
		core.FormatOptional(pattern.Name),
		core.FormatOptional(pattern.Type),
		core.FormatOptional(pattern.Value),
		formatOptionalBool(pattern.Enabled),
	}
}

func parsePatternID(raw string) (int64, error) {
	return core.ParseIDArg(raw, "pattern")
}
