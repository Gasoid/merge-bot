package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Gasoid/merge-bot/webhook"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// setupTestProviders registers test providers and returns a cleanup function
func setupTestProviders() func() {
	webhook.Register("test", newTestProvider)
	webhook.Register("failing", newFailingTestProvider)
	webhook.Register("concurrent", newTestProvider)
	webhook.Register("methodtest", newTestProvider)
	webhook.Register("emptybody", newTestProvider)

	// Return cleanup function (in a real scenario, you'd want to unregister)
	return func() {
		// No-op for now since webhook package doesn't provide unregister
	}
}

type testWebhookProvider struct {
	id        int
	projectID int
	cmd       string
	secret    string
	err       error
}

func (p *testWebhookProvider) GetCmd() string {
	return p.cmd
}

func (p *testWebhookProvider) GetID() int {
	return p.id
}

func (p *testWebhookProvider) GetProjectID() int {
	return p.projectID
}

func (p *testWebhookProvider) GetSecret() string {
	return p.secret
}

func (p *testWebhookProvider) ParseRequest(request *http.Request) error {
	return p.err
}

func newTestProvider() webhook.Provider {
	return &testWebhookProvider{
		id:        456,
		projectID: 123,
		cmd:       "test-cmd",
		secret:    "test-secret",
	}
}

func newFailingTestProvider() webhook.Provider {
	return &testWebhookProvider{
		err: webhook.PayloadError,
	}
}

func TestHandler(t *testing.T) {
	cleanup := setupTestProviders()
	defer cleanup()

	tests := []struct {
		name           string
		provider       string
		body           string
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "valid provider",
			provider:       "test",
			body:           `{"test": "data"}`,
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name:           "invalid provider",
			provider:       "nonexistent",
			body:           `{"test": "data"}`,
			expectedStatus: 0, // Error case
			expectedError:  true,
		},
		{
			name:           "failing provider parse",
			provider:       "failing",
			body:           `invalid json`,
			expectedStatus: 0, // Error case
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/mergebot/webhook/"+tt.provider+"/", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("provider")
			c.SetParamValues(tt.provider)

			err := Handler(c)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHandlerConcurrency(t *testing.T) {
	cleanup := setupTestProviders()
	defer cleanup()

	e := echo.New()
	const numRequests = 5

	// Use a channel to synchronize goroutines
	done := make(chan error, numRequests)

	// Test multiple concurrent requests
	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/mergebot/webhook/concurrent/", strings.NewReader(`{}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("provider")
			c.SetParamValues("concurrent")

			err := Handler(c)
			done <- err
		}()
	}

	// Wait for all goroutines to complete and check results
	for i := 0; i < numRequests; i++ {
		err := <-done
		assert.NoError(t, err)
	}
}

func TestHandlerWithDifferentMethods(t *testing.T) {
	cleanup := setupTestProviders()
	defer cleanup()

	tests := []struct {
		method         string
		expectedStatus int
	}{
		{http.MethodPost, http.StatusCreated},
		{http.MethodGet, http.StatusCreated},   // Handler doesn't validate method
		{http.MethodPut, http.StatusCreated},   // Handler doesn't validate method
		{http.MethodPatch, http.StatusCreated}, // Handler doesn't validate method
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tt.method, "/mergebot/webhook/methodtest/", strings.NewReader(`{}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("provider")
			c.SetParamValues("methodtest")

			err := Handler(c)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestHandlerWithEmptyBody(t *testing.T) {
	cleanup := setupTestProviders()
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/mergebot/webhook/emptybody/", bytes.NewReader([]byte{}))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("provider")
	c.SetParamValues("emptybody")

	err := Handler(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestHandlerWithNilRequest(t *testing.T) {
	cleanup := setupTestProviders()
	defer cleanup()

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(nil, rec)
	c.SetParamNames("provider")
	c.SetParamValues("test")

	err := Handler(c)
	// Should return an error since request is nil
	assert.Error(t, err)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "/healthy", HealthyEndpoint)
}
