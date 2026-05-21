package cli

import (
	"fmt"
	"math"
	"strconv"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// parseIDArg parses a positional argument as an int64 resource ID.
func parseIDArg(arg, resourceName string) (int64, error) {
	id, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s ID %q: must be a numeric ID", resourceName, arg)
	}

	return id, nil
}

// parseOptionalInt64Flag parses value as an int64 if non-empty, returning (0, false, nil) when empty.
// Returns an error when the value is out of the signed 32-bit range required by the upstream API.
func parseOptionalInt64Flag(name, value string) (int64, bool, error) {
	if value == "" {
		return 0, false, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("invalid --%s value %q: must be a numeric ID", name, value)
	}

	if parsed > math.MaxInt32 || parsed < math.MinInt32 {
		return 0, false, fmt.Errorf(
			"invalid --%s value %q: ID must fit in a signed 32-bit integer (max %d)",
			name, value, int64(math.MaxInt32),
		)
	}

	return parsed, true, nil
}

// buildAttributeResources converts --attribute flag values to []ItemAttributeResource.
func buildAttributeResources(attributes []string) ([]types.ItemAttributeResource, error) {
	resources := make([]types.ItemAttributeResource, 0, len(attributes))

	for _, attr := range attributes {
		key, value, found := parseAttribute(attr)
		if !found {
			return nil, fmt.Errorf("invalid attribute format %q (expected key:value)", attr)
		}

		resources = append(resources, types.ItemAttributeResource{
			Key:   &key,
			Value: value,
		})
	}

	return resources, nil
}
