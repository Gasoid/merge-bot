package metrics

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gasoid/merge-bot/logger"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	commandsCounter               *prometheus.CounterVec
	updateDuration                prometheus.Histogram
	backgroundTaskEnqueuedCounter *prometheus.CounterVec
	backgroundTaskCounter         *prometheus.CounterVec
)

const (
	commandSucceeded = "succeeded"
	commandFailed    = "failed"
)

func Handler(event string, f func() error) error {
	start := time.Now()
	err := f()

	if strings.HasPrefix(event, "!") {
		if err != nil {
			CommandFailedInc(event)
		} else {
			CommandSucceededInc(event)
		}

		if event == "!update" {
			duration := time.Since(start)
			UpdateDuration(duration)
		}
	}

	return err
}

func BackgroundRun(task string, f func()) func() {
	backgroundTaskEnqueuedCounter.WithLabelValues(task).Inc()

	return func() {
		backgroundTaskCounter.WithLabelValues(task).Inc()
		f()
	}
}

func CommandSucceededInc(command string) {
	commandsCounter.WithLabelValues(command, commandSucceeded).Inc()
}

func CommandFailedInc(command string) {
	commandsCounter.WithLabelValues(command, commandFailed).Inc()
}

func UpdateDuration(duration time.Duration) {
	updateDuration.Observe(duration.Seconds())
}

func initMetrics() error {
	commandsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mergebot_commands_total",
			Help: "How many webhook commands bot has received",
		},
		[]string{"command", "status"},
	)

	backgroundTaskCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mergebot_background_tasks_total",
			Help: "How many background tasks run",
		},
		[]string{"task"},
	)

	backgroundTaskEnqueuedCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mergebot_background_tasks_enqueued_total",
			Help: "How many background tasks enqueued/added",
		},
		[]string{"task"},
	)

	updateDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "mergebot_update_duration",
		Help:    "Time it has taken to update the branch",
		Buckets: prometheus.LinearBuckets(5, 4, 10),
	})

	if err := prometheus.Register(updateDuration); err != nil {
		return err
	}

	if err := prometheus.Register(commandsCounter); err != nil {
		return err
	}

	go func() {
		metrics := echo.New()
		metrics.GET("/metrics", echoprometheus.NewHandler())
		if err := metrics.Start(":8081"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(err.Error())
		}
	}()
	return nil
}

func init() {
	err := initMetrics()
	if err != nil {
		logger.Error("initMetrics", "err", err)
		panic(err)
	}
}
