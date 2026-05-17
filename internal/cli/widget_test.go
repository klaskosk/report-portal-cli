package cli //nolint:testpackage // white-box tests for widget CLI helpers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWidgetRow_allFieldsPopulated(t *testing.T) {
	t.Parallel()

	owner := "admin"
	desc := "A short description"
	filterID := int64(1)
	filterName := "My Filter"
	widget := &types.WidgetResource{
		Id:          42,
		Name:        "My Widget",
		WidgetType:  "launchStatistics",
		Owner:       &owner,
		Description: &desc,
		AppliedFilters: &[]types.UserFilterResource{
			{Id: filterID, Name: filterName},
		},
	}

	headers, row := widgetRow(widget)

	assert.Equal(t, widgetHeaders(), headers)
	assert.Equal(t, []string{
		"42",
		"My Widget",
		"launchStatistics",
		"admin",
		"A short description",
		"1",
	}, row)
}

func TestWidgetRow_nilOptionalFields(t *testing.T) {
	t.Parallel()

	widget := &types.WidgetResource{
		Id:         1,
		Name:       "Minimal Widget",
		WidgetType: "overallStatistics",
	}

	headers, row := widgetRow(widget)

	assert.Equal(t, widgetHeaders(), headers)
	assert.Equal(t, []string{
		"1",
		"Minimal Widget",
		"overallStatistics",
		"",
		"",
		"0",
	}, row)
}

func TestWidgetRow_descriptionTruncatedAt50(t *testing.T) {
	t.Parallel()

	long := make([]byte, 60)
	for i := range long {
		long[i] = 'a'
	}

	desc := string(long)
	widget := &types.WidgetResource{
		Id:          2,
		Name:        "W",
		WidgetType:  "launchStatistics",
		Description: &desc,
	}

	_, row := widgetRow(widget)

	expected := string(long[:50]) + "..."
	assert.Equal(t, expected, row[4])
}

func TestWidgetRow_nilAppliedFilters(t *testing.T) {
	t.Parallel()

	widget := &types.WidgetResource{
		Id:             3,
		Name:           "W",
		WidgetType:     "launchStatistics",
		AppliedFilters: nil,
	}

	_, row := widgetRow(widget)

	assert.Equal(t, "0", row[5])
}

func TestWidgetDataRow_contentKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  *map[string]any
		wantKeys string
	}{
		{
			name: "non-nil content with multiple keys sorted",
			content: &map[string]any{
				"chart":   map[string]any{"x": 1},
				"summary": "text",
			},
			wantKeys: "chart, summary",
		},
		{
			name:     "nil content",
			content:  nil,
			wantKeys: "(none)",
		},
		{
			name:     "empty map",
			content:  &map[string]any{},
			wantKeys: "(none)",
		},
		{
			name: "single key",
			content: &map[string]any{
				"data": 42,
			},
			wantKeys: "data",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			widget := &types.WidgetResource{
				Id:         10,
				Name:       "W",
				WidgetType: "launchStatistics",
				Content:    testCase.content,
			}

			headers, row := widgetDataRow(widget)

			assert.Equal(t, widgetDataHeaders(), headers)
			assert.Equal(t, testCase.wantKeys, row[3])
		})
	}
}

func TestReadWidgetRequestFromFile_happyPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "widget.json")

	desc := "my widget description"
	filterID := int64(7)
	itemsCount := int32(10)
	jsonContent := `{
  "name": "Test Widget",
  "widgetType": "launchStatistics",
  "description": "my widget description",
  "filterIds": [7],
  "contentParameters": {
    "itemsCount": 10,
    "contentFields": ["statistics$defects$product_bug$total"],
    "widgetOptions": {"zoom": true}
  }
}`

	require.NoError(t, os.WriteFile(path, []byte(jsonContent), 0o600))

	request, err := readWidgetRequestFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, "Test Widget", request.Name)
	assert.Equal(t, types.WidgetRQWidgetType("launchStatistics"), request.WidgetType)
	require.NotNil(t, request.Description)
	assert.Equal(t, desc, *request.Description)
	require.NotNil(t, request.FilterIds)
	assert.Equal(t, []int64{filterID}, *request.FilterIds)
	require.NotNil(t, request.ContentParameters)
	require.NotNil(t, request.ContentParameters.ItemsCount)
	assert.Equal(t, itemsCount, *request.ContentParameters.ItemsCount)
	require.NotNil(t, request.ContentParameters.ContentFields)
	assert.Equal(
		t,
		[]string{"statistics$defects$product_bug$total"},
		*request.ContentParameters.ContentFields,
	)
	require.NotNil(t, request.ContentParameters.WidgetOptions)
	assert.Equal(t, true, (*request.ContentParameters.WidgetOptions)["zoom"])
}

