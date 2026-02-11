package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"gorm.io/gorm"
)

// SilenceHandler manages manual alert silences.
type SilenceHandler struct {
	DB *gorm.DB
}

// CreateSilenceRequest body for POST /api/v1/alerts/:id/silence.
type CreateSilenceRequest struct {
	DurationMinutes int `json:"duration_minutes"` // e.g. 30, 60, 240, 1440
}

// Create adds or updates silence for an alert until now + duration.
func (h *SilenceHandler) Create(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alert_id required"})
		return
	}
	var req CreateSilenceRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.DurationMinutes <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "duration_minutes required and must be positive"})
		return
	}
	if req.DurationMinutes > 60*24*30 {
		req.DurationMinutes = 60 * 24 * 30 // cap 30 days
	}
	silenceUntil := time.Now().Add(time.Duration(req.DurationMinutes) * time.Minute)

	var s models.AlertSilence
	h.DB.Where("alert_id = ?", id).Limit(1).Find(&s)
	if s.ID != 0 {
		s.SilenceUntil = silenceUntil
		if err := h.DB.Save(&s).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		s = models.AlertSilence{AlertID: id, SilenceUntil: silenceUntil}
		if err := h.DB.Create(&s).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"id":            s.ID,
		"alert_id":      s.AlertID,
		"silence_until": s.SilenceUntil.Format(time.RFC3339),
	})
}

// List returns active silences (silence_until > now), with alert title when available.
func (h *SilenceHandler) List(c *gin.Context) {
	now := time.Now()
	var list []models.AlertSilence
	if err := h.DB.Where("silence_until > ?", now).Order("silence_until asc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type item struct {
		models.AlertSilence
		Title string `json:"title,omitempty"`
	}
	items := make([]item, 0, len(list))
	for _, s := range list {
		i := item{AlertSilence: s}
		var a models.Alert
		if h.DB.Where("id = ?", s.AlertID).First(&a).Error == nil {
			i.Title = a.Title
		}
		items = append(items, i)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Delete removes silence for the given alert_id.
func (h *SilenceHandler) Delete(c *gin.Context) {
	alertID := c.Param("alert_id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alert_id required"})
		return
	}
	res := h.DB.Where("alert_id = ?", alertID).Delete(&models.AlertSilence{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}
