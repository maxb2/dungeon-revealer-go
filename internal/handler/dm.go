package handler

import (
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
)

type DMHandler struct {
	maps *store.MapStore
	auth *auth.Auth
}

func NewDMHandler(maps *store.MapStore, a *auth.Auth) *DMHandler {
	return &DMHandler{maps: maps, auth: a}
}

func (h *DMHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	maps, _ := h.maps.List()
	chatName := h.auth.GetChatName(w, r, true)
	templates.DMWithMaps(maps, h.maps.ActiveID(), chatName).Render(r.Context(), w)
}
