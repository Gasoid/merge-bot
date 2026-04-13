package metrics

import (
	"strings"
	"time"

	"github.com/gasoid/merge-bot/logger"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	commandsCounter               *prometheus.CounterVec
	updateDuration                prometheus.Histogram
	backgroundTaskEnqueuedCounter *prometheus.CounterVec
	backgroundTaskCounter         *prometheus.CounterVec
	branchesDeletionCounter       *prometheus.CounterVec
	mrDeletionCounter             *prometheus.CounterVec
	branchesDeletionDuration      prometheus.Histogram
	mrDeletionDuration            prometheus.Histogram
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

func BranchDeletionInc() {
	branchesDeletionCounter.WithLabelValues().Inc()
}

func MrDeletionInc() {
	mrDeletionCounter.WithLabelValues().Inc()
}

func BranchDeletionDuration(duration time.Duration) {
	branchesDeletionDuration.Observe(duration.Seconds())
}

func MrDeletionDuration(duration time.Duration) {
	mrDeletionDuration.Observe(duration.Seconds())
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

	branchesDeletionCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mergebot_deleted_branches_total",
			Help: "How many branches has been deleted",
		},
		[]string{},
	)

	mrDeletionCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mergebot_deleted_mr_total",
			Help: "How many MRs has been deleted",
		},
		[]string{},
	)

	branchesDeletionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "mergebot_branch_deletion_duration",
		Help:    "Time it has taken to delete branches",
		Buckets: prometheus.LinearBuckets(5, 4, 10),
	})

	mrDeletionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "mergebot_mr_deletion_duration",
		Help:    "Time it has taken to delete MRs",
		Buckets: prometheus.LinearBuckets(5, 4, 10),
	})

	if err := prometheus.Register(backgroundTaskCounter); err != nil {
		return err
	}

	if err := prometheus.Register(backgroundTaskEnqueuedCounter); err != nil {
		return err
	}

	if err := prometheus.Register(updateDuration); err != nil {
		return err
	}

	if err := prometheus.Register(commandsCounter); err != nil {
		return err
	}

	if err := prometheus.Register(branchesDeletionCounter); err != nil {
		return err
	}

	if err := prometheus.Register(mrDeletionCounter); err != nil {
		return err
	}

	if err := prometheus.Register(branchesDeletionDuration); err != nil {
		return err
	}

	if err := prometheus.Register(mrDeletionDuration); err != nil {
		return err
	}

	return nil
}

func init() {
	err := initMetrics()
	if err != nil {
		logger.Error("initMetrics", "err", err)
		panic(err)
	}
}
