package launch //nolint:testpackage // white-box tests

import (
	"testing"

	"github.com/klaskosk/report-portal-cli/internal/cli/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLaunchReportCmd_Subcommands(t *testing.T) {
	t.Parallel()

	cmd := newLaunchReportCmd()

	assert.Equal(t, []string{"get"}, testutil.CommandNames(cmd))
}

func TestNewLaunchReportGetCmd_outputFileFlag(t *testing.T) {
	t.Parallel()

	cmd := newLaunchReportGetCmd()
	require.NoError(t, cmd.ParseFlags([]string{"--output-file", "report.html"}))

	outputFile, err := cmd.Flags().GetString("output-file")
	require.NoError(t, err)

	assert.Equal(t, "report.html", outputFile)
}

func TestRunLaunchReportGet_invalidID(t *testing.T) {
	t.Parallel()

	cmd := testutil.AttachCmdToRoot(t, newLaunchReportGetCmd())

	err := runLaunchReportGet(cmd, []string{"not-a-number"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid launch ID "not-a-number"`)
}
