package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Save original values
	originalSentryEnabled := sentryEnabled
	originalDebug := debug
	defer func() {
		sentryEnabled = originalSentryEnabled
		debug = originalDebug
	}()

	tests := []struct {
		name          string
		sentryEnabled bool
		debug         bool
	}{
		{
			name:          "sentry enabled, debug disabled",
			sentryEnabled: true,
			debug:         false,
		},
		{
			name:          "sentry disabled, debug enabled",
			sentryEnabled: false,
			debug:         true,
		},
		{
			name:          "both enabled",
			sentryEnabled: true,
			debug:         true,
		},
		{
			name:          "both disabled",
			sentryEnabled: false,
			debug:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentryEnabled = tt.sentryEnabled
			debug = tt.debug

			// This should not panic
			New()
		})
	}
}

func TestError(t *testing.T) {
	// Save original value
	originalSentryEnabled := sentryEnabled
	defer func() {
		sentryEnabled = originalSentryEnabled
	}()

	tests := []struct {
		name          string
		sentryEnabled bool
		msg           string
		args          []any
	}{
		{
			name:          "error with sentry enabled",
			sentryEnabled: true,
			msg:           "test error",
			args:          []any{"key", "value"},
		},
		{
			name:          "error with sentry disabled",
			sentryEnabled: false,
			msg:           "test error",
			args:          []any{"key", "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentryEnabled = tt.sentryEnabled

			// This should not panic
			Error(tt.msg, tt.args...)
		})
	}
}

func TestDebug(t *testing.T) {
	// This should not panic
	Debug("test debug message", "key", "value")
}

func TestInfo(t *testing.T) {
	// This should not panic
	Info("test info message", "key", "value")
}

func TestIsSentryEnabled(t *testing.T) {
	// Save original value
	originalSentryEnabled := sentryEnabled
	defer func() {
		sentryEnabled = originalSentryEnabled
	}()

	// Test enabled
	sentryEnabled = true
	assert.True(t, IsSentryEnabled())

	// Test disabled
	sentryEnabled = false
	assert.False(t, IsSentryEnabled())
}
