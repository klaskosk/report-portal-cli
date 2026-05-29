package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// integrationOps abstracts over global and project-scoped integration operations.
type integrationOps interface {
	list(ctx context.Context, plugin string) ([]types.IntegrationResource, error)
	get(ctx context.Context, integrationID int64) (*types.IntegrationResource, error)
	create(
		ctx context.Context,
		plugin string,
		request types.IntegrationRQ,
	) (*types.EntryCreatedRS, error)
	update(
		ctx context.Context,
		integrationID int64,
		request types.IntegrationRQ,
	) (*types.OperationCompletionRS, error)
	delete(ctx context.Context, integrationID int64) (*types.OperationCompletionRS, error)
}

type globalIntegrationOps struct {
	deps *cmdDeps
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (g *globalIntegrationOps) list(
	ctx context.Context,
	plugin string,
) ([]types.IntegrationResource, error) {
	if plugin != "" {
		return g.deps.client.Integrations().ListGlobalByPlugin(ctx, plugin)
	}

	return g.deps.client.Integrations().ListGlobal(ctx)
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (g *globalIntegrationOps) get(
	ctx context.Context,
	integrationID int64,
) (*types.IntegrationResource, error) {
	return g.deps.client.Integrations().GetGlobal(ctx, integrationID)
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (g *globalIntegrationOps) create(
	ctx context.Context,
	plugin string,
	request types.IntegrationRQ,
) (*types.EntryCreatedRS, error) {
	return g.deps.client.Integrations().CreateGlobal(ctx, plugin, request)
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (g *globalIntegrationOps) update(
	ctx context.Context,
	integrationID int64,
	request types.IntegrationRQ,
) (*types.OperationCompletionRS, error) {
	return g.deps.client.Integrations().UpdateGlobal(ctx, integrationID, request)
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (g *globalIntegrationOps) delete(
	ctx context.Context,
	integrationID int64,
) (*types.OperationCompletionRS, error) {
	return g.deps.client.Integrations().DeleteGlobal(ctx, integrationID)
}

type projectIntegrationOps struct {
	deps    *cmdDeps
	project string
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (p *projectIntegrationOps) list(
	ctx context.Context,
	plugin string,
) ([]types.IntegrationResource, error) {
	if plugin != "" {
		return p.deps.client.Project(p.project).Integrations().ListByPlugin(ctx, plugin)
	}

	return p.deps.client.Project(p.project).Integrations().List(ctx)
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (p *projectIntegrationOps) get(
	ctx context.Context,
	integrationID int64,
) (*types.IntegrationResource, error) {
	return p.deps.client.Project(p.project).Integrations().Get(ctx, integrationID)
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (p *projectIntegrationOps) create(
	ctx context.Context,
	plugin string,
	request types.IntegrationRQ,
) (*types.EntryCreatedRS, error) {
	return p.deps.client.Project(p.project).Integrations().Create(ctx, plugin, request)
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (p *projectIntegrationOps) update(
	ctx context.Context,
	integrationID int64,
	request types.IntegrationRQ,
) (*types.OperationCompletionRS, error) {
	return p.deps.client.Project(p.project).Integrations().Update(ctx, integrationID, request)
}

//nolint:wrapcheck // run function is responsible for wrapping errors with additional context
func (p *projectIntegrationOps) delete(
	ctx context.Context,
	integrationID int64,
) (*types.OperationCompletionRS, error) {
	return p.deps.client.Project(p.project).Integrations().Delete(ctx, integrationID)
}

// validateIntegrationProject checks the --project flag when it is explicitly set.
// It must be called before resolveCmd so that an empty --project value is rejected
// before a missing-connection error can surface.
func validateIntegrationProject(cmd *cobra.Command, printer *Printer) error {
	if !cmd.Flags().Changed("project") {
		return nil
	}

	project, _ := cmd.Flags().GetString("project")
	if project == "" {
		err := errors.New("project name cannot be empty (omit -p to use global scope)")
		printer.PrintError(err)

		return err
	}

	return nil
}

// resolveIntegrationOps returns the appropriate integration scope based on the --project flag.
// Omitting --project is intentional and means "global scope" — it does NOT fall back to env or
// profile defaults. Use cmd.Flags().Changed("project") to detect explicit use.
//
//nolint:ireturn // returns interface to abstract over global and project-scoped integration ops
func resolveIntegrationOps(cmd *cobra.Command, deps *cmdDeps) integrationOps {
	if !cmd.Flags().Changed("project") {
		return &globalIntegrationOps{deps: deps}
	}

	return &projectIntegrationOps{
		deps:    deps,
		project: mustGetStringFlag(cmd, "project"),
	}
}

// newIntegrationCmd builds the "integration" parent command that groups all integration subcommands.
func newIntegrationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration",
		Short: "Manage integrations",
		Long: `Manage Report Portal integrations.

Omit --project for global (instance-level) integrations — these typically require
administrator access. Include --project for project-scoped integrations.`,
	}
	cmd.PersistentFlags().StringP("project", "p", "", "project name (omit for global scope)")
	cmd.AddCommand(newIntegrationListCmd())
	cmd.AddCommand(newIntegrationGetCmd())
	cmd.AddCommand(newIntegrationCreateCmd())
	cmd.AddCommand(newIntegrationUpdateCmd())
	cmd.AddCommand(newIntegrationDeleteCmd())

	return cmd
}

// newIntegrationListCmd builds the "integration list" subcommand.
func newIntegrationListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List integrations",
		Long: `List integrations.

Omit --project for global integrations (admin-only). Include --project for
project-scoped integrations.

Examples:
  rp integration list                          List all global integrations
  rp integration list -o json                  List global integrations as JSON
  rp integration list --plugin jira            List global integrations by plugin
  rp integration list -p my_project            List project integrations`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runIntegrationList),
	}

	cmd.Flags().String("plugin", "", "filter integrations by plugin name")

	return cmd
}

// newIntegrationGetCmd builds the "integration get <id>" subcommand.
func newIntegrationGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get an integration by numeric ID",
		Long: `Get an integration by numeric ID.

Examples:
  rp integration get 1                         Get global integration
  rp integration get 10 -p my_project          Get project integration
  rp integration get 10 -o json -p my_project  Get integration as JSON`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runIntegrationGet),
	}
}

// newIntegrationCreateCmd builds the "integration create" subcommand.
func newIntegrationCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an integration from a JSON file",
		Long: `Create an integration from a JSON file.

Examples:
  rp integration create --plugin email --from-file email.json
  rp integration create --plugin jira --from-file jira.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: withPrintedErrors(runIntegrationCreate),
	}

	cmd.Flags().String("plugin", "", "plugin name (e.g. jira, email, rally)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing IntegrationRQ payload (use - for stdin)")

	return cmd
}

// newIntegrationUpdateCmd builds the "integration update <id>" subcommand.
func newIntegrationUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update an integration from a JSON file",
		Long: `Update an integration from a JSON file.

Examples:
  rp integration update 1 --from-file updated.json
  rp integration update 10 --from-file updated.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runIntegrationUpdate),
	}

	cmd.Flags().
		String("from-file", "", "path to JSON file containing IntegrationRQ payload (use - for stdin)")

	return cmd
}

