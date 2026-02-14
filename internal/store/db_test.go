package store

import (
	"testing"
)

func TestNew_Migrations(t *testing.T) {
	dir := t.TempDir()
	db, err := New(dir)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer db.Close()

	// Verify we can get a connection
	conn := db.Get()
	if conn == nil {
		t.Fatal("Get returned nil connection")
	}
	db.Put(conn)
}

func TestNew_MigrationsIdempotent(t *testing.T) {
	dir := t.TempDir()

	// First open — runs all migrations
	db1, err := New(dir)
	if err != nil {
		t.Fatalf("first New error: %v", err)
	}
	db1.Close()

	// Second open — migrations should be skipped (already applied)
	db2, err := New(dir)
	if err != nil {
		t.Fatalf("second New error: %v", err)
	}
	db2.Close()
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := New(dir)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
