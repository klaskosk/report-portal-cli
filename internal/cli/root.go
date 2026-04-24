package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Execute runs the root command and returns any error.
// Errors from command handlers are already printed by the handler via
// printer.PrintError and wrapped with printedError. Any other errors
// (Cobra-level failures, config errors) are printed here as a fallback.
func Execute() error {
	cmd := newRootCmd()

	err := cmd.Execute()
	if err == nil {
		return nil
	}

	var printed *printedError
	if !errors.As(err, &printed) {
		outputFmt, _ := cmd.PersistentFlags().GetString("output")
		newPrinter(outputFmt).PrintError(err)
	}

	return fmt.Errorf("executing command: %w", err)
}

// newRootCmd builds the root "rp" command with global flags and subcommands.
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
	flags.IntP("verbosity", "v", 0, "log verbosity level (0=off, 1=info, 2+=debug)")

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newDownloadCmd())
	cmd.AddCommand(newWhoamiCmd())

	return cmd
}
