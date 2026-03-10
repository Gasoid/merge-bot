package logger

import (
	"fmt"
	"log/slog"

	"github.com/gasoid/merge-bot/config"

	"github.com/getsentry/sentry-go"
	sentryslog "github.com/getsentry/sentry-go/slog"
)

var (
	log           = slog.Default()
	sentryEnabled = true
	debug         bool
	sentryDsn     = ""
)

func init() {
	config.BoolVar(&debug, "debug", false, "whether debug logging is enabled (also via DEBUG)")
	config.BoolVar(&sentryEnabled, "sentry-enabled", true, "whether sentry is enabled or not (also via SENTRY_ENABLED)")
}

func New() {
	if sentryDsn == "" {
		sentryEnabled = false
	}

	if sentryEnabled {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:           sentryDsn,
			EnableTracing: false,
		}); err != nil {
			Error("Sentry initialization failed", "err", err)
			return
		}

		handler := slog.NewMultiHandler(sentryslog.Option{Level: slog.LevelError}.NewSentryHandler(), slog.Default().Handler())
		log = slog.New(handler)

		sentry.CaptureMessage("merge-bot started")
		fmt.Println("Sentry is enabled")
	}

	//nolint:errcheck
	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		fmt.Println("Debug mode is enabled")
	}
}

func Error(msg string, args ...any) {
	log.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	log.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	log.Info(msg, args...)
}

func IsSentryEnabled() bool {
	return sentryEnabled
}
