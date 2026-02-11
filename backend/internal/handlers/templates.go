package handlers

import (
	"bytes"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"github.com/kk-alert/backend/internal/sender"
	"gorm.io/gorm"
)

// TemplateHandler CRUD and preview for templates.
type TemplateHandler struct {
	DB *gorm.DB
}

// List templates.
func (h *TemplateHandler) List(c *gin.Context) {
	var list []models.Template
	if err := h.DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GetDefault returns the default template (is_default=true); 404 if none.
func (h *TemplateHandler) GetDefault(c *gin.Context) {
	var t models.Template
	if err := h.DB.Where("is_default = ?", true).First(&t).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no default template"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// Get by ID.
func (h *TemplateHandler) Get(c *gin.Context) {
	var t models.Template
	if err := h.DB.First(&t, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// Create template.
func (h *TemplateHandler) Create(c *gin.Context) {
	var t models.Template
	if err := c.ShouldBindJSON(&t); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// If this is the first template or client set is_default, ensure only one default
	if t.IsDefault {
		h.DB.Model(&models.Template{}).Where("1 = 1").Update("is_default", false)
	}
	var count int64
	if h.DB.Model(&models.Template{}).Count(&count).Error == nil && count == 0 {
		t.IsDefault = true // first template becomes default
	}
	if err := h.DB.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

// Update template.
func (h *TemplateHandler) Update(c *gin.Context) {
	var t models.Template
	if err := h.DB.First(&t, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var body models.Template
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t.Name = body.Name
	t.ChannelType = body.ChannelType
	t.Body = body.Body
	t.IsDefault = body.IsDefault
	if t.IsDefault {
		h.DB.Model(&models.Template{}).Where("id != ?", t.ID).Update("is_default", false)
	}
	if err := h.DB.Save(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

// SetDefault sets this template as the default (only one default at a time).
func (h *TemplateHandler) SetDefault(c *gin.Context) {
	var t models.Template
	if err := h.DB.First(&t, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	h.DB.Model(&models.Template{}).Where("1 = 1").Update("is_default", false)
	t.IsDefault = true
	if err := h.DB.Save(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}

// Delete template.
func (h *TemplateHandler) Delete(c *gin.Context) {
	if err := h.DB.Delete(&models.Template{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RenderTemplate replaces {{.Labels.key}} and {{.AlertID}}, {{.Title}}, etc. with sample data.
func renderTemplate(body string, labels map[string]string, alertID, title, severity string) string {
	out := body
	// {{.Labels.xxx}}
	for k, v := range labels {
		out = replaceAll(out, "{{.Labels."+k+"}}", v)
	}
	out = replaceAll(out, "{{.AlertID}}", alertID)
	out = replaceAll(out, "{{.Title}}", title)
	out = replaceAll(out, "{{.Severity}}", severity)
	return out
}

func replaceAll(s, old, new string) string {
	return string(bytes.ReplaceAll([]byte(s), []byte(old), []byte(new)))
}

// PreviewRequest for template preview. All fields optional; defaults used for Go template rendering (including {{.RuleDescription}}, {{.SourceType}}, etc.).
type PreviewRequest struct {
	Labels           map[string]string `json:"labels"`
	AlertID          string            `json:"alert_id"`
	Title            string            `json:"title"`
	Severity         string            `json:"severity"`
	RuleDescription  string            `json:"rule_description"`
	SourceType       string            `json:"source_type"`
	StartAt          string            `json:"start_at"`
	Description      string            `json:"description"`
	Value            string            `json:"value"` // trigger value (当前值/阈值) for {{.Value}}
	IsRecovery       bool              `json:"is_recovery"`
	ResolvedAt       string            `json:"resolved_at"`
}

// Preview renders template with sample data using the same AlertTemplateData as real notifications.
func (h *TemplateHandler) Preview(c *gin.Context) {
	var req PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Labels == nil {
		req.Labels = make(map[string]string)
	}
	if req.AlertID == "" {
		req.AlertID = "sample-id"
	}
	if req.Title == "" {
		req.Title = "Sample Alert"
	}
	if req.Severity == "" {
		req.Severity = "warning"
	}
	if req.SourceType == "" {
		req.SourceType = "prometheus"
	}
	if req.StartAt == "" {
		req.StartAt = "2006-01-02 15:04:05"
	}
	if req.Description == "" {
		req.Description = "Sample alert description (e.g. CPU usage > 80%)"
	}
	if req.RuleDescription == "" {
		req.RuleDescription = "规则说明示例（规则描述，可在模板中用 {{.RuleDescription}} 引用）"
	}
	if req.Value == "" {
		req.Value = "80.5"
	}
	id := c.Param("id")
	var t models.Template
	if err := h.DB.First(&t, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	data := sender.AlertTemplateData{
		AlertID:         req.AlertID,
		Title:           req.Title,
		Severity:        req.Severity,
		Labels:          req.Labels,
		StartAt:         req.StartAt,
		SourceType:      req.SourceType,
		Description:     req.Description,
		Value:           req.Value,
		RuleDescription: req.RuleDescription,
		IsRecovery:      req.IsRecovery,
		ResolvedAt:      req.ResolvedAt,
		SentAt:          req.StartAt, // preview uses StartAt as sample send time when not provided
	}
	rendered, err := sender.RenderTemplate(t.Body, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template render failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rendered": rendered})
}

// ExpandTemplateForAlert renders template for an alert (used by rule engine). Uses regex for {{.Labels.xxx}}.
func ExpandTemplateForAlert(body string, labels map[string]string, alertID, title, severity string) string {
	return renderTemplate(body, labels, alertID, title, severity)
}

var labelRe = regexp.MustCompile(`\{\{\.Labels\.(\w+)\}\}`)

// ExpandTemplateWithLabels replaces all {{.Labels.key}} in body.
func ExpandTemplateWithLabels(body string, labels map[string]string) string {
	return labelRe.ReplaceAllStringFunc(body, func(m string) string {
		key := labelRe.FindStringSubmatch(m)
		if len(key) < 2 {
			return m
		}
		if v, ok := labels[key[1]]; ok {
			return v
		}
		return ""
	})
}
