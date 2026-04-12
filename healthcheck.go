package main

import (
	"net/http"

	"github.com/gasoid/merge-bot/handlers"

	"github.com/labstack/echo/v4"
)

//nolint:errcheck
func healthcheck(c echo.Context) error {
	if handlers.IsHealthy() {
		c.String(http.StatusOK, "ok")
	} else {
		c.String(http.StatusServiceUnavailable, "not healthy")
	}
	return nil
}
