package metrics

import (
	"expvar"
	"runtime"
	"time"
)

// Metrics holds all performance metrics for the system.
// Optimized for monitoring 1000+ concurrent rules.
type Metrics struct {
	// Alert processing metrics
	AlertsProcessed *expvar.Int
	AlertsQueued    *expvar.Int
	AlertsDropped   *expvar.Int
	AlertQueueDepth *expvar.Int

	// Rule evaluation metrics
	RulesEvaluated  *expvar.Int
	RuleCacheHits   *expvar.Int
	RuleCacheMisses *expvar.Int

	// Database metrics
	DBQueries     *expvar.Int
	DBQueryErrors *expvar.Int
	DBConnections *expvar.Int

	// Notification metrics
	NotificationsSent   *expvar.Int
	NotificationsFailed *expvar.Int
	RateLimiterWaits    *expvar.Int

	// System metrics
	Goroutines  *expvar.Int
	MemoryAlloc *expvar.Int
	Uptime      *expvar.String
}

// Global metrics instance
var M *Metrics

func init() {
	M = &Metrics{
		AlertsProcessed: expvar.NewInt("alerts_processed"),
		AlertsQueued:    expvar.NewInt("alerts_queued"),
		AlertsDropped:   expvar.NewInt("alerts_dropped"),
		AlertQueueDepth: expvar.NewInt("alert_queue_depth"),

		RulesEvaluated:  expvar.NewInt("rules_evaluated"),
		RuleCacheHits:   expvar.NewInt("rule_cache_hits"),
		RuleCacheMisses: expvar.NewInt("rule_cache_misses"),

		DBQueries:     expvar.NewInt("db_queries"),
		DBQueryErrors: expvar.NewInt("db_query_errors"),
		DBConnections: expvar.NewInt("db_connections"),

		NotificationsSent:   expvar.NewInt("notifications_sent"),
		NotificationsFailed: expvar.NewInt("notifications_failed"),
		RateLimiterWaits:    expvar.NewInt("rate_limiter_waits"),

		Goroutines:  expvar.NewInt("goroutines"),
		MemoryAlloc: expvar.NewInt("memory_alloc"),
		Uptime:      expvar.NewString("uptime"),
	}

	// Start system metrics collector
	go collectSystemMetrics()
}

// collectSystemMetrics periodically updates system-level metrics.
func collectSystemMetrics() {
	startTime := time.Now()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Update goroutine count
		M.Goroutines.Set(int64(runtime.NumGoroutine()))

		// Update memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		M.MemoryAlloc.Set(int64(m.Alloc))

		// Update uptime
		M.Uptime.Set(time.Since(startTime).String())
	}
}

// IncAlertsProcessed increments the processed alert counter.
func IncAlertsProcessed() {
	M.AlertsProcessed.Add(1)
}

// IncAlertsQueued increments the queued alert counter.
func IncAlertsQueued() {
	M.AlertsQueued.Add(1)
}

// IncAlertsDropped increments the dropped alert counter.
func IncAlertsDropped() {
	M.AlertsDropped.Add(1)
}

// SetAlertQueueDepth updates the current queue depth.
func SetAlertQueueDepth(depth int) {
	M.AlertQueueDepth.Set(int64(depth))
}

// IncRuleCacheHits increments the rule cache hit counter.
func IncRuleCacheHits() {
	M.RuleCacheHits.Add(1)
}

// IncRuleCacheMisses increments the rule cache miss counter.
func IncRuleCacheMisses() {
	M.RuleCacheMisses.Add(1)
}

// IncRulesEvaluated increments the rules evaluated counter.
func IncRulesEvaluated() {
	M.RulesEvaluated.Add(1)
}

// IncDBQueries increments the DB query counter.
func IncDBQueries() {
	M.DBQueries.Add(1)
}

// IncDBQueryErrors increments the DB query error counter.
func IncDBQueryErrors() {
	M.DBQueryErrors.Add(1)
}

// IncNotificationsSent increments the notifications sent counter.
func IncNotificationsSent() {
	M.NotificationsSent.Add(1)
}

// IncNotificationsFailed increments the notifications failed counter.
func IncNotificationsFailed() {
	M.NotificationsFailed.Add(1)
}

// IncRateLimiterWaits increments the rate limiter wait counter.
func IncRateLimiterWaits() {
	M.RateLimiterWaits.Add(1)
}
