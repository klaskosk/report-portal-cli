package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
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

	result, err := ReadRequestFromFile[testRequest](path, "test request")

	require.NoError(t, err)
	assert.Equal(t, "test", result.Name)
	assert.Equal(t, 42, result.Value)
}

func TestReadRequestFromFile_nonExistentFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.json")
	result, err := ReadRequestFromFile[testRequest](path, "test request")

	var fileError *FileError
	require.ErrorAs(t, err, &fileError)
	assert.Equal(t, FileRead, fileError.Op)
	assert.Equal(t, path, fileError.Path)
	require.ErrorIs(t, err, os.ErrNotExist)

	var pathError *os.PathError
	require.ErrorAs(t, err, &pathError)
	require.EqualError(t, err, "reading test request: "+pathError.Error())
	assert.Empty(t, result)
}

func TestReadRequestFromFile_malformedJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	err := os.WriteFile(path, []byte(`{invalid json`), 0o600)
	require.NoError(t, err)

	result, err := ReadRequestFromFile[testRequest](path, "test request")

	var fileError *FileError
	require.ErrorAs(t, err, &fileError)
	assert.Equal(t, FileParse, fileError.Op)
	assert.Equal(t, path, fileError.Path)

	var syntaxError *json.SyntaxError
	require.ErrorAs(t, err, &syntaxError)
	require.EqualError(
		t,
		err,
		"parsing test request JSON: "+path+": invalid character 'i' looking for beginning of object key string",
	)
	assert.Empty(t, result)
}

func TestReadRequestFromFile_emptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	err := os.WriteFile(path, []byte(``), 0o600)
	require.NoError(t, err)

	_, err = ReadRequestFromFile[testRequest](path, "test request")

	require.EqualError(t, err, "parsing test request JSON: "+path+": unexpected end of JSON input")
}

func TestRequestInput_usesFlags(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("from-file", "", "")
	cmd.Flags().String("name", "", "")
	require.NoError(t, cmd.ParseFlags([]string{"--name", "flags"}))

	result, err := RequestInput(cmd, "test request", func(cmd *cobra.Command) (testRequest, error) {
		name, err := cmd.Flags().GetString("name")
		require.NoError(t, err)

		return testRequest{Name: name}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, testRequest{Name: "flags"}, result)
}

func TestRequestInput_propagatesFlagError(t *testing.T) {
	t.Parallel()

	want := errors.New("invalid flags")
	cmd := &cobra.Command{}
	cmd.Flags().String("from-file", "", "")

	_, err := RequestInput(cmd, "test request", func(*cobra.Command) (testRequest, error) {
		return testRequest{}, want
	})
	require.ErrorIs(t, err, want)
}

func TestRequestInput_fileTakesPrecedence(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "request.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"file","value":42}`), 0o600))

	cmd := &cobra.Command{}
	cmd.Flags().String("from-file", "", "")
	require.NoError(t, cmd.Flags().Set("from-file", path))

	result, err := RequestInput(cmd, "test request", func(*cobra.Command) (testRequest, error) {
		return testRequest{}, errors.New("flag builder called")
	})
	require.NoError(t, err)
	assert.Equal(t, testRequest{Name: "file", Value: 42}, result)
}

func TestRequestInput_doesNotFallBackOnReadFailure(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("from-file", "", "")
	require.NoError(t, cmd.Flags().Set("from-file", ""))

	_, err := RequestInput(cmd, "test request", func(*cobra.Command) (testRequest, error) {
		return testRequest{Name: "flags"}, nil
	})

	var fileError *FileError
	require.ErrorAs(t, err, &fileError)
	assert.Equal(t, FileRead, fileError.Op)
	assert.Empty(t, fileError.Path)
}

func TestRequireFromFile_missingFlag(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("from-file", "", "")

	path, err := RequireFromFile(cmd, "--from-file is required for update")

	require.Error(t, err)
	assert.Empty(t, path)
	assert.Contains(t, err.Error(), "--from-file is required for update")
}

func TestRequireFromFile_returnsPath(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("from-file", "", "")
	require.NoError(t, cmd.Flags().Set("from-file", "/tmp/payload.json"))

	path, err := RequireFromFile(cmd, "--from-file is required")

	require.NoError(t, err)
	assert.Equal(t, "/tmp/payload.json", path)
}
