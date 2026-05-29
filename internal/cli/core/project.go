package core

import "github.com/spf13/cobra"

// RegisterProjectFlag adds the standard -p / --project flag to cmd.
func RegisterProjectFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP(
		"project", "p", "", "project name (overrides profile default)",
	)
}
