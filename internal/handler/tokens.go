package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
)

type TokenHandler struct {
	maps   *store.MapStore
	broker *realtime.Broker
}

func NewTokenHandler(maps *store.MapStore, broker *realtime.Broker) *TokenHandler {
	return &TokenHandler{maps: maps, broker: broker}
}

// ListTokens returns tokens as JSON. DM gets all, player gets visible only.
func (h *TokenHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")
	role := auth.RoleFromContext(r.Context())

	var tokens []store.Token
	var err error
	if role == auth.RoleAdmin {
		tokens, err = h.maps.GetTokens(mapID)
	} else {
		tokens, err = h.maps.GetVisibleTokens(mapID)
	}
	if err != nil {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}
	if tokens == nil {
		tokens = []store.Token{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

// CreateToken adds a new token to the map (DM only).
func (h *TokenHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")

	var t store.Token
	t.X, _ = strconv.ParseFloat(r.FormValue("x"), 64)
	t.Y, _ = strconv.ParseFloat(r.FormValue("y"), 64)
	t.Radius, _ = strconv.ParseFloat(r.FormValue("radius"), 64)
	if t.Radius <= 0 {
		t.Radius = 20
	}
	t.Label = r.FormValue("label")
	t.Color = r.FormValue("color")
	if t.Color == "" {
		t.Color = "#e94560"
	}
	t.Visible = r.FormValue("visible") == "true"
	t.Moveable = r.FormValue("moveable") == "true"
	t.Shape = r.FormValue("shape")
	if t.Shape != "circle" && t.Shape != "square" {
		t.Shape = ""
	}
	t.LabelSize, _ = strconv.ParseFloat(r.FormValue("labelSize"), 64)
	if t.LabelSize < 0 {
		t.LabelSize = 0
	}
	t.SightRadius, _ = strconv.ParseFloat(r.FormValue("sightRadius"), 64)
	if t.SightRadius < 0 {
		t.SightRadius = 0
	}

	token, err := h.maps.AddToken(mapID, t)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	h.publishTokenUpdate(mapID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

// UpdateToken modifies a token (DM can change all, player can only move moveable tokens).
func (h *TokenHandler) UpdateToken(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")
	tokenID := r.PathValue("tokenId")
	role := auth.RoleFromContext(r.Context())

	// Get existing token
	tokens, err := h.maps.GetTokens(mapID)
	if err != nil {
		http.Error(w, "Map not found", http.StatusNotFound)
		return
	}

	var existing *store.Token
	for _, t := range tokens {
		if t.ID == tokenID {
			existing = &t
			break
		}
	}
	if existing == nil {
		http.Error(w, "Token not found", http.StatusNotFound)
		return
	}

	// Players can only update position of moveable tokens
	if role != auth.RoleAdmin {
		if !existing.Moveable {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.FormValue("x") != "" {
			existing.X, _ = strconv.ParseFloat(r.FormValue("x"), 64)
		}
		if r.FormValue("y") != "" {
			existing.Y, _ = strconv.ParseFloat(r.FormValue("y"), 64)
		}
	} else {
		// DM can update everything
		r.ParseForm()
		if v := r.FormValue("x"); v != "" {
			existing.X, _ = strconv.ParseFloat(v, 64)
		}
		if v := r.FormValue("y"); v != "" {
			existing.Y, _ = strconv.ParseFloat(v, 64)
		}
		if v := r.FormValue("radius"); v != "" {
			existing.Radius, _ = strconv.ParseFloat(v, 64)
		}
		if _, ok := r.Form["label"]; ok {
			existing.Label = r.FormValue("label")
		}
		if v := r.FormValue("color"); v != "" {
			existing.Color = v
		}
		if r.FormValue("visible") != "" {
			existing.Visible = r.FormValue("visible") == "true"
		}
		if r.FormValue("moveable") != "" {
			existing.Moveable = r.FormValue("moveable") == "true"
		}
		if v := r.FormValue("shape"); v == "circle" || v == "square" {
			existing.Shape = v
		} else if r.FormValue("shape") == "" {
			existing.Shape = ""
		}
		if v := r.FormValue("labelSize"); v != "" {
			if ls, err := strconv.ParseFloat(v, 64); err == nil && ls >= 0 {
				existing.LabelSize = ls
			}
		}
		if v := r.FormValue("sightRadius"); v != "" {
			if sr, err := strconv.ParseFloat(v, 64); err == nil && sr >= 0 {
				existing.SightRadius = sr
			}
		}
	}

	if err := h.maps.UpdateToken(mapID, *existing); err != nil {
		http.Error(w, "Failed to update token", http.StatusInternalServerError)
		return
	}

	h.publishTokenUpdate(mapID)
	w.WriteHeader(http.StatusNoContent)
}

// DeleteToken removes a token (DM only).
func (h *TokenHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	mapID := r.PathValue("id")
	tokenID := r.PathValue("tokenId")

	if err := h.maps.DeleteToken(mapID, tokenID); err != nil {
		http.Error(w, "Token not found", http.StatusNotFound)
		return
	}

	h.publishTokenUpdate(mapID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *TokenHandler) publishTokenUpdate(mapID string) {
	h.broker.Publish(realtime.Event{
		Name: "tokenUpdate",
		Data: mapID,
	})
}
