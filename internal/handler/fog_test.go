package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
)

func newFogTestSetup(t *testing.T) (*store.MapStore, *store.Map, *realtime.Broker, *FogHandler) {
	t.Helper()
	ms := store.NewMapStore(t.TempDir())
	m, err := ms.Create("Fog Map", ".png", strings.NewReader("fake-png"))
	if err != nil {
		t.Fatalf("create map: %v", err)
	}
	broker := realtime.NewBroker()
	h := NewFogHandler(ms, broker)
	return ms, m, broker, h
}

func TestFogHandler_SaveProgress_MapNotFound(t *testing.T) {
	ms := store.NewMapStore(t.TempDir())
	broker := realtime.NewBroker()
	h := NewFogHandler(ms, broker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/nonexistent/fog/progress", strings.NewReader("data"))
	r.SetPathValue("id", "nonexistent")
	h.SaveProgress(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestFogHandler_SaveProgress_Success(t *testing.T) {
	ms, m, _, h := newFogTestSetup(t)

	body := "fake-fog-data"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/"+m.ID+"/fog/progress", strings.NewReader(body))
	r.SetPathValue("id", m.ID)
	h.SaveProgress(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify the file was written
	data, err := os.ReadFile(ms.FogProgressPath(m.ID))
	if err != nil {
		t.Fatalf("progress file not found: %v", err)
	}
	if string(data) != body {
		t.Errorf("file content = %q, want %q", string(data), body)
	}
}

func TestFogHandler_Push_NoProgressFile(t *testing.T) {
	_, m, _, h := newFogTestSetup(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/"+m.ID+"/fog/push", nil)
	r.SetPathValue("id", m.ID)
	h.Push(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestFogHandler_Push_Success(t *testing.T) {
	ms, m, _, h := newFogTestSetup(t)

	// Save progress first
	progressData := "fog-progress-bytes"
	os.WriteFile(ms.FogProgressPath(m.ID), []byte(progressData), 0644)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/maps/"+m.ID+"/fog/push", nil)
	r.SetPathValue("id", m.ID)
	h.Push(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify live file matches progress file
	live, err := os.ReadFile(ms.FogLivePath(m.ID))
	if err != nil {
		t.Fatalf("live file not found: %v", err)
	}
	if string(live) != progressData {
		t.Errorf("live content = %q, want %q", string(live), progressData)
	}
}

func TestFogHandler_ServeProgress_NoFile(t *testing.T) {
	_, m, _, h := newFogTestSetup(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/maps/"+m.ID+"/fog/progress", nil)
	r.SetPathValue("id", m.ID)
	h.ServeProgress(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestFogHandler_ServeLive_NoFile(t *testing.T) {
	_, m, _, h := newFogTestSetup(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/maps/"+m.ID+"/fog/live", nil)
	r.SetPathValue("id", m.ID)
	h.ServeLive(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
