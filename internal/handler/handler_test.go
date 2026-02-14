package handler

import (
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
)

func TestAuthHandler_LoginPage_NoPasswords(t *testing.T) {
	a := auth.New("secret", "", "")
	h := NewAuthHandler(a)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/login", nil)
	h.LoginPage(w, r)

	// No passwords = redirect to home
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want %q", loc, "/")
	}
}

func TestAuthHandler_LoginPage_WithPassword(t *testing.T) {
	a := auth.New("secret", "dmpass", "")
	h := NewAuthHandler(a)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/login", nil)
	h.LoginPage(w, r)

	// Should render login form (200 OK)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthHandler_LoginSubmit_Correct(t *testing.T) {
	a := auth.New("secret", "dmpass", "")
	h := NewAuthHandler(a)

	form := url.Values{"password": {"dmpass"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.LoginSubmit(w, r)

	// Admin login should redirect to /dm
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/dm" {
		t.Errorf("Location = %q, want %q", loc, "/dm")
	}
}

func TestAuthHandler_LoginSubmit_Wrong(t *testing.T) {
	a := auth.New("secret", "dmpass", "")
	h := NewAuthHandler(a)

	form := url.Values{"password": {"wrong"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.LoginSubmit(w, r)

	// Should render login page with error (200 OK, not redirect)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthHandler_LoginSubmit_PlayerRedirect(t *testing.T) {
	a := auth.New("secret", "dmpass", "playerpass")
	h := NewAuthHandler(a)

	form := url.Values{"password": {"playerpass"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.LoginSubmit(w, r)

	// Player login should redirect to /
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/" {
		t.Errorf("Location = %q, want %q", loc, "/")
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	a := auth.New("secret", "dmpass", "")
	h := NewAuthHandler(a)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/logout", nil)
	h.Logout(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want %q", loc, "/login")
	}
}

func TestChatHandler_Send_Empty(t *testing.T) {
	cs := store.NewChatStore(100)
	broker := realtime.NewBroker()
	h := NewChatHandler(cs, broker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/chat", strings.NewReader("message="))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Send(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// No message should be stored
	msgs := cs.Recent(0)
	if len(msgs) != 0 {
		t.Errorf("len(msgs) = %d, want 0", len(msgs))
	}
}

func TestChatHandler_Send_Message(t *testing.T) {
	cs := store.NewChatStore(100)
	broker := realtime.NewBroker()
	h := NewChatHandler(cs, broker)

	// Subscribe to verify broadcast
	sub := broker.Subscribe(false)
	defer broker.Unsubscribe(sub)

	form := url.Values{"message": {"Hello world!"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/chat", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.Send(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	msgs := cs.Recent(0)
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if msgs[0].Content != "Hello world!" {
		t.Errorf("content = %q, want %q", msgs[0].Content, "Hello world!")
	}
	if msgs[0].Author != "Player" {
		t.Errorf("author = %q, want %q", msgs[0].Author, "Player")
	}
}

func TestChatHandler_Send_DMAuthor(t *testing.T) {
	cs := store.NewChatStore(100)
	broker := realtime.NewBroker()
	h := NewChatHandler(cs, broker)

	form := url.Values{"message": {"DM says hi"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/chat", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Referer", "http://localhost:3000/dm")
	h.Send(w, r)

	msgs := cs.Recent(0)
	if len(msgs) != 1 {
		t.Fatalf("len(msgs) = %d, want 1", len(msgs))
	}
	if msgs[0].Author != "DM" {
		t.Errorf("author = %q, want %q", msgs[0].Author, "DM")
	}
}

func TestMapHandler_Upload(t *testing.T) {
	dir := t.TempDir()
	ms := store.NewMapStore(dir)
	h := NewMapHandler(ms)

	// Build multipart form
	var buf strings.Builder
	w := multipart.NewWriter(&buf)
	w.WriteField("title", "Test Map")
	fw, _ := w.CreateFormFile("map", "dungeon.png")
	fw.Write([]byte("fake-png-data"))
	w.Close()

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/upload", strings.NewReader(buf.String()))
	r.Header.Set("Content-Type", w.FormDataContentType())
	h.Upload(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Verify map was created
	maps, _ := ms.List()
	if len(maps) != 1 {
		t.Fatalf("len(maps) = %d, want 1", len(maps))
	}
	if maps[0].Title != "Test Map" {
		t.Errorf("title = %q, want %q", maps[0].Title, "Test Map")
	}
}

func TestMapHandler_Upload_MissingFile(t *testing.T) {
	dir := t.TempDir()
	ms := store.NewMapStore(dir)
	h := NewMapHandler(ms)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/upload", strings.NewReader(""))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	h.Upload(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMapHandler_SetActive_NotFound(t *testing.T) {
	dir := t.TempDir()
	ms := store.NewMapStore(dir)
	h := NewMapHandler(ms)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/nonexistent/active", nil)
	r.SetPathValue("id", "nonexistent")
	h.SetActive(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
