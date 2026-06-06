package main

import (
	"errors"
	"net/http"

	"github.com/gasoid/merge-bot/v3/handlers"
	"github.com/gasoid/merge-bot/v3/logger"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

func startMetricsEndpoint() {
	go func() {
		metrics := echo.New()
		metrics.HideBanner = true
		metrics.HidePort = true
		metrics.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			Skipper: func(c echo.Context) bool {
				return true
			},
			LogURI: true,
			LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
				logger.Info("request",
					"uri", values.URI,
				)
				return nil
			},
		}))
		metrics.GET("/metrics", echoprometheus.NewHandler())
		metrics.GET("/healthy", healthcheck)
		if err := metrics.Start(":8081"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(err.Error())
		}
	}()
}
