package cli //nolint:testpackage // serializes access to global viper in tests

import (
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// globalViperTestMu serializes tests that read or mutate the package-global viper
// singleton used by resolveConnection and initConfig.
var globalViperTestMu sync.Mutex

// runRootPersistentPreRunE runs the root command's PersistentPreRunE while holding
// globalViperTestMu so parallel tests that reset viper cannot race initConfig.
func runRootPersistentPreRunE(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	globalViperTestMu.Lock()
	defer globalViperTestMu.Unlock()

	require.NoError(t, cmd.PersistentPreRunE(cmd, nil))
}
