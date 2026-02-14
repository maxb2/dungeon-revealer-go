package handler

import (
	"io"
	"net/http"
	"os"

	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
)

type FogHandler struct {
	maps   *store.MapStore
	broker *realtime.Broker
}

func NewFogHandler(maps *store.MapStore, broker *realtime.Broker) *FogHandler {
	return &FogHandler{maps: maps, broker: broker}
}

// SaveProgress saves the DM's fog progress image (not yet visible to players).
func (h *FogHandler) SaveProgress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.maps.Get(id); err != nil {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		http.Error(w, "Request too large", http.StatusBadRequest)
		return
	}

	if err := os.WriteFile(h.maps.FogProgressPath(id), body, 0644); err != nil {
		http.Error(w, "Failed to save fog", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Push copies fog-progress to fog-live and notifies players via SSE.
func (h *FogHandler) Push(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.maps.Get(id); err != nil {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}

	progressPath := h.maps.FogProgressPath(id)
	livePath := h.maps.FogLivePath(id)

	data, err := os.ReadFile(progressPath)
	if err != nil {
		http.Error(w, "No fog progress to push", http.StatusBadRequest)
		return
	}

	if err := os.WriteFile(livePath, data, 0644); err != nil {
		http.Error(w, "Failed to push fog", http.StatusInternalServerError)
		return
	}

	// Notify players to refresh their fog
	h.broker.Publish(realtime.Event{
		Name: "fogUpdate",
		Data: id,
	})

	w.WriteHeader(http.StatusNoContent)
}

// ServeProgress serves the fog-progress image (DM only).
func (h *FogHandler) ServeProgress(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path := h.maps.FogProgressPath(id)
	if _, err := os.Stat(path); err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.ServeFile(w, r, path)
}

// ServeLive serves the fog-live image (players see this).
func (h *FogHandler) ServeLive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	path := h.maps.FogLivePath(id)
	if _, err := os.Stat(path); err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.ServeFile(w, r, path)
}
