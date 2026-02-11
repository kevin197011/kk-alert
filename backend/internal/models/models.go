package models

import (
	"time"

	"gorm.io/gorm"
)

// User for auth (minimal user store). Role: admin (all permissions), user (dashboard, alerts, reports only).
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash string         `gorm:"size:255" json:"-"`
	Role         string         `gorm:"size:32;default:user" json:"role"` // admin | user
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Datasource for alert ingestion (Prometheus, VictoriaMetrics, ES, Doris).
type Datasource struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128" json:"name"`
	Type      string         `gorm:"size:32" json:"type"` // prometheus, victoriametrics, elasticsearch, doris
	Endpoint  string         `gorm:"size:512" json:"endpoint"`
	AuthType  string         `gorm:"size:32" json:"auth_type,omitempty"`
	AuthValue string         `gorm:"size:512" json:"-"` // encrypted/masked in API
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Channel for notifications (Telegram, Lark).
type Channel struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128" json:"name"`
	Type      string         `gorm:"size:32" json:"type"` // telegram, lark
	Config    string         `gorm:"type:text" json:"-"`  // JSON, secrets stored encrypted
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Template for alert content (tag-based).
type Template struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:128" json:"name"`
	ChannelType string         `gorm:"size:32" json:"channel_type"` // generic, telegram, lark
	Body        string         `gorm:"type:text" json:"body"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"` // when true, used when rule has no template or template not found
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Rule for matching and routing alerts.
type Rule struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"size:128" json:"name"`
	Description     string         `gorm:"type:text" json:"description"`        // Human-readable purpose/usage for this rule, available in templates as {{.RuleDescription}}
	Enabled         bool           `gorm:"default:true" json:"enabled"`
	Priority        int            `gorm:"default:0" json:"priority"`
	DatasourceIDs    string         `gorm:"type:text" json:"datasource_ids"`    // JSON array of IDs, empty = all
	QueryLanguage    string         `gorm:"size:32" json:"query_language"`      // promql, elasticsearch_sql, sql, or empty
	QueryExpression  string         `gorm:"type:text" json:"query_expression"` // PromQL, ES SQL, or Doris SQL text
	MatchLabels      string         `gorm:"type:text" json:"match_labels"`     // JSON object
	MatchSeverity    string         `gorm:"size:32" json:"match_severity"`
	ChannelIDs      string         `gorm:"type:text" json:"channel_ids"`     // JSON array
	TemplateID      *uint          `json:"template_id"`
	CheckInterval   string         `gorm:"size:16" json:"check_interval"`    // e.g. 1m
	Duration        string         `gorm:"size:16" json:"duration"`          // e.g. 5m, 0 = immediate
	ExcludeWindows  string         `gorm:"type:text" json:"exclude_windows"`   // JSON array
	RecoveryNotify  bool           `gorm:"default:false" json:"recovery_notify"`
	SendInterval       string         `gorm:"size:16" json:"send_interval"`        // min interval per alert
	AggregationEnabled bool           `gorm:"default:false" json:"aggregation_enabled"` // when true, merge same-type alerts per window; default off to avoid merging different alerts
	AggregateBy        string         `gorm:"size:32" json:"aggregate_by"`         // hostname, instance, etc.
	AggregateWindow    string         `gorm:"size:16" json:"aggregate_window"`
	Suppression     string         `gorm:"type:text" json:"suppression"`      // JSON
	Thresholds      string         `gorm:"type:text" json:"thresholds"`       // JSON array of multi-level thresholds: [{operator,value,severity,channel_ids}]
	JiraEnabled     bool           `gorm:"default:false" json:"jira_enabled"`
	JiraAfterN      int            `gorm:"default:3" json:"jira_after_n"`
	JiraConfig      string         `gorm:"type:text" json:"jira_config,omitempty"` // Accepted on create/update; strip in List/Get for security
	LastRunAt       *time.Time     `json:"last_run_at,omitempty"`                 // last scheduler execution time for this rule
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// Alert unified model (stored for history).
type Alert struct {
	ID          string    `gorm:"primaryKey;size:64" json:"alert_id"`
	SourceID    uint      `gorm:"index" json:"source_id"`
	SourceType  string    `gorm:"size:32;index" json:"source_type"`
	ExternalID  string    `gorm:"size:128;index" json:"external_id,omitempty"`
	Title       string    `gorm:"size:256" json:"title"`
	Severity    string    `gorm:"size:32;index" json:"severity"`
	Status      string    `gorm:"size:32;index" json:"status"` // firing, resolved, suppressed
	FiringAt    time.Time  `gorm:"index" json:"firing_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Labels      string     `gorm:"type:text" json:"labels"`      // JSON
	Annotations string     `gorm:"type:text" json:"annotations"` // JSON
	Raw         string     `gorm:"type:text" json:"-"`           // optional full payload
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// JiraCreated records that we already created a Jira ticket for (rule_id, source_id, external_id) to avoid duplicates.
type JiraCreated struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	RuleID     uint      `gorm:"uniqueIndex:idx_jira_rule_source_ext" json:"rule_id"`
	SourceID   uint      `gorm:"uniqueIndex:idx_jira_rule_source_ext" json:"source_id"`
	ExternalID string   `gorm:"size:128;uniqueIndex:idx_jira_rule_source_ext" json:"external_id"`
	JiraKey    string   `gorm:"size:32" json:"jira_key"`
	CreatedAt  time.Time `json:"created_at"`
}

// AlertSendRecord tracks which channel received which alert (for history detail).
type AlertSendRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AlertID   string    `gorm:"index;size:64" json:"alert_id"`
	ChannelID uint      `gorm:"index:idx_send_rate,priority:1" json:"channel_id"`
	Success   bool      `gorm:"index:idx_send_rate,priority:2" json:"success"`
	Error     string    `gorm:"size:512" json:"error,omitempty"`
	CreatedAt time.Time `gorm:"index:idx_send_rate,priority:3" json:"created_at"`
}

// AlertSilence records manual silence-until time for an alert; no notifications are sent until then.
type AlertSilence struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AlertID      string    `gorm:"uniqueIndex:idx_silence_alert;size:64" json:"alert_id"`
	SilenceUntil time.Time `gorm:"index" json:"silence_until"`
	CreatedAt    time.Time `json:"created_at"`
}

// SystemConfig stores key-value system settings (e.g. retention_days).
type SystemConfig struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `gorm:"size:256" json:"value"`
}
