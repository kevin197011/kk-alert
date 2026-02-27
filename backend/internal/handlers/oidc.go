package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/auth"
	"github.com/kk-alert/backend/internal/models"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// OIDCHandler implements OIDC authorization code flow.
type OIDCHandler struct {
	DB *gorm.DB

	mu       sync.RWMutex
	provider *oidc.Provider
	oauth2   *oauth2.Config
	cfg      *models.OIDCConfig
	loadedAt time.Time
}

// oidcState stores CSRF state tokens with expiry.
var (
	stateMu    sync.Mutex
	stateStore = make(map[string]time.Time)
)

func generateState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := base64.URLEncoding.EncodeToString(b)
	stateMu.Lock()
	// Purge expired entries
	now := time.Now()
	for k, v := range stateStore {
		if now.After(v) {
			delete(stateStore, k)
		}
	}
	stateStore[state] = now.Add(10 * time.Minute)
	stateMu.Unlock()
	return state, nil
}

func validateState(state string) bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	exp, ok := stateStore[state]
	if !ok {
		return false
	}
	delete(stateStore, state)
	return time.Now().Before(exp)
}

// loadConfig refreshes the OIDC provider configuration from DB (cached for 5 minutes).
func (h *OIDCHandler) loadConfig() (*models.OIDCConfig, *oidc.Provider, *oauth2.Config, error) {
	h.mu.RLock()
	if h.cfg != nil && time.Since(h.loadedAt) < 5*time.Minute {
		cfg, prov, oa := h.cfg, h.provider, h.oauth2
		h.mu.RUnlock()
		return cfg, prov, oa, nil
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	var cfg models.OIDCConfig
	if err := h.DB.First(&cfg).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("OIDC not configured")
	}
	if !cfg.Enabled || cfg.Issuer == "" || cfg.ClientID == "" {
		return nil, nil, nil, fmt.Errorf("OIDC not enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}

	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	if cfg.Scopes != "" {
		scopes = strings.Fields(cfg.Scopes)
	}

	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	h.cfg = &cfg
	h.provider = provider
	h.oauth2 = oauthCfg
	h.loadedAt = time.Now()

	return &cfg, provider, oauthCfg, nil
}

// Status returns whether OIDC is configured and enabled (public endpoint for login page).
func (h *OIDCHandler) Status(c *gin.Context) {
	var cfg models.OIDCConfig
	if err := h.DB.First(&cfg).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"enabled": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":      cfg.Enabled && cfg.Issuer != "" && cfg.ClientID != "",
		"display_name": cfg.DisplayName,
	})
}

// Login redirects the user to the OIDC provider's authorization endpoint.
func (h *OIDCHandler) Login(c *gin.Context) {
	_, _, oauthCfg, err := h.loadConfig()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate state"})
		return
	}

	url := oauthCfg.AuthCodeURL(state)
	c.Redirect(http.StatusFound, url)
}

// Callback handles the OIDC provider's redirect, exchanges the code for tokens,
// looks up or creates the local user, and redirects to the frontend with a JWT.
func (h *OIDCHandler) Callback(c *gin.Context) {
	state := c.Query("state")
	if !validateState(state) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		errMsg := c.Query("error_description")
		if errMsg == "" {
			errMsg = c.Query("error")
		}
		if errMsg == "" {
			errMsg = "no authorization code"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	cfg, provider, oauthCfg, err := h.loadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		log.Printf("[oidc] token exchange failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "token exchange failed"})
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id_token"})
		return
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: oauthCfg.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("[oidc] id_token verification failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_token verification failed"})
		return
	}

	var claims struct {
		Sub               string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Email             string `json:"email"`
		Name              string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse claims"})
		return
	}

	// Determine username: preferred_username > email > name > sub
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Email
	}
	if username == "" {
		username = claims.Name
	}
	if username == "" {
		username = claims.Sub
	}

	// Look up existing user by OIDC sub+issuer, then by username
	var user models.User
	found := false

	if err := h.DB.Where("oidc_sub = ? AND oidc_issuer = ?", claims.Sub, cfg.Issuer).First(&user).Error; err == nil {
		found = true
	}

	if !found {
		if err := h.DB.Where("username = ?", username).First(&user).Error; err == nil {
			// Link existing local user to OIDC
			user.OIDCSub = claims.Sub
			user.OIDCIssuer = cfg.Issuer
			h.DB.Save(&user)
			found = true
		}
	}

	if !found {
		if !cfg.AutoRegister {
			c.JSON(http.StatusForbidden, gin.H{"error": "user not found and auto-registration is disabled"})
			return
		}
		role := cfg.DefaultRole
		if role == "" {
			role = "user"
		}
		user = models.User{
			Username:   username,
			Role:       role,
			OIDCSub:    claims.Sub,
			OIDCIssuer: cfg.Issuer,
		}
		if err := h.DB.Create(&user).Error; err != nil {
			// Username conflict — try appending sub suffix
			user.Username = username + "_" + claims.Sub[:8]
			if err2 := h.DB.Create(&user).Error; err2 != nil {
				log.Printf("[oidc] failed to create user: %v", err2)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}
		}
		log.Printf("[oidc] auto-registered user %s (role=%s)", user.Username, user.Role)
	}

	// Issue local JWT
	jwtToken, err := auth.IssueToken(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to issue token"})
		return
	}

	// Redirect to frontend with token in query param (frontend will store it)
	redirectURL := fmt.Sprintf("/login?oidc_token=%s", jwtToken)
	c.Redirect(http.StatusFound, redirectURL)
}

