// Package observability wires Prometheus metrics, health-check endpoints and
// Sentry error tracking. All collectors live on a dedicated registry so tests
// can construct independent instances without global state.
package observability

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds the bot's Prometheus collectors and their registry.
type Metrics struct {
	Registry *prometheus.Registry

	// CommandsTotal counts handled interactions, labelled by name and status.
	CommandsTotal *prometheus.CounterVec
	// CommandDuration observes handler latency in seconds, labelled by name.
	CommandDuration *prometheus.HistogramVec
	// InteractionsTotal counts raw gateway interactions by type.
	InteractionsTotal *prometheus.CounterVec
}

// NewMetrics constructs and registers the collectors on a fresh registry.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	m := &Metrics{
		Registry: reg,
		CommandsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "disgo",
			Subsystem: "commands",
			Name:      "total",
			Help:      "Total interactions handled, by command name and status.",
		}, []string{"command", "status"}),
		CommandDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "disgo",
			Subsystem: "commands",
			Name:      "duration_seconds",
			Help:      "Interaction handler latency in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"command"}),
		InteractionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "disgo",
			Subsystem: "interactions",
			Name:      "total",
			Help:      "Total gateway interactions received, by type.",
		}, []string{"type"}),
	}

	reg.MustRegister(m.CommandsTotal, m.CommandDuration, m.InteractionsTotal)
	return m
}

// ObserveCommand records a single command execution's outcome and latency.
func (m *Metrics) ObserveCommand(name string, ok bool, dur time.Duration) {
	m.CommandsTotal.WithLabelValues(name, status(ok)).Inc()
	m.CommandDuration.WithLabelValues(name).Observe(dur.Seconds())
}

// CountInteraction increments the raw interaction counter for a gateway type.
func (m *Metrics) CountInteraction(t int) {
	m.InteractionsTotal.WithLabelValues(strconv.Itoa(t)).Inc()
}

func status(ok bool) string {
	if ok {
		return "ok"
	}
	return "error"
}
