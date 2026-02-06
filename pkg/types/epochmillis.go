package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// maxMillisTimestamp is the upper bound for a value to be interpreted as milliseconds (corresponds to approximately
// year 2286). Values at or above this threshold are treated as microseconds. This cleanly separates the two because
// real-world millisecond timestamps top out around 10^12–10^13, while microsecond timestamps for modern dates start at
// ~10^14–10^15.
const maxMillisTimestamp int64 = 1e13

// EpochMillis represents a point in time serialized as an integer epoch timestamp. On deserialization it auto-detects
// whether the value is milliseconds or microseconds based on its magnitude. Serialization always produces milliseconds
// for backward compatibility.
type EpochMillis time.Time

// MarshalJSON serializes EpochMillis as Unix milliseconds timestamp.
func (e EpochMillis) MarshalJSON() ([]byte, error) {
	bytes, err := json.Marshal(time.Time(e).UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal epoch millis: %w", err)
	}

	return bytes, nil
}

// UnmarshalJSON deserializes an integer timestamp, auto-detecting milliseconds or microseconds.
func (e *EpochMillis) UnmarshalJSON(data []byte) error {
	var value int64

	err := json.Unmarshal(data, &value)
	if err != nil {
		return fmt.Errorf("failed to unmarshal epoch millis: %w", err)
	}

	if value >= maxMillisTimestamp {
		*e = EpochMillis(time.UnixMicro(value))
	} else {
		*e = EpochMillis(time.UnixMilli(value))
	}

	return nil
}
