// Package observ provides minimal observability helpers (structured logging
// and a placeholder metrics registry) shared by all services.
package observ

import (
	"log"
	"os"
	"sync"
	"time"
)

// Logger is a tiny structured logger suitable for the monorepo scope.
type Logger struct {
	service string
	level   string
}

// NewLogger creates a logger bound to a service name.
func NewLogger(service, level string) *Logger {
	if level == "" {
		level = "info"
	}
	return &Logger{service: service, level: level}
}

func (l *Logger) logf(lvl, format string, args ...interface{}) {
	if !l.allows(lvl) {
		return
	}
	prefix := time.Now().Format("2006-01-02T15:04:05Z07:00") + " [" + l.level + "] " + l.service
	logger := log.New(os.Stdout, prefix+" ", log.LstdFlags)
	logger.Printf(lvl+" "+format, args...)
}

func (l *Logger) allows(lvl string) bool {
	order := map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}
	return order[lvl] >= order[l.level]
}

func (l *Logger) Debug(format string, args ...interface{}) { l.logf("debug", format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.logf("info", format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.logf("warn", format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.logf("error", format, args...) }

// Metrics is a minimal in-process counter registry. A production-grade
// implementation (Prometheus) can replace this later.
type Metrics struct {
	mu     sync.Mutex
	counts map[string]int64
}

// NewMetrics creates an empty metrics registry.
func NewMetrics() *Metrics {
	return &Metrics{counts: make(map[string]int64)}
}

// Inc increments a named counter.
func (m *Metrics) Inc(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counts[name]++
}

// Snapshot returns a copy of all counters.
func (m *Metrics) Snapshot() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]int64, len(m.counts))
	for k, v := range m.counts {
		out[k] = v
	}
	return out
}
