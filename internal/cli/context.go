package cli

import (
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/spf13/cobra"
)

// cmdDeps holds the common dependencies resolved at the start of every command handler.
type cmdDeps struct {
	printer    *Printer
	client     *client.Client
	connection *Connection
	logger     *slog.Logger
}

// resolvePrinter returns a Printer configured from the root command's --output flag.
func resolvePrinter(cmd *cobra.Command) *Printer {
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")

	return newPrinter(outputFmt)
}

// resolveCmd resolves the common command preamble: logger, printer, connection, and client.
func resolveCmd(cmd *cobra.Command) (*cmdDeps, error) {
	logger := loggerFromCmd(cmd)
	printer := resolvePrinter(cmd)

	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return nil, err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return nil, err
	}

	return &cmdDeps{
		printer:    printer,
		client:     clientInstance,
		connection: conn,
		logger:     logger,
	}, nil
}

// resolveProject resolves the project name for project-scoped commands.
func (c *cmdDeps) resolveProject(cmd *cobra.Command) (string, error) {
	project, err := resolveProject(cmd, c.connection)
	if err != nil {
		c.printer.PrintError(err)

		return "", err
	}

	return project, nil
}
