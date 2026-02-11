package handlers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

var (
	exportHeaders = []string{"告警ID", "数据源ID", "数据源类型", "标题", "严重程度", "状态", "告警时间", "恢复时间", "影响时长", "创建时间", "当前值/阈值"}
	locShanghai   *time.Location
)

func init() {
	locShanghai, _ = time.LoadLocation("Asia/Shanghai")
	if locShanghai == nil {
		locShanghai = time.FixedZone("CST", 8*3600)
	}
}

// formatInShanghai formats t in Asia/Shanghai (UTC+8) for report export.
func formatInShanghai(t time.Time, layout string) string {
	if t.IsZero() {
		return ""
	}
	return t.In(locShanghai).Format(layout)
}

// alertValueFromAnnotations extracts "value" from alert annotations JSON for report export.
func alertValueFromAnnotations(annotations string) string {
	if annotations == "" {
		return ""
	}
	var ann map[string]string
	if err := json.Unmarshal([]byte(annotations), &ann); err != nil {
		return ""
	}
	return ann["value"]
}

const exportTimeLayout = "2006-01-02 15:04:05"

// formatImpactDuration returns human-readable impact duration: firing => now-firing_at, resolved => resolved_at-firing_at.
func formatImpactDuration(firingAt time.Time, resolvedAt *time.Time, status string, now time.Time) string {
	if firingAt.IsZero() {
		return ""
	}
	var end time.Time
	if status == "resolved" && resolvedAt != nil && !resolvedAt.IsZero() {
		end = *resolvedAt
	} else {
		end = now
	}
	d := end.Sub(firingAt)
	if d < 0 {
		return ""
	}
	sec := int(d.Seconds())
	days := sec / 86400
	hours := (sec % 86400) / 3600
	mins := (sec % 3600) / 60
	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d天", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d小时", hours))
	}
	if mins > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d分", mins))
	}
	prefix := "持续 "
	if status == "firing" {
		prefix = "已持续 "
	}
	return prefix + joinStrings(parts, "")
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	s := parts[0]
	for i := 1; i < len(parts); i++ {
		s += sep + parts[i]
	}
	return s
}

func writeAlertsCSV(w http.ResponseWriter, list []models.Alert) {
	enc := csv.NewWriter(w)
	enc.Write([]string{"alert_id", "source_id", "source_type", "title", "severity", "status", "firing_at", "resolved_at", "影响时长", "created_at", "value"})
	now := time.Now()
	for _, a := range list {
		firingAt := formatInShanghai(a.FiringAt, exportTimeLayout)
		resolvedAt := ""
		if a.ResolvedAt != nil {
			resolvedAt = formatInShanghai(*a.ResolvedAt, exportTimeLayout)
		}
		impactDur := formatImpactDuration(a.FiringAt, a.ResolvedAt, a.Status, now)
		createdAt := formatInShanghai(a.CreatedAt, exportTimeLayout)
		value := alertValueFromAnnotations(a.Annotations)
		enc.Write([]string{a.ID, fmt.Sprintf("%d", a.SourceID), a.SourceType, a.Title, a.Severity, a.Status, firingAt, resolvedAt, impactDur, createdAt, value})
	}
	enc.Flush()
}

func writeAlertsExcel(list []models.Alert) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	sheet := "告警列表"
	idx, _ := f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")
	for i, h := range exportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	styleHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#f0f0f0"}, Pattern: 1},
	})
	_ = f.SetCellStyle(sheet, "A1", "K1", styleHeader)
	now := time.Now()
	for row, a := range list {
		firingAt := formatInShanghai(a.FiringAt, exportTimeLayout)
		resolvedAt := ""
		if a.ResolvedAt != nil {
			resolvedAt = formatInShanghai(*a.ResolvedAt, exportTimeLayout)
		}
		impactDur := formatImpactDuration(a.FiringAt, a.ResolvedAt, a.Status, now)
		createdAt := formatInShanghai(a.CreatedAt, exportTimeLayout)
		value := alertValueFromAnnotations(a.Annotations)
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row+2), a.ID)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row+2), a.SourceID)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row+2), a.SourceType)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row+2), a.Title)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row+2), a.Severity)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row+2), a.Status)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row+2), firingAt)
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row+2), resolvedAt)
		_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row+2), impactDur)
		_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", row+2), createdAt)
		_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", row+2), value)
	}
	f.SetColWidth(sheet, "A", "A", 38)
	f.SetColWidth(sheet, "B", "B", 10)
	f.SetColWidth(sheet, "C", "C", 14)
	f.SetColWidth(sheet, "D", "D", 40)
	f.SetColWidth(sheet, "E", "E", 10)
	f.SetColWidth(sheet, "F", "F", 10)
	f.SetColWidth(sheet, "G", "G", 20)
	f.SetColWidth(sheet, "H", "H", 20)
	f.SetColWidth(sheet, "I", "I", 14)
	f.SetColWidth(sheet, "J", "J", 20)
	f.SetColWidth(sheet, "K", "K", 14)
	f.SetActiveSheet(idx)
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, err
	}
	return &buf, nil
}

