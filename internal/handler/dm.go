package handler

import (
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
)

type DMHandler struct {
	maps *store.MapStore
}

func NewDMHandler(maps *store.MapStore) *DMHandler {
	return &DMHandler{maps: maps}
}

func (h *DMHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	maps, _ := h.maps.List()
	templates.DMWithMaps(maps, h.maps.ActiveID()).Render(r.Context(), w)
}
