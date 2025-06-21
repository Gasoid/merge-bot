package main

import (
	"os"
	"path"
	"sync"

	"github.com/Gasoid/mergebot/config"
	"github.com/Gasoid/mergebot/handlers"
	"github.com/Gasoid/mergebot/logger"
	"github.com/Gasoid/mergebot/webhook"

	"net/http"

	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"
)

var (
	tlsEnabled bool
	tlsDomain  string
)

const (
	HealthyEndpoint = "/healthy"
)

func init() {
	config.BoolVar(&tlsEnabled, "tls-enabled", false, "whether tls enabled or not, bot will use Letsencrypt (also via TLS_ENABLED)")
	config.StringVar(&tlsDomain, "tls-domain", "", "which domain is used for ssl certificate (also via TLS_DOMAIN)")
}

func start() {
	e := echo.New()

	// Custom request logger middleware that skips /healthy endpoint
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == HealthyEndpoint
		},
		LogURI:    true,
		LogStatus: true,
		LogMethod: true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			logger.Info("request",
				"method", values.Method,
				"uri", values.URI,
				"status", values.Status,
			)
			return nil
		},
	}))
	e.Use(middleware.Recover())

	if logger.IsSentryEnabled() {
		e.Use(sentryecho.New(sentryecho.Options{Repanic: true, WaitForDelivery: false}))
	}

	e.GET(HealthyEndpoint, healthcheck)
	e.POST("/mergebot/webhook/:provider/", Handler)

	if tlsEnabled {
		tmpDir := path.Join(os.TempDir(), "tls", ".cache")

		if tlsDomain != "" {
			e.AutoTLSManager.HostPolicy = autocert.HostWhitelist(tlsDomain)
		}

		e.AutoTLSManager.Cache = autocert.DirCache(tmpDir)
		e.AutoTLSManager.Prompt = autocert.AcceptTOS
		e.Logger.Fatal(e.StartAutoTLS(":443"))
		return
	}

	e.Logger.Fatal(e.Start(":8080"))
}

var (
	handlerFuncs = map[string]func(*handlers.Request, *webhook.Webhook) error{}
	handlerMu    sync.RWMutex
)

//nolint:errcheck
func Handler(c echo.Context) error {
	c.String(http.StatusCreated, "")

	providerName := c.Param("provider")
	hook, err := webhook.New(providerName)
	if err != nil {
		logger.Error("webhook", "err", err)
		return err
	}

	if err = hook.ParseRequest(c.Request()); err != nil {
		logger.Error("ParseRequest", "err", err)
		return err
	}

	logger.Debug("handler", "event", hook.Event)

	handlerMu.RLock()
	defer handlerMu.RUnlock()

	if f, ok := handlerFuncs[hook.Event]; ok {
		go func() {
			command, err := handlers.New(providerName)
			if err != nil {
				logger.Error("can't initialize provider", "provider", providerName, "event", hook.Event, "err", err)
				return
			}

			// if err := command.LoadInfoAndConfig(hook.GetProjectID(), hook.GetID()); err != nil {
			// 	logger.Error("can't load repo config", "provider", providerName, "command", command, "err", err)
			// 	return
			// }

			if !command.ValidateSecret(hook.GetProjectID(), hook.GetSecret()) {
				logger.Error("webhook secret is not valid", "projectId", hook.GetProjectID(), "provider", providerName)
				return
			}

			if err := f(command, hook); err != nil {
				logger.Error("handlerFunc returns err", "provider", providerName, "event", hook.Event, "err", err)
				return
			}
		}()
	}

	return nil
}

func handle(onEvent string, funcHandler func(*handlers.Request, *webhook.Webhook) error) {
	handlerMu.Lock()
	defer handlerMu.Unlock()

	handlerFuncs[onEvent] = func(command *handlers.Request, hook *webhook.Webhook) error {
		return funcHandler(command, hook)
	}
}
