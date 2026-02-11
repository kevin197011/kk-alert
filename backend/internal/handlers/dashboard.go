package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"gorm.io/gorm"
)

// DashboardHandler provides stats for the dashboard (allowed for all authenticated users).
type DashboardHandler struct {
	DB *gorm.DB
}

// Stats returns counts for dashboard cards: alertTotal, firing, rules, datasources, channels, templates.
func (h *DashboardHandler) Stats(c *gin.Context) {
	var alertTotal, firingTotal int64
	h.DB.Model(&models.Alert{}).Count(&alertTotal)
	h.DB.Model(&models.Alert{}).Where("status = ?", "firing").Count(&firingTotal)

	var rules, datasources, channels, templates int64
	h.DB.Model(&models.Rule{}).Count(&rules)
	h.DB.Model(&models.Datasource{}).Count(&datasources)
	h.DB.Model(&models.Channel{}).Count(&channels)
	h.DB.Model(&models.Template{}).Count(&templates)

	c.JSON(http.StatusOK, gin.H{
		"alert_total":  alertTotal,
		"firing":       firingTotal,
		"rules":        rules,
		"datasources":  datasources,
		"channels":     channels,
		"templates":    templates,
	})
}
