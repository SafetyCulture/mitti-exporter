// Copyright (c) 2018 SafetyCulture Pty Ltd. All Rights Reserved.

package util_test

import (
	"context"
	"testing"

	"github.com/SafetyCulture/mitti-exporter/pkg/internal/util"
)

func TestGetRandomRequestIDFromContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			"Should return new non empty ContextKeyRequestID when encountering a nil context",
			nil,
		},
		{
			"Should return new non empty ContextKeyRequestID when an existing one is not found in context",
			context.Background(),
		},
		{
			"Should return new non empty ContextKeyRequestID when an empty one is found in context",
			context.WithValue(context.Background(), util.ContextKeyRequestID, ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := util.RequestIDFromContext(tt.ctx); got == "" {
				t.Errorf("RequestIDFromContext() = %v", got)
			}
		})
	}
}

func TestGetExistingRequestIDFromContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			"Should retrieve value from ContextKeyRequestID in context",
			context.WithValue(context.Background(), util.ContextKeyRequestID, "123"),
			"123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := util.RequestIDFromContext(tt.ctx); got != tt.want {
				t.Errorf("RequestIDFromContext() = %v, want %v", got, tt.want)
			}
		})
	}
}
