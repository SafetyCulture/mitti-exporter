package util_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeFromString(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
		err      error
	}{
		"RFC3339": {
			input:    "2023-04-02T15:04:05.000Z",
			expected: "2023-04-02 15:04:05 +0000 UTC",
		},
		"RFC1123": {
			input:    "Mon, 02 Jan 2006 15:04:05 MST",
			expected: "2006-01-02 15:04:05 +0000 MST",
		},
		"RFC822": {
			input:    "02 Jan 06 15:04 MST",
			expected: "2006-01-02 15:04:00 +0000 MST",
		},
		"RFC850": {
			input:    "Monday, 02-Jan-06 15:04:05 MST",
			expected: "2006-01-02 15:04:05 +0000 MST",
		},
		"ANSIC": {
			input:    "Mon Jan 2 15:04:05 2006",
			expected: "2006-01-02 15:04:05 +0000 UTC",
		},
		"UNIXDATE": {
			input:    "Mon Jan 2 15:04:05 MST 2006",
			expected: "2006-01-02 15:04:05 +0000 MST",
		},
		"ISO8601": {
			input:    "2006-01-02",
			expected: "2006-01-02 00:00:00 +0000 UTC",
		},
		"CUSTOM": {
			input:    "02 Jan 2006",
			expected: "2006-01-02 00:00:00 +0000 UTC",
		},
		"INVALID": {
			input: "invalid",
			err:   fmt.Errorf(`parsing time "invalid" as "02 Jan 2006": cannot parse "invalid" as "02"`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := util.TimeFromString(test.input)
			if test.err != nil {
				require.EqualError(t, err, test.err.Error())
			} else if test.err == nil {
				assert.EqualValues(t, test.expected, result.String())
			} else {
				require.Fail(t, "unexpected error")
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		"1 day": {
			input:    "1d",
			expected: 24 * time.Hour,
		},
		"7 days": {
			input:    "7d",
			expected: 7 * 24 * time.Hour,
		},
		"1 week": {
			input:    "1w",
			expected: 7 * 24 * time.Hour,
		},
		"2 weeks": {
			input:    "2w",
			expected: 14 * 24 * time.Hour,
		},
		"1 month": {
			input:    "1m",
			expected: 30 * 24 * time.Hour,
		},
		"3 months": {
			input:    "3m",
			expected: 90 * 24 * time.Hour,
		},
		"1 year": {
			input:    "1y",
			expected: 365 * 24 * time.Hour,
		},
		"empty string": {
			input:    "",
			hasError: true,
		},
		"invalid format": {
			input:    "invalid",
			hasError: true,
		},
		"zero value": {
			input:    "0d",
			hasError: true,
		},
		"negative value": {
			input:    "-1d",
			hasError: true,
		},
		"unsupported unit": {
			input:    "1h",
			hasError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := util.ParseDuration(test.input)
			if test.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestGenerateTimeBlocks(t *testing.T) {
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)
	blockSize := 24 * time.Hour // 1 day

	blocks := util.GenerateTimeBlocks(start, end, blockSize)

	assert.Len(t, blocks, 7)
	assert.Equal(t, start, blocks[0].Start)
	assert.Equal(t, time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC), blocks[0].End)
	assert.Equal(t, time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC), blocks[6].Start)
	assert.Equal(t, end, blocks[6].End)
}

func TestGenerateTimeBlocksFromString(t *testing.T) {
	start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC)

	t.Run("valid block size", func(t *testing.T) {
		blocks, err := util.GenerateTimeBlocksFromString(start, end, "1d")
		require.NoError(t, err)
		assert.Len(t, blocks, 3)
	})

	t.Run("empty block size", func(t *testing.T) {
		blocks, err := util.GenerateTimeBlocksFromString(start, end, "")
		require.NoError(t, err)
		assert.Len(t, blocks, 0)
	})

	t.Run("invalid block size", func(t *testing.T) {
		_, err := util.GenerateTimeBlocksFromString(start, end, "invalid")
		require.Error(t, err)
	})
}
