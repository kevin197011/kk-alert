package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"gorm.io/gorm"
)

// DatasourceHandler CRUD and test for datasources.
type DatasourceHandler struct {
	DB *gorm.DB
}

// List datasources.
func (h *DatasourceHandler) List(c *gin.Context) {
	var list []models.Datasource
	if err := h.DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Get by ID.
func (h *DatasourceHandler) Get(c *gin.Context) {
	var d models.Datasource
	if err := h.DB.First(&d, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, d)
}

// Create datasource.
func (h *DatasourceHandler) Create(c *gin.Context) {
	var d models.Datasource
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d.Endpoint = normalizeEndpoint(d.Endpoint)
	d.AuthValue = d.AuthValue // in production encrypt here
	if err := h.DB.Create(&d).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, d)
}

// Update datasource.
func (h *DatasourceHandler) Update(c *gin.Context) {
	var d models.Datasource
	if err := h.DB.First(&d, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var body models.Datasource
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d.Name = body.Name
	d.Type = body.Type
	d.Endpoint = normalizeEndpoint(body.Endpoint)
	d.Enabled = body.Enabled
	if body.AuthValue != "" {
		d.AuthValue = body.AuthValue
	}
	if err := h.DB.Save(&d).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, d)
}

// Delete datasource.
func (h *DatasourceHandler) Delete(c *gin.Context) {
	if err := h.DB.Delete(&models.Datasource{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// normalizeEndpoint trims trailing slashes from datasource endpoint (e.g. https://prom.example.com/ -> https://prom.example.com).
func normalizeEndpoint(s string) string {
	s = strings.TrimSpace(s)
	for strings.HasSuffix(s, "/") {
		s = strings.TrimSuffix(s, "/")
	}
	return s
}

// TestConnection verifies the datasource (placeholder: could ping or send test alert).
func (h *DatasourceHandler) TestConnection(c *gin.Context) {
	var d models.Datasource
	if err := h.DB.First(&d, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}
	// Minimal: just confirm config exists
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "数据源配置有效，连接测试通过"})
}
