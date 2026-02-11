package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kk-alert/backend/internal/jira"
	"github.com/kk-alert/backend/internal/models"
	"github.com/kk-alert/backend/internal/sender"
	"gorm.io/gorm"
)

// suppressionWindows holds per-rule suppression end times (ruleID -> endTime). When an alert matches
// source condition we set endTime = now+duration; when an alert matches suppressed condition and now < endTime we skip send.
var suppressionMu sync.RWMutex
var suppressionWindows = make(map[uint]time.Time)

// aggLastSent tracks last aggregated send time per (ruleID_typeFingerprint) so we send at most once per aggregate window.
var aggMu sync.RWMutex
var aggLastSent = make(map[string]time.Time)

// stripSystemAlertPrefix removes upstream "【系统告警】" prefix from title so notifications do not duplicate it.
func stripSystemAlertPrefix(s string) string {
	return strings.TrimSpace(strings.TrimPrefix(s, "【系统告警】"))
}

// IsSilenced returns true if alert_id has an active manual silence (no notifications until silence_until).
func IsSilenced(db *gorm.DB, alertID string) bool {
	var n int64
	db.Model(&models.AlertSilence{}).Where("alert_id = ? AND silence_until > ?", alertID, time.Now()).Count(&n)
	return n > 0
}

// alertJob represents a queued alert processing task.
type alertJob struct {
	db    *gorm.DB
	alert models.Alert
}

// Bounded notification worker pool (8 workers, 500-slot buffer).
// Prevents unbounded goroutine spawning and controls Lark API pressure.
var alertQueue = make(chan alertJob, 500)

func init() {
	const numWorkers = 8
	for i := 0; i < numWorkers; i++ {
		go func() {
			for job := range alertQueue {
				ProcessAlert(job.db, &job.alert)
			}
		}()
	}
}

// ProcessAlertAsync queues ProcessAlert to run asynchronously so the caller
// (scheduler) is not blocked by slow notification delivery (rate limiters, HTTP).
func ProcessAlertAsync(db *gorm.DB, alert *models.Alert) {
	// Copy the alert and create a fresh DB session to avoid data races
	// with the caller's subsequent modifications and DB session sharing.
	a := *alert
	freshDB := db.Session(&gorm.Session{NewDB: true})
	select {
	case alertQueue <- alertJob{db: freshDB, alert: a}:
		// queued successfully
	default:
		// queue full — run inline as fallback to avoid losing alerts
		log.Printf("[engine] alert queue full, processing inline for %s", a.ID)
		go ProcessAlert(freshDB, &a)
	}
}

