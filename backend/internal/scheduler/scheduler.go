package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kk-alert/backend/internal/dedup"
	"github.com/kk-alert/backend/internal/engine"
	"github.com/kk-alert/backend/internal/models"
	"github.com/kk-alert/backend/internal/query"
	"gorm.io/gorm"
)

type Scheduler struct {
	db       *gorm.DB
	tasks    map[uint]*RuleTask
	mu       sync.RWMutex
	stopChan chan struct{}
}

type RuleTask struct {
	ruleID   uint
	ticker   *time.Ticker
	stopChan chan struct{}
}

type queryState struct {
	mu            sync.RWMutex
	lastResults   map[string]queryResult
	lastCheckTime time.Time
}

type queryResult struct {
	Metric    map[string]string
	Value     float64
	Timestamp time.Time
	AlertID   string
	Severity  string // severity level from threshold match (critical/warning/info)
	MissCount int    // consecutive evaluations where this series was absent from query results
}

// resolveGracePeriod is how many consecutive absences before resolving an alert.
// Prevents flapping when Prometheus temporarily drops a series (scrape gap, network hiccup).
const resolveGracePeriod = 3

// roundValue rounds a float64 to 2 decimal places to avoid re-processing on tiny fluctuations.
func roundValue(v float64) float64 {
	return math.Round(v*100) / 100
}

var (
	stateCache = make(map[uint]*queryState)
	stateMu    sync.RWMutex
)