func TestReadWidgetRequestFromFile_ignoresUnknownFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "widget.json")

	jsonContent := `{
  "id": 99,
  "owner": "someone",
  "content": {"key": "value"},
  "appliedFilters": [],
  "name": "Round-trip Widget",
  "widgetType": "overallStatistics"
}`

	require.NoError(t, os.WriteFile(path, []byte(jsonContent), 0o600))

	request, err := readWidgetRequestFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, "Round-trip Widget", request.Name)
	assert.Equal(t, types.WidgetRQWidgetType("overallStatistics"), request.WidgetType)
}

func TestReadWidgetRequestFromFile_invalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte(`{`), 0o600))

	_, err := readWidgetRequestFromFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing widget JSON")
}

func TestReadWidgetRequestFromFile_missingFile(t *testing.T) {
	t.Parallel()

	_, err := readWidgetRequestFromFile(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading widget definition")
}

func TestReadWidgetRequestFromFile_stdin(t *testing.T) { //nolint:paralleltest // swaps os.Stdin
	jsonContent := `{"name":"stdin-widget","widgetType":"launchStatistics"}`

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

	request, err := readWidgetRequestFromFile("-")
	require.NoError(t, err)
	assert.Equal(t, "stdin-widget", request.Name)
	assert.Equal(t, types.WidgetRQWidgetType("launchStatistics"), request.WidgetType)
}

func TestRunWidgetCreate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newWidgetCreateCmd())

	err := runWidgetCreate(cmd, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestRunWidgetUpdate_requiresFromFileBeforeConnection(t *testing.T) {
	t.Setenv("RP_URL", "")
	t.Setenv("RP_TOKEN", "")
	t.Setenv("RP_PROJECT", "")
	t.Setenv("RP_INSECURE_SKIP_VERIFY", "")

	cmd := attachCmdToRoot(t, newWidgetUpdateCmd())

	err := runWidgetUpdate(cmd, []string{"42"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-file is required")
	assert.NotContains(t, err.Error(), "no Report Portal URL configured")
}

func TestApplyWidgetRequestFlagOverrides(t *testing.T) {
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
			name:     "no overrides preserves file values",
			flags:    nil,
			wantName: "from-file",
			wantDesc: "from-file",
		},
	}

	cmdFactories := []struct {
		cmdName string
		newCmd  func() *cobra.Command
	}{
		{"create", newWidgetCreateCmd},
		{"update", newWidgetUpdateCmd},
	}

	for _, factory := range cmdFactories {
		for _, testCase := range overrideCases {
			t.Run(factory.cmdName+"/"+testCase.name, func(t *testing.T) {
				t.Parallel()

				cmd := factory.newCmd()
				require.NoError(t, cmd.ParseFlags(testCase.flags))

				desc := "from-file"
				request := types.WidgetRQ{
					Name:        "from-file",
					Description: &desc,
					WidgetType:  "launchStatistics",
				}

				applyWidgetRequestFlagOverrides(cmd, &request)

				assert.Equal(t, testCase.wantName, request.Name)
				assert.Equal(t, testCase.wantDesc, formatStringPtr(request.Description))
			})
		}
	}
}

func TestParseWidgetID_valid(t *testing.T) {
	t.Parallel()

	id, err := parseWidgetID("42")
	require.NoError(t, err)
	assert.EqualValues(t, 42, id)
}

func TestParseWidgetID_invalid(t *testing.T) {
	t.Parallel()

	_, err := parseWidgetID("not-a-number")
	require.Error(t, err)
	assert.EqualError(t, err, `invalid widget ID "not-a-number": must be a numeric ID`)
}

func TestNewWidgetCmd_CommandTree(t *testing.T) {
	t.Parallel()

	cmd := newWidgetCmd()

	assert.Equal(t, []string{"create", "data", "get", "update"}, commandNames(cmd))

	var dataCmd *cobra.Command

	for _, sub := range cmd.Commands() {
		if sub.Name() == "data" {
			dataCmd = sub

			break
		}
	}

	require.NotNil(t, dataCmd, "data subcommand must exist")
	assert.Equal(t, []string{"get"}, commandNames(dataCmd))
}

func TestNewWidgetCmd_ProjectFlagInheritedBySubcommands(t *testing.T) {
	t.Parallel()

	cmd := newWidgetCmd()

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

func TestNewWidgetDataGetCmd_ProjectFlagInherited(t *testing.T) {
	t.Parallel()

	widgetCmd := newWidgetCmd()

	var dataCmd *cobra.Command

	for _, sub := range widgetCmd.Commands() {
		if sub.Name() == "data" {
			dataCmd = sub

			break
		}
	}

	require.NotNil(t, dataCmd)

	for _, sub := range dataCmd.Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			t.Parallel()

			flag := sub.InheritedFlags().Lookup("project")
			require.NotNil(
				t,
				flag,
				"data subcommand %q should inherit --project persistent flag",
				sub.Name(),
			)
			assert.Equal(t, "p", flag.Shorthand)
		})
	}
}
