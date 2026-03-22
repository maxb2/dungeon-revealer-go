package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
)

func newNotesTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.New(t.TempDir())
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNotesHandler_PlayerView_PrivateNote_Returns404(t *testing.T) {
	db := newNotesTestDB(t)
	note, err := db.CreateNote("Private", "secret content", false, false /* isPublic=false */)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	broker := realtime.NewBroker()
	h := NewNotesHandler(db, broker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/notes/"+note.ID, nil)
	r.SetPathValue("id", note.ID)
	h.PlayerView(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (private note must not be visible to player)", w.Code, http.StatusNotFound)
	}
}

func TestNotesHandler_PlayerView_PublicNote_Returns200(t *testing.T) {
	db := newNotesTestDB(t)
	note, err := db.CreateNote("Public", "public content", false, true /* isPublic=true */)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	broker := realtime.NewBroker()
	h := NewNotesHandler(db, broker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/notes/"+note.ID, nil)
	r.SetPathValue("id", note.ID)
	h.PlayerView(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNotesHandler_PlayerEditForm_LockedNote_Returns403(t *testing.T) {
	db := newNotesTestDB(t)
	note, err := db.CreateNote("Locked", "locked content", false, true /* isPublic=true */)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if err := db.LockNote(note.ID, true); err != nil {
		t.Fatalf("LockNote: %v", err)
	}

	broker := realtime.NewBroker()
	h := NewNotesHandler(db, broker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/notes/"+note.ID+"/edit", nil)
	r.SetPathValue("id", note.ID)
	h.PlayerEditForm(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (locked note cannot be edited by player)", w.Code, http.StatusForbidden)
	}
}

func TestNotesHandler_PlayerSave_PrivateNote_Returns404(t *testing.T) {
	db := newNotesTestDB(t)
	note, err := db.CreateNote("Private", "secret content", false, false /* isPublic=false */)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	broker := realtime.NewBroker()
	h := NewNotesHandler(db, broker)

	form := url.Values{"title": {"updated"}, "content": {"new content"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/notes/"+note.ID, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", note.ID)
	h.PlayerSave(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d (private note cannot be saved by player)", w.Code, http.StatusNotFound)
	}
}

func TestNotesHandler_PlayerSave_LockedNote_Returns403(t *testing.T) {
	db := newNotesTestDB(t)
	note, err := db.CreateNote("Locked", "locked content", false, true /* isPublic=true */)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	if err := db.LockNote(note.ID, true); err != nil {
		t.Fatalf("LockNote: %v", err)
	}

	broker := realtime.NewBroker()
	h := NewNotesHandler(db, broker)

	form := url.Values{"title": {"updated"}, "content": {"new content"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/notes/"+note.ID, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", note.ID)
	h.PlayerSave(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (locked note cannot be saved by player)", w.Code, http.StatusForbidden)
	}
}
