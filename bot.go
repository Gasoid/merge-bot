package main

import (
	"log/slog"
	"mergebot/webhook"
	"os"
	"path"

	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

func start() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/healthy", healthcheck)
	e.POST("/mergebot/webhook/:provider/:owner/:repo/", Handler)

	if os.Getenv("TLS_ENABLED") == "true" {
		tmpDir := path.Join(os.TempDir(), "tls", ".cache")

		e.AutoTLSManager.HostPolicy = autocert.HostWhitelist(os.Getenv("TLS_DOMAIN"))
		e.AutoTLSManager.Cache = autocert.DirCache(tmpDir)
		e.AutoTLSManager.Prompt = autocert.AcceptTOS
		e.Logger.Fatal(e.StartAutoTLS(":443"))
		return
	}

	e.Logger.Fatal(e.Start(":8080"))
}

//nolint:errcheck
func Handler(c echo.Context) error {
	c.String(http.StatusCreated, "")

	providerName := c.Param("provider")
	hook, err := webhook.New(providerName)
	if err != nil {
		slog.Error("webhook", "err", err)
		return err
	}

	if err = hook.ParseRequest(c.Request()); err != nil {
		slog.Error("ParseRequest", "err", err)
		return err
	}

	slog.Debug("handler", "event", hook.Event)

	if f, ok := handlerFuncs[hook.Event]; ok {
		go func() {
			if err := f(providerName, hook); err != nil {
				slog.Error("handlerFunc", "err", err)
			}
		}()
	}

	return nil
}

var (
	handlerFuncs = map[string]func(string, *webhook.Webhook) error{}
)

func handle(onEvent string, funcHandler func(string, *webhook.Webhook) error) {
	handlerFuncs[onEvent] = func(provider string, hook *webhook.Webhook) error {
		return funcHandler(provider, hook)
	}
}
