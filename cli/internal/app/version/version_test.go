package version_test

import (
	"testing"

	"github.com/SafetyCulture/mitti-exporter/internal/app/version"
	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	r := version.GetVersion()
	assert.EqualValues(t, "0.0.0-dev", r)
}
