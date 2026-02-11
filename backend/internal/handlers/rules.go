package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"github.com/kk-alert/backend/internal/query"
	"github.com/kk-alert/backend/internal/scheduler"
	"gorm.io/gorm"
)

// clearEmptyTimeFields removes created_at/updated_at/last_run_at when they are empty string
// so that JSON unmarshal into time.Time does not fail.
func clearEmptyTimeFields(m map[string]interface{}) {
	for _, key := range []string{"created_at", "updated_at", "last_run_at"} {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok && s == "" {
				delete(m, key)
			}
		}
	}
}

// normalizeRuleTemplateID ensures template_id in the map is a number so it unmarshals into Rule.TemplateID (*uint).
func normalizeRuleTemplateID(m map[string]interface{}) {
	v, ok := m["template_id"]
	if !ok || v == nil {
		return
	}
	switch x := v.(type) {
	case float64:
		m["template_id"] = uint(x)
	case string:
		if x == "" {
			delete(m, "template_id")
			return
		}
		n, err := strconv.ParseUint(x, 10, 64)
		if err == nil {
			m["template_id"] = uint(n)
		}
	}
}

// RuleHandler CRUD, import/export, batch for rules.
type RuleHandler struct {
	DB        *gorm.DB
	Scheduler *scheduler.Scheduler // optional; when set, RunRuleNow is called after create/update
}

// stripJiraConfig clears JiraConfig so it is not returned to the client.
func stripJiraConfig(r *models.Rule) { r.JiraConfig = "" }

// List rules. Returns { "rules": [...], "firing_counts": { "ruleId": count } } so UI can show red/green per rule.
// Query: name — fuzzy match on rule name (LIKE %name%).
func (h *RuleHandler) List(c *gin.Context) {
	name := strings.TrimSpace(c.Query("name"))
	q := h.DB.Model(&models.Rule{})
	if name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	var list []models.Rule
	if err := q.Order("priority asc, id asc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for i := range list {
		stripJiraConfig(&list[i])
	}
	out := gin.H{"rules": list}
	if h.Scheduler != nil {
		counts := scheduler.FiringCountByRule()
		countsStr := make(map[string]*scheduler.SeverityCounts, len(counts))
		for id, sc := range counts {
			countsStr[strconv.FormatUint(uint64(id), 10)] = sc
		}
		out["firing_counts"] = countsStr
	} else {
		out["firing_counts"] = map[string]*scheduler.SeverityCounts{}
	}
	c.JSON(http.StatusOK, out)
}