// newIntegrationDeleteCmd builds the "integration delete <id>" subcommand.
func newIntegrationDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an integration by numeric ID",
		Long: `Delete an integration by numeric ID.

Examples:
  rp integration delete 1                      Delete global integration
  rp integration delete 10 -p my_project       Delete project integration`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runIntegrationDelete),
	}
}

func runIntegrationList(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	err := validateIntegrationProject(cmd, printer)
	if err != nil {
		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	ops := resolveIntegrationOps(cmd, deps)

	plugin := ""
	if cmd.Flags().Changed("plugin") {
		plugin, _ = cmd.Flags().GetString("plugin")
	}

	deps.logger.Debug("listing integrations", "plugin", plugin)

	integrations, err := ops.list(cmd.Context(), plugin)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("listing integrations: %w", err)
	}

	headers := integrationHeaders()
	rows := make([][]string, 0, len(integrations))

	for idx := range integrations {
		_, row := integrationRow(&integrations[idx])
		rows = append(rows, row)
	}

	return deps.printer.Print(integrations, headers, rows)
}

func runIntegrationGet(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	integrationID, err := parseIntegrationID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	err = validateIntegrationProject(cmd, printer)
	if err != nil {
		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	ops := resolveIntegrationOps(cmd, deps)

	deps.logger.Debug("getting integration", "id", integrationID)

	result, err := ops.get(cmd.Context(), integrationID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("getting integration %d: %w", integrationID, err)
	}

	headers, row := integrationRow(result)

	return deps.printer.Print(result, headers, [][]string{row})
}

func runIntegrationCreate(cmd *cobra.Command, _ []string) error {
	printer := resolvePrinter(cmd)

	if !cmd.Flags().Changed("plugin") {
		err := errors.New("--plugin is required for integration creation")
		printer.PrintError(err)

		return err
	}

	plugin, _ := cmd.Flags().GetString("plugin")
	if plugin == "" {
		err := errors.New("--plugin cannot be empty")
		printer.PrintError(err)

		return err
	}

	path, err := requireFromFile(
		cmd,
		printer,
		"--from-file is required for integration creation (integration parameters are specified in a JSON file)",
	)
	if err != nil {
		return err
	}

	request, err := readIntegrationRequestFromFile(path)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	err = validateIntegrationProject(cmd, printer)
	if err != nil {
		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	ops := resolveIntegrationOps(cmd, deps)

	deps.logger.Debug("creating integration", "plugin", plugin)

	result, err := ops.create(cmd.Context(), plugin, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("creating integration: %w", err)
	}

	headers := []string{"ID"}
	rows := [][]string{{formatOptional(result.Id)}}

	return deps.printer.Print(result, headers, rows)
}

func runIntegrationUpdate(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	integrationID, err := parseIntegrationID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	path, err := requireFromFile(
		cmd,
		printer,
		"--from-file is required for integration update",
	)
	if err != nil {
		return err
	}

	request, err := readIntegrationRequestFromFile(path)
	if err != nil {
		printer.PrintError(err)

		return err
	}

	err = validateIntegrationProject(cmd, printer)
	if err != nil {
		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	ops := resolveIntegrationOps(cmd, deps)

	deps.logger.Debug("updating integration", "id", integrationID)

	result, err := ops.update(cmd.Context(), integrationID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("updating integration %d: %w", integrationID, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func runIntegrationDelete(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	integrationID, err := parseIntegrationID(args[0])
	if err != nil {
		printer.PrintError(err)

		return err
	}

	err = validateIntegrationProject(cmd, printer)
	if err != nil {
		return err
	}

	deps, err := resolveCmd(cmd)
	if err != nil {
		return err
	}

	ops := resolveIntegrationOps(cmd, deps)

	deps.logger.Debug("deleting integration", "id", integrationID)

	result, err := ops.delete(cmd.Context(), integrationID)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("deleting integration %d: %w", integrationID, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

// readIntegrationRequestFromFile reads JSON from path (or stdin when path is "-") into an IntegrationRQ.
func readIntegrationRequestFromFile(path string) (types.IntegrationRQ, error) {
	result, err := readRequestFromFile[types.IntegrationRQ](path)
	if err != nil {
		return result, wrapFileError(err, "integration definition", "integration JSON")
	}

	return result, nil
}

// integrationHeaders returns the table column headers for integration rows.
func integrationHeaders() []string {
	return []string{"ID", "NAME", "PLUGIN", "ENABLED", "CREATOR"}
}

// integrationRow formats an IntegrationResource as a table row, returning both headers and column values.
func integrationRow(resource *types.IntegrationResource) ([]string, []string) {
	plugin := ""
	if resource.IntegrationType != nil {
		plugin = formatOptional(resource.IntegrationType.Name)
	}

	return integrationHeaders(), []string{
		formatOptional(resource.Id),
		formatOptional(resource.Name),
		plugin,
		formatOptional(resource.Enabled),
		formatOptional(resource.Creator),
	}
}

// parseIntegrationID parses an integration ID from CLI input, which must be numeric.
func parseIntegrationID(raw string) (int64, error) {
	return parseIDArg(raw, "integration")
}
