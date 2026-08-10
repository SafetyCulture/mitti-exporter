package version_test

import (
	"testing"

	"github.com/SafetyCulture/mitti-exporter-ui/internal/version"
	"github.com/stretchr/testify/assert"
)

func TestShouldUpdate(t *testing.T) {
	type test struct {
		name         string
		current      string
		new          string
		shouldUpdate bool
	}

	tests := []test{
		{
			name:         "Dev",
			current:      "v0.0.0-dev",
			new:          "v1.1.3",
			shouldUpdate: true,
		},
		{
			name:         "Same Major One Minor",
			current:      "v1.0.0",
			new:          "v1.1.3",
			shouldUpdate: false,
		},
		{
			name:         "Same Major Two Minor PRE",
			current:      "v1.0.0",
			new:          "v1.2.3-alpha.12",
			shouldUpdate: false,
		},
		{
			name:         "Prerelease Current",
			current:      "v1.0.0-alpha.1",
			new:          "v1.2.3-alpha.12",
			shouldUpdate: false,
		},
		{
			name:         "Prerelease Current #2",
			current:      "v1.0.0-alpha.1",
			new:          "v1.0.0",
			shouldUpdate: true,
		},
		{
			name:         "Prerelease Current #3",
			current:      "v1.0.0-alpha.1",
			new:          "v1.0.1",
			shouldUpdate: true,
		},
		{
			name:         "Prerelease Current #4",
			current:      "v2.2.0-alpha.1",
			new:          "v2.0.0",
			shouldUpdate: false,
		},
		{
			name:         "Prerelease Current #5",
			current:      "v2.2.1-alpha.1",
			new:          "v2.2.0",
			shouldUpdate: false,
		},
		{
			name:         "Prerelease Current #6",
			current:      "v1.10.0",
			new:          "v1.10.1-alpha.1",
			shouldUpdate: false,
		},

		{
			name:         "Same Major Two Minor",
			current:      "v1.0.0",
			new:          "v1.2.3",
			shouldUpdate: true,
		},
		{
			name:         "Bigger Major Same Minor",
			current:      "v1.0.0",
			new:          "v2.0.3",
			shouldUpdate: true,
		},
		{
			name:         "Smaller Major Same Minor (unlikely)",
			current:      "v3.0.0",
			new:          "v1.0.3",
			shouldUpdate: false,
		},
		{
			name:         "Same Major Smaller Minor (unlikely)",
			current:      "v3.9.0",
			new:          "v3.8.3",
			shouldUpdate: false,
		},
		{
			name:         "0 Major",
			current:      "v0.0.1",
			new:          "v0.2.3",
			shouldUpdate: true,
		},
		{
			name:         "Bad format #1",
			current:      "v.1.0.0",
			new:          "v1.2.3",
			shouldUpdate: true,
		},
		{
			name:         "Bad format #2",
			current:      "v1.0.0",
			new:          "v.1.2.3",
			shouldUpdate: false,
		},
		{
			name:         "Bad format #3",
			current:      "v.1.0.0",
			new:          "v.1.2.3",
			shouldUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := version.ShouldUpdate(tt.current, tt.new)
			assert.EqualValues(t, tt.shouldUpdate, res)
		})
	}
}
