package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// AlertHandler query and detail for alert history.
type AlertHandler struct {
	DB *gorm.DB
}

// AlertListItem is an alert with notification send counts for list API.
type AlertListItem struct {
	models.Alert
	NotifySuccessCount int `json:"notify_success_count"`
	NotifyFailCount    int `json:"notify_fail_count"`
}

// List alerts with filters and pagination.
func (h *AlertHandler) List(c *gin.Context) {
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
	q = applyAlertFilters(q, c)
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}
	var total int64
	q.Count(&total)
	var list []models.Alert
	offset := (page - 1) * pageSize
	if err := q.Order("firing_at desc, created_at desc").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Aggregate notification success/fail counts per alert
	notifyCounts := make(map[string]struct{ Success, Fail int })
	if len(list) > 0 {
		ids := make([]string, 0, len(list))
		for _, a := range list {
			ids = append(ids, a.ID)
		}
		var recs []struct {
			AlertID string
			Success bool
			Count   int64
		}
		h.DB.Model(&models.AlertSendRecord{}).
			Select("alert_id, success, count(*) as count").
			Where("alert_id in ?", ids).
			Group("alert_id, success").
			Scan(&recs)
		for _, r := range recs {
			c := notifyCounts[r.AlertID]
			if r.Success {
				c.Success += int(r.Count)
			} else {
				c.Fail += int(r.Count)
			}
			notifyCounts[r.AlertID] = c
		}
	}

	items := make([]AlertListItem, 0, len(list))
	for _, a := range list {
		c := notifyCounts[a.ID]
		items = append(items, AlertListItem{
			Alert:              a,
			NotifySuccessCount: c.Success,
			NotifyFailCount:    c.Fail,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// NotifyTotal returns total count of notification send records (all channels, success + fail).
func (h *AlertHandler) NotifyTotal(c *gin.Context) {
	var n int64
	if err := h.DB.Model(&models.AlertSendRecord{}).Count(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"total": n})
}

// Export alerts matching current filters as Excel file.
func (h *AlertHandler) Export(c *gin.Context) {
	q := h.DB.Model(&models.Alert{})
	q = applyAlertFilters(q, c)

	var list []models.Alert
	if err := q.Order("firing_at desc, created_at desc").Limit(10000).Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ef, err := writeAlertExportExcel(list)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var buf bytes.Buffer
	if _, err := ef.WriteTo(&buf); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dateStr := time.Now().Format("2006-01-02")
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=alerts-"+dateStr+".xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// applyAlertFilters applies common alert query filters from request params.
func applyAlertFilters(q *gorm.DB, c *gin.Context) *gorm.DB {
	if id := c.Query("alert_id"); id != "" {
		q = q.Where("id LIKE ?", "%"+strings.TrimSpace(id)+"%")
	}
	if title := c.Query("title"); title != "" {
		q = q.Where("title LIKE ?", "%"+strings.TrimSpace(title)+"%")
	}
	if ds := c.Query("datasource_id"); ds != "" {
		q = q.Where("source_id = ?", ds)
	}
	if sev := c.Query("severity"); sev != "" {
		q = q.Where("severity = ?", sev)
	}
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	return q
}

// writeAlertExportExcel generates an Excel file from alert list.
func writeAlertExportExcel(list []models.Alert) (*excelize.File, error) {
	f := excelize.NewFile()
	sheet := "告警列表"
	idx, _ := f.NewSheet(sheet)
	f.DeleteSheet("Sheet1")

	headers := []string{"告警ID", "数据源ID", "数据源类型", "标题", "告警值", "严重程度", "状态", "标签", "告警时间", "恢复时间", "影响时长", "创建时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	styleHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#f0f0f0"}, Pattern: 1},
	})
	lastCol, _ := excelize.CoordinatesToCellName(len(headers), 1)
	_ = f.SetCellStyle(sheet, "A1", lastCol, styleHeader)

	now := time.Now()
	loc := time.FixedZone("CST", 8*3600)
	fmtTime := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.In(loc).Format("2006-01-02 15:04:05")
	}
	fmtDuration := func(a models.Alert) string {
		if a.FiringAt.IsZero() {
			return ""
		}
		end := now
		if a.Status == "resolved" && a.ResolvedAt != nil {
			end = *a.ResolvedAt
		}
		d := end.Sub(a.FiringAt)
		if d < 0 {
			return ""
		}
		totalSec := int(d.Seconds())
		days := totalSec / 86400
		hours := (totalSec % 86400) / 3600
		minutes := (totalSec % 3600) / 60
		parts := []string{}
		if days > 0 {
			parts = append(parts, fmt.Sprintf("%d天", days))
		}
		if hours > 0 {
			parts = append(parts, fmt.Sprintf("%d小时", hours))
		}
		if minutes > 0 || len(parts) == 0 {
			parts = append(parts, fmt.Sprintf("%d分", minutes))
		}
		return strings.Join(parts, "")
	}

	// extractAnnotationValue extracts the "value" field from the annotations JSON.
	extractValue := func(ann string) string {
		if ann == "" {
			return ""
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(ann), &m); err != nil {
			return ""
		}
		if v, ok := m["value"]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	// formatLabels converts label JSON to a readable key=value string.
	formatLabels := func(raw string) string {
		if raw == "" {
			return ""
		}
		var m map[string]string
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return raw
		}
		parts := make([]string, 0, len(m))
		for k, v := range m {
			parts = append(parts, k+"="+v)
		}
		return strings.Join(parts, ", ")
	}

	for row, a := range list {
		r := row + 2
		resolvedAt := ""
		if a.ResolvedAt != nil {
			resolvedAt = fmtTime(*a.ResolvedAt)
		}
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", r), a.ID)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", r), a.SourceID)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", r), a.SourceType)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", r), a.Title)
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", r), extractValue(a.Annotations))
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", r), a.Severity)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", r), a.Status)
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", r), formatLabels(a.Labels))
		_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", r), fmtTime(a.FiringAt))
		_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", r), resolvedAt)
		_ = f.SetCellValue(sheet, fmt.Sprintf("K%d", r), fmtDuration(a))
		_ = f.SetCellValue(sheet, fmt.Sprintf("L%d", r), fmtTime(a.CreatedAt))
	}

	f.SetColWidth(sheet, "A", "A", 38)
	f.SetColWidth(sheet, "B", "B", 10)
	f.SetColWidth(sheet, "C", "C", 14)
	f.SetColWidth(sheet, "D", "D", 40)
	f.SetColWidth(sheet, "E", "E", 14)
	f.SetColWidth(sheet, "F", "F", 10)
	f.SetColWidth(sheet, "G", "G", 10)
	f.SetColWidth(sheet, "H", "H", 40)
	f.SetColWidth(sheet, "I", "I", 20)
	f.SetColWidth(sheet, "J", "J", 20)
	f.SetColWidth(sheet, "K", "K", 14)
	f.SetColWidth(sheet, "L", "L", 20)
	f.SetActiveSheet(idx)
	return f, nil
}

// Get alert detail including send records.
func (h *AlertHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var a models.Alert
	if err := h.DB.First(&a, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var records []models.AlertSendRecord
	h.DB.Where("alert_id = ?", id).Find(&records)
	c.JSON(http.StatusOK, gin.H{
		"alert":  a,
		"sends":  records,
	})
}
