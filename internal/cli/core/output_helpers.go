package core

import (
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// MessageHeaders returns the table column headers for mutation results that only contain a message.
func MessageHeaders() []string {
	return []string{"MESSAGE"}
}

// MessageRows formats a message string as a single table row.
func MessageRows(message string) [][]string {
	return [][]string{{message}}
}

// FormatOptional returns a string representation of the pointed-to value, or empty if nil.
func FormatOptional[T any](value *T) string {
	if value == nil {
		return ""
	}

	return fmt.Sprintf("%v", *value)
}

// OperationResultHeaders returns the table column headers for mutation results that only contain a message.
func OperationResultHeaders() []string {
	return MessageHeaders()
}

// OperationResultRows formats an operation-completion response as a single table row.
func OperationResultRows(result *types.OperationCompletionRS) [][]string {
	return MessageRows(FormatOptional(result.Message))
}

// OperationResultsRows formats multiple operation-completion responses as table rows.
func OperationResultsRows(results []types.OperationCompletionRS) [][]string {
	rows := make([][]string, 0, len(results))
	for i := range results {
		rows = append(rows, []string{FormatOptional(results[i].Message)})
	}

	return rows
}

// SuccessfulUpdateRows formats a SuccessfulUpdate as a single table row.
func SuccessfulUpdateRows(result *types.SuccessfulUpdate) [][]string {
	if result == nil {
		return MessageRows("")
	}

	return MessageRows(FormatOptional(result.Message))
}

// MustGetStringFlag returns the string value for name, ignoring impossible Cobra access errors.
func MustGetStringFlag(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)

	return value
}

// OptionalBoolFlag returns a pointer to the explicit bool flag value, or nil when the flag was omitted.
func OptionalBoolFlag(cmd *cobra.Command, name string) *bool {
	if !cmd.Flags().Changed(name) {
		return nil
	}

	value, _ := cmd.Flags().GetBool(name)

	return &value
}

// OptionalStringFlag returns a pointer to the explicit string flag value, or nil when the flag was omitted.
func OptionalStringFlag(cmd *cobra.Command, name string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}

	value, _ := cmd.Flags().GetString(name)

	return &value
}

// FormatDeleteBulkResult formats a DeleteBulkRS response as table headers and rows.
func FormatDeleteBulkResult(
	result *types.DeleteBulkRS,
	resourceName string,
) ([]string, [][]string) {
	headers := []string{"STATUS", "MESSAGE"}

	if result.SuccessfullyDeleted != nil && len(*result.SuccessfullyDeleted) > 0 {
		return headers, [][]string{{"Deleted", resourceName + " deleted successfully"}}
	}

	if result.Errors != nil && len(*result.Errors) > 0 {
		return headers, [][]string{{"Error", (*result.Errors)[0].Message}}
	}

	if result.NotFound != nil && len(*result.NotFound) > 0 {
		return headers, [][]string{
			{"Not Found", fmt.Sprintf("ID %d not found", (*result.NotFound)[0])},
		}
	}

	return headers, [][]string{{"Unknown", "No status information in response"}}
}

// DeleteSuccessMessage is the JSON shape for void-response delete operations.
type DeleteSuccessMessage struct {
	Message string `json:"message"`
}

// DeleteSuccessResult returns a JSON-serializable result and table output for a successful delete.
func DeleteSuccessResult(msg string) (DeleteSuccessMessage, []string, [][]string) {
	return DeleteSuccessMessage{Message: msg}, OperationResultHeaders(), MessageRows(msg)
}
