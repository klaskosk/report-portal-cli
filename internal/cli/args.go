package cli

import (
	"fmt"
	"strconv"
)

// parseIDArg parses a positional argument as an int64 resource ID.
func parseIDArg(arg, resourceName string) (int64, error) {
	id, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s ID %q: must be a numeric ID", resourceName, arg)
	}

	return id, nil
}
