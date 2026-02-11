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

// GenericWebhook for ES / Doris or any JSON with alerts array.
type GenericWebhook struct {
	Alerts []struct {
		Title       string            `json:"title"`
		Severity    string            `json:"severity"`
		Status      string            `json:"status"`
		Labels      map[string]string  `json:"labels"`
		Annotations map[string]string  `json:"annotations"`
		StartsAt    string            `json:"starts_at"`
		EndsAt      string            `json:"ends_at"`
		Fingerprint string            `json:"fingerprint"`
	} `json:"alerts"`
}

// GenericHandler handles POST /inbound/elasticsearch and /inbound/doris with source_type set.
type GenericHandler struct {
	DB         *gorm.DB
	SourceType string // "elasticsearch" or "doris"
}

// Serve parses JSON and stores alerts with the handler's source type.
func (h *GenericHandler) Serve(c *gin.Context) {
	var payload GenericWebhook
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(400, gin.H{"error": "invalid json"})
		return
	}
	sourceID := uint(1)
	if id := c.Query("source_id"); id != "" {
		var u uint
		if _, _ = fmt.Sscanf(id, "%d", &u); u != 0 {
			sourceID = u
		}
	}
	created := 0
	for _, a := range payload.Alerts {
		labelsJSON, _ := json.Marshal(a.Labels)
		annotationsJSON, _ := json.Marshal(a.Annotations)
		if labelsJSON == nil {
			labelsJSON = []byte("{}")
		}
		if annotationsJSON == nil {
			annotationsJSON = []byte("{}")
		}
		status := a.Status
		if status == "" {
			status = "firing"
		}
		title := a.Title
		if title == "" {
			title = "Alert"
		}
		severity := a.Severity
		if severity == "" {
			severity = "warning"
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
		labelsMap := a.Labels
		if labelsMap == nil {
			labelsMap = make(map[string]string)
		}
		// Uniqueness: datasource + title + all labels (same => same alert, reuse ID until resolved)
		externalID := dedup.Key(sourceID, title, labelsMap)

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
				alert.Title = title
				alert.Severity = severity
				alert.FiringAt = firingAt
				alert.Labels = string(labelsJSON)
				alert.Annotations = string(annotationsJSON)
				h.DB.Save(&alert)
			} else {
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
