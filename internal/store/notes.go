package store

import (
	"fmt"
	"time"

	"github.com/rs/xid"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Note struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	IsEntryPoint bool      `json:"isEntryPoint"`
	IsPublic     bool      `json:"isPublic"`
	IsLocked     bool      `json:"isLocked"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (db *DB) CreateNote(title, content string, isEntryPoint, isPublic bool) (*Note, error) {
	conn := db.Get()
	defer db.Put(conn)

	id := xid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	err := sqlitex.ExecuteTransient(conn, `INSERT INTO notes (id, title, content, is_entry_point, is_public, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, &sqlitex.ExecOptions{
		Args: []any{id, title, content, boolToInt(isEntryPoint), boolToInt(isPublic), now, now},
	})
	if err != nil {
		return nil, fmt.Errorf("insert note: %w", err)
	}

	// Update FTS
	sqlitex.ExecuteTransient(conn, `INSERT INTO notes_fts(rowid, title, content) SELECT rowid, title, content FROM notes WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{id},
	})

	return &Note{
		ID: id, Title: title, Content: content,
		IsEntryPoint: isEntryPoint, IsPublic: isPublic,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (db *DB) UpdateNote(id, title, content string, isEntryPoint, isPublic bool) error {
	conn := db.Get()
	defer db.Put(conn)

	now := time.Now().UTC().Format(time.RFC3339)

	// Remove old FTS entry
	sqlitex.ExecuteTransient(conn, `DELETE FROM notes_fts WHERE rowid = (SELECT rowid FROM notes WHERE id = ?)`, &sqlitex.ExecOptions{
		Args: []any{id},
	})

	err := sqlitex.ExecuteTransient(conn, `UPDATE notes SET title = ?, content = ?, is_entry_point = ?, is_public = ?, updated_at = ? WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{title, content, boolToInt(isEntryPoint), boolToInt(isPublic), now, id},
	})
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}

	// Re-add FTS entry
	sqlitex.ExecuteTransient(conn, `INSERT INTO notes_fts(rowid, title, content) SELECT rowid, title, content FROM notes WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{id},
	})

	return nil
}

func (db *DB) LockNote(id string, locked bool) error {
	conn := db.Get()
	defer db.Put(conn)

	return sqlitex.ExecuteTransient(conn, `UPDATE notes SET is_locked = ? WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{boolToInt(locked), id},
	})
}

func (db *DB) DeleteNote(id string) error {
	conn := db.Get()
	defer db.Put(conn)

	sqlitex.ExecuteTransient(conn, `DELETE FROM notes_fts WHERE rowid = (SELECT rowid FROM notes WHERE id = ?)`, &sqlitex.ExecOptions{
		Args: []any{id},
	})

	return sqlitex.ExecuteTransient(conn, `DELETE FROM notes WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{id},
	})
}

func (db *DB) GetNote(id string) (*Note, error) {
	conn := db.Get()
	defer db.Put(conn)

	var note Note
	found := false
	err := sqlitex.ExecuteTransient(conn, `SELECT id, title, content, is_entry_point, is_public, is_locked, created_at, updated_at FROM notes WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			found = true
			note.ID = stmt.ColumnText(0)
			note.Title = stmt.ColumnText(1)
			note.Content = stmt.ColumnText(2)
			note.IsEntryPoint = stmt.ColumnInt(3) == 1
			note.IsPublic = stmt.ColumnInt(4) == 1
			note.IsLocked = stmt.ColumnInt(5) == 1
			note.CreatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(6))
			note.UpdatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(7))
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("note not found: %s", id)
	}
	return &note, nil
}

func (db *DB) ListNotes(publicOnly bool) ([]Note, error) {
	conn := db.Get()
	defer db.Put(conn)

	query := `SELECT id, title, content, is_entry_point, is_public, is_locked, created_at, updated_at FROM notes ORDER BY updated_at DESC`
	if publicOnly {
		query = `SELECT id, title, content, is_entry_point, is_public, is_locked, created_at, updated_at FROM notes WHERE is_public = 1 ORDER BY updated_at DESC`
	}

	var notes []Note
	err := sqlitex.ExecuteTransient(conn, query, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			n := Note{
				ID:           stmt.ColumnText(0),
				Title:        stmt.ColumnText(1),
				Content:      stmt.ColumnText(2),
				IsEntryPoint: stmt.ColumnInt(3) == 1,
				IsPublic:     stmt.ColumnInt(4) == 1,
				IsLocked:     stmt.ColumnInt(5) == 1,
			}
			n.CreatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(6))
			n.UpdatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(7))
			notes = append(notes, n)
			return nil
		},
	})
	return notes, err
}

func (db *DB) SearchNotes(query string, publicOnly bool) ([]Note, error) {
	conn := db.Get()
	defer db.Put(conn)

	sql := `SELECT n.id, n.title, n.content, n.is_entry_point, n.is_public, n.is_locked, n.created_at, n.updated_at
		FROM notes_fts f JOIN notes n ON f.rowid = n.rowid
		WHERE notes_fts MATCH ?`
	if publicOnly {
		sql += ` AND n.is_public = 1`
	}
	sql += ` ORDER BY rank`

	var notes []Note
	err := sqlitex.ExecuteTransient(conn, sql, &sqlitex.ExecOptions{
		Args: []any{query},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			n := Note{
				ID:           stmt.ColumnText(0),
				Title:        stmt.ColumnText(1),
				Content:      stmt.ColumnText(2),
				IsEntryPoint: stmt.ColumnInt(3) == 1,
				IsPublic:     stmt.ColumnInt(4) == 1,
				IsLocked:     stmt.ColumnInt(5) == 1,
			}
			n.CreatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(6))
			n.UpdatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(7))
			notes = append(notes, n)
			return nil
		},
	})
	return notes, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