// Get by ID.
func (h *RuleHandler) Get(c *gin.Context) {
	var r models.Rule
	if err := h.DB.First(&r, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	stripJiraConfig(&r)
	c.JSON(http.StatusOK, r)
}

// Create rule.
func (h *RuleHandler) Create(c *gin.Context) {
	var m map[string]interface{}
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	normalizeRuleTemplateID(m)
	b, _ := json.Marshal(m)
	var r models.Rule
	if err := json.Unmarshal(b, &r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.DB.Create(&r).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.Scheduler != nil {
		h.Scheduler.RunRuleNow(r.ID)
	}
	stripJiraConfig(&r)
	c.JSON(http.StatusCreated, r)
}

// Update rule.
func (h *RuleHandler) Update(c *gin.Context) {
	var r models.Rule
	if err := h.DB.First(&r, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var m map[string]interface{}
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	normalizeRuleTemplateID(m)
	if jira, ok := m["jira_config"]; !ok || jira == "" {
		m["jira_config"] = r.JiraConfig
	}
	b, _ := json.Marshal(m)
	var body models.Rule
	if err := json.Unmarshal(b, &body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	body.ID = r.ID
	if err := h.DB.Save(&body).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.Scheduler != nil {
		h.Scheduler.RunRuleNow(body.ID)
	}
	stripJiraConfig(&body)
	c.JSON(http.StatusOK, body)
}

// Trigger runs a rule immediately (manual trigger from UI).
func (h *RuleHandler) Trigger(c *gin.Context) {
	var r models.Rule
	if err := h.DB.First(&r, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if !r.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule is disabled"})
		return
	}
	if r.QueryExpression == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rule has no query expression"})
		return
	}
	if h.Scheduler != nil {
		h.Scheduler.RunRuleNow(r.ID)
		c.JSON(http.StatusOK, gin.H{"ok": true, "message": "rule triggered"})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "scheduler not available"})
	}
}

// Delete rule.
func (h *RuleHandler) Delete(c *gin.Context) {
	if err := h.DB.Delete(&models.Rule{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// BatchRequest for enable/disable/delete.
type BatchRequest struct {
	IDs    []uint `json:"ids" binding:"required"`
	Action string `json:"action" binding:"required"` // enable, disable, delete
}

// Batch updates rules.
func (h *RuleHandler) Batch(c *gin.Context) {
	var req BatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var ok, fail int
	switch req.Action {
	case "enable":
		res := h.DB.Model(&models.Rule{}).Where("id IN ?", req.IDs).Update("enabled", true)
		ok = int(res.RowsAffected)
		fail = len(req.IDs) - ok
	case "disable":
		res := h.DB.Model(&models.Rule{}).Where("id IN ?", req.IDs).Update("enabled", false)
		ok = int(res.RowsAffected)
		fail = len(req.IDs) - ok
	case "delete":
		res := h.DB.Delete(&models.Rule{}, req.IDs)
		ok = int(res.RowsAffected)
		fail = len(req.IDs) - ok
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": ok, "failed": fail})
}

// ExportBody optional body with ids.
type ExportBody struct {
	IDs []uint `json:"ids"`
}

// Export returns selected rules as JSON.
func (h *RuleHandler) Export(c *gin.Context) {
	var body ExportBody
	_ = c.ShouldBindJSON(&body)
	var list []models.Rule
	q := h.DB.Model(&models.Rule{})
	if len(body.IDs) > 0 {
		q = q.Where("id IN ?", body.IDs)
	}
	if err := q.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Strip JiraConfig for export
	out := make([]map[string]interface{}, 0, len(list))
	for _, r := range list {
		b, _ := json.Marshal(r)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		delete(m, "jira_config")
		out = append(out, m)
	}
	c.JSON(http.StatusOK, gin.H{"rules": out})
}

// ImportRequest for rule import. Rules are raw maps so empty time strings do not break unmarshal.
type ImportRequest struct {
	Rules []map[string]interface{} `json:"rules" binding:"required"`
	Mode  string                  `json:"mode"` // add, overwrite
}

// Import creates or updates rules from JSON.
func (h *RuleHandler) Import(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Mode == "" {
		req.Mode = "add"
	}
	var imported, failed int
	for _, ruleMap := range req.Rules {
		clearEmptyTimeFields(ruleMap)
		normalizeRuleTemplateID(ruleMap)
		delete(ruleMap, "id")
		b, _ := json.Marshal(ruleMap)
		var r models.Rule
		if err := json.Unmarshal(b, &r); err != nil {
			failed++
			continue
		}
		r.ID = 0
		if err := h.DB.Create(&r).Error; err != nil {
			failed++
			continue
		}
		imported++
	}
	c.JSON(http.StatusOK, gin.H{"imported": imported, "failed": failed})
}

// TestMatchRequest for testing rule match.
type TestMatchRequest struct {
	DatasourceIDs   string `json:"datasource_ids"`
	QueryLanguage   string `json:"query_language"`
	QueryExpression string `json:"query_expression"`
	MatchLabels     string `json:"match_labels"`
	MatchSeverity   string `json:"match_severity"`
	Thresholds      string `json:"thresholds"` // JSON array of multi-level thresholds
}

// TestMatchResponse for test match result.
type TestMatchResponse struct {
	Matched                    bool           `json:"matched"`
	TotalAlerts                int            `json:"total_alerts"`
	MatchedAlerts              []MatchedAlert `json:"matched_alerts"`
	Message                    string         `json:"message"`
	AlertsFromSelectedDS       int            `json:"alerts_from_selected_datasource,omitempty"`
	AlertsWithSelectedSeverity int           `json:"alerts_with_selected_severity,omitempty"`
	RawSeriesCount             int            `json:"raw_series_count,omitempty"` // total series from PromQL before value filter
}

// MatchedAlert represents a matched alert in test.
type MatchedAlert struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Severity string            `json:"severity"`
	Labels   map[string]string `json:"labels"`
	Status   string            `json:"status"`
	Value    float64           `json:"value"` // metric value for threshold display
}

// TestMatch runs PromQL (or other query) on selected datasources in real time and returns
// matching series. No DB fallback — always queries datasources.
func (h *RuleHandler) TestMatch(c *gin.Context) {
	var req TestMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := &models.Rule{
		DatasourceIDs:   req.DatasourceIDs,
		QueryLanguage:   req.QueryLanguage,
		QueryExpression: req.QueryExpression,
		MatchLabels:     req.MatchLabels,
		MatchSeverity:   req.MatchSeverity,
		Thresholds:      req.Thresholds,
	}

	var dsIDs []uint
	_ = json.Unmarshal([]byte(rule.DatasourceIDs), &dsIDs)
	expr := strings.TrimSpace(rule.QueryExpression)

	if expr == "" || len(dsIDs) == 0 {
		c.JSON(http.StatusOK, TestMatchResponse{
			Matched:       false,
			TotalAlerts:   0,
			MatchedAlerts: nil,
			Message:       "请选择数据源并填写 PromQL 表达式后再测试匹配。",
		})
		return
	}
	if rule.QueryLanguage != "" && rule.QueryLanguage != "promql" {
		c.JSON(http.StatusOK, TestMatchResponse{
			Matched:       false,
			TotalAlerts:   0,
			MatchedAlerts: nil,
			Message:       "当前仅支持 PromQL 的测试匹配，请选择查询语言为 PromQL。",
		})
		return
	}

	matched, total, rawSeries, message, fromDS, withSev, err := h.runTestMatchPromQL(c.Request.Context(), rule, dsIDs)
	if err != nil {
		c.JSON(http.StatusOK, TestMatchResponse{
			Matched:       false,
			TotalAlerts:   0,
			MatchedAlerts: nil,
			Message:       "执行 PromQL 失败: " + err.Error(),
		})
		return
	}
	if len(matched) > 0 {
		message = "匹配成功"
	}
	c.JSON(http.StatusOK, TestMatchResponse{
		Matched:                    len(matched) > 0,
		TotalAlerts:                total,
		MatchedAlerts:              matched,
		Message:                    message,
		AlertsFromSelectedDS:       fromDS,
		AlertsWithSelectedSeverity: withSev,
		RawSeriesCount:             rawSeries,
	})
}

// runTestMatchPromQL runs PromQL on each selected Prometheus/VictoriaMetrics datasource and returns
// synthetic matched alerts. When thresholds are configured, applies threshold filtering and assigns
// severity per level — mirroring the real scheduler evaluation.
func (h *RuleHandler) runTestMatchPromQL(ctx context.Context, rule *models.Rule, dsIDs []uint) (
	matched []MatchedAlert, total int, rawSeriesCount int, message string, fromDS int, withSev int, err error,
) {
	thresholds := scheduler.ParseThresholds(rule.Thresholds)
	var allCandidates []MatchedAlert
	var lastErr error
	for _, id := range dsIDs {
		var ds models.Datasource
		if err := h.DB.First(&ds, id).Error; err != nil {
			lastErr = fmt.Errorf("数据源 %d 不存在", id)
			continue
		}
		if ds.Type != "prometheus" && ds.Type != "victoriametrics" {
			lastErr = fmt.Errorf("数据源 %d 类型 %s 不支持 PromQL 测试", id, ds.Type)
			continue
		}
		client := query.NewPrometheusClient(ds.Endpoint)
		result, qerr := client.Query(ctx, rule.QueryExpression)
		if qerr != nil {
			lastErr = qerr
			continue
		}
		for _, r := range result.Data.Result {
			rawSeriesCount++
			labels := r.Metric
			if labels == nil {
				labels = make(map[string]string)
			}
			value := query.GetValue(r.Value)

			severity := rule.MatchSeverity
			if severity == "" {
				severity = "warning"
			}

			// Apply multi-level threshold matching (same logic as scheduler)
			if thresholds != nil {
				m := scheduler.MatchThreshold(thresholds, value)
				if m == nil {
					// Value below all thresholds — skip (normal)
					continue
				}
				severity = m.Severity
				if severity == "" {
					severity = "warning"
				}
			}

			total++
			title := formatMetricForTest(r.Metric)
			alertID := generateFingerprintForTest(r.Metric, id)
			allCandidates = append(allCandidates, MatchedAlert{
				ID:       alertID,
				Title:    title,
				Severity: severity,
				Labels:   labels,
				Status:   "firing",
				Value:    value,
			})
		}
	}
	if total == 0 && lastErr != nil {
		return nil, 0, rawSeriesCount, "", 0, 0, lastErr
	}
	fromDS = total
	withSev = 0
	for _, a := range allCandidates {
		if rule.MatchSeverity == "" || rule.MatchSeverity == a.Severity {
			withSev++
		}
		if !matchLabelsForTest(rule.MatchLabels, a.Labels) {
			continue
		}
		if rule.MatchSeverity != "" && rule.MatchSeverity != a.Severity {
			continue
		}
		matched = append(matched, a)
	}
	if rawSeriesCount == 0 {
		message = "PromQL 返回 0 条结果（无满足条件的序列）。请检查表达式与数据源。"
	} else if len(matched) == 0 {
		if thresholds != nil {
			message = fmt.Sprintf("PromQL 返回 %d 条序列，经阈值过滤后 %d 条匹配，但无符合标签/严重程度过滤的告警。", rawSeriesCount, total)
		} else {
			message = fmt.Sprintf("PromQL 返回 %d 条序列，但无符合「严重程度/标签」过滤的告警。请调整匹配条件。", rawSeriesCount)
		}
	} else {
		if thresholds != nil {
			message = fmt.Sprintf("PromQL 返回 %d 条序列，经阈值过滤后 %d 条匹配。", rawSeriesCount, len(matched))
		} else {
			message = fmt.Sprintf("PromQL 返回 %d 条序列，其中 %d 条符合规则。", rawSeriesCount, len(matched))
		}
	}
	return matched, total, rawSeriesCount, message, fromDS, withSev, nil
}

func formatMetricForTest(metric map[string]string) string {
	if len(metric) == 0 {
		return "(无标签)"
	}
	parts := make([]string, 0, len(metric))
	for k, v := range metric {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ", ")
}

func generateFingerprintForTest(metric map[string]string, dsID uint) string {
	return fmt.Sprintf("test-%d-%s", dsID, formatMetricForTest(metric))
}

func matchLabelsForTest(matchLabelsJSON string, labels map[string]string) bool {
	if matchLabelsJSON == "" || matchLabelsJSON == "{}" {
		return true
	}
	var want map[string]string
	if err := json.Unmarshal([]byte(matchLabelsJSON), &want); err != nil || len(want) == 0 {
		return true
	}
	for k, v := range want {
		if labels[k] != v {
			return false
		}
	}
	return true
}

