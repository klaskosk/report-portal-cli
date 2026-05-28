package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
		RunE: withPrintedErrors(runItemStart),
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
		RunE: withPrintedErrors(runItemFinish),
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
	printer := resolvePrinter(cmd)

	request, err := buildItemStartRequest(cmd)
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

	parentUUID, _ := cmd.Flags().GetString("parent-uuid")
	deps.logger.Debug(
		"starting test item",
		"project",
		project,
		"name",
		request.Name,
		"parentUuid",
		parentUUID,
	)

	scope := deps.client.Project(project).Items()

	result, err := dispatchItemStart(cmd.Context(), scope, parentUUID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("starting test item in project %s: %w", project, err)
	}

	return deps.printer.Print(result, startTestItemHeaders(), startTestItemRows(result))
}

func runItemFinish(cmd *cobra.Command, args []string) error {
	printer := resolvePrinter(cmd)

	request, err := buildItemFinishRequest(cmd)
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

	itemUUID := args[0]
	deps.logger.Debug("finishing test item", "project", project, "uuid", itemUUID)

	result, err := deps.client.Project(project).Items().Finish(cmd.Context(), itemUUID, request)
	if err != nil {
		deps.printer.PrintError(err)

		return fmt.Errorf("finishing test item %s in project %s: %w", itemUUID, project, err)
	}

	return deps.printer.Print(result, operationResultHeaders(), operationResultRows(result))
}

func buildItemStartRequest(cmd *cobra.Command) (types.StartTestItemRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		req, err := readRequestFromFile[types.StartTestItemRQ](path)
		if err != nil {
			return types.StartTestItemRQ{}, wrapFileError(
				err,
				"item start file",
				"item start JSON",
			)
		}

		return req, nil
	}

	return buildItemStartRequestFromFlags(cmd)
}

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

	if description := optionalStringFlag(cmd, "description"); description != nil {
		request.Description = description
	}

	hasStats, _ := cmd.Flags().GetBool("has-stats")
	request.HasStats = &hasStats

	if cmd.Flags().Changed("attribute") {
		rawAttrs, _ := cmd.Flags().GetStringArray("attribute")

		attrs, attrErr := buildItemAttributesRQ(rawAttrs)
		if attrErr != nil {
			return types.StartTestItemRQ{}, attrErr
		}

		request.Attributes = &attrs
	}

	return request, nil
}

func buildItemFinishRequest(cmd *cobra.Command) (types.FinishTestItemRQ, error) {
	if cmd.Flags().Changed("from-file") {
		path, _ := cmd.Flags().GetString("from-file")

		req, err := readRequestFromFile[types.FinishTestItemRQ](path)
		if err != nil {
			return types.FinishTestItemRQ{}, wrapFileError(
				err,
				"item finish file",
				"item finish JSON",
			)
		}

		return req, nil
	}

	return buildItemFinishRequestFromFlags(cmd)
}

func buildItemFinishRequestFromFlags(cmd *cobra.Command) (types.FinishTestItemRQ, error) {
	launchUUID, _ := cmd.Flags().GetString("launch-uuid")
	if strings.TrimSpace(launchUUID) == "" {
		return types.FinishTestItemRQ{}, errors.New("--launch-uuid is required")
	}

	request := types.FinishTestItemRQ{
		LaunchUuid: launchUUID,
		EndTime:    time.Now(),
	}

	request.Description = optionalStringFlag(cmd, "description")
	request.Status = resolveFinishTestItemStatus(cmd)

	if cmd.Flags().Changed("attribute") {
		rawAttrs, _ := cmd.Flags().GetStringArray("attribute")

		attrs, err := buildItemAttributesRQ(rawAttrs)
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
	case types.AFTERCLASS,
		types.AFTERGROUPS,
		types.AFTERMETHOD,
		types.AFTERSUITE,
		types.AFTERTEST,
		types.BEFORECLASS,
		types.BEFOREGROUPSBEFOREMETHOD,
		types.BEFORESUITE,
		types.BEFORETEST,
		types.SCENARIO,
		types.STEP,
		types.STORY,
		types.SUITE,
		types.TEST:
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
	return [][]string{{formatOptional(result.Id)}}
}

// compile-time check that *client.TestItemScope satisfies itemStartScope.
var _ itemStartScope = (*client.TestItemScope)(nil)
