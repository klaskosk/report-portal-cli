package core

import "github.com/spf13/cobra"

// GroupAnnotationKey marks noun-group commands that only group subcommands.
const GroupAnnotationKey = "rp-group"

// IsGroupCommand returns true for commands that only exist to group
// subcommands and show help. These skip config initialization so that
// help is always reachable even with a broken config file.
func IsGroupCommand(cmd *cobra.Command) bool {
	_, ok := cmd.Annotations[GroupAnnotationKey]

	return ok
}

// MarkGroupCommand marks cmd as help-only and skips config initialization.
func MarkGroupCommand(cmd *cobra.Command) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}

	cmd.Annotations[GroupAnnotationKey] = "true"
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return c.Help()
	}

	return cmd
}
