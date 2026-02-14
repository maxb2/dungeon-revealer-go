package store

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/xid"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type MediaFile struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"contentType"`
	SHA256      string    `json:"sha256"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"createdAt"`
}

type MediaStore struct {
	db      *DB
	dataDir string
}

func NewMediaStore(db *DB, dataDir string) *MediaStore {
	os.MkdirAll(filepath.Join(dataDir, "files"), 0755)
	os.MkdirAll(filepath.Join(dataDir, "token-images"), 0755)
	return &MediaStore{db: db, dataDir: dataDir}
}

func (s *MediaStore) Upload(filename, contentType string, r io.Reader) (*MediaFile, error) {
	// Read into temp file and compute hash
	tmpPath := filepath.Join(s.dataDir, "files", ".tmp-"+xid.New().String())
	tmp, err := os.Create(tmpPath)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpPath)

	h := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, h), r)
	tmp.Close()
	if err != nil {
		return nil, err
	}

	hash := fmt.Sprintf("%x", h.Sum(nil))

	// Check for duplicate
	conn := s.db.Get()
	defer s.db.Put(conn)

	var existing *MediaFile
	sqlitex.ExecuteTransient(conn, `SELECT id, filename, content_type, sha256, size, created_at FROM media WHERE sha256 = ?`, &sqlitex.ExecOptions{
		Args: []any{hash},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			existing = &MediaFile{
				ID:          stmt.ColumnText(0),
				Filename:    stmt.ColumnText(1),
				ContentType: stmt.ColumnText(2),
				SHA256:      stmt.ColumnText(3),
				Size:        stmt.ColumnInt64(4),
			}
			existing.CreatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(5))
			return nil
		},
	})
	if existing != nil {
		return existing, nil
	}

	// Save file
	id := xid.New().String()
	ext := filepath.Ext(filename)
	destPath := filepath.Join(s.dataDir, "files", id+ext)
	if err := os.Rename(tmpPath, destPath); err != nil {
		// Rename may fail across filesystems, fall back to copy
		data, _ := os.ReadFile(tmpPath)
		os.WriteFile(destPath, data, 0644)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	err = sqlitex.ExecuteTransient(conn, `INSERT INTO media (id, filename, content_type, sha256, size, created_at) VALUES (?, ?, ?, ?, ?, ?)`, &sqlitex.ExecOptions{
		Args: []any{id, filename, contentType, hash, size, now},
	})
	if err != nil {
		return nil, err
	}

	return &MediaFile{
		ID: id, Filename: filename, ContentType: contentType,
		SHA256: hash, Size: size, CreatedAt: time.Now(),
	}, nil
}

func (s *MediaStore) Get(id string) (*MediaFile, error) {
	conn := s.db.Get()
	defer s.db.Put(conn)

	var f MediaFile
	found := false
	err := sqlitex.ExecuteTransient(conn, `SELECT id, filename, content_type, sha256, size, created_at FROM media WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			found = true
			f.ID = stmt.ColumnText(0)
			f.Filename = stmt.ColumnText(1)
			f.ContentType = stmt.ColumnText(2)
			f.SHA256 = stmt.ColumnText(3)
			f.Size = stmt.ColumnInt64(4)
			f.CreatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(5))
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("media not found: %s", id)
	}
	return &f, nil
}

func (s *MediaStore) List() ([]MediaFile, error) {
	conn := s.db.Get()
	defer s.db.Put(conn)

	var files []MediaFile
	err := sqlitex.ExecuteTransient(conn, `SELECT id, filename, content_type, sha256, size, created_at FROM media ORDER BY created_at DESC`, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			f := MediaFile{
				ID:          stmt.ColumnText(0),
				Filename:    stmt.ColumnText(1),
				ContentType: stmt.ColumnText(2),
				SHA256:      stmt.ColumnText(3),
				Size:        stmt.ColumnInt64(4),
			}
			f.CreatedAt, _ = time.Parse(time.RFC3339, stmt.ColumnText(5))
			files = append(files, f)
			return nil
		},
	})
	return files, err
}

func (s *MediaStore) FilePath(id string) (string, error) {
	f, err := s.Get(id)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(f.Filename)
	return filepath.Join(s.dataDir, "files", id+ext), nil
}

func (s *MediaStore) Delete(id string) error {
	path, err := s.FilePath(id)
	if err != nil {
		return err
	}
	os.Remove(path)

	conn := s.db.Get()
	defer s.db.Put(conn)
	return sqlitex.ExecuteTransient(conn, `DELETE FROM media WHERE id = ?`, &sqlitex.ExecOptions{
		Args: []any{id},
	})
}