// ReportHandler aggregation and export for alerts.
type ReportHandler struct {
	DB *gorm.DB
}

// AggregationResult for charts.
type AggregationResult struct {
	Dimension string `json:"dimension"`
	Count     int64  `json:"count"`
}

// Trend returns alert count per hour for the last N hours (default 24), ending at current time (no truncation to hour).
// Response: { "data": [ { "hour": "2026-02-10T12:37:00Z", "count": 5 }, ... ] }
func (h *ReportHandler) Trend(c *gin.Context) {
	hours := 24
	if n := c.Query("hours"); n != "" {
		if v, err := parseIntDefault(n, 24); err == nil && v > 0 && v <= 168 {
			hours = v
		}
	}
	now := time.Now().UTC()
	from := now.Add(-time.Duration(hours) * time.Hour)

	var data []gin.H
	for i := 0; i < hours; i++ {
		bucketStart := from.Add(time.Duration(i) * time.Hour)
		bucketEnd := bucketStart.Add(time.Hour)
		if bucketEnd.After(now) {
			bucketEnd = now
		}
		var count int64
		// Use firing_at (when alert started firing) instead of created_at which can be
		// corrupted to zero by GORM Save. This also matches reports/export filter behaviour.
		h.DB.Model(&models.Alert{}).Where("firing_at >= ? AND firing_at < ?", bucketStart, bucketEnd).Count(&count)
		data = append(data, gin.H{"hour": bucketStart.Format(time.RFC3339), "count": count})
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func parseIntDefault(s string, defaultVal int) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return defaultVal, err
	}
	return n, nil
}

// Aggregate returns counts by time, datasource, or severity.
func (h *ReportHandler) Aggregate(c *gin.Context) {
	groupBy := c.Query("group_by") // time, datasource, severity
	from := c.Query("from")
	to := c.Query("to")
	var fromT, toT time.Time
	if from != "" {
		fromT, _ = time.Parse(time.RFC3339, from)
	} else {
		fromT = time.Now().AddDate(0, 0, -7)
	}
	if to != "" {
		toT, _ = time.Parse(time.RFC3339, to)
	} else {
		toT = time.Now()
	}

	// Use firing_at so aggregate matches "alerts that fired in this range" (same as export and alert history)
	q := h.DB.Model(&models.Alert{}).Where("firing_at >= ? AND firing_at <= ?", fromT, toT)
	var results []AggregationResult
	switch groupBy {
	case "severity":
		var rows []struct {
			Severity string
			Count    int64
		}
		q.Select("severity as severity, count(*) as count").Group("severity").Scan(&rows)
		for _, r := range rows {
			results = append(results, AggregationResult{Dimension: r.Severity, Count: r.Count})
		}
	case "datasource", "source_id":
		var rows []struct {
			SourceID uint
			Count    int64
		}
		q.Select("source_id as source_id, count(*) as count").Group("source_id").Scan(&rows)
		for _, r := range rows {
			results = append(results, AggregationResult{Dimension: fmt.Sprintf("%d", r.SourceID), Count: r.Count})
		}
	case "time", "day":
		var rows []struct {
			Day   string
			Count int64
		}
		q.Select("date(firing_at) as day, count(*) as count").Group("day").Scan(&rows)
		for _, r := range rows {
			results = append(results, AggregationResult{Dimension: r.Day, Count: r.Count})
		}
	default:
		var total int64
		q.Count(&total)
		results = []AggregationResult{{Dimension: "total", Count: total}}
	}
	c.JSON(http.StatusOK, gin.H{"data": results})
}

