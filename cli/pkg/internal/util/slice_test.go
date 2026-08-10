package util_test

import (
	"fmt"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitSliceInBatch(t *testing.T) {
	tests := map[string]struct {
		size        int
		collection  []string
		fn          func([]string) error
		expectedErr error
	}{
		"batch size is 0": {
			size:        0,
			expectedErr: fmt.Errorf("batch size cannot be 0"),
		},
		"batch size is greater than the collection size": {
			size:       100,
			collection: []string{"a", "b", "c", "d", "e", "f", "g"},
			fn: func(strings []string) error {
				require.True(t, len(strings) == 7)
				return nil
			},
		},
		"when not divisible": {
			size:       3,
			collection: []string{"a", "b", "c", "d", "e", "f", "g"},
			fn: func(strings []string) error {
				require.True(t, len(strings) == 3 || len(strings) == 1)
				return nil
			},
		},
		"when is divisible": {
			size:       3,
			collection: []string{"a", "b", "c", "d", "e", "f"},
			fn: func(strings []string) error {
				require.True(t, len(strings) == 3)
				return nil
			},
		},
		"when errors out": {
			size:       3,
			collection: []string{"a", "b", "c", "d", "e", "f"},
			fn: func(strings []string) error {
				if strings[0] == "d" {
					return fmt.Errorf("error in processing function")
				}
				return nil
			},
			expectedErr: fmt.Errorf("error in processing function"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := util.SplitSliceInBatch(tt.size, tt.collection, tt.fn)
			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}
