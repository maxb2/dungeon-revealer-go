package handler

import (
	"encoding/json"
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
)

type WallHandler struct {
	maps   *store.MapStore
	broker *realtime.Broker
}

func NewWallHandler(maps *store.MapStore, broker *realtime.Broker) *WallHandler {
	return &WallHandler{maps: maps, broker: broker}
}

// ListWalls returns all walls as JSON (player-accessible; walls are not secret data).
func (h *WallHandler) ListWalls(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")
	walls, err := h.maps.GetWalls(mapID)
	if err != nil {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}
	if walls == nil {
		walls = []store.Wall{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(walls)
}

// CreateWall adds a polygon wall (DM only). Expects JSON body: {"points":[{"x":…,"y":…},…]}
func (h *WallHandler) CreateWall(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")

	var req struct {
		Points []store.WallPoint `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Points) < 3 {
		http.Error(w, "Need at least 3 points", http.StatusBadRequest)
		return
	}

	wall := store.Wall{Points: req.Points}
	created, err := h.maps.AddWall(mapID, wall)
	if err != nil {
		http.Error(w, "Failed to create wall", http.StatusInternalServerError)
		return
	}

	h.broker.Publish(realtime.Event{Name: "wallUpdate", Data: mapID})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(created)
}

// DeleteWall removes a wall (DM only).
func (h *WallHandler) DeleteWall(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")
	wallID := r.PathValue("wallId")

	if err := h.maps.DeleteWall(mapID, wallID); err != nil {
		http.Error(w, "Wall not found", http.StatusNotFound)
		return
	}

	h.broker.Publish(realtime.Event{Name: "wallUpdate", Data: mapID})
	w.WriteHeader(http.StatusNoContent)
}

// UpdateWall replaces a wall's points (DM only). Expects JSON body: {"points":[…]}
func (h *WallHandler) UpdateWall(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")
	wallID := r.PathValue("wallId")

	var req struct {
		Points []store.WallPoint `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Points) < 3 {
		http.Error(w, "Need at least 3 points", http.StatusBadRequest)
		return
	}

	if err := h.maps.UpdateWall(mapID, wallID, req.Points); err != nil {
		http.Error(w, "Wall not found", http.StatusNotFound)
		return
	}

	h.broker.Publish(realtime.Event{Name: "wallUpdate", Data: mapID})
	w.WriteHeader(http.StatusNoContent)
}

// GetSettings returns map settings as JSON.
func (h *WallHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")
	m, err := h.maps.Get(mapID)
	if err != nil {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dynamicLighting": m.DynamicLighting,
	})
}

// UpdateSettings changes map settings (DM only).
func (h *WallHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")
	dynamicLighting := r.FormValue("dynamicLighting") == "true"

	if err := h.maps.SetDynamicLighting(mapID, dynamicLighting); err != nil {
		http.Error(w, "Failed to update settings", http.StatusInternalServerError)
		return
	}

	h.broker.Publish(realtime.Event{Name: "mapSettingsUpdate", Data: mapID})
	w.WriteHeader(http.StatusNoContent)
}
