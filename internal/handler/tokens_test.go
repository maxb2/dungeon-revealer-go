package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
)

// loginCookies logs in with the given password and returns the session cookies.
func loginCookies(t *testing.T, a *auth.Auth, pass string) []*http.Cookie {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", strings.NewReader("password="+pass))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := a.Login(w, r, pass); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	return w.Result().Cookies()
}

// withCookies attaches cookies to a request.
func withCookies(r *http.Request, cookies []*http.Cookie) *http.Request {
	for _, c := range cookies {
		r.AddCookie(c)
	}
	return r
}

// serveWithAuth wraps the handler with auth middleware before serving.
func serveWithAuth(a *auth.Auth, h http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	a.Middleware(http.HandlerFunc(h)).ServeHTTP(w, r)
}

func newTestMapStore(t *testing.T) (*store.MapStore, *store.Map) {
	t.Helper()
	ms := store.NewMapStore(t.TempDir())
	m, err := ms.Create("Test Map", ".png", strings.NewReader("fake-png"))
	if err != nil {
		t.Fatalf("create map: %v", err)
	}
	return ms, m
}

func TestTokenHandler_ListTokens_AdminGetsAll(t *testing.T) {
	ms, m := newTestMapStore(t)
	ms.AddToken(m.ID, store.Token{Visible: true, Label: "visible"})
	ms.AddToken(m.ID, store.Token{Visible: false, Label: "hidden"})

	broker := realtime.NewBroker()
	h := NewTokenHandler(ms, broker)
	a := auth.New("secret", "dmpass", "playerpass")
	cookies := loginCookies(t, a, "dmpass")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/maps/"+m.ID+"/tokens", nil)
	r.SetPathValue("id", m.ID)
	r = withCookies(r, cookies)
	serveWithAuth(a, h.ListTokens, w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var tokens []store.Token
	if err := json.NewDecoder(w.Body).Decode(&tokens); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("len(tokens) = %d, want 2 (admin sees all)", len(tokens))
	}
}

func TestTokenHandler_ListTokens_PlayerGetsVisibleOnly(t *testing.T) {
	ms, m := newTestMapStore(t)
	ms.AddToken(m.ID, store.Token{Visible: true, Label: "visible"})
	ms.AddToken(m.ID, store.Token{Visible: false, Label: "hidden"})

	broker := realtime.NewBroker()
	h := NewTokenHandler(ms, broker)
	a := auth.New("secret", "dmpass", "playerpass")
	cookies := loginCookies(t, a, "playerpass")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/maps/"+m.ID+"/tokens", nil)
	r.SetPathValue("id", m.ID)
	r = withCookies(r, cookies)
	serveWithAuth(a, h.ListTokens, w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var tokens []store.Token
	if err := json.NewDecoder(w.Body).Decode(&tokens); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("len(tokens) = %d, want 1 (player sees visible only)", len(tokens))
	}
	if tokens[0].Label != "visible" {
		t.Errorf("token label = %q, want %q", tokens[0].Label, "visible")
	}
}

func TestTokenHandler_CreateToken_DefaultRadiusAndColor(t *testing.T) {
	ms, m := newTestMapStore(t)
	broker := realtime.NewBroker()
	h := NewTokenHandler(ms, broker)

	form := url.Values{"x": {"10"}, "y": {"20"}} // no radius, no color
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/"+m.ID+"/tokens", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", m.ID)
	h.CreateToken(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var token store.Token
	if err := json.NewDecoder(w.Body).Decode(&token); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if token.Radius != 20 {
		t.Errorf("radius = %v, want 20", token.Radius)
	}
	if token.Color != "#e94560" {
		t.Errorf("color = %q, want %q", token.Color, "#e94560")
	}
}

func TestTokenHandler_CreateToken_InvalidShapeBecomesEmpty(t *testing.T) {
	ms, m := newTestMapStore(t)
	broker := realtime.NewBroker()
	h := NewTokenHandler(ms, broker)

	form := url.Values{"shape": {"triangle"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/"+m.ID+"/tokens", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", m.ID)
	h.CreateToken(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var token store.Token
	if err := json.NewDecoder(w.Body).Decode(&token); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if token.Shape != "" {
		t.Errorf("shape = %q, want empty string", token.Shape)
	}
}

func TestTokenHandler_UpdateToken_Player_Forbidden_NonMoveable(t *testing.T) {
	ms, m := newTestMapStore(t)
	tok, _ := ms.AddToken(m.ID, store.Token{Visible: true, Moveable: false, Label: "fixed"})
	broker := realtime.NewBroker()
	h := NewTokenHandler(ms, broker)
	a := auth.New("secret", "dmpass", "playerpass")
	cookies := loginCookies(t, a, "playerpass")

	form := url.Values{"x": {"50"}, "y": {"50"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/maps/"+m.ID+"/tokens/"+tok.ID, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", m.ID)
	r.SetPathValue("tokenId", tok.ID)
	r = withCookies(r, cookies)
	serveWithAuth(a, h.UpdateToken, w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (player cannot move non-moveable token)", w.Code, http.StatusForbidden)
	}
}

func TestTokenHandler_UpdateToken_Player_CanMove_Moveable(t *testing.T) {
	ms, m := newTestMapStore(t)
	tok, _ := ms.AddToken(m.ID, store.Token{Visible: true, Moveable: true, Label: "moveable"})
	broker := realtime.NewBroker()
	h := NewTokenHandler(ms, broker)
	a := auth.New("secret", "dmpass", "playerpass")
	cookies := loginCookies(t, a, "playerpass")

	form := url.Values{"x": {"99"}, "y": {"88"}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/maps/"+m.ID+"/tokens/"+tok.ID, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", m.ID)
	r.SetPathValue("tokenId", tok.ID)
	r = withCookies(r, cookies)
	serveWithAuth(a, h.UpdateToken, w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d (player can move moveable token)", w.Code, http.StatusNoContent)
	}

	// Verify position was updated
	tokens, _ := ms.GetTokens(m.ID)
	if tokens[0].X != 99 || tokens[0].Y != 88 {
		t.Errorf("position = (%v,%v), want (99,88)", tokens[0].X, tokens[0].Y)
	}
}

func TestTokenHandler_DeleteToken_NotFound(t *testing.T) {
	ms, m := newTestMapStore(t)
	broker := realtime.NewBroker()
	h := NewTokenHandler(ms, broker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/maps/"+m.ID+"/tokens/nonexistent", nil)
	r.SetPathValue("id", m.ID)
	r.SetPathValue("tokenId", "nonexistent")
	h.DeleteToken(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
