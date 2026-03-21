package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
)

type MapHandler struct {
	maps   *store.MapStore
	broker *realtime.Broker
}

func NewMapHandler(maps *store.MapStore, broker *realtime.Broker) *MapHandler {
	return &MapHandler{maps: maps, broker: broker}
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

func (h *MapHandler) UpdateGrid(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	parseFloat := func(key string) float64 {
		v, _ := strconv.ParseFloat(r.FormValue(key), 64)
		return v
	}
	gridSize := parseFloat("gridSize")
	offsetX := parseFloat("gridOffsetX")
	offsetY := parseFloat("gridOffsetY")
	gridEnabled := r.FormValue("gridEnabled") == "true"

	if err := h.maps.UpdateGrid(id, gridSize, offsetX, offsetY, gridEnabled); err != nil {
		http.Error(w, "Failed to update grid", http.StatusInternalServerError)
		return
	}

	type gridEvent struct {
		MapID       string  `json:"mapId"`
		GridSize    float64 `json:"gridSize"`
		GridOffsetX float64 `json:"gridOffsetX"`
		GridOffsetY float64 `json:"gridOffsetY"`
		GridEnabled bool    `json:"gridEnabled"`
	}
	if h.broker != nil {
		data, _ := json.Marshal(gridEvent{MapID: id, GridSize: gridSize, GridOffsetX: offsetX, GridOffsetY: offsetY, GridEnabled: gridEnabled})
		h.broker.Publish(realtime.Event{Name: "gridUpdate", Data: string(data)})
	}

	w.WriteHeader(http.StatusNoContent)
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
