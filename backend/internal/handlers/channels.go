package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"github.com/kk-alert/backend/internal/sender"
	"gorm.io/gorm"
)

// ChannelHandler CRUD and test send for channels.
type ChannelHandler struct {
	DB *gorm.DB
}

// List channels (config without secrets).
func (h *ChannelHandler) List(c *gin.Context) {
	var list []models.Channel
	if err := h.DB.Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Don't send Config (secrets) in list
	out := make([]map[string]interface{}, len(list))
	for i := range list {
		out[i] = map[string]interface{}{
			"id":         list[i].ID,
			"name":       list[i].Name,
			"type":       list[i].Type,
			"enabled":    list[i].Enabled,
			"created_at": list[i].CreatedAt,
			"updated_at": list[i].UpdatedAt,
		}
	}
	c.JSON(http.StatusOK, out)
}

// Get by ID (config masked).
func (h *ChannelHandler) Get(c *gin.Context) {
	var ch models.Channel
	if err := h.DB.First(&ch, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         ch.ID,
		"name":       ch.Name,
		"type":       ch.Type,
		"enabled":    ch.Enabled,
		"created_at": ch.CreatedAt,
		"updated_at": ch.UpdatedAt,
		"config_set": ch.Config != "",
	})
}

// Create channel.
func (h *ChannelHandler) Create(c *gin.Context) {
	var body struct {
		Name    string `json:"name" binding:"required"`
		Type    string `json:"type" binding:"required"`
		Config  string `json:"config"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ch := models.Channel{Name: body.Name, Type: body.Type, Config: body.Config, Enabled: body.Enabled}
	if err := h.DB.Create(&ch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": ch.ID, "name": ch.Name, "type": ch.Type, "enabled": ch.Enabled})
}

// Update channel.
func (h *ChannelHandler) Update(c *gin.Context) {
	var ch models.Channel
	if err := h.DB.First(&ch, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var body struct {
		Name    *string `json:"name"`
		Type    *string `json:"type"`
		Config  *string `json:"config"`
		Enabled *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.Name != nil {
		ch.Name = *body.Name
	}
	if body.Type != nil {
		ch.Type = *body.Type
	}
	if body.Config != nil && *body.Config != "" {
		ch.Config = *body.Config
	}
	if body.Enabled != nil {
		ch.Enabled = *body.Enabled
	}
	if err := h.DB.Save(&ch).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": ch.ID, "name": ch.Name, "type": ch.Type, "enabled": ch.Enabled})
}

// Delete channel.
func (h *ChannelHandler) Delete(c *gin.Context) {
	if err := h.DB.Delete(&models.Channel{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// TestSend sends a test message via the channel (Telegram/Lark).
func (h *ChannelHandler) TestSend(c *gin.Context) {
	var ch models.Channel
	if err := h.DB.First(&ch, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "通知渠道不存在"})
		return
	}

	log.Printf("[channel test] sending test message to channel %d (type=%s, config_set=%v)", ch.ID, ch.Type, ch.Config != "")

	if err := sender.Send(ch.Type, ch.Config, "KK Alert – 测试", "这是一条来自 KK Alert 的测试消息。", false); err != nil {
		log.Printf("[channel test] failed to send test message to channel %d: %v", ch.ID, err)
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "测试发送失败：" + err.Error()})
		return
	}

	log.Printf("[channel test] test message sent successfully to channel %d", ch.ID)
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "测试消息已发送成功"})
}