// ProcessAlert loads enabled rules, matches the alert, applies duration threshold, and sends to channels via Telegram/Lark.
func ProcessAlert(db *gorm.DB, alert *models.Alert) {
	if IsSilenced(db, alert.ID) {
		return
	}
	var rules []models.Rule
	if err := db.Where("enabled = ?", true).Order("priority asc").Find(&rules).Error; err != nil {
		return
	}
	var labels map[string]string
	_ = json.Unmarshal([]byte(alert.Labels), &labels)
	if labels == nil {
		labels = make(map[string]string)
	}
	for _, r := range rules {
		// If this alert matches suppression source condition, start or refresh the suppression window for this rule.
		updateSuppressionWindow(&r, labels)

		if !matchRule(&r, alert, labels) {
			continue
		}
		// Determine channels: prefer per-threshold channels from annotations, fall back to rule-level channels.
		var channelIDs []uint
		if thChStr := annotationValue(alert, "threshold_channel_ids"); thChStr != "" {
			_ = json.Unmarshal([]byte(thChStr), &channelIDs)
		}
		if len(channelIDs) == 0 {
			_ = json.Unmarshal([]byte(r.ChannelIDs), &channelIDs)
		}
		if len(channelIDs) == 0 {
			continue
		}

		// Recovery: when alert is resolved and rule has recovery notify, send by template only (no extra title).
		// Deduplicate by (alert_id, channel_id): if another rule already sent recovery to this channel, skip to avoid duplicate notifications.
		if alert.Status == "resolved" && r.RecoveryNotify {
			title := ""
			sendAt := time.Now()
			body := resolveBody(db, &r, alert, labels, true, sendAt) + "\n\n发送时间: " + formatSendTime(sendAt)
			for _, chID := range channelIDs {
				if recoveryAlreadySent(db, alert.ID, chID) {
					continue
				}
				var ch models.Channel
				if err := db.First(&ch, chID).Error; err != nil || !ch.Enabled {
					continue
				}
				if err := sender.Send(ch.Type, ch.Config, title, body, true); err != nil {
					log.Printf("[engine] recovery send alert %s to channel %d failed: %v", alert.ID, chID, err)
					db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: false, Error: err.Error()})
				} else {
					db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: true})
				}
			}
			continue
		}
		if alert.Status != "firing" {
			continue
		}

		if !durationSatisfied(&r, alert) {
			continue
		}
		if inExcludeWindow(&r) {
			continue
		}
		if suppressed(&r, labels) {
			continue
		}
		sendAt := time.Now()
		body := resolveBody(db, &r, alert, labels, false, sendAt) + "\n\n发送时间: " + formatSendTime(sendAt)
		title := stripSystemAlertPrefix(alert.Title)
		if title == "" {
			title = "Alert"
		}
		tryCreateJiraTicket(db, &r, alert, title, body)
		if r.AggregationEnabled && r.AggregateBy != "" && r.AggregateWindow != "" {
			sendAggregated(db, &r, alert, labels, title, body, channelIDs)
		} else {
			for _, chID := range channelIDs {
				if sendRateLimited(db, &r, alert.ID, chID) {
					continue
				}
				var ch models.Channel
				if err := db.First(&ch, chID).Error; err != nil || !ch.Enabled {
					db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: false, Error: "channel not found or disabled"})
					continue
				}
				err := sender.Send(ch.Type, ch.Config, title, body, false)
				if err != nil {
					log.Printf("[engine] send alert %s to channel %d failed: %v", alert.ID, chID, err)
					db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: false, Error: err.Error()})
					continue
				}
				db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: true})
			}
		}
	}
}

func durationSatisfied(r *models.Rule, a *models.Alert) bool {
	if r.Duration == "" || r.Duration == "0" {
		return true
	}
	d, err := time.ParseDuration(r.Duration)
	if err != nil {
		return true
	}
	// Require alert to have been firing for at least d
	elapsed := time.Since(a.FiringAt)
	return elapsed >= d
}

var locCST = time.FixedZone("CST", 8*3600)

func formatSendTime(t time.Time) string {
	return t.In(locCST).Format("2006-01-02 15:04:05")
}

