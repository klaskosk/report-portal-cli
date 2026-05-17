package cli //nolint:testpackage // white-box tests for fromfile CLI helper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRequest struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestReadRequestFromFile_validJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "request.json")

	err := os.WriteFile(path, []byte(`{"name": "test", "value": 42}`), 0o600)
	require.NoError(t, err)

	result, err := readRequestFromFile[testRequest](path)

	require.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestReadRequestFromFile_nonExistentFile(t *testing.T) {
	t.Parallel()

	_, err := readRequestFromFile[testRequest]("/nonexistent/path/file.json")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading file")
}

func TestReadRequestFromFile_malformedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	err := os.WriteFile(path, []byte(`{invalid json`), 0o600)
	require.NoError(t, err)

	_, err = readRequestFromFile[testRequest](path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing JSON from")
	assert.Contains(t, err.Error(), "bad.json")
}

func TestReadRequestFromFile_emptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	err := os.WriteFile(path, []byte(``), 0o600)
	require.NoError(t, err)

	_, err = readRequestFromFile[testRequest](path)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing JSON from")
}
