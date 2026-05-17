package cli //nolint:testpackage // white-box tests for args CLI helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIDArg_valid(t *testing.T) {
	t.Parallel()

	id, err := parseIDArg("42", "filter")

	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
}

func TestParseIDArg_nonNumeric(t *testing.T) {
	t.Parallel()

	_, err := parseIDArg("abc", "filter")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid filter ID")
	assert.Contains(t, err.Error(), `"abc"`)
	assert.Contains(t, err.Error(), "must be a numeric ID")
}

func TestParseIDArg_emptyString(t *testing.T) {
	t.Parallel()

	_, err := parseIDArg("", "widget")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid widget ID")
}

func TestParseIDArg_overflow(t *testing.T) {
	t.Parallel()

	_, err := parseIDArg("99999999999999999999", "group")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid group ID")
}

func TestParseIDArg_resourceNameInError(t *testing.T) {
	t.Parallel()

	_, err := parseIDArg("xyz", "dashboard")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid dashboard ID")
}
