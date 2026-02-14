package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHasAccess(t *testing.T) {
	tests := []struct {
		have, need Role
		want       bool
	}{
		{RoleAdmin, RoleAdmin, true},
		{RoleAdmin, RolePlayer, true},
		{RoleAdmin, RoleUnauthenticated, true},
		{RolePlayer, RoleAdmin, false},
		{RolePlayer, RolePlayer, true},
		{RolePlayer, RoleUnauthenticated, true},
		{RoleUnauthenticated, RoleAdmin, false},
		{RoleUnauthenticated, RolePlayer, false},
		{RoleUnauthenticated, RoleUnauthenticated, true},
	}
	for _, tt := range tests {
		got := hasAccess(tt.have, tt.need)
		if got != tt.want {
			t.Errorf("hasAccess(%s, %s) = %v, want %v", tt.have, tt.need, got, tt.want)
		}
	}
}

func TestNeedsLogin(t *testing.T) {
	tests := []struct {
		dm, player string
		want       bool
	}{
		{"", "", false},
		{"secret", "", true},
		{"", "secret", true},
		{"dm", "player", true},
	}
	for _, tt := range tests {
		a := New("test-secret", tt.dm, tt.player)
		got := a.NeedsLogin()
		if got != tt.want {
			t.Errorf("NeedsLogin(dm=%q, player=%q) = %v, want %v", tt.dm, tt.player, got, tt.want)
		}
	}
}

func TestRoleFromContext(t *testing.T) {
	// Empty context returns unauthenticated
	role := RoleFromContext(context.Background())
	if role != RoleUnauthenticated {
		t.Errorf("empty context role = %s, want %s", role, RoleUnauthenticated)
	}

	// Round-trip through context
	ctx := context.WithValue(context.Background(), contextKey{}, RoleAdmin)
	role = RoleFromContext(ctx)
	if role != RoleAdmin {
		t.Errorf("context role = %s, want %s", role, RoleAdmin)
	}
}

func TestLogin(t *testing.T) {
	a := New("test-secret", "dmpass", "playerpass")

	tests := []struct {
		password string
		wantRole Role
	}{
		{"dmpass", RoleAdmin},
		{"playerpass", RolePlayer},
		{"wrong", RoleUnauthenticated},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/login", nil)
		role, err := a.Login(w, r, tt.password)
		if err != nil {
			t.Fatalf("Login(%q) error: %v", tt.password, err)
		}
		if role != tt.wantRole {
			t.Errorf("Login(%q) = %s, want %s", tt.password, role, tt.wantRole)
		}
	}
}

func TestLogin_NoPasswords(t *testing.T) {
	a := New("test-secret", "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)
	role, err := a.Login(w, r, "anything")
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if role != RoleAdmin {
		t.Errorf("role = %s, want %s (no passwords = admin)", role, RoleAdmin)
	}
}

func TestLogout(t *testing.T) {
	a := New("test-secret", "dmpass", "")

	// Login first
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)
	a.Login(w, r, "dmpass")

	// Extract cookies for next request
	cookies := w.Result().Cookies()

	// Logout
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/logout", nil)
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	if err := a.Logout(w2, r2); err != nil {
		t.Fatalf("Logout error: %v", err)
	}

	// Verify session is cleared by checking role on a new request with the logout cookies
	w3 := httptest.NewRecorder()
	_ = w3
	r3 := httptest.NewRequest("GET", "/", nil)
	for _, c := range w2.Result().Cookies() {
		r3.AddCookie(c)
	}
	role := a.GetRole(r3)
	if role != RoleUnauthenticated {
		t.Errorf("after logout role = %s, want %s", role, RoleUnauthenticated)
	}
}

func TestMiddleware_NoPasswords(t *testing.T) {
	a := New("test-secret", "", "")

	var gotRole Role
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole = RoleFromContext(r.Context())
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if gotRole != RoleAdmin {
		t.Errorf("middleware role = %s, want %s (no passwords)", gotRole, RoleAdmin)
	}
}

func TestMiddleware_WithSession(t *testing.T) {
	a := New("test-secret", "dmpass", "")

	// Login to get session cookie
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)
	a.Login(w, r, "dmpass")
	cookies := w.Result().Cookies()

	// Make request with session cookie through middleware
	var gotRole Role
	handler := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRole = RoleFromContext(r.Context())
	}))

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/", nil)
	for _, c := range cookies {
		r2.AddCookie(c)
	}
	handler.ServeHTTP(w2, r2)

	if gotRole != RoleAdmin {
		t.Errorf("middleware role = %s, want %s", gotRole, RoleAdmin)
	}
}

func TestRequireRole_Allowed(t *testing.T) {
	a := New("test-secret", "", "")

	called := false
	handler := a.RequireRole(RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	// Inject admin role via context
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/dm", nil)
	ctx := context.WithValue(r.Context(), contextKey{}, RoleAdmin)
	handler.ServeHTTP(w, r.WithContext(ctx))

	if !called {
		t.Error("handler was not called for admin role")
	}
}

func TestRequireRole_Denied(t *testing.T) {
	a := New("test-secret", "dmpass", "")

	called := false
	handler := a.RequireRole(RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	// Inject player role via context
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/dm", nil)
	ctx := context.WithValue(r.Context(), contextKey{}, RolePlayer)
	handler.ServeHTTP(w, r.WithContext(ctx))

	if called {
		t.Error("handler was called for insufficient role")
	}
	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}
