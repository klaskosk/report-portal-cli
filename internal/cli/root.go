package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// Execute runs the root command and returns any error.
func Execute() error {
	err := newRootCmd().Execute()
	if err != nil {
		return fmt.Errorf("executing command: %w", err)
	}

	return nil
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "rp",
		Short:         "Report Portal CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			verbosity, _ := cmd.Flags().GetInt("verbosity")
			logger := newLogger(verbosity)
			cmd.SetContext(context.WithValue(cmd.Context(), loggerKey{}, logger))

			return initConfig(cmd)
		},
	}

	flags := cmd.PersistentFlags()
	flags.String("config", "", "config file (default: $XDG_CONFIG_HOME/rp/config.yaml)")
	flags.String("profile", "", "config profile to use (overrides default-profile)")
	flags.String("url", "", "Report Portal base URL (overrides profile)")
	flags.String("token", "", "API bearer token (overrides profile)")
	flags.Bool("insecure-skip-verify", false, "skip TLS certificate verification")
	flags.StringP("output", "o", "table", "output format: table, json, go-template=<tmpl>")
	flags.IntP("verbosity", "v", 0, "log verbosity level (0=off, 1=info, 2=debug, 3+=trace)")

	cmd.AddCommand(newGetCmd())

	return cmd
}
