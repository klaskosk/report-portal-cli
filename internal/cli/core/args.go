package core

import (
	"fmt"
	"math"
	"strconv"

	"github.com/klaskosk/report-portal-cli/pkg/types"
)

// ParseIDArg parses a positional argument as an int64 resource ID.
func ParseIDArg(arg, resourceName string) (int64, error) {
	id, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s ID %q: must be a numeric ID", resourceName, arg)
	}

	return id, nil
}

// ParseOptionalInt64Flag parses value as an int64 if non-empty, returning (0, false, nil) when empty.
func ParseOptionalInt64Flag(name, value string) (int64, bool, error) {
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

// ParseAttribute parses an attribute string in "key:value" format.
func ParseAttribute(attr string) (string, string, bool) {
	for i := range len(attr) {
		if attr[i] == ':' {
			if i == 0 {
				return "", "", false
			}

			return attr[:i], attr[i+1:], true
		}
	}

	return "", "", false
}

// BuildAttributeResources converts --attribute flag values to []ItemAttributeResource.
func BuildAttributeResources(attributes []string) ([]types.ItemAttributeResource, error) {
	resources := make([]types.ItemAttributeResource, 0, len(attributes))

	for _, attr := range attributes {
		key, value, found := ParseAttribute(attr)
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

// BuildItemAttributesRQ converts --attribute flag values to []ItemAttributesRQ.
func BuildItemAttributesRQ(attributes []string) ([]types.ItemAttributesRQ, error) {
	resources, err := BuildAttributeResources(attributes)
	if err != nil {
		return nil, err
	}

	attrs := make([]types.ItemAttributesRQ, len(resources))
	for i, resource := range resources {
		attrs[i] = types.ItemAttributesRQ{
			Key:   resource.Key,
			Value: resource.Value,
		}
	}

	return attrs, nil
}
