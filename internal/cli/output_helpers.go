package cli

import (
	"fmt"

	"github.com/klaskosk/report-portal-cli/pkg/types"
	"github.com/spf13/cobra"
)

// messageHeaders returns the table column headers for mutation results that only contain a message.
func messageHeaders() []string {
	return []string{"MESSAGE"}
}

// messageRows formats a message string as a single table row.
func messageRows(message string) [][]string {
	return [][]string{{message}}
}

// formatOptional returns a string representation of the pointed-to value, or empty if nil.
func formatOptional[T any](value *T) string {
	if value == nil {
		return ""
	}

	return fmt.Sprintf("%v", *value)
}

// operationResultHeaders returns the table column headers for mutation results that only contain a message.
func operationResultHeaders() []string {
	return messageHeaders()
}

// operationResultRows formats an operation-completion response as a single table row.
func operationResultRows(result *types.OperationCompletionRS) [][]string {
	return messageRows(formatOptional(result.Message))
}

// operationResultsRows formats multiple operation-completion responses as table rows.
func operationResultsRows(results []types.OperationCompletionRS) [][]string {
	rows := make([][]string, 0, len(results))
	for i := range results {
		rows = append(rows, []string{formatOptional(results[i].Message)})
	}

	return rows
}

// successfulUpdateRows formats a SuccessfulUpdate as a single table row.
func successfulUpdateRows(result *types.SuccessfulUpdate) [][]string {
	if result == nil {
		return messageRows("")
	}

	return messageRows(formatOptional(result.Message))
}

// mustGetStringFlag returns the string value for name, ignoring impossible Cobra access errors.
func mustGetStringFlag(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)

	return value
}

// optionalStringFlag returns a pointer to the explicit string flag value, or nil when the flag was omitted.
func optionalStringFlag(cmd *cobra.Command, name string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}

	value, _ := cmd.Flags().GetString(name)

	return &value
}

// formatDeleteBulkResult formats a DeleteBulkRS response as table headers and rows.
func formatDeleteBulkResult(
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

// deleteSuccessMessage is the JSON shape for void-response delete operations.
type deleteSuccessMessage struct {
	Message string `json:"message"`
}

// deleteSuccessResult returns a JSON-serializable result and table output for a successful delete.
func deleteSuccessResult(msg string) (deleteSuccessMessage, []string, [][]string) {
	return deleteSuccessMessage{Message: msg}, operationResultHeaders(), messageRows(msg)
}
