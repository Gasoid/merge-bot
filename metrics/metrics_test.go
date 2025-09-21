package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func TestCommandSucceededInc(t *testing.T) {
	// Reset the counter
	commandsCounter.Reset()

	// Call the function
	CommandSucceededInc("test-command")

	// Get the metric
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	// Find our metric
	var commandsMetric *dto.MetricFamily
	for _, mf := range metricFamilies {
		if mf.GetName() == "mergebot_commands_total" {
			commandsMetric = mf
			break
		}
	}

	assert.NotNil(t, commandsMetric)
	assert.Equal(t, 1, len(commandsMetric.GetMetric()))

	metric := commandsMetric.GetMetric()[0]
	assert.Equal(t, float64(1), metric.GetCounter().GetValue())

	// Check labels
	labels := metric.GetLabel()
	labelMap := make(map[string]string)
	for _, label := range labels {
		labelMap[label.GetName()] = label.GetValue()
	}

	assert.Equal(t, "test-command", labelMap["command"])
	assert.Equal(t, commandSucceeded, labelMap["status"])
}

func TestCommandFailedInc(t *testing.T) {
	// Reset the counter
	commandsCounter.Reset()

	// Call the function
	CommandFailedInc("test-command")

	// Get the metric
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	// Find our metric
	var commandsMetric *dto.MetricFamily
	for _, mf := range metricFamilies {
		if mf.GetName() == "mergebot_commands_total" {
			commandsMetric = mf
			break
		}
	}

	assert.NotNil(t, commandsMetric)

	metric := commandsMetric.GetMetric()[0]
	assert.Equal(t, float64(1), metric.GetCounter().GetValue())

	// Check labels
	labels := metric.GetLabel()
	labelMap := make(map[string]string)
	for _, label := range labels {
		labelMap[label.GetName()] = label.GetValue()
	}

	assert.Equal(t, "test-command", labelMap["command"])
	assert.Equal(t, commandFailed, labelMap["status"])
}

func TestInitMetrics(t *testing.T) {
	// initMetrics should have been called during init()
	// Let's verify that commandsCounter is not nil
	assert.NotNil(t, commandsCounter)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "succeeded", commandSucceeded)
	assert.Equal(t, "failed", commandFailed)
}
