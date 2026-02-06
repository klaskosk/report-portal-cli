package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

const defaultPageSize = 10

func newGetLaunchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch [id-or-uuid]",
		Short: "List launches or get a specific launch",
		Long: `List recent launches in a project, or get a single launch by numeric ID or UUID.

Examples:
  rp get launch                     List the 10 most recent launches
  rp get launch 42                  Get launch by numeric ID
  rp get launch abc-def-123         Get launch by UUID
  rp get launch -p other_project    List launches in a different project`,
		Args: cobra.MaximumNArgs(1),
		RunE: runGetLaunch,
	}

	cmd.Flags().StringP("project", "p", "", "project name (overrides profile default)")

	return cmd
}

func runGetLaunch(cmd *cobra.Command, args []string) error {
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
		logger.Debug("getting single launch", "project", project, "idOrUUID", args[0])

		return getSingleLaunch(cmd, clientInstance, project, args[0], printer)
	}

	logger.Debug("listing launches", "project", project)

	return listLaunches(cmd, clientInstance, project, printer)
}

func getSingleLaunch(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project, idOrUUID string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Launches()

	var (
		launch *types.LaunchResource
		err    error
	)

	if isNumericID(idOrUUID) {
		launch, err = scope.GetByID(ctx, idOrUUID)
	} else {
		launch, err = scope.GetByUUID(ctx, idOrUUID)
	}

	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("getting launch %s: %w", idOrUUID, err)
	}

	headers, row := launchRow(launch)

	return printer.Print(launch, headers, [][]string{row})
}

func listLaunches(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project string,
	printer *Printer,
) error {
	ctx := cmd.Context()
	scope := clientInstance.Project(project).Launches()

	pageSize := int32(defaultPageSize)

	result, err := scope.List(ctx,
		client.WithPageSize(pageSize),
		client.WithSort("startTime,desc"),
	)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("listing launches: %w", err)
	}

	headers := launchHeaders()
	rows := make([][]string, 0, len(result.Content))

	for i := range result.Content {
		_, row := launchRow(&result.Content[i])
		rows = append(rows, row)
	}

	return printer.Print(result.Content, headers, rows)
}

func launchHeaders() []string {
	return []string{"ID", "UUID", "NAME", "STATUS", "START TIME", "NUMBER"}
}

func launchRow(l *types.LaunchResource) ([]string, []string) {
	startTime := time.Time(l.StartTime).UTC().Format(time.DateTime)

	return launchHeaders(), []string{
		strconv.FormatInt(l.Id, 10),
		l.Uuid,
		l.Name,
		l.Status,
		startTime,
		strconv.FormatInt(l.Number, 10),
	}
}

func isNumericID(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)

	return err == nil
}
