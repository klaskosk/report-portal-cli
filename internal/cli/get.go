package cli

import "github.com/spf13/cobra"

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}

	cmd.AddCommand(newGetLaunchCmd())

	return cmd
}