// Preview returns paginated alert list + summary stats for the selected date range.
// Used by the reports UI to show a data table before exporting.
func (h *ReportHandler) Preview(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	status := c.Query("status")
	severity := c.Query("severity")

	var page, pageSize int
	if p := c.Query("page"); p != "" {
		_, _ = fmt.Sscanf(p, "%d", &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		_, _ = fmt.Sscanf(ps, "%d", &pageSize)
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	q := h.DB.Model(&models.Alert{})
	if from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("firing_at >= ?", t)
		}
	}
	if to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("firing_at <= ?", t)
		}
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}

	// Total count
	var total int64
	q.Count(&total)

	// Summary stats (severity + status breakdown)
	var sevRows []struct {
		Severity string
		Count    int64
	}
	q.Session(&gorm.Session{NewDB: false}).Select("severity, count(*) as count").Group("severity").Scan(&sevRows)

	var statusRows []struct {
		Status string
		Count  int64
	}
	q.Session(&gorm.Session{NewDB: false}).Select("status, count(*) as count").Group("status").Scan(&statusRows)

	sevMap := make(map[string]int64)
	for _, r := range sevRows {
		sevMap[r.Severity] = r.Count
	}
	statusMap := make(map[string]int64)
	for _, r := range statusRows {
		statusMap[r.Status] = r.Count
	}

	// Paginated list
	var list []models.Alert
	offset := (page - 1) * pageSize
	q.Order("firing_at desc").Offset(offset).Limit(pageSize).Find(&list)

	now := time.Now()
	alerts := make([]gin.H, 0, len(list))
	for _, a := range list {
		duration := formatImpactDuration(a.FiringAt, a.ResolvedAt, a.Status, now)
		value := alertValueFromAnnotations(a.Annotations)
		alerts = append(alerts, gin.H{
			"alert_id":        a.ID,
			"title":           a.Title,
			"severity":        a.Severity,
			"status":          a.Status,
			"firing_at":       formatInShanghai(a.FiringAt, exportTimeLayout),
			"resolved_at":     func() string { if a.ResolvedAt != nil { return formatInShanghai(*a.ResolvedAt, exportTimeLayout) }; return "" }(),
			"impact_duration": duration,
			"value":           value,
			"labels":          a.Labels,
			"source_type":     a.SourceType,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"alerts":   alerts,
		"summary": gin.H{
			"severity": sevMap,
			"status":   statusMap,
		},
	})
}

// Export alerts as JSON or CSV based on format= query (default json).
func (h *ReportHandler) Export(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	format := c.Query("format")
	if format == "" {
		format = "json"
	}
	// Filter by firing_at so export matches "alerts that fired in this range" (same as alert history semantics)
	q := h.DB.Model(&models.Alert{})
	if from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("firing_at >= ?", t)
		}
	}
	if to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("firing_at <= ?", t)
		}
	}
	var list []models.Alert
	if err := q.Order("firing_at desc, created_at desc").Limit(10000).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dateStr := time.Now().UTC().Format("2006-01-02")
	if format == "csv" {
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=alerts-"+dateStr+".csv")
		writeAlertsCSV(c.Writer, list)
		return
	}
	if format == "xlsx" || format == "excel" {
		buf, err := writeAlertsExcel(list)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename=alerts-"+dateStr+".xlsx")
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=alerts-"+dateStr+".json")
	now := time.Now()
	out := make([]map[string]interface{}, 0, len(list))
	for _, a := range list {
		b, _ := json.Marshal(a)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		m["firing_at"] = formatInShanghai(a.FiringAt, exportTimeLayout)
		if a.ResolvedAt != nil {
			m["resolved_at"] = formatInShanghai(*a.ResolvedAt, exportTimeLayout)
		} else {
			m["resolved_at"] = ""
		}
		m["impact_duration"] = formatImpactDuration(a.FiringAt, a.ResolvedAt, a.Status, now)
		m["created_at"] = formatInShanghai(a.CreatedAt, exportTimeLayout)
		out = append(out, m)
	}
	c.JSON(http.StatusOK, out)
}
