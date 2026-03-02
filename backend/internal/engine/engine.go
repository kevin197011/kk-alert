package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/kk-alert/backend/internal/cache"
	"github.com/kk-alert/backend/internal/models"
	"github.com/kk-alert/backend/internal/sender"
	"gorm.io/gorm"
)

var suppressionMu sync.RWMutex
var suppressionWindows = make(map[uint]time.Time)

var aggMu sync.RWMutex
var aggLastSent = make(map[string]time.Time)

func stripSystemAlertPrefix(s string) string {
	return strings.TrimSpace(strings.TrimPrefix(s, "【系统告警】"))
}

func IsSilenced(db *gorm.DB, alertID string) bool {
	var n int64
	db.Model(&models.AlertSilence{}).Where("alert_id = ? AND silence_until > ?", alertID, time.Now()).Count(&n)
	return n > 0
}

type alertJob struct {
	db    *gorm.DB
	alert models.Alert
}

var alertQueue = make(chan alertJob, 2000)

func init() {
	const numWorkers = 32
	for i := 0; i < numWorkers; i++ {
		go func() {
			for job := range alertQueue {
				ProcessAlert(job.db, &job.alert)
			}
		}()
	}
}

func ProcessAlertAsync(db *gorm.DB, alert *models.Alert) {
	a := *alert
	freshDB := db.Session(&gorm.Session{NewDB: true})
	select {
	case alertQueue <- alertJob{db: freshDB, alert: a}:
	default:
		log.Printf("[engine] alert queue full, processing inline for %s", a.ID)
		go ProcessAlert(freshDB, &a)
	}
}

