package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listFlags holds the resolved values of standard list/pagination flags.
type listFlags struct {
	pageSize int32
	page     int32
	sort     []string
}

// listFlagDefaults allows commands to override the default sort order.
type listFlagDefaults struct {
	sort []string
}

// registerListFlags adds --page, --page-size, and --sort flags to cmd
// with the standard default sort of startTime,desc.
func registerListFlags(cmd *cobra.Command) {
	registerListFlagsWithDefaults(cmd, listFlagDefaults{sort: []string{"startTime,desc"}})
}

// registerListFlagsWithDefaults adds --page, --page-size, and --sort flags to cmd
// with configurable defaults.
func registerListFlagsWithDefaults(cmd *cobra.Command, defaults listFlagDefaults) {
	cmd.Flags().Int32("page-size", 10, "number of results per page")
	cmd.Flags().Int32("page", 1, "page number (1-based, must be >= 1)")
	cmd.Flags().StringArray(
		"sort",
		defaults.sort,
		`sort criteria as field,direction (e.g. "startTime,desc"); default varies by command`,
	)
}

// resolveListFlags reads the standard list flags from cmd and returns them.
// The page number is converted from 1-based (user-facing) to 0-based (API).
func resolveListFlags(cmd *cobra.Command) (listFlags, error) {
	pageSize, _ := cmd.Flags().GetInt32("page-size")
	page, _ := cmd.Flags().GetInt32("page")
	sort, _ := cmd.Flags().GetStringArray("sort")

	if page < 1 {
		return listFlags{}, fmt.Errorf("page must be at least 1, got %d", page)
	}

	return listFlags{pageSize: pageSize, page: page - 1, sort: sort}, nil
}
