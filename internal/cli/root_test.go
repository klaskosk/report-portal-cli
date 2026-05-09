package cli //nolint:testpackage // white-box tests for unexported root command helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_JSONFlagOverridesOutput(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"--json"}))
	require.NoError(t, cmd.PersistentPreRunE(cmd, nil))

	output, _ := cmd.PersistentFlags().GetString("output")
	assert.Equal(t, "json", output)
}

func TestRootCmd_JSONFlagWinsOverTableOutput(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.ParseFlags([]string{"-o", "table", "--json"}))
	require.NoError(t, cmd.PersistentPreRunE(cmd, nil))

	output, _ := cmd.PersistentFlags().GetString("output")
	assert.Equal(t, "json", output, "--json should override -o table")
}

func TestRootCmd_JSONFlagFalseDoesNotOverrideOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantFmt string
	}{
		{"json=false with table keeps table", []string{"-o", "table", "--json=false"}, "table"},
		{"json=false alone keeps default table", []string{"--json=false"}, "table"},
		{"no json flag keeps default", []string{}, "table"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cmd := newRootCmd()
			cmd.SetContext(context.Background())
			require.NoError(t, cmd.ParseFlags(test.args))
			require.NoError(t, cmd.PersistentPreRunE(cmd, nil))

			output, _ := cmd.PersistentFlags().GetString("output")
			assert.Equal(t, test.wantFmt, output)
		})
	}
}

func TestRootCmd_JSONFlagDefaultsToFalse(t *testing.T) {
	t.Parallel()

	cmd := newRootCmd()

	useJSON, err := cmd.PersistentFlags().GetBool("json")
	require.NoError(t, err)
	assert.False(t, useJSON)
}

func TestExecute_JSONFlagProducesJSONErrorBeforePersistentPreRunE(t *testing.T) {
	t.Parallel()

	// "launch get" requires exactly one positional arg. Cobra validates args before
	// PersistentPreRunE runs, so --json never gets a chance to rewrite the output flag
	// through the normal path. execute() must apply effectiveOutputFormat in its own
	// fallback to guarantee consistent JSON errors for machine consumers.
	var errBuf bytes.Buffer

	err := execute([]string{"launch", "get", "--json"}, &errBuf)
	require.Error(t, err)

	var result map[string]string
	require.NoError(
		t,
		json.Unmarshal(errBuf.Bytes(), &result),
		"error output should be JSON when --json is set, got: %s",
		errBuf.String(),
	)
	assert.NotEmpty(t, result["error"])
}

func TestNewLaunchGetCmd_RequiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	cmd := newLaunchGetCmd()

	require.Error(t, cmd.Args(cmd, []string{}), "should reject zero args")
	require.NoError(t, cmd.Args(cmd, []string{"42"}), "should accept exactly one arg")
	require.Error(t, cmd.Args(cmd, []string{"42", "43"}), "should reject two args")
}

func TestNewLaunchListCmd_RejectsPositionalArgs(t *testing.T) {
	t.Parallel()

	cmd := newLaunchListCmd()

	require.Error(t, cmd.Args(cmd, []string{"unexpected"}), "list should reject positional args")
}
