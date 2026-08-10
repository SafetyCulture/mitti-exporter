// Copyright (c) 2018 SafetyCulture Pty Ltd. All Rights Reserved.

package util_test

import (
	"net/http"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "Should not panic if there is no error",
			err:  nil,
			msg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			util.Check(tt.err, tt.msg)
		})
	}
}

func TestCheckFeedError_ShouldReturnIfNoError(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.ErrorLevel)
	observedLogger := zap.New(observedZapCore)

	util.CheckFeedError(observedLogger.Sugar(), nil, "some message")

	assert.Empty(t, observedLogs)
}

func TestCheckFeedError_ShouldCapture403(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.ErrorLevel)
	observedLogger := zap.New(observedZapCore)

	util.CheckFeedError(observedLogger.Sugar(), util.HTTPError{
		StatusCode: http.StatusForbidden,
		Resource:   "/test",
		Message:    "test",
	}, "some message")

	require.NotEmpty(t, observedLogs)
	assert.Equal(t, `some message: {"status_code":403,"resource":"/test","message":"test"}`, observedLogs.All()[0].Message)
}
