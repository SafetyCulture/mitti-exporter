package events_test

import (
	"fmt"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/events"
	"github.com/stretchr/testify/assert"
)

func TestBuildEventError_INFO(t *testing.T) {
	infoNonFatal := events.NewEventError(
		fmt.Errorf("some error"),
		events.ErrorSeverityInfo,
		events.ErrorSubSystemDB,
		false).(*events.EventError)

	assert.True(t, infoNonFatal.IsInfo())
	assert.False(t, infoNonFatal.IsWarn())
	assert.False(t, infoNonFatal.IsError())
	assert.False(t, infoNonFatal.IsFatal())
	assert.EqualValues(t, "some error", infoNonFatal.Error())
}

func TestBuildEventError_WARNING(t *testing.T) {
	warnNonFatal := events.NewEventError(
		fmt.Errorf("some error"),
		events.ErrorSeverityWarning,
		events.ErrorSubSystemDB,
		false).(*events.EventError)

	assert.True(t, warnNonFatal.IsWarn())
	assert.False(t, warnNonFatal.IsInfo())
	assert.False(t, warnNonFatal.IsFatal())
	assert.EqualValues(t, "some error", warnNonFatal.Error())
}

func TestBuildEventError_ERROR(t *testing.T) {
	errorFatal := events.NewEventError(
		fmt.Errorf("some error"),
		events.ErrorSeverityError,
		events.ErrorSubSystemDB,
		true).(*events.EventError)
	assert.True(t, errorFatal.IsError())
	assert.True(t, errorFatal.IsFatal())
	assert.False(t, errorFatal.IsInfo())
	assert.False(t, errorFatal.IsWarn())
	assert.EqualValues(t, "some error", errorFatal.Error())
}
