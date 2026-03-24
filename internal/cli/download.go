package cli

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/spf13/cobra"
)

// newDownloadCmd builds the "download" subcommand for fetching file attachments.
func newDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <data-id>",
		Short: "Download an attachment by data ID",
		Long: `Download an attachment by its data store ID (from a log entry's binaryContent.id).

If --output-file is specified, the file is written to that path and a
confirmation is printed to stderr. Otherwise, raw bytes are written to stdout.

If --log is specified, the argument is treated as a log ID and the attachment's
data ID is resolved automatically from the log's binaryContent field.

Examples:
  rp download 12345 -p my_project                   Write file bytes to stdout
  rp download 12345 -p my_project --output-file out.png  Save to a file
  rp download --log 678 -p my_project                Resolve data ID from log entry`,
		Args: cobra.ExactArgs(1),
		RunE: withPrintedErrors(runDownload),
	}

	cmd.Flags().StringP("project", "p", "", "project name (overrides profile default)")
	cmd.Flags().String("output-file", "", "write to file instead of stdout")
	cmd.Flags().Bool("log", false, "treat argument as a log ID and resolve binaryContent.id")

	return cmd
}

// runDownload executes the download command, resolving the data ID and writing file content to either a named file or
// stdout.
func runDownload(cmd *cobra.Command, args []string) error {
	logger := loggerFromCmd(cmd)
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	printer := newPrinter(outputFmt)

	clientInstance, project, err := resolveDownloadClientAndProject(cmd, printer, logger)
	if err != nil {
		return err
	}

	argumentValue := args[0]
	shouldResolveFromLog, _ := cmd.Flags().GetBool("log")

	var dataID int64

	if shouldResolveFromLog {
		dataID, err = resolveDataIDFromLog(cmd, clientInstance, project, argumentValue, printer)
		if err != nil {
			return err
		}
	} else {
		dataID, err = strconv.ParseInt(argumentValue, 10, 64)
		if err != nil {
			parseErr := fmt.Errorf("invalid data ID %q: %w", argumentValue, err)
			printer.PrintError(parseErr)

			return parseErr
		}
	}

	logger.Debug("downloading file", "project", project, "dataId", dataID)

	content, err := clientInstance.Project(project).DownloadFile(cmd.Context(), dataID)
	if err != nil {
		printer.PrintError(err)

		return fmt.Errorf("downloading file %d: %w", dataID, err)
	}

	return writeDownloadedContent(cmd, content)
}

// resolveDataIDFromLog fetches a log entry by ID and extracts the data store ID from its binaryContent field.
func resolveDataIDFromLog(
	cmd *cobra.Command,
	clientInstance *client.Client,
	project, logID string,
	printer *Printer,
) (int64, error) {
	numericID, parseErr := strconv.ParseInt(logID, 10, 64)
	if parseErr != nil {
		return 0, fmt.Errorf("invalid log id %q: %w", logID, parseErr)
	}

	logEntry, err := clientInstance.Project(project).Logs().GetByID(cmd.Context(), numericID)
	if err != nil {
		printer.PrintError(err)

		return 0, fmt.Errorf("getting log %s: %w", logID, err)
	}

	if logEntry.BinaryContent == nil {
		noAttachErr := fmt.Errorf("log %s has no attachment", logID)
		printer.PrintError(noAttachErr)

		return 0, noAttachErr
	}

	rawID := logEntry.BinaryContent.Id
	if rawID == "" {
		emptyErr := fmt.Errorf("log %s has empty binaryContent.id", logID)
		printer.PrintError(emptyErr)

		return 0, emptyErr
	}

	dataID, convErr := strconv.ParseInt(rawID, 10, 64)
	if convErr != nil {
		convErr = fmt.Errorf("parsing binaryContent.id for log %s: %w", logID, convErr)
		printer.PrintError(convErr)

		return 0, convErr
	}

	return dataID, nil
}

// resolveDownloadOutputPath returns the destination path from --output-file.
func resolveDownloadOutputPath(cmd *cobra.Command) string {
	outputPath, _ := cmd.Flags().GetString("output-file")

	return outputPath
}

// resolveDownloadClientAndProject resolves connection, project, and API client dependencies for downloads.
func resolveDownloadClientAndProject(
	cmd *cobra.Command,
	printer *Printer,
	logger *slog.Logger,
) (*client.Client, string, error) {
	conn, err := resolveConnection(cmd)
	if err != nil {
		printer.PrintError(err)

		return nil, "", err
	}

	project, err := resolveProject(cmd, conn)
	if err != nil {
		printer.PrintError(err)

		return nil, "", err
	}

	clientInstance, err := newClient(conn, logger)
	if err != nil {
		printer.PrintError(err)

		return nil, "", err
	}

	return clientInstance, project, nil
}

// writeDownloadedContent writes content to the requested output file or stdout when no file is configured.
func writeDownloadedContent(
	cmd *cobra.Command,
	content *client.FileContent,
) error {
	outputFile := resolveDownloadOutputPath(cmd)

	if outputFile != "" {
		return writeToFile(outputFile, content)
	}

	_, err := os.Stdout.Write(content.Data)
	if err != nil {
		return fmt.Errorf("writing to stdout: %w", err)
	}

	return nil
}

// writeToFile writes the downloaded content to disk and prints a confirmation to stderr.
func writeToFile(path string, content *client.FileContent) error {
	err := os.WriteFile(path, content.Data, 0o600)
	if err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}

	_, _ = fmt.Fprintf(os.Stderr, "Wrote %d bytes to %s (%s)\n",
		len(content.Data), path, content.ContentType)

	return nil
}
