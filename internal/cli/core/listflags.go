package core

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ListFlags holds the resolved values of standard list/pagination flags.
type ListFlags struct {
	PageSize int32
	Page     int32
	Sort     []string
}

// ListFlagDefaults allows commands to override the default sort order.
type ListFlagDefaults struct {
	Sort []string
}

// RegisterListFlags adds --page, --page-size, and --sort flags to cmd
// with the standard default sort of startTime,desc.
func RegisterListFlags(cmd *cobra.Command) {
	RegisterListFlagsWithDefaults(cmd, ListFlagDefaults{Sort: []string{"startTime,desc"}})
}

// RegisterListFlagsWithDefaults adds --page, --page-size, and --sort flags to cmd
// with configurable defaults.
func RegisterListFlagsWithDefaults(cmd *cobra.Command, defaults ListFlagDefaults) {
	cmd.Flags().Int32("page-size", 10, "number of results per page")
	cmd.Flags().Int32("page", 1, "page number (1-based, must be >= 1)")
	cmd.Flags().StringArray(
		"sort",
		defaults.Sort,
		`sort criteria as field,direction (e.g. "startTime,desc"); default varies by command`,
	)
}

// ResolveListFlags reads the standard list flags from cmd and returns them.
// The page number is converted from 1-based (user-facing) to 0-based (API).
func ResolveListFlags(cmd *cobra.Command) (ListFlags, error) {
	pageSize, _ := cmd.Flags().GetInt32("page-size")
	page, _ := cmd.Flags().GetInt32("page")
	sort, _ := cmd.Flags().GetStringArray("sort")

	if page < 1 {
		return ListFlags{}, fmt.Errorf("page must be at least 1, got %d", page)
	}

	return ListFlags{PageSize: pageSize, Page: page - 1, Sort: sort}, nil
}
