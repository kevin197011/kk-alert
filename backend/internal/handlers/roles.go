package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/models"
	"gorm.io/gorm"
)

// RoleHandler handles role and permission management.
type RoleHandler struct {
	DB *gorm.DB
}

// ListRoles returns all roles with their permissions.
func (h *RoleHandler) ListRoles(c *gin.Context) {
	var roles []models.Role
	if err := h.DB.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, roles)
}

// GetRole returns a single role by ID.
func (h *RoleHandler) GetRole(c *gin.Context) {
	var role models.Role
	if err := h.DB.First(&role, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	c.JSON(http.StatusOK, role)
}

// CreateRoleRequest for creating a role.
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// CreateRole creates a new role.
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if role name exists
	var count int64
	h.DB.Model(&models.Role{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "role name already exists"})
		return
	}

	permissionsJSON, _ := json.Marshal(req.Permissions)
	role := models.Role{
		Name:        req.Name,
		Description: req.Description,
		Permissions: string(permissionsJSON),
	}

	if err := h.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, role)
}

// UpdateRoleRequest for updating a role.
type UpdateRoleRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateRole updates a role.
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	var role models.Role
	if err := h.DB.First(&role, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check name uniqueness if changing name
	if req.Name != "" && req.Name != role.Name {
		var count int64
		h.DB.Model(&models.Role{}).Where("name = ?", req.Name).Count(&count)
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "role name already exists"})
			return
		}
		role.Name = req.Name
	}

	if req.Description != "" {
		role.Description = req.Description
	}

	if req.Permissions != nil {
		permissionsJSON, _ := json.Marshal(req.Permissions)
		role.Permissions = string(permissionsJSON)
	}

	if err := h.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, role)
}

// DeleteRole deletes a role.
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id := c.Param("id")

	// Check if role is in use
	var count int64
	h.DB.Model(&models.User{}).Where("role = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "cannot delete role that is assigned to users"})
		return
	}

	if err := h.DB.Delete(&models.Role{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListPermissions returns all available permissions.
func (h *RoleHandler) ListPermissions(c *gin.Context) {
	var permissions []models.Permission
	if err := h.DB.Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, permissions)
}

// SeedDefaultRoles creates default roles if they don't exist.
func SeedDefaultRoles(db *gorm.DB) {
	// Check if roles exist
	var count int64
	db.Model(&models.Role{}).Count(&count)
	if count > 0 {
		return
	}

	// Create admin role with all permissions
	adminPerms := []string{
		"dashboard:view",
		"alerts:view", "alerts:manage",
		"reports:view",
		"rules:view", "rules:create", "rules:update", "rules:delete",
		"datasources:view", "datasources:create", "datasources:update", "datasources:delete",
		"channels:view", "channels:create", "channels:update", "channels:delete",
		"templates:view", "templates:create", "templates:update", "templates:delete",
		"users:view", "users:create", "users:update", "users:delete",
		"roles:view", "roles:create", "roles:update", "roles:delete",
		"settings:view", "settings:update",
	}
	adminPermsJSON, _ := json.Marshal(adminPerms)
	db.Create(&models.Role{
		Name:        "admin",
		Description: "Administrator with full access",
		Permissions: string(adminPermsJSON),
	})

	// Create user role with limited permissions
	userPerms := []string{
		"dashboard:view",
		"alerts:view",
		"reports:view",
	}
	userPermsJSON, _ := json.Marshal(userPerms)
	db.Create(&models.Role{
		Name:        "user",
		Description: "Standard user with read-only access to alerts and reports",
		Permissions: string(userPermsJSON),
	})
}

// SeedPermissions creates default permissions.
func SeedPermissions(db *gorm.DB) {
	var count int64
	db.Model(&models.Permission{}).Count(&count)
	if count > 0 {
		return
	}

	permissions := []models.Permission{
		// Dashboard
		{Code: "dashboard:view", Name: "View Dashboard", Category: "menu"},

		// Alerts
		{Code: "alerts:view", Name: "View Alerts", Category: "menu"},
		{Code: "alerts:manage", Name: "Manage Alerts (silence)", Category: "action"},

		// Reports
		{Code: "reports:view", Name: "View Reports", Category: "menu"},

		// Rules
		{Code: "rules:view", Name: "View Rules", Category: "menu"},
		{Code: "rules:create", Name: "Create Rules", Category: "action"},
		{Code: "rules:update", Name: "Update Rules", Category: "action"},
		{Code: "rules:delete", Name: "Delete Rules", Category: "action"},

		// Datasources
		{Code: "datasources:view", Name: "View Datasources", Category: "menu"},
		{Code: "datasources:create", Name: "Create Datasources", Category: "action"},
		{Code: "datasources:update", Name: "Update Datasources", Category: "action"},
		{Code: "datasources:delete", Name: "Delete Datasources", Category: "action"},

		// Channels
		{Code: "channels:view", Name: "View Channels", Category: "menu"},
		{Code: "channels:create", Name: "Create Channels", Category: "action"},
		{Code: "channels:update", Name: "Update Channels", Category: "action"},
		{Code: "channels:delete", Name: "Delete Channels", Category: "action"},

		// Templates
		{Code: "templates:view", Name: "View Templates", Category: "menu"},
		{Code: "templates:create", Name: "Create Templates", Category: "action"},
		{Code: "templates:update", Name: "Update Templates", Category: "action"},
		{Code: "templates:delete", Name: "Delete Templates", Category: "action"},

		// Users
		{Code: "users:view", Name: "View Users", Category: "menu"},
		{Code: "users:create", Name: "Create Users", Category: "action"},
		{Code: "users:update", Name: "Update Users", Category: "action"},
		{Code: "users:delete", Name: "Delete Users", Category: "action"},

		// Roles
		{Code: "roles:view", Name: "View Roles", Category: "menu"},
		{Code: "roles:create", Name: "Create Roles", Category: "action"},
		{Code: "roles:update", Name: "Update Roles", Category: "action"},
		{Code: "roles:delete", Name: "Delete Roles", Category: "action"},

		// Settings
		{Code: "settings:view", Name: "View Settings", Category: "menu"},
		{Code: "settings:update", Name: "Update Settings", Category: "action"},
	}

	for _, p := range permissions {
		db.Create(&p)
	}
}
