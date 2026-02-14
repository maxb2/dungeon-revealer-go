package handler

import (
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/templates"
)

type AuthHandler struct {
	auth *auth.Auth
}

func NewAuthHandler(a *auth.Auth) *AuthHandler {
	return &AuthHandler{auth: a}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if !h.auth.NeedsLogin() {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	templates.Login("").Render(r.Context(), w)
}

func (h *AuthHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")

	role, err := h.auth.Login(w, r, password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if role == auth.RoleUnauthenticated {
		templates.Login("Invalid password.").Render(r.Context(), w)
		return
	}

	if role == auth.RoleAdmin {
		http.Redirect(w, r, "/dm", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.auth.Logout(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
