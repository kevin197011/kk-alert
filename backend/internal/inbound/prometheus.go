package inbound

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kk-alert/backend/internal/dedup"
	"github.com/kk-alert/backend/internal/engine"
	"github.com/kk-alert/backend/internal/models"
	"gorm.io/gorm"
)

// Prometheus webhook payload (Alertmanager format).
// https://prometheus.io/docs/alerting/latest/configuration/#webhook_config
type PrometheusWebhook struct {
	Alerts []struct {
		Status      string            `json:"status"`
		Labels      map[string]string  `json:"labels"`
		Annotations map[string]string  `json:"annotations"`
		StartsAt    string            `json:"startsAt"`
		EndsAt      string            `json:"endsAt"`
		Fingerprint string            `json:"fingerprint"`
	} `json:"alerts"`
}

// PrometheusHandler receives Alertmanager webhooks and normalizes to unified alert model.
type PrometheusHandler struct {
	DB         *gorm.DB
	SourceID   uint
	SourceType string
}

// ServeHTTP handles POST /inbound/prometheus (or with source id in path/query).
func (h *PrometheusHandler) Serve(c *gin.Context) {
	var payload PrometheusWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	sourceID := h.SourceID
	if id := c.Query("source_id"); id != "" {
		var u uint
		if _, _ = fmt.Sscanf(id, "%d", &u); u != 0 {
			sourceID = u
		}
	}
	if sourceID == 0 {
		sourceID = 1
	}
	created := 0
	for _, a := range payload.Alerts {
		labelsJSON, _ := json.Marshal(a.Labels)
		annotationsJSON, _ := json.Marshal(a.Annotations)
		status := "firing"
		if a.Status == "resolved" {
			status = "resolved"
		}
		var resolvedAt *time.Time
		if a.EndsAt != "" {
			if t, err := time.Parse(time.RFC3339, a.EndsAt); err == nil {
				resolvedAt = &t
			}
		}
		var firingAt time.Time
		if a.StartsAt != "" {
			firingAt, _ = time.Parse(time.RFC3339, a.StartsAt)
		}
		if firingAt.IsZero() {
			firingAt = time.Now()
		}
		title := a.Annotations["summary"]
		if title == "" {
			title = a.Annotations["alertname"]
		}
		if title == "" {
			title = "Alert"
		}
		severity := a.Labels["severity"]
		if severity == "" {
			severity = "warning"
		}
		// Uniqueness: datasource + title + all labels (same => same alert, reuse ID until resolved)
		externalID := dedup.Key(sourceID, title, a.Labels)

		// Reuse same alert ID while previous alert with same (source_id, external_id) is still firing; only new ID after resolved.
		var alert models.Alert
		hasFiring := h.DB.Where("source_id = ? AND external_id = ? AND status = ?", sourceID, externalID, "firing").First(&alert).Error == nil

		if status == "resolved" {
			if hasFiring {
				alert.Status = "resolved"
				alert.ResolvedAt = resolvedAt
				alert.Title = title
				alert.Labels = string(labelsJSON)
				alert.Annotations = string(annotationsJSON)
				h.DB.Save(&alert)
			} else {
				// No prior firing row: create resolved-only record for history
				alert = models.Alert{
					ID:          uuid.New().String(),
					SourceID:    sourceID,
					SourceType:  h.SourceType,
					ExternalID:  externalID,
					Title:       title,
					Severity:    severity,
					Status:      "resolved",
					FiringAt:    firingAt,
					ResolvedAt:  resolvedAt,
					Labels:      string(labelsJSON),
					Annotations: string(annotationsJSON),
				}
				h.DB.Create(&alert)
			}
		} else {
			if hasFiring {
				// Existing firing: update in place (keep same ID)
				alert.Title = title
				alert.Severity = severity
				alert.FiringAt = firingAt
				alert.Labels = string(labelsJSON)
				alert.Annotations = string(annotationsJSON)
				h.DB.Save(&alert)
			} else {
				// No firing for this fingerprint: create new
				alert = models.Alert{
					ID:          uuid.New().String(),
					SourceID:    sourceID,
					SourceType:  h.SourceType,
					ExternalID:  externalID,
					Title:       title,
					Severity:    severity,
					Status:      "firing",
					FiringAt:    firingAt,
					ResolvedAt:  nil,
					Labels:      string(labelsJSON),
					Annotations: string(annotationsJSON),
				}
				if err := h.DB.Create(&alert).Error; err != nil {
					continue
				}
				created++
			}
		}
		engine.ProcessAlert(h.DB, &alert)
	}
	c.JSON(200, gin.H{"received": len(payload.Alerts), "created": created})
}
