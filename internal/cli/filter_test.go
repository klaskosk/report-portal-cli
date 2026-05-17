package cli //nolint:testpackage // white-box tests for filter CLI helpers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilterCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newFilterCmd()

	assert.Equal(t, []string{"create", "delete", "get", "list", "update"}, commandNames(cmd))
}

func TestNewFilterCmd_ProjectFlagInheritedBySubcommands(t *testing.T) {
	t.Parallel()

	cmd := newFilterCmd()

	for _, sub := range cmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			t.Parallel()

			flag := sub.InheritedFlags().Lookup("project")
			require.NotNil(
				t,
				flag,
				"subcommand %q should inherit --project persistent flag",
				sub.Name(),
			)
			assert.Equal(t, "p", flag.Shorthand)
		})
	}
}

func TestNewFilterListCmd_DefaultSort(t *testing.T) {
	t.Parallel()

	cmd := newFilterListCmd()

	flags, err := resolveListFlags(cmd)
	require.NoError(t, err)

	assert.Equal(t, []string{"name,ASC"}, flags.sort)
}

func TestFilterRow_populated(t *testing.T) {
	t.Parallel()

	desc := "A short description"
	filterResource := &types.UserFilterResource{
		Id:          7,
		Name:        "My Filter",
		Type:        "launch",
		Owner:       "admin",
		Description: &desc,
		Conditions: []types.UserFilterCondition{
			{Condition: "cnt", FilteringField: "name", Value: "x"},
			{Condition: "eq", FilteringField: "status", Value: "FAILED"},
		},
	}

	headers, row := filterRow(filterResource)

	assert.Equal(t, filterHeaders(), headers)
	assert.Equal(t, []string{
		"7",
		"My Filter",
		"launch",
		"admin",
		"A short description",
		"2",
	}, row)
}

func TestFilterRow_nilDescription(t *testing.T) {
	t.Parallel()

	filterResource := &types.UserFilterResource{
		Id:     1,
		Name:   "F",
		Type:   "log",
		Owner:  "u",
		Orders: []types.Order{},
	}

	_, row := filterRow(filterResource)

	assert.Empty(t, row[4])
}

func TestFilterRow_descriptionTruncatedAt50(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", 60)
	filterResource := &types.UserFilterResource{
		Id:          2,
		Name:        "n",
		Type:        "testitem",
		Owner:       "o",
		Description: &long,
	}

	_, row := filterRow(filterResource)

	expected := strings.Repeat("a", 50) + "..."
	assert.Equal(t, expected, row[4])
}

func TestParseFilterID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseFilterID("42")
	require.NoError(t, err)
	assert.EqualValues(t, 42, id)
}

func TestParseFilterID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseFilterID("not-a-number")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid filter ID "not-a-number": must be a numeric ID`)
}

func TestReadFilterRequestFromFile_happyPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "filter.json")

	jsonContent := `{
  "name": "launch failures",
  "type": "launch",
  "description": "failed runs",
  "conditions": [
    {"condition": "eq", "filteringField": "status", "value": "FAILED"}
  ],
  "orders": [{"isAsc": true, "sortingColumn": "name"}]
}`

	require.NoError(t, os.WriteFile(path, []byte(jsonContent), 0o600))

	request, err := readFilterRequestFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, "launch failures", request.Name)
	assert.Equal(t, types.UpdateUserFilterRQTypeLaunch, request.Type)
	require.NotNil(t, request.Description)
	assert.Equal(t, "failed runs", *request.Description)
	require.Len(t, request.Conditions, 1)
	assert.Equal(t, "eq", request.Conditions[0].Condition)
	assert.Equal(t, "status", request.Conditions[0].FilteringField)
	assert.Equal(t, "FAILED", request.Conditions[0].Value)
	require.Len(t, request.Orders, 1)
	assert.True(t, request.Orders[0].IsAsc)
	assert.Equal(t, "name", request.Orders[0].SortingColumn)
}

func TestReadFilterRequestFromFile_ignoresUnknownFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "filter.json")

	jsonContent := `{
  "id": 99,
  "owner": "someone",
  "name": "n",
  "type": "log",
  "conditions": [],
  "orders": []
}`

	require.NoError(t, os.WriteFile(path, []byte(jsonContent), 0o600))

	request, err := readFilterRequestFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, "n", request.Name)
	assert.Equal(t, types.UpdateUserFilterRQTypeLog, request.Type)
}

func TestReadFilterRequestFromFile_invalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{`), 0o600))

	_, err := readFilterRequestFromFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing filter JSON")
}

func TestReadFilterRequestFromFile_missingFile(t *testing.T) {
	t.Parallel()

	_, err := readFilterRequestFromFile(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading filter definition")
}

func TestReadFilterRequestFromFile_stdin(t *testing.T) { //nolint:paralleltest // swaps os.Stdin
	jsonContent := `{"name":"stdin-filter","type":"launch","conditions":[],"orders":[]}`

	pipeRead, pipeWrite, err := os.Pipe()
	require.NoError(t, err)

	_, err = pipeWrite.WriteString(jsonContent)
	require.NoError(t, err)
	require.NoError(t, pipeWrite.Close())

	origStdin := os.Stdin
	os.Stdin = pipeRead

	defer func() {
		os.Stdin = origStdin

		require.NoError(t, pipeRead.Close())
	}()

	request, err := readFilterRequestFromFile("-")
	require.NoError(t, err)
	assert.Equal(t, "stdin-filter", request.Name)
	assert.Equal(t, types.UpdateUserFilterRQTypeLaunch, request.Type)
}

func TestApplyFilterRequestFlagOverrides_nameAndDescription(t *testing.T) {
	t.Parallel()

	overrideCases := []struct {
		name     string
		flags    []string
		wantName string
		wantDesc string
	}{
		{
			name:     "name override only",
			flags:    []string{"--name", "from-flag"},
			wantName: "from-flag",
			wantDesc: "from-file",
		},
		{
			name:     "description override only",
			flags:    []string{"--description", "flag-desc"},
			wantName: "from-file",
			wantDesc: "flag-desc",
		},
		{
			name:     "both overrides",
			flags:    []string{"--name", "n2", "--description", "d2"},
			wantName: "n2",
			wantDesc: "d2",
		},
		{
			name:     "no overrides preserves file",
			flags:    nil,
			wantName: "from-file",
			wantDesc: "from-file",
		},
	}

	cmdFactories := []struct {
		cmdName string
		newCmd  func() *cobra.Command
	}{
		{"create", newFilterCreateCmd},
		{"update", newFilterUpdateCmd},
	}

	for _, factory := range cmdFactories {
		for _, testCase := range overrideCases {
			t.Run(factory.cmdName+"/"+testCase.name, func(t *testing.T) {
				t.Parallel()

				cmd := factory.newCmd()
				require.NoError(t, cmd.ParseFlags(testCase.flags))

				desc := "from-file"
				request := types.UpdateUserFilterRQ{
					Name:        "from-file",
					Description: &desc,
					Type:        types.UpdateUserFilterRQTypeLaunch,
				}

				applyFilterRequestFlagOverrides(cmd, &request)

				assert.Equal(t, testCase.wantName, request.Name)
				assert.Equal(t, testCase.wantDesc, formatStringPtr(request.Description))
			})
		}
	}
}
