package core

import (
	"fmt"
	"os"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/spf13/cobra"
)

// WriteDownloadedContent writes content to the requested output file or stdout when no file is configured.
func WriteDownloadedContent(cmd *cobra.Command, content *client.FileContent) error {
	outputFile, _ := cmd.Flags().GetString("output-file")

	if outputFile != "" {
		err := os.WriteFile(outputFile, content.Data, 0o600)
		if err != nil {
			return fmt.Errorf("writing file %s: %w", outputFile, err)
		}

		_, _ = fmt.Fprintf(os.Stderr, "Wrote %d bytes to %s (%s)\n",
			len(content.Data), outputFile, content.ContentType)

		return nil
	}

	_, err := os.Stdout.Write(content.Data)
	if err != nil {
		return fmt.Errorf("writing to stdout: %w", err)
	}

	return nil
}