func ProcessAlert(db *gorm.DB, alert *models.Alert) {
	if IsSilenced(db, alert.ID) {
		return
	}

	var rules []models.Rule
	if cached := cache.Rules.Get(); cached != nil {
		rules = cached
	} else {
		if err := db.Where("enabled = ?", true).Order("priority asc").Find(&rules).Error; err != nil {
			return
		}
		cache.Rules.Set(rules)
	}

	var labels map[string]string
	_ = json.Unmarshal([]byte(alert.Labels), &labels)
	if labels == nil {
		labels = make(map[string]string)
	}

	for _, r := range rules {
		updateSuppressionWindow(&r, labels)

		if !matchRule(&r, alert, labels) {
			continue
		}

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

		if alert.Status == "resolved" && r.RecoveryNotify {
			title := ""
			sendAt := time.Now()
			body := resolveBody(db, &r, alert, labels, true, sendAt) + "\n\n发送时间: " + formatSendTime(sendAt)
			for _, chID := range channelIDs {
				if recoveryAlreadySent(db, alert.ID, chID) {
					continue
				}
				var ch models.Channel
				if cachedCh, ok := cache.Channels.Get(chID); ok {
					ch = cachedCh
				} else {
					if err := db.First(&ch, chID).Error; err != nil || !ch.Enabled {
						continue
					}
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
				if cachedCh, ok := cache.Channels.Get(chID); ok {
					ch = cachedCh
				} else {
					if err := db.First(&ch, chID).Error; err != nil || !ch.Enabled {
						db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: false, Error: "channel not found or disabled"})
						continue
					}
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

func annotationValue(alert *models.Alert, key string) string {
	var annotations map[string]string
	_ = json.Unmarshal([]byte(alert.Annotations), &annotations)
	return annotations[key]
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

func durationSatisfied(r *models.Rule, a *models.Alert) bool {
	if r.Duration == "" || r.Duration == "0" {
		return true
	}
	d, err := time.ParseDuration(r.Duration)
	if err != nil {
		return true
	}
	return time.Since(a.FiringAt) >= d
}

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
	fmt.Sscanf(s, "%d:%d", &h, &m)
	return h*60 + m
}

func suppressed(r *models.Rule, labels map[string]string) bool {
	if r.Suppression == "" {
		return false
	}
	var suppression struct {
		Enabled  bool   `json:"enabled"`
		Duration int    `json:"duration"`
		Labels   string `json:"labels"`
	}
	_ = json.Unmarshal([]byte(r.Suppression), &suppression)
	if !suppression.Enabled {
		return false
	}

	suppressionLabels := make(map[string]string)
	for _, pair := range strings.Split(suppression.Labels, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			suppressionLabels[parts[0]] = parts[1]
		}
	}

	for k, v := range suppressionLabels {
		if labels[k] != v {
			return false
		}
	}

	suppressionMu.RLock()
	endTime, ok := suppressionWindows[r.ID]
	suppressionMu.RUnlock()

	if ok && time.Now().Before(endTime) {
		return true
	}

	return false
}

func updateSuppressionWindow(r *models.Rule, labels map[string]string) {
	if r.Suppression == "" {
		return
	}
	var suppression struct {
		Enabled  bool   `json:"enabled"`
		Duration int    `json:"duration"`
		Labels   string `json:"labels"`
	}
	_ = json.Unmarshal([]byte(r.Suppression), &suppression)
	if !suppression.Enabled || suppression.Duration == 0 {
		return
	}

	suppressionLabels := make(map[string]string)
	for _, pair := range strings.Split(suppression.Labels, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			suppressionLabels[parts[0]] = parts[1]
		}
	}

	match := true
	for k, v := range suppressionLabels {
		if labels[k] != v {
			match = false
			break
		}
	}

	if match {
		suppressionMu.Lock()
		suppressionWindows[r.ID] = time.Now().Add(time.Duration(suppression.Duration) * time.Second)
		suppressionMu.Unlock()
	}
}

func sendRateLimited(db *gorm.DB, r *models.Rule, alertID string, channelID uint) bool {
	if r.SendInterval == "" {
		return false
	}
	sendInterval, err := time.ParseDuration(r.SendInterval)
	if err != nil {
		return false
	}
	var count int64
	db.Model(&models.AlertSendRecord{}).
		Where("alert_id = ? AND channel_id = ? AND success = ? AND created_at > ?",
			alertID, channelID, true, time.Now().Add(-sendInterval)).
		Count(&count)
	return count > 0
}

func recoveryAlreadySent(db *gorm.DB, alertID string, channelID uint) bool {
	var count int64
	db.Model(&models.AlertSendRecord{}).
		Where("alert_id = ? AND channel_id = ? AND success = ?", alertID, channelID, true).
		Count(&count)
	return count > 0
}

func sendAggregated(db *gorm.DB, r *models.Rule, alert *models.Alert, labels map[string]string, title, body string, channelIDs []uint) {
	var aggKeys []string
	_ = json.Unmarshal([]byte(r.AggregateBy), &aggKeys)

	fingerprintParts := []string{fmt.Sprintf("rule_%d", r.ID)}
	for _, key := range aggKeys {
		if val, ok := labels[key]; ok {
			fingerprintParts = append(fingerprintParts, fmt.Sprintf("%s=%s", key, val))
		}
	}
	fingerprint := strings.Join(fingerprintParts, ",")

	window, _ := time.ParseDuration(r.AggregateWindow)
	if window <= 0 {
		window = 5 * time.Minute
	}

	aggMu.Lock()
	lastSent, ok := aggLastSent[fingerprint]
	shouldSend := !ok || time.Since(lastSent) >= window
	if shouldSend {
		aggLastSent[fingerprint] = time.Now()
	}
	aggMu.Unlock()

	if !shouldSend {
		return
	}

	for _, chID := range channelIDs {
		var ch models.Channel
		if cachedCh, ok := cache.Channels.Get(chID); ok {
			ch = cachedCh
		} else {
			if err := db.First(&ch, chID).Error; err != nil || !ch.Enabled {
				continue
			}
		}
		if err := sender.Send(ch.Type, ch.Config, title, body, false); err != nil {
			log.Printf("[engine] aggregated send alert %s to channel %d failed: %v", alert.ID, chID, err)
			db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: false, Error: err.Error()})
		} else {
			db.Create(&models.AlertSendRecord{AlertID: alert.ID, ChannelID: chID, Success: true})
		}
	}
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
	if data.Description == "" && r.Description != "" {
		data.Description = r.Description
	}

	// Try rule's template from cache first
	if r.TemplateID != nil && *r.TemplateID != 0 {
		if tpl, ok := cache.Templates.Get(*r.TemplateID); ok && tpl.Body != "" {
			out, err := sender.RenderTemplate(tpl.Body, data)
			if err == nil {
				return out
			}
			log.Printf("[engine] template render failed, using simple replace: %v", err)
			return sender.RenderBody(tpl.Body, labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
		}
		// Cache miss - load from DB and refresh cache
		var t models.Template
		db.Where("id = ?", *r.TemplateID).Limit(1).Find(&t)
		if t.ID != 0 && t.Body != "" {
			var allTemplates []models.Template
			db.Find(&allTemplates)
			cache.Templates.Set(allTemplates)

			out, err := sender.RenderTemplate(t.Body, data)
			if err == nil {
				return out
			}
			return sender.RenderBody(t.Body, labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
		}
		if t.ID == 0 {
			log.Printf("[engine] template id=%d not found (rule %d), falling back to default template", *r.TemplateID, r.ID)
			if defaultT, ok := cache.Templates.GetDefault(); ok {
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

	// Fallback: load default template from cache
	if defaultT, ok := cache.Templates.GetDefault(); ok && defaultT.Body != "" {
		out, err := sender.RenderTemplate(defaultT.Body, data)
		if err == nil {
			return out
		}
		return sender.RenderBody(defaultT.Body, labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
	}

	// Fallback: load from DB and refresh cache
	var defaultT models.Template
	db.Where("is_default = ?", true).Limit(1).Find(&defaultT)
	if defaultT.ID != 0 && defaultT.Body != "" {
		var allTemplates []models.Template
		db.Find(&allTemplates)
		cache.Templates.Set(allTemplates)

		out, err := sender.RenderTemplate(defaultT.Body, data)
		if err == nil {
			return out
		}
		return sender.RenderBody(defaultT.Body, labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
	}
	return sender.RenderBody("AlertID: {{.AlertID}}\nTitle: {{.Title}}\nSeverity: {{.Severity}}", labels, alert.ID, stripSystemAlertPrefix(alert.Title), alert.Severity)
}

func tryCreateJiraTicket(db *gorm.DB, r *models.Rule, alert *models.Alert, title, body string) {
	if !r.JiraEnabled || r.JiraAfterN == 0 {
		return
	}
	var sendCount int64
	db.Model(&models.AlertSendRecord{}).Where("alert_id = ? AND success = ?", alert.ID, true).Count(&sendCount)
	if int(sendCount) < r.JiraAfterN {
		return
	}
	var count int64
	db.Model(&models.JiraCreated{}).Where("alert_id = ?", alert.ID).Count(&count)
	if count > 0 {
		return
	}
	log.Printf("[engine] would create Jira ticket for alert %s after %d sends", alert.ID, sendCount)
}
