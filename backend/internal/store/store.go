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
	); err != nil {
		return err
	}
	return migrateAlertSuppressionsToSilences(db)
}

// migrateAlertSuppressionsToSilences one-time: copy alert_suppressions -> alert_silences, drop old table.
func migrateAlertSuppressionsToSilences(db *gorm.DB) error {
	if res := db.Exec("SELECT 1 FROM alert_suppressions LIMIT 1"); res.Error != nil {
		return nil // old table does not exist
	}
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
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(50)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
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
