package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
)

type MapHandler struct {
	maps *store.MapStore
}

func NewMapHandler(maps *store.MapStore) *MapHandler {
	return &MapHandler{maps: maps}
}

func (h *MapHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("map")
	if err != nil {
		http.Error(w, "Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	title := r.FormValue("title")
	if title == "" {
		title = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".png"
	}

	_, err = h.maps.Create(title, ext, file)
	if err != nil {
		http.Error(w, "Failed to save map", http.StatusInternalServerError)
		return
	}

	// Return updated map list partial for HTMX
	maps, _ := h.maps.List()
	templates.MapList(maps, h.maps.ActiveID()).Render(r.Context(), w)
}

func (h *MapHandler) List(w http.ResponseWriter, r *http.Request) {
	maps, _ := h.maps.List()
	templates.MapList(maps, h.maps.ActiveID()).Render(r.Context(), w)
}

func (h *MapHandler) SetActive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.maps.Get(id); err != nil {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}
	h.maps.SetActive(id)

	// Trigger DM map area refresh via HTMX event
	w.Header().Set("HX-Trigger", "mapChanged")

	// Return updated map list
	maps, _ := h.maps.List()
	templates.MapList(maps, h.maps.ActiveID()).Render(r.Context(), w)
}

func (h *MapHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.maps.Delete(id)

	maps, _ := h.maps.List()
	templates.MapList(maps, h.maps.ActiveID()).Render(r.Context(), w)
}

func (h *MapHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	imgPath, err := h.maps.ImagePath(id)
	if err != nil {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, imgPath)
}

func (h *MapHandler) ActiveMapView(w http.ResponseWriter, r *http.Request) {
	activeID := h.maps.ActiveID()
	if activeID == "" {
		templates.MapAreaEmpty().Render(r.Context(), w)
		return
	}
	m, err := h.maps.Get(activeID)
	if err != nil {
		templates.MapAreaEmpty().Render(r.Context(), w)
		return
	}
	templates.MapAreaImage(m).Render(r.Context(), w)
}

func (h *MapHandler) PlayerMapView(w http.ResponseWriter, r *http.Request) {
	activeID := h.maps.ActiveID()
	if activeID == "" {
		templates.PlayerMapEmpty().Render(r.Context(), w)
		return
	}
	m, err := h.maps.Get(activeID)
	if err != nil {
		templates.PlayerMapEmpty().Render(r.Context(), w)
		return
	}
	templates.PlayerMapImage(m).Render(r.Context(), w)
}
