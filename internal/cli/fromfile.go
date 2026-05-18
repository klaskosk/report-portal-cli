package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

//nolint:ireturn // generic helper intentionally returns type parameter T
func readRequestFromFile[T any](path string) (T, error) {
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

// wrapFileError rewrites generic readRequestFromFile errors into resource-specific messages.
func wrapFileError(err error, readContext, parseContext string) error {
	msg := err.Error()

	if strings.HasPrefix(msg, "reading file ") {
		return fmt.Errorf("reading %s: %w", readContext, err)
	}

	if strings.HasPrefix(msg, "parsing JSON from ") {
		return fmt.Errorf("parsing %s: %w", parseContext, err)
	}

	return err
}
