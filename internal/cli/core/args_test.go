package core

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOptionalInt64Flag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		flagName  string
		value     string
		wantVal   int64
		wantFound bool
		wantErr   string
	}{
		{
			name:      "empty value returns not found",
			flagName:  "launch",
			value:     "",
			wantVal:   0,
			wantFound: false,
		},
		{
			name:      "valid positive ID",
			flagName:  "launch",
			value:     "101",
			wantVal:   101,
			wantFound: true,
		},
		{
			name:      "valid negative ID",
			flagName:  "parent",
			value:     "-1",
			wantVal:   -1,
			wantFound: true,
		},
		{
			name:      "max int32 is accepted",
			flagName:  "launch",
			value:     "2147483647",
			wantVal:   math.MaxInt32,
			wantFound: true,
		},
		{
			name:      "min int32 is accepted",
			flagName:  "launch",
			value:     "-2147483648",
			wantVal:   math.MinInt32,
			wantFound: true,
		},
		{
			name:     "non-numeric value",
			flagName: "launch",
			value:    "abc",
			wantErr:  `invalid --launch value "abc": must be a numeric ID`,
		},
		{
			name:     "value exceeds int32 max",
			flagName: "launch",
			value:    "2147483648",
			wantErr:  `invalid --launch value "2147483648": ID must fit in a signed 32-bit integer`,
		},
		{
			name:     "value below int32 min",
			flagName: "parent",
			value:    "-2147483649",
			wantErr:  `invalid --parent value "-2147483649": ID must fit in a signed 32-bit integer`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, found, err := ParseOptionalInt64Flag(test.flagName, test.value)

			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantVal, got)
			assert.Equal(t, test.wantFound, found)
		})
	}
}
