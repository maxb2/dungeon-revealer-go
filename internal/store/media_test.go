package store

import (
	"strings"
	"testing"
)

func TestMedia_UploadGetListDelete(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	ms := NewMediaStore(db, dir)

	// Upload
	mf, err := ms.Upload("test.png", "image/png", strings.NewReader("fake-image-data"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}
	if mf.ID == "" {
		t.Error("media ID is empty")
	}
	if mf.Filename != "test.png" {
		t.Errorf("filename = %q, want %q", mf.Filename, "test.png")
	}
	if mf.ContentType != "image/png" {
		t.Errorf("contentType = %q, want %q", mf.ContentType, "image/png")
	}
	if mf.SHA256 == "" {
		t.Error("SHA256 is empty")
	}
	if mf.Size != int64(len("fake-image-data")) {
		t.Errorf("size = %d, want %d", mf.Size, len("fake-image-data"))
	}

	// Get
	got, err := ms.Get(mf.ID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Filename != "test.png" {
		t.Errorf("Get filename = %q, want %q", got.Filename, "test.png")
	}

	// List
	files, err := ms.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("len(List) = %d, want 1", len(files))
	}

	// FilePath
	path, err := ms.FilePath(mf.ID)
	if err != nil {
		t.Fatalf("FilePath error: %v", err)
	}
	if !strings.HasSuffix(path, mf.ID+".png") {
		t.Errorf("FilePath = %q, want suffix %q", path, mf.ID+".png")
	}

	// Delete
	err = ms.Delete(mf.ID)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	files, _ = ms.List()
	if len(files) != 0 {
		t.Errorf("after delete len(List) = %d, want 0", len(files))
	}
}

func TestMedia_Dedup(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	ms := NewMediaStore(db, dir)

	content := "identical-content"

	// Upload same content twice
	mf1, err := ms.Upload("file1.png", "image/png", strings.NewReader(content))
	if err != nil {
		t.Fatalf("first Upload error: %v", err)
	}

	mf2, err := ms.Upload("file2.png", "image/png", strings.NewReader(content))
	if err != nil {
		t.Fatalf("second Upload error: %v", err)
	}

	// Should return the same ID (dedup by SHA256)
	if mf1.ID != mf2.ID {
		t.Errorf("dedup failed: IDs differ (%q vs %q)", mf1.ID, mf2.ID)
	}

	// List should only show one entry
	files, _ := ms.List()
	if len(files) != 1 {
		t.Errorf("len(List) = %d, want 1 (dedup)", len(files))
	}
}

func TestMedia_GetNotFound(t *testing.T) {
	db := newTestDB(t)
	dir := t.TempDir()
	ms := NewMediaStore(db, dir)

	_, err := ms.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent media")
	}
}
