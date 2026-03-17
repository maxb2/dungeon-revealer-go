package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type DB struct {
	pool *sqlitex.Pool
}

func New(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "db.sqlite")
	pool, err := sqlitex.NewPool(dbPath, sqlitex.PoolOptions{
		PoolSize: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db := &DB{pool: pool}
	if err := db.migrate(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.pool.Close()
}

func (db *DB) Get() *sqlite.Conn {
	return db.pool.Get(nil)
}

func (db *DB) Put(conn *sqlite.Conn) {
	db.pool.Put(conn)
}

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY
	);`,
	`CREATE TABLE IF NOT EXISTS notes (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '',
		is_entry_point INTEGER NOT NULL DEFAULT 0,
		is_public INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(
		title, content, content=notes, content_rowid=rowid
	);`,
	`CREATE TABLE IF NOT EXISTS media (
		id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		content_type TEXT NOT NULL DEFAULT '',
		sha256 TEXT NOT NULL DEFAULT '',
		size INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`,
	`ALTER TABLE notes ADD COLUMN is_locked INTEGER NOT NULL DEFAULT 0;`,
}

func (db *DB) migrate() error {
	conn := db.pool.Get(nil)
	defer db.pool.Put(conn)

	// Ensure migrations table exists
	if err := sqlitex.ExecuteTransient(conn, migrations[0], nil); err != nil {
		return err
	}

	// Get current version
	var currentVersion int
	if err := sqlitex.ExecuteTransient(conn, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			currentVersion = stmt.ColumnInt(0)
			return nil
		},
	}); err != nil {
		return err
	}

	// Run pending migrations (skip index 0 which is the migrations table itself)
	for i := currentVersion + 1; i < len(migrations); i++ {
		log.Printf("Running migration %d", i)
		if err := sqlitex.ExecuteTransient(conn, migrations[i], nil); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
		if err := sqlitex.ExecuteTransient(conn, "INSERT INTO schema_migrations (version) VALUES (?)", &sqlitex.ExecOptions{
			Args: []any{i},
		}); err != nil {
			return fmt.Errorf("record migration %d: %w", i, err)
		}
	}

	return nil
}
