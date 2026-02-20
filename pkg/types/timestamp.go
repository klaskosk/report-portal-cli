package types

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// UnmarshalJSON supports both legacy object timestamps and epoch millisecond
// numeric timestamps returned by some Report Portal endpoints.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*t = Timestamp{}
		return nil
	}

	var unixMillis int64
	if err := json.Unmarshal(trimmed, &unixMillis); err == nil {
		*t = Timestamp{Time: &unixMillis}
		return nil
	}

	type timestampAlias Timestamp

	var parsed timestampAlias
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return fmt.Errorf("failed to unmarshal timestamp: %w", err)
	}

	*t = Timestamp(parsed)
	return nil
}
