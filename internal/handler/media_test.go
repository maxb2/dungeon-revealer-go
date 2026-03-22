package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matt/dungeon-revealer-go/internal/store"
)

func newMediaTestSetup(t *testing.T) (*store.MediaStore, *MediaHandler) {
	t.Helper()
	db, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ms := store.NewMediaStore(db, t.TempDir())
	return ms, NewMediaHandler(ms)
}

func uploadMediaFile(t *testing.T, h *MediaHandler, filename string, data []byte) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", filename)
	fw.Write(data)
	mw.Close()

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/media/upload", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	h.Upload(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload failed: status %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestMediaHandler_Upload_MissingFile(t *testing.T) {
	_, h := newMediaTestSetup(t)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/media/upload", bytes.NewReader(nil))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	h.Upload(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMediaHandler_Upload_Success(t *testing.T) {
	_, h := newMediaTestSetup(t)
	uploadMediaFile(t, h, "photo.png", []byte("fake-image-data"))
}

func TestMediaHandler_Serve_NotFound(t *testing.T) {
	_, h := newMediaTestSetup(t)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/media/nonexistent", nil)
	r.SetPathValue("id", "nonexistent")
	h.Serve(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMediaHandler_Serve_SetsHeaders(t *testing.T) {
	ms, h := newMediaTestSetup(t)

	uploadMediaFile(t, h, "photo.png", []byte("fake-image-data"))

	files, err := ms.List()
	if err != nil || len(files) == 0 {
		t.Fatalf("expected uploaded file in store")
	}
	id := files[0].ID

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/media/"+id, nil)
	r.SetPathValue("id", id)
	h.Serve(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	// CreateFormFile doesn't set a Content-Type on the part, so the handler
	// falls back to "application/octet-stream".
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/octet-stream")
	}
	if cd := rec.Header().Get("Content-Disposition"); cd == "" {
		t.Error("Content-Disposition header missing")
	}
}

func TestMediaHandler_Delete_Returns200(t *testing.T) {
	ms, h := newMediaTestSetup(t)

	uploadMediaFile(t, h, "todelete.png", []byte("data"))

	files, _ := ms.List()
	if len(files) == 0 {
		t.Fatal("expected 1 file after upload")
	}
	id := files[0].ID

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/media/"+id, nil)
	r.SetPathValue("id", id)
	h.Delete(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
