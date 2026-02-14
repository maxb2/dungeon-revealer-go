package auth

import (
	"context"
	"log"
	"net/http"

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
