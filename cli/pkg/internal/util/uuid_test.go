package util_test

import (
	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConvertS12ToUUID(t *testing.T) {
	expectedUUID, err := uuid.FromString("ada3042f-16a4-4249-915d-dc088adef92a")
	require.NoError(t, err)

	tests := []struct {
		name     string
		stringID string
		expected uuid.UUID
	}{
		{
			name:     "should pass as S12",
			stringID: "role_ada3042f16a44249915ddc088adef92a",
			expected: expectedUUID,
		},
		{
			name:     "should pass as UUID",
			stringID: "ada3042f-16a4-4249-915d-dc088adef92a",
			expected: expectedUUID,
		},
		{
			name:     "should return UUID ZERO in case of error",
			stringID: "xyz",
			expected: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := util.ConvertS12ToUUID(tt.stringID)
			if res != tt.expected {
				t.Errorf("got %s, want %s", res, tt.expected)
			}
		})
	}
}