// --- Admin OIDC Config CRUD ---

// GetConfig returns the current OIDC configuration (client_secret masked).
func (h *OIDCHandler) GetConfig(c *gin.Context) {
	var cfg models.OIDCConfig
	if err := h.DB.First(&cfg).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled":       false,
			"issuer":        "",
			"client_id":     "",
			"redirect_uri":  "",
			"scopes":        "openid profile email",
			"display_name":  "SSO",
			"auto_register": true,
			"default_role":  "user",
		})
		return
	}
	// Mask secret
	maskedSecret := ""
	if cfg.ClientSecret != "" {
		maskedSecret = "••••••••"
	}
	c.JSON(http.StatusOK, gin.H{
		"id":            cfg.ID,
		"enabled":       cfg.Enabled,
		"issuer":        cfg.Issuer,
		"client_id":     cfg.ClientID,
		"client_secret": maskedSecret,
		"redirect_uri":  cfg.RedirectURI,
		"scopes":        cfg.Scopes,
		"display_name":  cfg.DisplayName,
		"auto_register": cfg.AutoRegister,
		"default_role":  cfg.DefaultRole,
	})
}

// SaveConfig creates or updates the OIDC configuration.
func (h *OIDCHandler) SaveConfig(c *gin.Context) {
	var req struct {
		Enabled      bool   `json:"enabled"`
		Issuer       string `json:"issuer"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURI  string `json:"redirect_uri"`
		Scopes       string `json:"scopes"`
		DisplayName  string `json:"display_name"`
		AutoRegister *bool  `json:"auto_register"`
		DefaultRole  string `json:"default_role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	var cfg models.OIDCConfig
	h.DB.First(&cfg)

	cfg.Enabled = req.Enabled
	cfg.Issuer = strings.TrimRight(req.Issuer, "/")
	cfg.ClientID = req.ClientID
	// Only update secret if not masked placeholder
	if req.ClientSecret != "" && req.ClientSecret != "••••••••" {
		cfg.ClientSecret = req.ClientSecret
	}
	cfg.RedirectURI = req.RedirectURI
	if req.Scopes != "" {
		cfg.Scopes = req.Scopes
	}
	if req.DisplayName != "" {
		cfg.DisplayName = req.DisplayName
	}
	if req.AutoRegister != nil {
		cfg.AutoRegister = *req.AutoRegister
	}
	if req.DefaultRole != "" && (req.DefaultRole == "admin" || req.DefaultRole == "user") {
		cfg.DefaultRole = req.DefaultRole
	}

	var err error
	if cfg.ID == 0 {
		err = h.DB.Create(&cfg).Error
	} else {
		err = h.DB.Save(&cfg).Error
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Invalidate cached provider
	h.mu.Lock()
	h.cfg = nil
	h.provider = nil
	h.oauth2 = nil
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// TestConfig attempts OIDC discovery against the configured issuer.
func (h *OIDCHandler) TestConfig(c *gin.Context) {
	var cfg models.OIDCConfig
	if err := h.DB.First(&cfg).Error; err != nil || cfg.Issuer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OIDC not configured"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	var discoveryRaw json.RawMessage
	if err := provider.Claims(&discoveryRaw); err == nil {
		var disc map[string]interface{}
		_ = json.Unmarshal(discoveryRaw, &disc)
		c.JSON(http.StatusOK, gin.H{
			"success":                true,
			"authorization_endpoint": disc["authorization_endpoint"],
			"token_endpoint":         disc["token_endpoint"],
			"userinfo_endpoint":      disc["userinfo_endpoint"],
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