func resolveBody(db *gorm.DB, r *models.Rule, alert *models.Alert, labels map[string]string, isRecovery bool, sendAt time.Time) string {
	data := sender.AlertTemplateData{
		AlertID:         alert.ID,
		Title:           stripSystemAlertPrefix(alert.Title),
		Severity:        alert.Severity,
		Labels:          labels,
		StartAt:         alert.FiringAt.Format("2006-01-02 15:04:05"),
		SourceType:      alert.SourceType,
		IsRecovery:      isRecovery,
		RuleDescription: r.Description,
		SentAt:          formatSendTime(sendAt),
	}
	if isRecovery && alert.ResolvedAt != nil {
		data.ResolvedAt = alert.ResolvedAt.Format("2006-01-02 15:04:05")
	}
	if alert.Annotations != "" {
		var ann map[string]string
		if _ = json.Unmarshal([]byte(alert.Annotations), &ann); ann != nil {
			if d := ann["description"]; d != "" {
				data.Description = d
			}
			if data.Description == "" && ann["summary"] != "" {
				data.Description = ann["summary"]
			}
			if v := ann["value"]; v != "" {
				data.Value = v
			}
		}
	}
	// When alert has no description/summary, use rule description only (do not use alert title — it is already shown as 🔔 header)
	if data.Description == "" && r.Description != "" {
		data.Description = r.Description
	}
	// Try rule's template first (use Find to avoid GORM logging "record not found" when template was deleted)
	if r.TemplateID != nil && *r.TemplateID != 0 {
		var t models.Template
		db.Where("id = ?", *r.TemplateID).Limit(1).Find(&t)
		if t.ID != 0 && t.Body != "" {
			out, err := sender.RenderTemplate(t.Body, data)
			if err == nil {
				return out
			}
			log.Printf("[engine] template render failed, using simple replace: %v", err)
			return sender.RenderBody(t.Body, labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
		}
		if t.ID == 0 {
			log.Printf("[engine] template id=%d not found (rule %d), falling back to default template", *r.TemplateID, r.ID)
			// Auto-fix: bind rule to default template in DB so next run uses it without fallback
			var defaultT models.Template
			db.Where("is_default = ?", true).Limit(1).Find(&defaultT)
			if defaultT.ID != 0 {
				_ = db.Model(&models.Rule{}).Where("id = ?", r.ID).Update("template_id", defaultT.ID).Error
				r.TemplateID = &defaultT.ID
				out, err := sender.RenderTemplate(defaultT.Body, data)
				if err == nil {
					return out
				}
				return sender.RenderBody(defaultT.Body, labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
			}
		}
	}
	// Fallback: load default template (is_default=true); if none, use hardcoded body
	var defaultT models.Template
	db.Where("is_default = ?", true).Limit(1).Find(&defaultT)
	if defaultT.ID != 0 && defaultT.Body != "" {
		out, err := sender.RenderTemplate(defaultT.Body, data)
		if err == nil {
			return out
		}
		return sender.RenderBody(defaultT.Body, labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
	}
	return sender.RenderBody("AlertID: {{.AlertID}}\nTitle: {{.Title}}\nSeverity: {{.Severity}}", labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
}

func matchRule(r *models.Rule, a *models.Alert, labels map[string]string) bool {
	if r.MatchSeverity != "" && r.MatchSeverity != a.Severity {
		return false
	}
	if r.DatasourceIDs != "" {
		var ids []uint
		if err := json.Unmarshal([]byte(r.DatasourceIDs), &ids); err == nil && len(ids) > 0 {
			found := false
			for _, id := range ids {
				if id == uint(a.SourceID) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	if r.MatchLabels != "" {
		var want map[string]string
		if err := json.Unmarshal([]byte(r.MatchLabels), &want); err == nil && len(want) > 0 {
			for k, v := range want {
				if labels[k] != v {
					return false
				}
			}
		}
	}
	return true
}

// inExcludeWindow returns true if current time (local) falls inside any rule exclude window.
// ExcludeWindows JSON: [{"start":"22:00","end":"08:00"}] for daily 22:00-08:00.
func inExcludeWindow(r *models.Rule) bool {
	if r.ExcludeWindows == "" {
		return false
	}
	var windows []struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}
	if err := json.Unmarshal([]byte(r.ExcludeWindows), &windows); err != nil || len(windows) == 0 {
		return false
	}
	now := time.Now()
	hm := now.Hour()*60 + now.Minute()
	for _, w := range windows {
		startMin := parseHM(w.Start)
		endMin := parseHM(w.End)
		if startMin < 0 || endMin < 0 {
			continue
		}
		if startMin <= endMin {
			if hm >= startMin && hm < endMin {
				return true
			}
		} else {
			if hm >= startMin || hm < endMin {
				return true
			}
		}
	}
	return false
}

func parseHM(s string) int {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return -1
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return -1
	}
	return h*60 + m
}

// suppressionConfig is the JSON shape for Rule.Suppression.
type suppressionConfig struct {
	SourceLabels    map[string]string `json:"source_labels"`
	SuppressedLabels map[string]string `json:"suppressed_labels"`
	Duration        string            `json:"duration"`
}

// labelsMatch returns true if alert labels contain all key-value pairs in want.
func labelsMatch(labels, want map[string]string) bool {
	for k, v := range want {
		if labels[k] != v {
			return false
		}
	}
	return len(want) > 0
}

func updateSuppressionWindow(r *models.Rule, labels map[string]string) {
	if r.Suppression == "" {
		return
	}
	var cfg suppressionConfig
	if err := json.Unmarshal([]byte(r.Suppression), &cfg); err != nil || cfg.Duration == "" {
		return
	}
	if !labelsMatch(labels, cfg.SourceLabels) {
		return
	}
	d, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		return
	}
	suppressionMu.Lock()
	suppressionWindows[r.ID] = time.Now().Add(d)
	suppressionMu.Unlock()
}

// suppressed returns true if this rule has an active suppression window and the alert matches suppressed_labels (so we skip send).
func suppressed(r *models.Rule, labels map[string]string) bool {
	if r.Suppression == "" {
		return false
	}
	var cfg suppressionConfig
	if err := json.Unmarshal([]byte(r.Suppression), &cfg); err != nil {
		return false
	}
	if len(cfg.SuppressedLabels) == 0 {
		return false
	}
	suppressionMu.RLock()
	endTime := suppressionWindows[r.ID]
	suppressionMu.RUnlock()
	if time.Now().After(endTime) {
		return false
	}
	return labelsMatch(labels, cfg.SuppressedLabels)
}

// annotationValue extracts a single string value from the alert's Annotations JSON.
func annotationValue(alert *models.Alert, key string) string {
	if alert.Annotations == "" {
		return ""
	}
	var ann map[string]string
	if err := json.Unmarshal([]byte(alert.Annotations), &ann); err != nil {
		return ""
	}
	return ann[key]
}

// recoveryAlreadySent returns true if this alert was already sent to this channel successfully in the last 2 minutes (avoids duplicate recovery when multiple rules match).
func recoveryAlreadySent(db *gorm.DB, alertID string, chID uint) bool {
	var count int64
	db.Model(&models.AlertSendRecord{}).Where("alert_id = ? AND channel_id = ? AND success = ? AND created_at > ?",
		alertID, chID, true, time.Now().Add(-2*time.Minute)).Count(&count)
	return count > 0
}

// sendRateLimited returns true if we already sent this alert (same alert_id) to this channel within rule's send_interval.
// Interval is per alert only: different alerts matching the same rule can each send; the same alert is throttled.
func sendRateLimited(db *gorm.DB, r *models.Rule, alertID string, chID uint) bool {
	if r.SendInterval == "" || r.SendInterval == "0" {
		return false
	}
	d, err := time.ParseDuration(r.SendInterval)
	if err != nil {
		return false
	}
	var count int64
	db.Model(&models.AlertSendRecord{}).Where("alert_id = ? AND channel_id = ? AND success = ? AND created_at > ?",
		alertID, chID, true, time.Now().Add(-d)).Count(&count)
	return count > 0
}

// tryCreateJiraTicket creates a Jira issue when the same alert (source_id + external_id) has been seen at least JiraAfterN times and we have not created a ticket yet.
func tryCreateJiraTicket(db *gorm.DB, r *models.Rule, alert *models.Alert, title, body string) {
	if !r.JiraEnabled || r.JiraAfterN <= 0 || r.JiraConfig == "" {
		return
	}
	var count int64
	db.Model(&models.Alert{}).Where("source_id = ? AND external_id = ?", alert.SourceID, alert.ExternalID).Count(&count)
	if count < int64(r.JiraAfterN) {
		return
	}
	var existing models.JiraCreated
	if err := db.Where("rule_id = ? AND source_id = ? AND external_id = ?", r.ID, alert.SourceID, alert.ExternalID).First(&existing).Error; err == nil {
		return // already created
	}
	var cfg jira.Config
	if err := json.Unmarshal([]byte(r.JiraConfig), &cfg); err != nil {
		log.Printf("[engine] jira config parse error rule %d: %v", r.ID, err)
		return
	}
	summary := fmt.Sprintf("[Alert] %s", title)
	if len(summary) > 255 {
		summary = summary[:252] + "..."
	}
	key, err := jira.CreateIssue(&cfg, summary, body)
	if err != nil {
		log.Printf("[engine] jira create issue rule %d: %v", r.ID, err)
		return
	}
	if err := db.Create(&models.JiraCreated{RuleID: r.ID, SourceID: alert.SourceID, ExternalID: alert.ExternalID, JiraKey: key}).Error; err != nil {
		log.Printf("[engine] jira record save error rule %d: %v", r.ID, err)
	}
}

// aggregationDimensionKeys returns label keys to exclude when computing "same type" for the given dimension.
func aggregationDimensionKeys(aggregateBy string) []string {
	switch strings.ToLower(aggregateBy) {
	case "hostname":
		return []string{"hostname", "host", "instance"}
	case "ip":
		return []string{"ip", "instance"}
	case "port":
		return []string{"port", "instance"}
	default:
		return []string{aggregateBy}
	}
}

// typeFingerprint returns a stable string for "same type" (labels minus aggregation dimension).
func typeFingerprint(labels map[string]string, aggregateBy string) string {
	exclude := aggregationDimensionKeys(aggregateBy)
	excl := make(map[string]bool)
	for _, k := range exclude {
		excl[k] = true
	}
	copied := make(map[string]string, len(labels))
	for k, v := range labels {
		if !excl[k] {
			copied[k] = v
		}
	}
	keys := make([]string, 0, len(copied))
	for k := range copied {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(copied[k])
		b.WriteString(";")
	}
	return b.String()
}

// aggregationKey extracts the dimension value from labels (e.g. hostname, ip, port).
func aggregationKey(labels map[string]string, aggregateBy string) string {
	switch strings.ToLower(aggregateBy) {
	case "hostname":
		if v := labels["hostname"]; v != "" {
			return v
		}
		if v := labels["host"]; v != "" {
			return v
		}
		if v := labels["instance"]; v != "" {
			return strings.Split(v, ":")[0]
		}
		return labels["instance"]
	case "ip":
		if v := labels["ip"]; v != "" {
			return v
		}
		return strings.Split(labels["instance"], ":")[0]
	case "port":
		if v := labels["port"]; v != "" {
			return v
		}
		if inst := labels["instance"]; inst != "" {
			if idx := strings.LastIndex(inst, ":"); idx >= 0 && idx < len(inst)-1 {
				return inst[idx+1:]
			}
		}
		return ""
	default:
		return labels[aggregateBy]
	}
}

// labelsSameType returns true if a and b match except for the aggregation dimension keys.
func labelsSameType(a, b map[string]string, aggregateBy string) bool {
	exclude := aggregationDimensionKeys(aggregateBy)
	excl := make(map[string]bool)
	for _, k := range exclude {
		excl[k] = true
	}
	for k, v := range a {
		if excl[k] {
			continue
		}
		if b[k] != v {
			return false
		}
	}
	for k := range b {
		if excl[k] {
			continue
		}
		if a[k] != b[k] {
			return false
		}
	}
	return true
}

// sendAggregated collects same-type alerts in the rule's aggregate window and sends one notification per (rule, type) per window.
func sendAggregated(db *gorm.DB, r *models.Rule, alert *models.Alert, labels map[string]string, title, body string, channelIDs []uint) {
	d, err := time.ParseDuration(r.AggregateWindow)
	if err != nil {
		d = 5 * time.Minute
	}
	since := time.Now().Add(-d)
	var candidates []models.Alert
	if err := db.Where("firing_at >= ? AND status = ?", since, "firing").Find(&candidates).Error; err != nil || len(candidates) == 0 {
		return
	}
	typeFP := typeFingerprint(labels, r.AggregateBy)
	aggKey := aggregationKey(labels, r.AggregateBy)
	if aggKey == "" {
		aggKey = alert.ID
	}
	keysSeen := map[string]bool{aggKey: true}
	for _, a := range candidates {
		if a.ID == alert.ID {
			continue
		}
		var la map[string]string
		_ = json.Unmarshal([]byte(a.Labels), &la)
		if la == nil {
			continue
		}
		if !matchRule(r, &a, la) {
			continue
		}
		if !labelsSameType(labels, la, r.AggregateBy) {
			continue
		}
		k := aggregationKey(la, r.AggregateBy)
		if k != "" {
			keysSeen[k] = true
		}
	}
	aggStateKey := fmt.Sprintf("%d_%s", r.ID, typeFP)
	aggMu.Lock()
	lastSent := aggLastSent[aggStateKey]
	aggMu.Unlock()
	if !lastSent.IsZero() && time.Since(lastSent) < d {
		return // already sent in this window
	}
	dimName := r.AggregateBy
	if dimName == "" {
		dimName = "items"
	}
	aggTitle := fmt.Sprintf("%s (%d %s)", title, len(keysSeen), dimName)
	keyList := make([]string, 0, len(keysSeen))
	for k := range keysSeen {
		keyList = append(keyList, k)
	}
	sort.Strings(keyList)
	aggBody := body + "\n\n" + dimName + " list: " + strings.Join(keyList, ", ")
	for _, chID := range channelIDs {
		if sendRateLimited(db, r, alert.ID, chID) {
			continue
		}
		var ch models.Channel
		if err := db.First(&ch, chID).Error; err != nil || !ch.Enabled {
			db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: false, Error: "channel not found or disabled"})
			continue
		}
		if err := sender.Send(ch.Type, ch.Config, aggTitle, aggBody, false); err != nil {
			log.Printf("[engine] aggregated send alert to channel %d failed: %v", chID, err)
			db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: false, Error: err.Error()})
			continue
		}
		db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: true})
	}
	aggMu.Lock()
	aggLastSent[aggStateKey] = time.Now()
	aggMu.Unlock()
}
