package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// ReadRequestFromFile reads and unmarshals a JSON request from path ("-" for stdin).
//
//nolint:ireturn // generic helper intentionally returns type parameter T
func ReadRequestFromFile[T any](path string) (T, error) {
	var zero T

	var (
		data []byte
		err  error
		name = path
	)

	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
		name = "stdin"
	} else {
		//nolint:gosec // path is user-provided --from-file argument
		data, err = os.ReadFile(path)
	}

	if err != nil {
		return zero, fmt.Errorf("reading file %s: %w", name, err)
	}

	var request T

	err = json.Unmarshal(data, &request)
	if err != nil {
		return zero, fmt.Errorf("parsing JSON from %s: %w", name, err)
	}

	return request, nil
}

// RequireFromFile returns the --from-file path after validating the flag was set.
func RequireFromFile(cmd *cobra.Command, printer *Printer, missingMessage string) (string, error) {
	if !cmd.Flags().Changed("from-file") {
		err := errors.New(missingMessage)
		printer.PrintError(err)

		return "", err
	}

	path, _ := cmd.Flags().GetString("from-file")

	return path, nil
}

// WrapFileError rewrites generic ReadRequestFromFile errors into resource-specific messages.
func WrapFileError(err error, readContext, parseContext string) error {
	msg := err.Error()

	if strings.HasPrefix(msg, "reading file ") {
		return fmt.Errorf("reading %s: %w", readContext, err)
	}

	if strings.HasPrefix(msg, "parsing JSON from ") {
		return fmt.Errorf("parsing %s: %w", parseContext, err)
	}

	return err
}
