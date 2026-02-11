package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kk-alert/backend/internal/auth"
	"github.com/kk-alert/backend/internal/handlers"
	"github.com/kk-alert/backend/internal/inbound"
	"github.com/kk-alert/backend/internal/models"
	"github.com/kk-alert/backend/internal/scheduler"
	"github.com/kk-alert/backend/internal/store"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

//go:embed docs/openapi.json docs/swagger.html
var docsFS embed.FS

func serveOpenAPI(c *gin.Context) {
	b, _ := fs.ReadFile(docsFS, "docs/openapi.json")
	c.Data(http.StatusOK, "application/json", b)
}

func main() {
	db, err := store.NewDB()
	if err != nil {
		log.Fatal(err)
	}
	seedUser(db.DB)
	seedDefaultTemplate(db.DB)
	seedSettings(db.DB)
	fixTemplatesRuleDescriptionHeader(db.DB)

	sched := scheduler.NewScheduler(db.DB)
	sched.Start()

	go runRetentionCleanupLoop(db.DB)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		sched.Stop()
		os.Exit(0)
	}()

	r := gin.Default()
	r.Use(gin.Recovery())

	// Public
	r.POST("/api/v1/auth/login", wrapAuth(db.DB).Login)
	r.GET("/api/v1/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	// Swagger: OpenAPI spec and UI (no auth); token via Authorize in Swagger UI
	r.GET("/api/openapi.json", serveOpenAPI)
	swaggerHTML, _ := fs.ReadFile(docsFS, "docs/swagger.html")
	r.GET("/swagger", func(c *gin.Context) { c.Data(http.StatusOK, "text/html; charset=utf-8", swaggerHTML) })
	r.GET("/swagger/", func(c *gin.Context) { c.Data(http.StatusOK, "text/html; charset=utf-8", swaggerHTML) })
	r.GET("/swagger/index.html", func(c *gin.Context) { c.Data(http.StatusOK, "text/html; charset=utf-8", swaggerHTML) })

	// Inbound webhooks (no auth for Alertmanager / VM / ES / Doris)
	inboundGroup := r.Group("/api/v1/inbound")
	{
		prom := &inbound.PrometheusHandler{DB: db.DB, SourceType: "prometheus"}
		inboundGroup.POST("/prometheus", prom.Serve)
		vm := &inbound.PrometheusHandler{DB: db.DB, SourceType: "victoriametrics"}
		inboundGroup.POST("/victoriametrics", vm.Serve)
		elasticsearchHandler := &inbound.GenericHandler{DB: db.DB, SourceType: "elasticsearch"}
		inboundGroup.POST("/elasticsearch", elasticsearchHandler.Serve)
		dorisHandler := &inbound.GenericHandler{DB: db.DB, SourceType: "doris"}
		inboundGroup.POST("/doris", dorisHandler.Serve)
	}

	// Fill role from DB when JWT has no role (e.g. old tokens before role was added)
	fillRole := fillRoleFromDB(db.DB)

	// Protected API (all authenticated)
	api := r.Group("/api/v1")
	api.Use(auth.RequireAuth(), fillRole)
	{
		api.POST("/auth/logout", wrapAuth(db.DB).Logout)
		api.GET("/auth/me", wrapAuth(db.DB).Me)

		dash := &handlers.DashboardHandler{DB: db.DB}
		api.GET("/dashboard/stats", dash.Stats)

		al := &handlers.AlertHandler{DB: db.DB}
		api.GET("/alerts", al.List)
		api.GET("/alerts/export", al.Export)
		api.GET("/alerts/notify-total", al.NotifyTotal)
		api.GET("/alerts/:id", al.Get)
		sil := &handlers.SilenceHandler{DB: db.DB}
		api.POST("/alerts/:id/silence", sil.Create)
		api.GET("/silences", sil.List)
		api.DELETE("/silences/:alert_id", sil.Delete)

		rep := &handlers.ReportHandler{DB: db.DB}
		api.GET("/reports/aggregate", rep.Aggregate)
		api.GET("/reports/trend", rep.Trend)
		api.GET("/reports/preview", rep.Preview)
		api.GET("/reports/export", rep.Export)

		set := &handlers.SettingsHandler{DB: db.DB}
		api.GET("/settings", set.Get)
	}

	// Admin-only API
	admin := r.Group("/api/v1")
	admin.Use(auth.RequireAuth(), fillRole, auth.RequireAdmin())
	{
		ds := &handlers.DatasourceHandler{DB: db.DB}
		admin.GET("/datasources", ds.List)
		admin.GET("/datasources/:id", ds.Get)
		admin.POST("/datasources", ds.Create)
		admin.PUT("/datasources/:id", ds.Update)
		admin.DELETE("/datasources/:id", ds.Delete)
		admin.POST("/datasources/:id/test", ds.TestConnection)

		ch := &handlers.ChannelHandler{DB: db.DB}
		admin.GET("/channels", ch.List)
		admin.GET("/channels/:id", ch.Get)
		admin.POST("/channels", ch.Create)
		admin.PUT("/channels/:id", ch.Update)
		admin.DELETE("/channels/:id", ch.Delete)
		admin.POST("/channels/:id/test", ch.TestSend)

		tpl := &handlers.TemplateHandler{DB: db.DB}
		admin.GET("/templates", tpl.List)
		admin.GET("/templates/default", tpl.GetDefault)
		admin.GET("/templates/:id", tpl.Get)
		admin.POST("/templates", tpl.Create)
		admin.PUT("/templates/:id/set-default", tpl.SetDefault)
		admin.PUT("/templates/:id", tpl.Update)
		admin.DELETE("/templates/:id", tpl.Delete)
		admin.POST("/templates/:id/preview", tpl.Preview)

		rule := &handlers.RuleHandler{DB: db.DB, Scheduler: sched}
		admin.GET("/rules", rule.List)
		admin.GET("/rules/:id", rule.Get)
		admin.POST("/rules", rule.Create)
		admin.PUT("/rules/:id", rule.Update)
		admin.DELETE("/rules/:id", rule.Delete)
		admin.POST("/rules/batch", rule.Batch)
		admin.POST("/rules/export", rule.Export)
		admin.POST("/rules/import", rule.Import)
		admin.POST("/rules/test-match", rule.TestMatch)
		admin.POST("/rules/:id/trigger", rule.Trigger)

		uh := &handlers.UserHandler{DB: db.DB}
		admin.GET("/users", uh.List)
		admin.POST("/users", uh.Create)
		admin.PUT("/users/:id", uh.Update)
		admin.DELETE("/users/:id", uh.Delete)

		set := &handlers.SettingsHandler{DB: db.DB}
		admin.PUT("/settings", set.Update)
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Println("Listening on", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func wrapAuth(db *gorm.DB) *handlers.AuthHandler {
	return &handlers.AuthHandler{DB: db}
}

// fillRoleFromDB sets role in context from DB when JWT role is empty (backfill for old tokens).
func fillRoleFromDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != nil && role != "" {
			c.Next()
			return
		}
		userID, ok := c.Get("user_id")
		if !ok {
			c.Next()
			return
		}
		var u struct{ Role string }
		if err := db.Table("users").Select("role").Where("id = ?", userID).First(&u).Error; err == nil {
			r := u.Role
			if r == "" {
				r = "user"
			}
			c.Set("role", r)
		}
		c.Next()
	}
}

func seedSettings(db *gorm.DB) {
	var c int64
	db.Model(&models.SystemConfig{}).Where("key = ?", "retention_days").Count(&c)
	if c > 0 {
		return
	}
	db.Create(&models.SystemConfig{Key: "retention_days", Value: "90"})
}

func runRetentionCleanupLoop(db *gorm.DB) {
	// Run once after 1 min, then every 24h
	time.Sleep(1 * time.Minute)
	handlers.RunRetentionCleanup(db)
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		handlers.RunRetentionCleanup(db)
	}
}

