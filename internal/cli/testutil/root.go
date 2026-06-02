package testutil

import (
	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/spf13/cobra"
)

// NewMinimalRootCmd returns a root command with the same global flags as rp for unit tests.
func NewMinimalRootCmd() *cobra.Command {
	return core.NewMinimalRootCmd()
}
