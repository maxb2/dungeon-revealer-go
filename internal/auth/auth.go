package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gorilla/sessions"
)

type Role string

const (
	RoleAdmin          Role = "admin"
	RolePlayer         Role = "player"
	RoleUnauthenticated Role = "unauthenticated"
)

type contextKey struct{}

const sessionName = "dr-session"

type Auth struct {
	store          sessions.Store
	dmPassword     string
	playerPassword string
	playerCounter  int64
}

func New(secret, dmPassword, playerPassword string) *Auth {
	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30, // 30 days
	}
	return &Auth{
		store:          store,
		dmPassword:     dmPassword,
		playerPassword: playerPassword,
	}
}

func (a *Auth) NeedsLogin() bool {
	return a.dmPassword != "" || a.playerPassword != ""
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request, password string) (Role, error) {
	role := RoleUnauthenticated

	if a.dmPassword != "" && password == a.dmPassword {
		role = RoleAdmin
	} else if a.playerPassword != "" && password == a.playerPassword {
		role = RolePlayer
	} else if a.dmPassword == "" && a.playerPassword == "" {
		// No passwords configured — everyone is admin
		role = RoleAdmin
	} else {
		return RoleUnauthenticated, nil
	}

	session, _ := a.store.Get(r, sessionName)
	session.Values["role"] = string(role)
	if err := session.Save(r, w); err != nil {
		return RoleUnauthenticated, err
	}
	return role, nil
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) error {
	session, _ := a.store.Get(r, sessionName)
	session.Values["role"] = ""
	session.Options.MaxAge = -1
	return session.Save(r, w)
}

func (a *Auth) GetRole(r *http.Request) Role {
	session, err := a.store.Get(r, sessionName)
	if err != nil {
		return RoleUnauthenticated
	}
	roleStr, ok := session.Values["role"].(string)
	if !ok || roleStr == "" {
		return RoleUnauthenticated
	}
	return Role(roleStr)
}

// Middleware injects the role into the request context.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := a.GetRole(r)

		// If no passwords set, everyone is admin
		if !a.NeedsLogin() {
			role = RoleAdmin
		}

		ctx := context.WithValue(r.Context(), contextKey{}, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole returns middleware that enforces a minimum role.
func (a *Auth) RequireRole(minRole Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromContext(r.Context())
			if !hasAccess(role, minRole) {
				log.Printf("Auth: denied %s %s (have=%s, need=%s)", r.Method, r.URL.Path, role, minRole)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RoleFromContext(ctx context.Context) Role {
	role, ok := ctx.Value(contextKey{}).(Role)
	if !ok {
		return RoleUnauthenticated
	}
	return role
}

func hasAccess(have, need Role) bool {
	levels := map[Role]int{
		RoleUnauthenticated: 0,
		RolePlayer:          1,
		RoleAdmin:           2,
	}
	return levels[have] >= levels[need]
}

// GetChatName returns the chat display name from the session.
// If none is set, it generates a default based on isDM and saves it.
func (a *Auth) GetChatName(w http.ResponseWriter, r *http.Request, isDM bool) string {
	session, _ := a.store.Get(r, sessionName)
	if name, ok := session.Values["chatName"].(string); ok && name != "" {
		return name
	}

	var name string
	if isDM {
		name = "DM"
	} else {
		n := atomic.AddInt64(&a.playerCounter, 1)
		name = fmt.Sprintf("Player %d", n)
	}

	session.Values["chatName"] = name
	session.Save(r, w)
	return name
}

// SetChatName updates the chat display name in the session.
func (a *Auth) SetChatName(w http.ResponseWriter, r *http.Request, name string) error {
	name = strings.TrimSpace(name)
	if len(name) > 20 {
		name = name[:20]
	}
	session, _ := a.store.Get(r, sessionName)
	session.Values["chatName"] = name
	return session.Save(r, w)
}
