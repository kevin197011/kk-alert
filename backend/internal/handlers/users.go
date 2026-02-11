package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserHandler CRUD for users (admin only).
type UserHandler struct {
	DB *gorm.DB
}

// List returns all users (id, username, role, created_at). Password hashes omitted.
func (h *UserHandler) List(c *gin.Context) {
	var list []models.User
	if err := h.DB.Select("id", "username", "role", "created_at").Order("id asc").Find(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// CreateRequest for creating a user.
type CreateRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role"`
}

// Create a new user.
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	if req.Role != "admin" && req.Role != "user" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or user"})
		return
	}
	var exists int64
	h.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&exists)
	if exists > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	u := models.User{Username: req.Username, PasswordHash: string(hash), Role: req.Role}
	if err := h.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "username": u.Username, "role": u.Role})
}

// UpdateRequest for updating a user (password and/or role).
type UpdateRequest struct {
	Password *string `json:"password"`
	Role     *string `json:"role"`
}

// Update user by id (path :id).
func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	var u models.User
	if err := h.DB.First(&u, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if req.Role != nil {
		r := *req.Role
		if r != "admin" && r != "user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role must be admin or user"})
			return
		}
		u.Role = r
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
			return
		}
		u.PasswordHash = string(hash)
	}
	if err := h.DB.Save(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "username": u.Username, "role": u.Role})
}

// Delete user by id.
func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var u models.User
	if err := h.DB.First(&u, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if u.Username == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete admin user"})
		return
	}
	if err := h.DB.Delete(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
