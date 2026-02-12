package cli //nolint:testpackage // white-box tests for unexported required key constants

import (
	"reflect"
	"strings"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsonFieldNames returns the set of JSON field names for a struct type, ignoring any
// ",omitempty" or ",string" suffixes. Panics if t is not a struct.
func jsonFieldNames(structType reflect.Type) map[string]bool {
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}

	names := make(map[string]bool, structType.NumField())

	for i := range structType.NumField() {
		field := structType.Field(i)

		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		name, _, _ := strings.Cut(tag, ",")
		names[name] = true
	}

	return names
}

func TestLaunchRequiredJSONKeys_AreValidStructFields(t *testing.T) {
	t.Parallel()

	structFields := jsonFieldNames(reflect.TypeFor[types.LaunchResource]())
	require.NotEmpty(t, structFields, "LaunchResource should have JSON fields")

	for _, key := range launchRequiredJSONKeys {
		assert.True(t, structFields[key],
			"launchRequiredJSONKeys contains %q which is not a JSON field on LaunchResource", key)
	}
}

func TestItemRequiredJSONKeys_AreValidStructFields(t *testing.T) {
	t.Parallel()

	structFields := jsonFieldNames(reflect.TypeFor[types.TestItemResource]())
	require.NotEmpty(t, structFields, "TestItemResource should have JSON fields")

	for _, key := range itemRequiredJSONKeys {
		assert.True(t, structFields[key],
			"itemRequiredJSONKeys contains %q which is not a JSON field on TestItemResource", key)
	}
}

func TestLogRequiredJSONKeys_AreValidStructFields(t *testing.T) {
	t.Parallel()

	structFields := jsonFieldNames(reflect.TypeFor[types.LogResource]())
	require.NotEmpty(t, structFields, "LogResource should have JSON fields")

	for _, key := range logRequiredJSONKeys {
		assert.True(t, structFields[key],
			"logRequiredJSONKeys contains %q which is not a JSON field on LogResource", key)
	}
}

func TestRequiredJSONKeys_NoDuplicates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		keys []string
	}{
		{"launch", launchRequiredJSONKeys},
		{"item", itemRequiredJSONKeys},
		{"log", logRequiredJSONKeys},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			seen := make(map[string]bool, len(test.keys))
			for _, key := range test.keys {
				assert.False(t, seen[key],
					"duplicate required key %q in %s required keys", key, test.name)
				seen[key] = true
			}
		})
	}
}
