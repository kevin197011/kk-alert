package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"gorm.io/gorm"
)

const (
	ConfigKeyRetentionDays = "retention_days"
	DefaultRetentionDays   = 90
)

// SettingsHandler provides GET/PUT for system settings (admin only).
type SettingsHandler struct {
	DB *gorm.DB
}

// Get returns current system settings (e.g. retention_days). All authenticated users can read.
func (h *SettingsHandler) Get(c *gin.Context) {
	var cfg models.SystemConfig
	err := h.DB.Where("key = ?", ConfigKeyRetentionDays).First(&cfg).Error
	retentionDays := DefaultRetentionDays
	if err == nil && cfg.Value != "" {
		if v, e := strconv.Atoi(cfg.Value); e == nil && v > 0 {
			retentionDays = v
		}
	}
	c.JSON(http.StatusOK, gin.H{"retention_days": retentionDays})
}

// SettingsUpdateRequest for updating settings.
type SettingsUpdateRequest struct {
	RetentionDays *int `json:"retention_days"`
}

// Update saves system settings. Admin only.
func (h *SettingsHandler) Update(c *gin.Context) {
	var req SettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.RetentionDays != nil {
		v := *req.RetentionDays
		if v < 1 || v > 3650 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "retention_days must be between 1 and 3650"})
			return
		}
		err := h.DB.Save(&models.SystemConfig{Key: ConfigKeyRetentionDays, Value: strconv.Itoa(v)}).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// Return current state
	h.Get(c)
}

// RunRetentionCleanup deletes alerts and their send records older than retention days. Call periodically (e.g. daily).
func RunRetentionCleanup(db *gorm.DB) {
	var cfg models.SystemConfig
	err := db.Where("key = ?", ConfigKeyRetentionDays).First(&cfg).Error
	retentionDays := DefaultRetentionDays
	if err == nil && cfg.Value != "" {
		if v, e := strconv.Atoi(cfg.Value); e == nil && v > 0 {
			retentionDays = v
		}
	}
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	var ids []string
	if err := db.Model(&models.Alert{}).Where("created_at < ?", cutoff).Pluck("id", &ids).Error; err != nil {
		log.Printf("[retention] list old alerts: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	// Delete send records for those alerts first
	if res := db.Where("alert_id in ?", ids).Delete(&models.AlertSendRecord{}); res.Error != nil {
		log.Printf("[retention] delete send records: %v", res.Error)
		return
	}
	// Then delete alerts
	if res := db.Where("created_at < ?", cutoff).Delete(&models.Alert{}); res.Error != nil {
		log.Printf("[retention] delete alerts: %v", res.Error)
		return
	}
	log.Printf("[retention] cleaned up %d alert(s) older than %s (retention %d days)", len(ids), cutoff.Format(time.RFC3339), retentionDays)
}
