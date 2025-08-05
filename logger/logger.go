package logger

import (
	"fmt"
	"log/slog"

	"github.com/gasoid/merge-bot/config"

	"github.com/getsentry/sentry-go"
	sentryslog "github.com/getsentry/sentry-go/slog"
)

var (
	log           = slog.New(sentryslog.Option{Level: slog.LevelError}.NewSentryHandler())
	sentryEnabled = true
	debug         bool
)

const (
	sentryDsn = "https://11a97d0fb2804c34db705b2c2088f298@o4509393813897216.ingest.de.sentry.io/4509393818288208"
)

func init() {
	config.BoolVar(&debug, "debug", false, "whether debug logging is enabled (also via DEBUG)")
	config.BoolVar(&sentryEnabled, "sentry-enabled", true, "whether sentry is enabled or not (also via SENTRY_ENABLED)")
}

func New() {
	if sentryEnabled {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:           sentryDsn,
			EnableTracing: false,
		}); err != nil {
			Error("Sentry initialization failed", "err", err)
			return
		}

		sentry.CaptureMessage("merge-bot started")
		fmt.Println("Sentry is enabled")
	}

	//nolint:errcheck
	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		fmt.Println("Debug mode is enabled")
		// os.Setenv("GIT_CURL_VERBOSE", "True")
		// os.Setenv("GIT_TRACE", "True")
	}
}

func Error(msg string, args ...any) {
	if sentryEnabled {
		log.Error(msg, args...)
	}

	slog.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func IsSentryEnabled() bool {
	return sentryEnabled
}
