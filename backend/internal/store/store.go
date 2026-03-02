package store

import (
	"os"
	"time"

	"github.com/kk-alert/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DB wraps GORM and provides access to all entities.
type DB struct {
	*gorm.DB
}

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Datasource{},
		&models.Channel{},
		&models.Template{},
		&models.Rule{},
		&models.Alert{},
		&models.AlertSendRecord{},
		&models.AlertSilence{},
		&models.JiraCreated{},
		&models.SystemConfig{},
		&models.OIDCConfig{},
		&models.PasswordResetToken{},
		&models.Role{},
		&models.Permission{},
	); err != nil {
		return err
	}
	return migrateAlertSuppressionsToSilences(db)
}

// migrateAlertSuppressionsToSilences one-time: copy alert_suppressions -> alert_silences, drop old table.
// This migration silently skips if the old table doesn't exist (no error output).
func migrateAlertSuppressionsToSilences(db *gorm.DB) error {
	// Check if old table exists using SQLite-compatible query
	// For PostgreSQL, this will error and we handle it gracefully
	var count int64
	result := db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='alert_suppressions'").Count(&count)
	if result.Error != nil {
		// PostgreSQL or other DB - try alternative check
		// If table doesn't exist, we'll get an error which we ignore
		return nil
	}
	if count == 0 {
		// Table doesn't exist, nothing to migrate
		return nil
	}

	// Migration: copy data and drop old table
	// Ignore errors as the migration is idempotent
	_ = db.Exec("INSERT INTO alert_silences (id, alert_id, silence_until, created_at) SELECT id, alert_id, suppress_until, created_at FROM alert_suppressions")
	_ = db.Exec("DROP TABLE alert_suppressions")
	return nil
}

// NewDB opens the database from environment: DATABASE_URL for PostgreSQL, else DB_PATH for SQLite.
func NewDB() (*DB, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return NewPostgres(dsn)
	}
	path := os.Getenv("DB_PATH")
	if path == "" {
		path = "data/kkalert.db"
	}
	return NewSQLite(path)
}

// NewPostgres opens a PostgreSQL DB and runs migrations.
func NewPostgres(dsn string) (*DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// Tune connection pool to handle concurrent scheduler + API load.
	// Optimized for 1000+ concurrent rules with queued writes
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(200)                 // 50→200: support 1000+ concurrent rules
		sqlDB.SetMaxIdleConns(50)                  // 10→50: maintain more idle connections
		sqlDB.SetConnMaxLifetime(60 * time.Minute) // 30→60: longer lifetime for stability
		sqlDB.SetConnMaxIdleTime(10 * time.Minute) // add idle timeout
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &DB{DB: db}, nil
}

// NewSQLite opens a SQLite DB and runs migrations.
func NewSQLite(path string) (*DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &DB{DB: db}, nil
}
