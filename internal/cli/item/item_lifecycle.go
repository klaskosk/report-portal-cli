package item

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// itemStartScope is the narrow interface used to dispatch test item start.
type itemStartScope interface {
	StartRoot(
		ctx context.Context,
		request types.StartTestItemRQ,
	) (*types.EntryCreatedAsyncRS, error)
	StartChild(
		ctx context.Context,
		parentItemUUID string,
		request types.StartTestItemRQ,
	) (*types.EntryCreatedAsyncRS, error)
}

// dispatchItemStart calls StartRoot or StartChild depending on whether parentUUID is set.
func dispatchItemStart(
	ctx context.Context,
	scope itemStartScope,
	parentUUID string,
	request types.StartTestItemRQ,
) (*types.EntryCreatedAsyncRS, error) {
	if parentUUID != "" {
		result, err := scope.StartChild(ctx, parentUUID, request)
		if err != nil {
			return nil, fmt.Errorf("start child test item: %w", err)
		}

		return result, nil
	}

	result, err := scope.StartRoot(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("start root test item: %w", err)
	}

	return result, nil
}

func newItemStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a test item",
		Long: `Start a root or child test item under a launch.

Examples:
  rp item start --name "Login Suite" --launch-uuid abc-uuid --type SUITE -p my_project
  rp item start --name "Login Test" --launch-uuid abc-uuid --type TEST --parent-uuid def-uuid -p my_project
  rp item start --from-file item_start.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: runItemStart,
	}

	cmd.Flags().String("name", "", "test item name")
	cmd.Flags().String("launch-uuid", "", "UUID of the parent launch")
	cmd.Flags().String("type", "", "item type (SUITE, STORY, TEST, STEP, etc.)")
	cmd.Flags().String("parent-uuid", "", "parent item UUID (creates a child item when set)")
	cmd.Flags().String("description", "", "item description")
	cmd.Flags().StringArray("attribute", []string{}, "attribute in key:value format (repeatable)")
	cmd.Flags().Bool("has-stats", true, "whether the item contributes to launch statistics")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing StartTestItemRQ (use - for stdin)")

	return cmd
}

func newItemFinishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finish <uuid>",
		Short: "Finish a test item by UUID",
		Long: `Finish a test item using the UUID returned by item start.

Examples:
  rp item finish item-uuid-789 --launch-uuid abc-uuid --status PASSED -p my_project
  rp item finish item-uuid-789 --from-file item_finish.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: runItemFinish,
	}

	cmd.Flags().String("status", "", "final item status (PASSED, FAILED, SKIPPED, etc.)")
	cmd.Flags().String("description", "", "final item description")
	cmd.Flags().String("launch-uuid", "", "UUID of the parent launch")
	cmd.Flags().StringArray("attribute", []string{}, "attribute in key:value format (repeatable)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing FinishTestItemRQ (use - for stdin)")

	return cmd
}

func runItemStart(cmd *cobra.Command, _ []string) error {
	request, err := buildItemStartRequest(cmd)
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

	parentUUID, _ := cmd.Flags().GetString("parent-uuid")
	deps.Logger.Debug(
		"starting test item",
		"project",
		project,
		"name",
		request.Name,
		"parentUuid",
		parentUUID,
	)

	scope := deps.Client.Project(project).Items()

	result, err := dispatchItemStart(cmd.Context(), scope, parentUUID, request)
	if err != nil {
		return fmt.Errorf("starting test item in project %s: %w", project, err)
	}

	return deps.Printer.Print(result, startTestItemHeaders(), startTestItemRows(result))
}

func runItemFinish(cmd *cobra.Command, args []string) error {
	request, err := buildItemFinishRequest(cmd)
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

	itemUUID := args[0]
	deps.Logger.Debug("finishing test item", "project", project, "uuid", itemUUID)

	result, err := deps.Client.Project(project).Items().Finish(cmd.Context(), itemUUID, request)
	if err != nil {
		return fmt.Errorf("finishing test item %s in project %s: %w", itemUUID, project, err)
	}

	return deps.Printer.Print(
		result,
		core.OperationResultHeaders(),
		core.OperationResultRows(result),
	)
}

// buildItemStartRequest builds the item start request from flags or a file.
func buildItemStartRequest(cmd *cobra.Command) (types.StartTestItemRQ, error) {
	return core.RequestInput(cmd, "item start", buildItemStartRequestFromFlags)
}

