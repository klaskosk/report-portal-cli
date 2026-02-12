package cli //nolint:testpackage // white-box tests for unexported json contract helpers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureJSONObjectKeys_PreservesLargeInt64(t *testing.T) {
	t.Parallel()

	// Value above 2^53 that loses precision when round-tripped through float64.
	type resource struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	input := resource{ID: 1<<53 + 1, Name: "test"}

	result, err := ensureJSONObjectKeys(input, "id", "name")
	require.NoError(t, err)

	// Re-encode the normalized map to JSON and verify the integer survives.
	output, err := json.Marshal(result)
	require.NoError(t, err)

	assert.Contains(t, string(output), "9007199254740993",
		"int64 above 2^53 must survive the normalization round-trip")
}

func TestEnsureJSONObjectKeys_AddsMissingKeysAsNull(t *testing.T) {
	t.Parallel()

	type resource struct {
		Name string `json:"name"`
	}

	input := resource{Name: "test"}

	result, err := ensureJSONObjectKeys(input, "name", "missing1", "missing2")
	require.NoError(t, err)

	assert.Contains(t, result, "name")
	assert.Contains(t, result, "missing1")
	assert.Contains(t, result, "missing2")

	// Missing keys should marshal as null.
	output, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(output, &decoded))

	assert.Nil(t, decoded["missing1"])
	assert.Nil(t, decoded["missing2"])
	assert.Equal(t, "test", decoded["name"])
}

func TestEnsureJSONObjectKeys_PreservesExistingKeys(t *testing.T) {
	t.Parallel()

	type resource struct {
		ID     int64   `json:"id"`
		Name   string  `json:"name"`
		Status *string `json:"status"`
	}

	input := resource{ID: 42, Name: "launch"}

	result, err := ensureJSONObjectKeys(input, "id", "name", "status")
	require.NoError(t, err)

	output, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(output, &decoded))

	assert.InDelta(t, float64(42), decoded["id"], 0)
	assert.Equal(t, "launch", decoded["name"])
	assert.Nil(t, decoded["status"])
}

func TestEnsureJSONArrayObjectKeys_PreservesLargeInt64(t *testing.T) {
	t.Parallel()

	type resource struct {
		ID int64 `json:"id"`
	}

	input := []resource{{ID: 1<<53 + 1}}

	result, err := ensureJSONArrayObjectKeys(input, "id")
	require.NoError(t, err)
	require.Len(t, result, 1)

	output, err := json.Marshal(result)
	require.NoError(t, err)

	assert.Contains(t, string(output), "9007199254740993",
		"int64 above 2^53 must survive the normalization round-trip")
}

func TestEnsureJSONArrayObjectKeys_AddsMissingKeysAsNull(t *testing.T) {
	t.Parallel()

	type resource struct {
		Name string `json:"name"`
	}

	input := []resource{{Name: "first"}, {Name: "second"}}

	result, err := ensureJSONArrayObjectKeys(input, "name", "missing")
	require.NoError(t, err)
	require.Len(t, result, 2)

	for i, object := range result {
		output, err := json.Marshal(object)
		require.NoError(t, err)

		var decoded map[string]any
		require.NoError(t, json.Unmarshal(output, &decoded))

		assert.Nil(t, decoded["missing"], "object at index %d should have null for missing key", i)
		assert.NotEmpty(t, decoded["name"], "object at index %d should preserve name", i)
	}
}

func TestEnsureJSONObjectKeys_PreservesNestedObjects(t *testing.T) {
	t.Parallel()

	type nested struct {
		Inner map[string]string `json:"inner"`
		Value int64             `json:"value"`
	}

	input := nested{
		Inner: map[string]string{"key": "val"},
		Value: 123,
	}

	result, err := ensureJSONObjectKeys(input, "inner", "value", "extra")
	require.NoError(t, err)

	output, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(output, &decoded))

	innerMap, ok := decoded["inner"].(map[string]any)
	require.True(t, ok, "nested object should be preserved")
	assert.Equal(t, "val", innerMap["key"])
	assert.Nil(t, decoded["extra"])
}
