package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewSQLite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := NewSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	if db.DB == nil {
		t.Fatal("db is nil")
	}
	_ = os.Remove(path)
}