// buildItemStartRequestFromFlags builds the item start request from individual flags.
func buildItemStartRequestFromFlags(cmd *cobra.Command) (types.StartTestItemRQ, error) {
	name, _ := cmd.Flags().GetString("name")
	if strings.TrimSpace(name) == "" {
		return types.StartTestItemRQ{}, errors.New("--name is required")
	}

	launchUUID, _ := cmd.Flags().GetString("launch-uuid")
	if strings.TrimSpace(launchUUID) == "" {
		return types.StartTestItemRQ{}, errors.New("--launch-uuid is required")
	}

	rawType, _ := cmd.Flags().GetString("type")

	itemType, err := resolveStartTestItemType(rawType)
	if err != nil {
		return types.StartTestItemRQ{}, err
	}

	request := types.StartTestItemRQ{
		Name:       name,
		LaunchUuid: launchUUID,
		Type:       itemType,
		StartTime:  time.Now(),
		Uuid:       uuid.NewString(),
	}

	if description := core.OptionalStringFlag(cmd, "description"); description != nil {
		request.Description = description
	}

	hasStats, _ := cmd.Flags().GetBool("has-stats")
	request.HasStats = &hasStats

	if cmd.Flags().Changed("attribute") {
		rawAttrs, _ := cmd.Flags().GetStringArray("attribute")

		attrs, attrErr := core.BuildItemAttributesRQ(rawAttrs)
		if attrErr != nil {
			return types.StartTestItemRQ{}, attrErr
		}

		request.Attributes = &attrs
	}

	return request, nil
}

// buildItemFinishRequest builds the item finish request from flags or a file.
func buildItemFinishRequest(cmd *cobra.Command) (types.FinishTestItemRQ, error) {
	return core.RequestInput(cmd, "item finish", buildItemFinishRequestFromFlags)
}

// buildItemFinishRequestFromFlags builds the item finish request from individual flags.
func buildItemFinishRequestFromFlags(cmd *cobra.Command) (types.FinishTestItemRQ, error) {
	launchUUID, _ := cmd.Flags().GetString("launch-uuid")
	if strings.TrimSpace(launchUUID) == "" {
		return types.FinishTestItemRQ{}, errors.New("--launch-uuid is required")
	}

	request := types.FinishTestItemRQ{
		LaunchUuid: launchUUID,
		EndTime:    time.Now(),
	}

	request.Description = core.OptionalStringFlag(cmd, "description")
	request.Status = resolveFinishTestItemStatus(cmd)

	if cmd.Flags().Changed("attribute") {
		rawAttrs, _ := cmd.Flags().GetStringArray("attribute")

		attrs, err := core.BuildItemAttributesRQ(rawAttrs)
		if err != nil {
			return types.FinishTestItemRQ{}, err
		}

		request.Attributes = &attrs
	}

	return request, nil
}

func resolveStartTestItemType(raw string) (types.StartTestItemRQType, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	if normalized == "" {
		return "", errors.New("--type is required")
	}

	switch types.StartTestItemRQType(normalized) {
	case types.StartTestItemRQTypeAFTERCLASS,
		types.StartTestItemRQTypeAFTERGROUPS,
		types.StartTestItemRQTypeAFTERMETHOD,
		types.StartTestItemRQTypeAFTERSUITE,
		types.StartTestItemRQTypeAFTERTEST,
		types.StartTestItemRQTypeBEFORECLASS,
		types.StartTestItemRQTypeBEFOREGROUPS,
		types.StartTestItemRQTypeBEFOREMETHOD,
		types.StartTestItemRQTypeBEFORESUITE,
		types.StartTestItemRQTypeBEFORETEST,
		types.StartTestItemRQTypeSCENARIO,
		types.StartTestItemRQTypeSTEP,
		types.StartTestItemRQTypeSTORY,
		types.StartTestItemRQTypeSUITE,
		types.StartTestItemRQTypeTEST:
		return types.StartTestItemRQType(normalized), nil
	default:
		return "", fmt.Errorf("invalid type %q: must be a known test item type", raw)
	}
}

func resolveFinishTestItemStatus(cmd *cobra.Command) *types.FinishTestItemRQStatus {
	if !cmd.Flags().Changed("status") {
		return nil
	}

	raw, _ := cmd.Flags().GetString("status")
	normalized := types.FinishTestItemRQStatus(strings.ToUpper(strings.TrimSpace(raw)))

	return &normalized
}

func startTestItemHeaders() []string {
	return []string{"ID"}
}

func startTestItemRows(result *types.EntryCreatedAsyncRS) [][]string {
	return [][]string{{core.FormatOptional(result.Id)}}
}

// compile-time check that *client.TestItemScope satisfies itemStartScope.
var _ itemStartScope = (*client.TestItemScope)(nil)
