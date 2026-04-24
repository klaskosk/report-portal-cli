package cli

import "github.com/spf13/cobra"

// newGetCmd builds the parent "get" subcommand that groups resource-listing commands.
func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}

	cmd.AddCommand(newGetLaunchCmd())
	cmd.AddCommand(newGetItemCmd())
	cmd.AddCommand(newGetLogCmd())
	cmd.AddCommand(newGetProjectCmd())
	cmd.AddCommand(newGetUserCmd())

	return cmd
}
