package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHealthcheck(t *testing.T) {
	// Create a new Echo instance
	e := echo.New()

	// Create a request
	req := httptest.NewRequest(http.MethodGet, "/healthy", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call the healthcheck function
	err := healthcheck(c)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestHealthcheckEndpoint(t *testing.T) {
	// Create a new Echo instance and register the healthcheck endpoint
	e := echo.New()
	e.GET(HealthyEndpoint, healthcheck)

	// Create a request to the healthcheck endpoint
	req := httptest.NewRequest(http.MethodGet, HealthyEndpoint, nil)
	rec := httptest.NewRecorder()

	// Serve the request
	e.ServeHTTP(rec, req)

	// Assertions
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}
