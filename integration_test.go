package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Gasoid/merge-bot/handlers"
	"github.com/Gasoid/merge-bot/webhook"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// Integration test for complete webhook flow
type integrationTestProvider struct {
	projectID int
	id        int
	secret    string
	cmd       string
	event     string
}

func (p *integrationTestProvider) GetCmd() string {
	return p.cmd
}

func (p *integrationTestProvider) GetID() int {
	return p.id
}

func (p *integrationTestProvider) GetProjectID() int {
	return p.projectID
}

func (p *integrationTestProvider) GetSecret() string {
	return p.secret
}

func (p *integrationTestProvider) ParseRequest(request *http.Request) error {
	return nil
}

func newIntegrationTestProvider() webhook.Provider {
	return &integrationTestProvider{
		projectID: 123,
		id:        456,
		secret:    "test-secret",
		cmd:       "!merge",
		event:     "!merge",
	}
}

func TestIntegrationWebhookFlow(t *testing.T) {
	// Register test provider
	webhook.Register("integration", newIntegrationTestProvider)

	// Save original handlers
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

	// Register a test handler
	handle("!merge", func(command *handlers.Request) error {
		return nil
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/merge-bot/webhook/integration/", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("provider")
	c.SetParamValues("integration")

	err := Handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Give some time for the goroutine to execute
	time.Sleep(200 * time.Millisecond)
}

func TestServerSetup(t *testing.T) {
	// Test that HealthyEndpoint constant is properly defined
	assert.Equal(t, "/healthy", HealthyEndpoint)

	// Test that global variables are initialized
	assert.NotNil(t, handlerFuncs)

	// Test that init function registered handlers
	handlerMu.RLock()
	defer handlerMu.RUnlock()

	// These should be registered by the init() function in commands.go
	expectedHandlers := []string{"!merge", "!check", "!update"}
	for _, handler := range expectedHandlers {
		_, exists := handlerFuncs[handler]
		assert.True(t, exists, "Handler %s should be registered by init", handler)
	}
}

func TestTLSConfiguration(t *testing.T) {
	// Test default TLS configuration
	assert.False(t, tlsEnabled, "TLS should be disabled by default")
	assert.Empty(t, tlsDomain, "TLS domain should be empty by default")
}