func seedUser(db *gorm.DB) {
	var c int64
	db.Model(&models.User{}).Count(&c)
	if c > 0 {
		// Ensure existing admin user has role admin (migration helper)
		db.Model(&models.User{}).Where("username = ?", "admin").Update("role", "admin")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	db.Create(&models.User{Username: "admin", PasswordHash: string(hash), Role: "admin"})
}

// defaultTemplateBody: recovery block uses same layout as alert block (数据源/规则/当前值/标签, then 告警ID/严重程度/时间).
const defaultTemplateBody = `{{if .IsRecovery}}
━━━━━━━━━━━━━━━━━━━━━
✅ {{.Title}}
━━━━━━━━━━━━━━━━━━━━━
📊 数据源: {{.SourceType}}
📈 当前值/阈值: {{.Value}}
📍 标签:
{{range $key, $value := .Labels -}}
• {{$key}}: {{$value}}
{{end -}}
━━━━━━━━━━━━━━━━━━━━━
📌 告警ID: {{.AlertID}} 
⚠️ 严重程度: {{.Severity}} 
⏰ 发生时间: {{.StartAt}}{{if .ResolvedAt}} 
🕐 恢复时间: {{.ResolvedAt}}{{end}}
━━━━━━━━━━━━━━━━━━━━━
此告警由 KK Alert 系统自动发送
{{else}}
━━━━━━━━━━━━━━━━━━━━━
🔔 {{.Title}}
━━━━━━━━━━━━━━━━━━━━━
📊 数据源: {{.SourceType}}
📈 当前值/阈值: {{.Value}}
📍 标签:
{{range $key, $value := .Labels -}}
• {{$key}}: {{$value}}
{{end -}}
━━━━━━━━━━━━━━━━━━━━━
📌 告警ID: {{.AlertID}} 
⚠️ 严重程度: {{.Severity}} 
⏰ 发生时间: {{.StartAt}}
━━━━━━━━━━━━━━━━━━━━━
此告警由 KK Alert 系统自动发送
{{end}}`

func seedDefaultTemplate(db *gorm.DB) {
	var c int64
	db.Model(&models.Template{}).Where("is_default = ?", true).Count(&c)
	if c > 0 {
		return
	}
	// No default template: create one (or set first existing as default)
	var first models.Template
	if err := db.Order("id asc").First(&first).Error; err == nil {
		first.IsDefault = true
		db.Save(&first)
		return
	}
	db.Create(&models.Template{
		Name:        "默认告警模板",
		ChannelType: "generic",
		Body:        defaultTemplateBody,
		IsDefault:   true,
	})
}

// fixTemplatesRuleDescriptionHeader updates existing templates that use 🔔 {{.RuleDescription}} as alert header to 🔔 {{.Title}} so each alert shows its own title.
func fixTemplatesRuleDescriptionHeader(db *gorm.DB) {
	var list []models.Template
	if err := db.Find(&list).Error; err != nil {
		return
	}
	for i := range list {
		body := list[i].Body
		if !strings.Contains(body, "🔔 {{.RuleDescription}}") {
			continue
		}
		newBody := strings.Replace(body, "🔔 {{.RuleDescription}}", "🔔 {{.Title}}", 1)
		if newBody != body {
			db.Model(&list[i]).Update("body", newBody)
			log.Printf("[main] updated template id=%d: alert header to {{.Title}}", list[i].ID)
		}
	}
}
