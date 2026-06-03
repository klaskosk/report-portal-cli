package core

import (
	"log/slog"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/spf13/cobra"
)

// Deps holds the common dependencies resolved at the start of every command handler.
type Deps struct {
	Printer    *Printer
	Client     *client.Client
	Connection *Connection
	Logger     *slog.Logger
}

// NewPrinterFromCmd returns a Printer configured from the root command's output flags.
func NewPrinterFromCmd(cmd *cobra.Command) *Printer {
	return NewTestPrinter(
		EffectiveOutputFormat(cmd.Root()),
		cmd.OutOrStdout(),
		cmd.ErrOrStderr(),
	)
}

// ResolveCmd resolves the common command preamble: logger, printer, connection, and client.
func ResolveCmd(cmd *cobra.Command) (*Deps, error) {
	logger := LoggerFromCmd(cmd)
	printer := NewPrinterFromCmd(cmd)

	conn, err := ResolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return nil, err
	}

	clientInstance, err := NewClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return nil, err
	}

	return &Deps{
		Printer:    printer,
		Client:     clientInstance,
		Connection: conn,
		Logger:     logger,
	}, nil
}

// ResolveProject resolves the project name for project-scoped commands.
func (d *Deps) ResolveProject(cmd *cobra.Command) (string, error) {
	project, err := resolveProject(cmd, d.Connection)
	if err != nil {
		d.Printer.PrintError(err)

		return "", err
	}

	return project, nil
}
