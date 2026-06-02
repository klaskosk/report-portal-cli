package core

import "github.com/spf13/cobra"

// NewMinimalRootCmd returns a root command with the same global flags as rp for unit tests.
func NewMinimalRootCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "rp"}

	flags := cmd.PersistentFlags()
	flags.String("config", "", "config file")
	flags.String("profile", "", "config profile")
	flags.String("url", "", "Report Portal base URL")
	flags.String("token", "", "API token")
	flags.Bool("insecure-skip-verify", false, "skip TLS verification")
	flags.StringP("output", "o", "table", "output format")
	flags.IntP("verbosity", "v", 0, "verbosity")
	flags.Bool("json", false, "shorthand for -o json")

	return cmd
}