func NewScheduler(db *gorm.DB) *Scheduler {
	return &Scheduler{
		db:       db,
		tasks:    make(map[uint]*RuleTask),
		stopChan: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	log.Println("[scheduler] starting rule scheduler")
	s.loadRules()

	// Reload rules every 5 minutes to pick up changes
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.loadRules()
			case <-s.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	log.Println("[scheduler] stopping rule scheduler")
	close(s.stopChan)

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.tasks {
		close(task.stopChan)
	}
	s.tasks = make(map[uint]*RuleTask)
}

// SeverityCounts holds alert counts broken down by severity level.
type SeverityCounts struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

// FiringCountByRule returns the current number of firing series per rule
// broken down by severity level (from scheduler state).
func FiringCountByRule() map[uint]*SeverityCounts {
	stateMu.RLock()
	defer stateMu.RUnlock()
	out := make(map[uint]*SeverityCounts, len(stateCache))
	for ruleID, state := range stateCache {
		state.mu.RLock()
		sc := &SeverityCounts{Total: len(state.lastResults)}
		for _, r := range state.lastResults {
			switch r.Severity {
			case "critical":
				sc.Critical++
			case "warning":
				sc.Warning++
			case "info":
				sc.Info++
			}
		}
		state.mu.RUnlock()
		out[ruleID] = sc
	}
	return out
}

// RunRuleNow runs the given rule once immediately (non-blocking). Used after create/update so new rules run without waiting for next interval.
func (s *Scheduler) RunRuleNow(ruleID uint) {
	var rule models.Rule
	if err := s.db.First(&rule, ruleID).Error; err != nil {
		log.Printf("[scheduler] RunRuleNow: rule %d not found", ruleID)
		return
	}
	if !rule.Enabled || rule.QueryExpression == "" {
		return
	}
	go func() {
		s.evaluateRule(&rule)
		s.updateLastRunAt(ruleID)
	}()
}

func (s *Scheduler) loadRules() {
	var rules []models.Rule
	if err := s.db.Where("enabled = ? AND query_expression != ?", true, "").Find(&rules).Error; err != nil {
		log.Printf("[scheduler] failed to load rules: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Track current rule IDs
	currentIDs := make(map[uint]bool)
	for _, rule := range rules {
		currentIDs[rule.ID] = true

		// Check if task already exists
		if _, exists := s.tasks[rule.ID]; exists {
			continue
		}

		// Create new task
		interval := parseInterval(rule.CheckInterval)
		if interval <= 0 {
			interval = time.Minute
		}

		task := &RuleTask{
			ruleID:   rule.ID,
			stopChan: make(chan struct{}),
		}
		s.tasks[rule.ID] = task

		// Start the task
		go s.runTask(task, rule, interval)
		log.Printf("[scheduler] scheduled rule %d with interval %v", rule.ID, interval)
	}

	// Stop tasks for rules that no longer exist or are disabled
	for id, task := range s.tasks {
		if !currentIDs[id] {
			close(task.stopChan)
			delete(s.tasks, id)
			log.Printf("[scheduler] stopped rule %d", id)
		}
	}
}

// runTask runs one rule in its own goroutine; each rule has independent schedule and fixed interval (no drift).
func (s *Scheduler) runTask(task *RuleTask, rule models.Rule, interval time.Duration) {
	s.evaluateRule(&rule)
	s.updateLastRunAt(task.ruleID)
	nextRun := time.Now().Add(interval)
	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		wait := time.Until(nextRun)
		if wait < 0 {
			wait = 0
		}
		timer.Reset(wait)
		select {
		case <-timer.C:
			nextRun = nextRun.Add(interval)
			if time.Now().After(nextRun) {
				nextRun = time.Now().Add(interval)
			}
			var currentRule models.Rule
			if err := s.db.First(&currentRule, task.ruleID).Error; err != nil {
				log.Printf("[scheduler] rule %d not found, stopping", task.ruleID)
				return
			}
			if !currentRule.Enabled || currentRule.QueryExpression == "" {
				log.Printf("[scheduler] rule %d disabled or no query, stopping", task.ruleID)
				return
			}
			s.evaluateRule(&currentRule)
			s.updateLastRunAt(task.ruleID)
		case <-task.stopChan:
			return
		}
	}
}

func (s *Scheduler) updateLastRunAt(ruleID uint) {
	now := time.Now()
	_ = s.db.Model(&models.Rule{}).Where("id = ?", ruleID).Update("last_run_at", now).Error
}

func (s *Scheduler) evaluateRule(rule *models.Rule) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a fresh DB session for this goroutine to avoid shared-session
	// race conditions when multiple rules run concurrently.
	db := s.db.Session(&gorm.Session{NewDB: true})

	// Get datasource
	var datasourceID uint
	if rule.DatasourceIDs != "" {
		var ids []uint
		if err := json.Unmarshal([]byte(rule.DatasourceIDs), &ids); err == nil && len(ids) > 0 {
			datasourceID = ids[0]
		}
	}

	if datasourceID == 0 {
		log.Printf("[scheduler] rule %d has no datasource", rule.ID)
		return
	}

	var ds models.Datasource
	if err := db.First(&ds, datasourceID).Error; err != nil {
		log.Printf("[scheduler] rule %d datasource %d not found", rule.ID, datasourceID)
		return
	}

	if !ds.Enabled {
		log.Printf("[scheduler] rule %d datasource %d disabled", rule.ID, datasourceID)
		return
	}

	// Query based on datasource type
	switch ds.Type {
	case "prometheus", "victoriametrics":
		s.queryPrometheus(ctx, rule, &ds, db)
	default:
		log.Printf("[scheduler] rule %d unsupported datasource type: %s", rule.ID, ds.Type)
	}
}

func (s *Scheduler) queryPrometheus(ctx context.Context, rule *models.Rule, ds *models.Datasource, db *gorm.DB) {
	client := query.NewPrometheusClient(ds.Endpoint)

	result, err := client.Query(ctx, rule.QueryExpression)
	if err != nil {
		log.Printf("[scheduler] rule %d (%s) query failed: %v", rule.ID, rule.Name, err)
		return
	}

	// Get or create state for this rule
	stateMu.Lock()
	state, exists := stateCache[rule.ID]
	if !exists {
		state = &queryState{
			lastResults: make(map[string]queryResult),
		}
		stateCache[rule.ID] = state
	}
	stateMu.Unlock()

	state.mu.Lock()
	defer state.mu.Unlock()

	// Uniqueness key: datasource + title + all labels (same => same alert, reuse ID until resolved).
	// When labels lack instance/job, KeyForSeries uses result index so each series gets its own alert.
	currentKeys := make(map[string]bool)
	numResults := len(result.Data.Result)
	if numResults > 0 {
		log.Printf("[scheduler] rule %d (%s) query returned %d series", rule.ID, rule.Name, numResults)
	}
	// 0 series is normal when no condition is met (e.g. no disk > threshold); no log to avoid noise

	thresholds := ParseThresholds(rule.Thresholds)

	for i, r := range result.Data.Result {
		metric := r.Metric
		if metric == nil {
			metric = make(map[string]string)
		}
		labels, _ := json.Marshal(metric)
		value := query.GetValue(r.Value)

		severity := rule.MatchSeverity
		if severity == "" {
			severity = "warning"
		}

		// Build annotations map
		annotations := map[string]string{"value": fmt.Sprintf("%v", value)}

		// Multi-level threshold evaluation: first matching level wins.
		// If thresholds are configured but none match, this series is "normal" (skip / resolve).
		if thresholds != nil {
			matched := MatchThreshold(thresholds, value)
			if matched == nil {
				// Value below all thresholds — don't add to currentKeys so existing alert gets resolved
				continue
			}
			severity = matched.Severity
			if severity == "" {
				severity = "warning"
			}
			// Carry per-level channel_ids in annotations for engine to pick up
			if len(matched.ChannelIDs) > 0 {
				chJSON, _ := json.Marshal(matched.ChannelIDs)
				annotations["threshold_channel_ids"] = string(chJSON)
			}
		}

		title := fmt.Sprintf("%s: %s", rule.Name, formatMetric(metric))
		// Include rule ID so different rules get different alerts for the same instance (avoid 3 rules x 7 instances => 7 alerts)
		extKey := dedup.KeyForSeriesWithRule(uint(ds.ID), uint(rule.ID), title, metric, i)
		currentKeys[extKey] = true

		lastResult, hadResult := state.lastResults[extKey]

		// Determine if this alert needs (re-)processing:
		// 1. First time seeing this series (!hadResult)
		// 2. Value changed (metric fluctuation)
		// 3. Stable alert needs periodic re-process so engine can re-send per send_interval
		valueChanged := !hadResult || roundValue(lastResult.Value) != roundValue(value)
		needsReprocess := false
		if hadResult && !valueChanged && lastResult.AlertID != "" {
			// Re-process stable alerts every 60s so the engine's sendRateLimited
			// can decide whether to send a repeat notification.
			needsReprocess = time.Since(lastResult.Timestamp) >= 60*time.Second
		}
		if valueChanged || needsReprocess {
			alertID := lastResult.AlertID
			if alertID == "" {
				// After restart, in-memory state is lost. Reuse existing firing alert with same (source_id, external_id).
				var existingFiring models.Alert
				db.Where("source_id = ? AND external_id = ? AND status = ?", ds.ID, extKey, "firing").Limit(1).Find(&existingFiring)
				if existingFiring.ID != "" {
					alertID = existingFiring.ID
				} else {
					alertID = uuid.New().String()
				}
			}

			annotationsJSON, _ := json.Marshal(annotations)
			alert := models.Alert{
				ID:           alertID,
				SourceID:     uint(ds.ID),
				SourceType:   ds.Type,
				ExternalID:   extKey,
				Title:        title,
				Severity:     severity,
				Status:       "firing",
				FiringAt:     time.Now(),
				Labels:       string(labels),
				Annotations:  string(annotationsJSON),
			}

			// New key: Create so alert appears in history/reports. Existing (from memory or DB lookup): Save to update but preserve FiringAt and CreatedAt.
			if !hadResult {
				// Check again: we may have set alertID from existingFiring above
				var exists models.Alert
				db.Where("id = ?", alertID).Limit(1).Find(&exists)
				if exists.ID != "" {
					// Reuse existing row (e.g. after restart) — preserve FiringAt and CreatedAt
					if !exists.FiringAt.IsZero() {
						alert.FiringAt = exists.FiringAt
					}
					alert.CreatedAt = exists.CreatedAt
					if res := db.Save(&alert); res.Error != nil {
						log.Printf("[scheduler] rule %d failed to update alert %s: %v", rule.ID, alertID, res.Error)
						continue
					}
				} else {
					if res := db.Create(&alert); res.Error != nil {
						log.Printf("[scheduler] rule %d failed to create alert %s: %v", rule.ID, alertID, res.Error)
						continue
					} else if res.RowsAffected == 0 {
						log.Printf("[scheduler] rule %d alert %s Create returned 0 rows, retrying", rule.ID, alertID)
						if res2 := db.Save(&alert); res2.Error != nil {
							log.Printf("[scheduler] rule %d failed to save alert %s on retry: %v", rule.ID, alertID, res2.Error)
							continue
						}
					}
				}
			} else {
				var existing models.Alert
				db.Where("id = ?", alertID).Limit(1).Find(&existing)
				if existing.ID != "" {
					alert.FiringAt = existing.FiringAt // preserve so duration (e.g. 5m) is satisfied when re-processing
					alert.CreatedAt = existing.CreatedAt
				}
				if res := db.Save(&alert); res.Error != nil {
					log.Printf("[scheduler] rule %d failed to update alert %s: %v", rule.ID, alertID, res.Error)
					continue
				}
			}

			// Process alert through engine asynchronously so notification
			// delivery (rate limiters, HTTP) does not block the scheduler.
			engine.ProcessAlertAsync(db, &alert)

			// Update state (reset MissCount since series is present)
			state.lastResults[extKey] = queryResult{
				Metric:    metric,
				Value:     value,
				Timestamp: time.Now(),
				AlertID:   alertID,
				Severity:  severity,
				MissCount: 0,
			}

			if !hadResult {
				log.Printf("[scheduler] rule %d new alert %s (value=%.2f)", rule.ID, alertID, value)
			} else {
				log.Printf("[scheduler] rule %d updated alert %s (value=%.2f)", rule.ID, alertID, value)
			}
		} else if hadResult && lastResult.MissCount > 0 {
			// Series reappeared after being absent — reset miss counter
			lastResult.MissCount = 0
			state.lastResults[extKey] = lastResult
		}
	}
	if numResults > 0 {
		uniqueKeys := len(currentKeys)
		if uniqueKeys < numResults {
			log.Printf("[scheduler] rule %d: %d series produced %d unique keys (threshold filtered); this is normal when only some series match the threshold", rule.ID, numResults, uniqueKeys)
		}
	}

	// Check for resolved alerts (keys that no longer appear in result).
	// Grace period: only resolve after resolveGracePeriod consecutive absences to handle
	// temporary Prometheus scrape gaps / network hiccups that would otherwise cause flapping.
	for extKey, lastResult := range state.lastResults {
		if !currentKeys[extKey] {
			lastResult.MissCount++
			state.lastResults[extKey] = lastResult

			if lastResult.MissCount < resolveGracePeriod {
				log.Printf("[scheduler] rule %d alert %s absent (%d/%d), waiting before resolve",
					rule.ID, lastResult.AlertID, lastResult.MissCount, resolveGracePeriod)
				continue
			}

			// Exceeded grace period — actually resolve
			if lastResult.AlertID != "" {
				var alert models.Alert
				if err := db.First(&alert, "id = ?", lastResult.AlertID).Error; err == nil {
					if alert.Status == "firing" {
						now := time.Now()
						alert.Status = "resolved"
						alert.ResolvedAt = &now
						db.Save(&alert)

						// Process resolved alert (recovery notification) asynchronously
						engine.ProcessAlertAsync(db, &alert)

						log.Printf("[scheduler] rule %d resolved alert %s (absent %d checks)",
							rule.ID, alert.ID, lastResult.MissCount)
					}
				}
			}
			delete(state.lastResults, extKey)
		}
	}

	state.lastCheckTime = time.Now()
}

// ThresholdLevel represents one level in a multi-threshold rule.
type ThresholdLevel struct {
	Operator   string  `json:"operator"`    // >, <, >=, <=, ==, !=
	Value      float64 `json:"value"`
	Severity   string  `json:"severity"`    // critical, warning, info
	ChannelIDs []uint  `json:"channel_ids"`
}

// ParseThresholds parses the rule's Thresholds JSON into a slice. Returns nil if empty or invalid.
func ParseThresholds(raw string) []ThresholdLevel {
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var levels []ThresholdLevel
	if err := json.Unmarshal([]byte(raw), &levels); err != nil {
		return nil
	}
	if len(levels) == 0 {
		return nil
	}
	return levels
}

// MatchThreshold evaluates value against threshold levels in order (first match wins).
func MatchThreshold(levels []ThresholdLevel, value float64) *ThresholdLevel {
	for i, l := range levels {
		matched := false
		switch l.Operator {
		case ">":
			matched = value > l.Value
		case ">=":
			matched = value >= l.Value
		case "<":
			matched = value < l.Value
		case "<=":
			matched = value <= l.Value
		case "==":
			matched = value == l.Value
		case "!=":
			matched = value != l.Value
		default:
			matched = value > l.Value // default to >
		}
		if matched {
			return &levels[i]
		}
	}
	return nil
}

func parseInterval(s string) time.Duration {
	if s == "" {
		return time.Minute
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Minute
	}
	if d < time.Minute {
		return time.Minute
	}
	return d
}

func formatMetric(metric map[string]string) string {
	if instance, ok := metric["instance"]; ok {
		return instance
	}
	if name, ok := metric["__name__"]; ok {
		return name
	}
	return "unknown"
}
