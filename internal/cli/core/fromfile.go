package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// FileOp identifies the stage that failed while loading a JSON request.
type FileOp uint8

const (
	// FileRead indicates that reading the file or stdin failed.
	FileRead FileOp = iota + 1
	// FileParse indicates that unmarshaling the JSON failed.
	FileParse
)

// FileError reports a read or parse failure for a file-backed request.
type FileError struct {
	Op       FileOp
	Path     string
	Err      error
	resource string
}

// Error describes the failing operation and resource.
func (e *FileError) Error() string {
	if e.Op == FileParse {
		source := e.Path
		if source == "-" {
			source = "stdin"
		}

		return fmt.Sprintf("parsing %s JSON: %s: %v", e.resource, source, e.Err)
	}

	return fmt.Sprintf("reading %s: %v", e.resource, e.Err)
}

// Unwrap returns the underlying I/O or JSON error.
func (e *FileError) Unwrap() error {
	return e.Err
}

// RequestInput builds a request from flags unless --from-file was set.
//
//nolint:ireturn // generic helper intentionally returns type parameter T
func RequestInput[T any](
	cmd *cobra.Command,
	resource string,
	fromFlags func(*cobra.Command) (T, error),
) (T, error) {
	if !cmd.Flags().Changed("from-file") {
		return fromFlags(cmd)
	}

	path, err := cmd.Flags().GetString("from-file")
	if err != nil {
		var zero T

		return zero, fmt.Errorf("reading --from-file flag: %w", err)
	}

	return ReadRequestFromFile[T](path, resource)
}

// ReadRequestFromFile reads and unmarshals a resource's JSON from path ("-" for stdin).
//
//nolint:ireturn // generic helper intentionally returns type parameter T
func ReadRequestFromFile[T any](path, resource string) (T, error) {
	var zero T

	var (
		data []byte
		err  error
	)

	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		//nolint:gosec // path is user-provided --from-file argument
		data, err = os.ReadFile(path)
	}

	if err != nil {
		return zero, &FileError{Op: FileRead, Path: path, Err: err, resource: resource}
	}

	var request T

	err = json.Unmarshal(data, &request)
	if err != nil {
		return zero, &FileError{Op: FileParse, Path: path, Err: err, resource: resource}
	}

	return request, nil
}

// RequireFromFile returns the --from-file path after validating the flag was set.
func RequireFromFile(cmd *cobra.Command, missingMessage string) (string, error) {
	if !cmd.Flags().Changed("from-file") {
		return "", errors.New(missingMessage)
	}

	path, err := cmd.Flags().GetString("from-file")
	if err != nil {
		return "", fmt.Errorf("reading --from-file flag: %w", err)
	}

	return path, nil
}
