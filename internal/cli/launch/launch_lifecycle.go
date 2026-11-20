package launch

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

func newLaunchStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new launch",
		Long: `Start a new launch in a project.

Examples:
  rp launch start --name "Nightly Run" --mode DEFAULT -p my_project
  rp launch start --name "Debug Run" --mode debug -p my_project
  rp launch start --from-file launch_start.json -p my_project`,
		Args: cobra.NoArgs,
		RunE: runLaunchStart,
	}

	cmd.Flags().String("name", "", "launch name")
	cmd.Flags().String("description", "", "launch description")
	cmd.Flags().String("mode", "", "launch mode (DEFAULT or DEBUG)")
	cmd.Flags().StringArray("attribute", []string{}, "attribute in key:value format (repeatable)")
	cmd.Flags().String("uuid", "", "optional user-supplied launch UUID")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing StartLaunchRQ (use - for stdin)")

	return cmd
}

func newLaunchFinishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finish <uuid>",
		Short: "Finish a launch by UUID",
		Long: `Finish a launch using the UUID returned by launch start.

Examples:
  rp launch finish abc-uuid-123 -p my_project
  rp launch finish abc-uuid-123 --status PASSED -p my_project
  rp launch finish abc-uuid-123 --from-file launch_finish.json -p my_project`,
		Args: cobra.ExactArgs(1),
		RunE: runLaunchFinish,
	}

	cmd.Flags().String("status", "", "final launch status")
	cmd.Flags().String("description", "", "final launch description")
	cmd.Flags().StringArray("attribute", []string{}, "attribute in key:value format (repeatable)")
	cmd.Flags().
		String("from-file", "", "path to JSON file containing FinishExecutionRQ (use - for stdin)")

	return cmd
}

func runLaunchStart(cmd *cobra.Command, _ []string) error {
	request, err := buildLaunchStartRequest(cmd)
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

	deps.Logger.Debug("starting launch", "project", project, "name", request.Name)

	result, err := deps.Client.Project(project).Launches().Start(cmd.Context(), request)
	if err != nil {
		return fmt.Errorf("starting launch in project %s: %w", project, err)
	}

	return deps.Printer.Print(result, startLaunchHeaders(), startLaunchRows(result))
}

func runLaunchFinish(cmd *cobra.Command, args []string) error {
	request, err := buildLaunchFinishRequest(cmd)
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

	launchUUID := args[0]
	deps.Logger.Debug("finishing launch", "project", project, "uuid", launchUUID)

	result, err := deps.Client.Project(project).
		Launches().
		Finish(cmd.Context(), launchUUID, request)
	if err != nil {
		return fmt.Errorf("finishing launch %s in project %s: %w", launchUUID, project, err)
	}

	return deps.Printer.Print(result, finishLaunchHeaders(), finishLaunchRows(result))
}

// buildLaunchStartRequest builds the launch start request from flags or a file.
func buildLaunchStartRequest(cmd *cobra.Command) (types.StartLaunchRQ, error) {
	return core.RequestInput(cmd, "launch start", buildLaunchStartRequestFromFlags)
}

// buildLaunchStartRequestFromFlags builds the launch start request from individual flags.
func buildLaunchStartRequestFromFlags(cmd *cobra.Command) (types.StartLaunchRQ, error) {
	name, _ := cmd.Flags().GetString("name")
	if strings.TrimSpace(name) == "" {
		return types.StartLaunchRQ{}, errors.New("--name is required")
	}

	request := types.StartLaunchRQ{
		Name:      name,
		StartTime: time.Now(),
	}

	if description := core.OptionalStringFlag(cmd, "description"); description != nil {
		request.Description = description
	}

	if cmd.Flags().Changed("mode") {
		raw, _ := cmd.Flags().GetString("mode")

		mode, err := resolveStartLaunchMode(raw)
		if err != nil {
			return types.StartLaunchRQ{}, err
		}

		request.Mode = &mode
	}

	if cmd.Flags().Changed("attribute") {
		rawAttrs, _ := cmd.Flags().GetStringArray("attribute")

		attrs, err := core.BuildItemAttributesRQ(rawAttrs)
		if err != nil {
			return types.StartLaunchRQ{}, err
		}

		request.Attributes = &attrs
	}

	if cmd.Flags().Changed("uuid") {
		request.Uuid, _ = cmd.Flags().GetString("uuid")
	} else {
		request.Uuid = uuid.NewString()
	}

	return request, nil
}

// buildLaunchFinishRequest builds the launch finish request from flags or a file.
func buildLaunchFinishRequest(cmd *cobra.Command) (types.FinishExecutionRQ, error) {
	return core.RequestInput(cmd, "launch finish", buildLaunchFinishRequestFromFlags)
}

// buildLaunchFinishRequestFromFlags builds the launch finish request from individual flags.
func buildLaunchFinishRequestFromFlags(cmd *cobra.Command) (types.FinishExecutionRQ, error) {
	request := types.FinishExecutionRQ{
		EndTime: time.Now(),
	}

	request.Description = core.OptionalStringFlag(cmd, "description")
	request.Status = resolveFinishExecutionStatus(cmd)

	if cmd.Flags().Changed("attribute") {
		rawAttrs, _ := cmd.Flags().GetStringArray("attribute")

		attrs, err := core.BuildItemAttributesRQ(rawAttrs)
		if err != nil {
			return types.FinishExecutionRQ{}, err
		}

		request.Attributes = &attrs
	}

	return request, nil
}

func resolveFinishExecutionStatus(cmd *cobra.Command) *types.FinishExecutionRQStatus {
	if !cmd.Flags().Changed("status") {
		return nil
	}

	raw, _ := cmd.Flags().GetString("status")
	normalized := types.FinishExecutionRQStatus(strings.ToUpper(strings.TrimSpace(raw)))

	return &normalized
}

func startLaunchHeaders() []string {
	return []string{"ID", "NUMBER"}
}

func startLaunchRows(result *types.StartLaunchRS) [][]string {
	return [][]string{
		{
			core.FormatOptional(result.Id),
			core.FormatOptional(result.Number),
		},
	}
}

func finishLaunchHeaders() []string {
	return []string{"ID", "NUMBER", "LINK"}
}

func finishLaunchRows(result *types.FinishLaunchRS) [][]string {
	return [][]string{
		{
			core.FormatOptional(result.Id),
			core.FormatOptional(result.Number),
			core.FormatOptional(result.Link),
		},
	}
}
