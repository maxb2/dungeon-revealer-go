package handler

import (
	"net/http"

	"github.com/matt/dungeon-revealer-go/templates"
)

type HomeHandler struct{}

func NewHomeHandler() *HomeHandler {
	return &HomeHandler{}
}

func (h *HomeHandler) PlayerPage(w http.ResponseWriter, r *http.Request) {
	templates.Home().Render(r.Context(), w)
}
