package main

import (
	"testing"

	"github.com/Gasoid/mergebot/handlers"
	"github.com/Gasoid/mergebot/webhook"
	"github.com/stretchr/testify/assert"
)

func TestHandle(t *testing.T) {
	// Save original state
	handlerMu.Lock()
	originalHandlers := make(map[string]func(*handlers.Request) error)
	for k, v := range handlerFuncs {
		originalHandlers[k] = v
	}
	handlerMu.Unlock()

	// Clean up after test
	defer func() {
		handlerMu.Lock()
		handlerFuncs = originalHandlers
		handlerMu.Unlock()
	}()

	testFunc := func(command *handlers.Request) error {
		return nil
	}

	handle("test-event", testFunc)

	handlerMu.RLock()
	_, exists := handlerFuncs["test-event"]
	handlerMu.RUnlock()

	assert.True(t, exists, "Handler should be registered")
}

func TestInitCommands(t *testing.T) {
	// Test that init function registers the expected handlers
	handlerMu.RLock()
	defer handlerMu.RUnlock()

	expectedHandlers := []string{
		"!merge",
		"!check",
		"!update",
		webhook.OnNewMR,
		webhook.OnMerge,
	}

	for _, handler := range expectedHandlers {
		_, exists := handlerFuncs[handler]
		assert.True(t, exists, "Handler %s should be registered", handler)
	}
}
