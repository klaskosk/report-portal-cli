package cli

import "strconv"

// messageHeaders returns the table column headers for mutation results that only contain a message.
func messageHeaders() []string {
	return []string{"MESSAGE"}
}

// messageRows formats a message string as a single table row.
func messageRows(message string) [][]string {
	return [][]string{{message}}
}

// formatMessage returns the dereferenced string or an empty string if the pointer is nil.
func formatMessage(msg *string) string {
	if msg == nil {
		return ""
	}

	return *msg
}

// formatStringPtr returns the dereferenced string or an empty string if nil.
func formatStringPtr(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

// formatInt64Ptr returns the formatted int64 or an empty string if nil.
func formatInt64Ptr(value *int64) string {
	if value == nil {
		return ""
	}

	return strconv.FormatInt(*value, 10)
}

// formatBoolPtr returns the dereferenced bool as a string or an empty string if nil.
func formatBoolPtr(value *bool) string {
	if value == nil {
		return ""
	}

	return strconv.FormatBool(*value)
}
