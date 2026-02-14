package handler

import (
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/templates"
)

type HomeHandler struct {
	auth *auth.Auth
}

func NewHomeHandler(a *auth.Auth) *HomeHandler {
	return &HomeHandler{auth: a}
}

func (h *HomeHandler) PlayerPage(w http.ResponseWriter, r *http.Request) {
	chatName := h.auth.GetChatName(w, r, false)
	templates.Home(chatName).Render(r.Context(), w)
}
