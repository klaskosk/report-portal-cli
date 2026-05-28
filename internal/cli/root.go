package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// Execute runs the root command with arguments from os.Args and returns any error.
func Execute() error {
	return execute(nil, os.Stderr)
}

// execute runs the root command. args are passed to cobra (nil means os.Args[1:]). Errors from
// command handlers are already printed by the handler via printer.PrintError and wrapped with
// printedError. Any other errors (Cobra-level failures, config errors) are written to errOut.
func execute(args []string, errOut io.Writer) error {
	cmd := newRootCmd()
	if args != nil {
		cmd.SetArgs(args)
	}

	err := cmd.Execute()
	if err == nil {
		return nil
	}

	var printed *printedError
	if !errors.As(err, &printed) {
		newPrinterWithErrOut(effectiveOutputFormat(cmd.Root()), errOut).PrintError(err)
	}

	return fmt.Errorf("executing command: %w", err)
}

// effectiveOutputFormat derives the output format from the root command's persistent flags,
// giving --json precedence over -o/--output.
func effectiveOutputFormat(cmd *cobra.Command) string {
	if useJSON, _ := cmd.PersistentFlags().GetBool("json"); useJSON {
		return "json"
	}

	outputFmt, _ := cmd.PersistentFlags().GetString("output")

	return outputFmt
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

			_ = cmd.Root().PersistentFlags().Set("output", effectiveOutputFormat(cmd.Root()))

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
	flags.Bool("json", false, "shorthand for -o json")

	cmd.AddCommand(newUserCmd())
	cmd.AddCommand(newAuthCmd())
	cmd.AddCommand(newFilterCmd())
	cmd.AddCommand(newGroupCmd())
	cmd.AddCommand(newIntegrationCmd())
	cmd.AddCommand(newLaunchCmd())
	cmd.AddCommand(newItemCmd())
	cmd.AddCommand(newLogCmd())
	cmd.AddCommand(newLogTypeCmd())
	cmd.AddCommand(newProjectCmd())
	cmd.AddCommand(newDownloadCmd())
	cmd.AddCommand(newWhoamiCmd())
	cmd.AddCommand(newWidgetCmd())
	cmd.AddCommand(newPluginCmd())
	cmd.AddCommand(newDashboardCmd())
	cmd.AddCommand(newActivityEventCmd())
	cmd.AddCommand(newActivityCmd())
	cmd.AddCommand(newBtsCmd())
	cmd.AddCommand(newDemoDataCmd())
	cmd.AddCommand(newSettingCmd())

	return cmd
}
