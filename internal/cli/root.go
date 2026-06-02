package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/klaskosk/report-portal-cli/internal/cli/activity"
	"github.com/klaskosk/report-portal-cli/internal/cli/activityevent"
	"github.com/klaskosk/report-portal-cli/internal/cli/auth"
	"github.com/klaskosk/report-portal-cli/internal/cli/bts"
	"github.com/klaskosk/report-portal-cli/internal/cli/core"
	"github.com/klaskosk/report-portal-cli/internal/cli/dashboard"
	"github.com/klaskosk/report-portal-cli/internal/cli/demodata"
	"github.com/klaskosk/report-portal-cli/internal/cli/download"
	"github.com/klaskosk/report-portal-cli/internal/cli/filter"
	"github.com/klaskosk/report-portal-cli/internal/cli/group"
	"github.com/klaskosk/report-portal-cli/internal/cli/integration"
	"github.com/klaskosk/report-portal-cli/internal/cli/item"
	"github.com/klaskosk/report-portal-cli/internal/cli/launch"
	"github.com/klaskosk/report-portal-cli/internal/cli/log"
	"github.com/klaskosk/report-portal-cli/internal/cli/logtype"
	"github.com/klaskosk/report-portal-cli/internal/cli/plugin"
	"github.com/klaskosk/report-portal-cli/internal/cli/project"
	"github.com/klaskosk/report-portal-cli/internal/cli/setting"
	"github.com/klaskosk/report-portal-cli/internal/cli/user"
	"github.com/klaskosk/report-portal-cli/internal/cli/whoami"
	"github.com/klaskosk/report-portal-cli/internal/cli/widget"
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

	if !core.IsPrintedError(err) {
		core.NewPrinterWithErrOut(core.EffectiveOutputFormat(cmd.Root()), errOut).PrintError(err)
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
			if core.IsGroupCommand(cmd) {
				return nil
			}

			verbosity, _ := cmd.Flags().GetInt("verbosity")
			logger := core.NewLogger(verbosity)
			cmd.SetContext(core.WithLogger(cmd.Context(), logger))

			_ = cmd.Root().PersistentFlags().Set("output", core.EffectiveOutputFormat(cmd.Root()))

			return core.InitConfig(cmd)
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

	registerNounCommands(cmd)

	return cmd
}

func registerNounCommands(root *cobra.Command) {
	root.AddCommand(user.NewCommand())
	root.AddCommand(auth.NewCommand())
	root.AddCommand(filter.NewCommand())
	root.AddCommand(group.NewCommand())
	root.AddCommand(integration.NewCommand())
	root.AddCommand(launch.NewCommand())
	root.AddCommand(item.NewCommand())
	root.AddCommand(log.NewCommand())
	root.AddCommand(logtype.NewCommand())
	root.AddCommand(project.NewCommand())
	root.AddCommand(download.NewCommand())
	root.AddCommand(whoami.NewCommand())
	root.AddCommand(widget.NewCommand())
	root.AddCommand(plugin.NewCommand())
	root.AddCommand(dashboard.NewCommand())
	root.AddCommand(activityevent.NewCommand())
	root.AddCommand(activity.NewCommand())
	root.AddCommand(bts.NewCommand())
	root.AddCommand(demodata.NewCommand())
	root.AddCommand(setting.NewCommand())
}
