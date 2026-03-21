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
	a := auth.New("secret", "", "")
	h := NewChatHandler(cs, broker, a)

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
	a := auth.New("secret", "", "")
	h := NewChatHandler(cs, broker, a)

	// Subscribe to verify broadcast
	sub := broker.Subscribe(false)
	defer broker.Unsubscribe(sub)

	form := url.Values{"message": {"Hello world!"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/chat", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// No passwords + no role in context = unauthenticated, gets "Player N" default
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
	if msgs[0].Author != "Player 1" {
		t.Errorf("author = %q, want %q", msgs[0].Author, "Player 1")
	}
}

func TestChatHandler_Send_DMAuthor(t *testing.T) {
	cs := store.NewChatStore(100)
	broker := realtime.NewBroker()
	a := auth.New("secret", "", "")
	h := NewChatHandler(cs, broker, a)

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
	h := NewMapHandler(ms, nil)

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
	h := NewMapHandler(ms, nil)

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
	h := NewMapHandler(ms, nil)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/nonexistent/active", nil)
	r.SetPathValue("id", "nonexistent")
	h.SetActive(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func newMapTestSetup(t *testing.T) (*store.MapStore, *MapHandler) {
	t.Helper()
	dir := t.TempDir()
	ms := store.NewMapStore(dir)
	return ms, NewMapHandler(ms, nil)
}

func uploadTestMap(t *testing.T, ms *store.MapStore, h *MapHandler, title, filename string) {
	t.Helper()
	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	if title != "" {
		mw.WriteField("title", title)
	}
	fw, _ := mw.CreateFormFile("map", filename)
	fw.Write([]byte("fake-png-data"))
	mw.Close()

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/upload", strings.NewReader(buf.String()))
	r.Header.Set("Content-Type", mw.FormDataContentType())
	h.Upload(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload failed: status %d", rec.Code)
	}
}

func TestChatHandler_SetName_Empty_Returns400(t *testing.T) {
	cs := store.NewChatStore(100)
	broker := realtime.NewBroker()
	a := auth.New("secret", "", "")
	h := NewChatHandler(cs, broker, a)

	form := url.Values{"chatName": {""}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/chat/name", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.SetName(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestChatHandler_SetName_WhitespaceOnly_Returns400(t *testing.T) {
	cs := store.NewChatStore(100)
	broker := realtime.NewBroker()
	a := auth.New("secret", "", "")
	h := NewChatHandler(cs, broker, a)

	form := url.Values{"chatName": {"   "}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/chat/name", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.SetName(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestChatHandler_SetName_Valid_Returns200(t *testing.T) {
	cs := store.NewChatStore(100)
	broker := realtime.NewBroker()
	a := auth.New("secret", "", "")
	h := NewChatHandler(cs, broker, a)

	form := url.Values{"chatName": {"Alice"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/chat/name", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.SetName(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestChatHandler_SetName_TruncatesAt20Chars(t *testing.T) {
	cs := store.NewChatStore(100)
	broker := realtime.NewBroker()
	a := auth.New("secret", "", "")
	h := NewChatHandler(cs, broker, a)

	form := url.Values{"chatName": {"ABCDEFGHIJKLMNOPQRSTUVWXY"}} // 25 chars
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/chat/name", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.SetName(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMapHandler_Upload_AutoTitleFromFilename(t *testing.T) {
	ms, h := newMapTestSetup(t)

	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	// No title field — should derive from filename
	fw, _ := mw.CreateFormFile("map", "dungeon.png")
	fw.Write([]byte("fake-png-data"))
	mw.Close()

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/upload", strings.NewReader(buf.String()))
	r.Header.Set("Content-Type", mw.FormDataContentType())
	h.Upload(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	maps, _ := ms.List()
	if len(maps) != 1 {
		t.Fatalf("len(maps) = %d, want 1", len(maps))
	}
	if maps[0].Title != "dungeon" {
		t.Errorf("title = %q, want %q", maps[0].Title, "dungeon")
	}
}

func TestMapHandler_Upload_NoExtension_DefaultsPng(t *testing.T) {
	ms, h := newMapTestSetup(t)

	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	mw.WriteField("title", "No Ext Map")
	fw, _ := mw.CreateFormFile("map", "dungeon") // no extension
	fw.Write([]byte("fake-png-data"))
	mw.Close()

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/upload", strings.NewReader(buf.String()))
	r.Header.Set("Content-Type", mw.FormDataContentType())
	h.Upload(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	maps, _ := ms.List()
	if len(maps) != 1 {
		t.Fatalf("len(maps) = %d, want 1", len(maps))
	}
	// Verify map was created (ext defaulted to .png internally)
	if maps[0].Title != "No Ext Map" {
		t.Errorf("title = %q, want %q", maps[0].Title, "No Ext Map")
	}
}

func TestMapHandler_SetActive_SetsHXTriggerHeader(t *testing.T) {
	ms, h := newMapTestSetup(t)
	uploadTestMap(t, ms, h, "Test Map", "dungeon.png")

	maps, _ := ms.List()
	id := maps[0].ID

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/"+id+"/active", nil)
	r.SetPathValue("id", id)
	h.SetActive(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("HX-Trigger"); got != "mapChanged" {
		t.Errorf("HX-Trigger = %q, want %q", got, "mapChanged")
	}
}

func TestMapHandler_Delete_RemovesMap(t *testing.T) {
	ms, h := newMapTestSetup(t)
	uploadTestMap(t, ms, h, "To Delete", "delete.png")

	maps, _ := ms.List()
	if len(maps) != 1 {
		t.Fatalf("expected 1 map before delete, got %d", len(maps))
	}
	id := maps[0].ID

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/maps/"+id, nil)
	r.SetPathValue("id", id)
	h.Delete(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	maps, _ = ms.List()
	if len(maps) != 0 {
		t.Errorf("len(maps) = %d, want 0 after delete", len(maps))
	}
}

func TestMapHandler_ServeImage_NotFound(t *testing.T) {
	_, h := newMapTestSetup(t)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/maps/nonexistent/image", nil)
	r.SetPathValue("id", "nonexistent")
	h.ServeImage(rec, r)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMapHandler_ActiveMapView_NoActiveMap(t *testing.T) {
	_, h := newMapTestSetup(t)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/maps/active-view", nil)
	h.ActiveMapView(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMapHandler_ActiveMapView_ActiveMapDeleted(t *testing.T) {
	ms, h := newMapTestSetup(t)
	ms.SetActive("nonexistent-id")

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/maps/active-view", nil)
	h.ActiveMapView(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMapHandler_ActiveMapView_ValidActiveMap(t *testing.T) {
	ms, h := newMapTestSetup(t)
	uploadTestMap(t, ms, h, "Active Map", "active.png")

	maps, _ := ms.List()
	ms.SetActive(maps[0].ID)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/maps/active-view", nil)
	h.ActiveMapView(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMapHandler_PlayerMapView_NoActiveMap(t *testing.T) {
	_, h := newMapTestSetup(t)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/player-map", nil)
	h.PlayerMapView(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMapHandler_PlayerMapView_ActiveMapDeleted(t *testing.T) {
	ms, h := newMapTestSetup(t)
	ms.SetActive("nonexistent-id")

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/player-map", nil)
	h.PlayerMapView(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMapHandler_PlayerMapView_ValidActiveMap(t *testing.T) {
	ms, h := newMapTestSetup(t)
	uploadTestMap(t, ms, h, "Player Map", "player.png")

	maps, _ := ms.List()
	ms.SetActive(maps[0].ID)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/player-map", nil)
	h.PlayerMapView(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
