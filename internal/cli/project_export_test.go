package cli

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectExportCmd_outputFileFlag(t *testing.T) {
	t.Parallel()

	cmd := newProjectExportCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--output-file", "projects.csv"}))

	outputFile, err := cmd.Flags().GetString("output-file")
	require.NoError(t, err)

	assert.Equal(t, "projects.csv", outputFile)
}

func TestResolveProjectExportOptions_WithFilters(t *testing.T) {
	t.Parallel()

	cmd := newProjectExportCmd()
	require.NoError(t, cmd.ParseFlags([]string{
		"--view", "csv",
		"--name", "my_project",
		"--type", "INTERNAL",
		"--last-run", "2025-06-15",
	}))

	options, err := resolveProjectExportOptions(cmd)
	require.NoError(t, err)
	require.Len(t, options, 4)

	for _, option := range options {
		assert.Implements(t, (*client.ExportProjectsOption)(nil), option)
	}
}

func TestResolveProjectExportOptions_invalidLastRun(t *testing.T) {
	t.Parallel()

	cmd := newProjectExportCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--last-run", "not-a-date"}))

	_, err := resolveProjectExportOptions(cmd)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --last-run")
}
