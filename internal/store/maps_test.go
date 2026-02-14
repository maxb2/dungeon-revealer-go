package store

import (
	"strings"
	"testing"
)

func TestMapStore_CreateGetListDelete(t *testing.T) {
	dir := t.TempDir()
	ms := NewMapStore(dir)

	// Create
	m, err := ms.Create("Test Map", ".png", strings.NewReader("fake-image-data"))
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if m.Title != "Test Map" {
		t.Errorf("title = %q, want %q", m.Title, "Test Map")
	}
	if m.Ext != ".png" {
		t.Errorf("ext = %q, want %q", m.Ext, ".png")
	}
	if m.ID == "" {
		t.Error("ID is empty")
	}

	// Get
	got, err := ms.Get(m.ID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Title != "Test Map" {
		t.Errorf("Get title = %q, want %q", got.Title, "Test Map")
	}

	// List
	maps, err := ms.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(maps) != 1 {
		t.Fatalf("len(List) = %d, want 1", len(maps))
	}

	// Create another
	m2, err := ms.Create("Map 2", ".jpg", strings.NewReader("more-data"))
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	maps, _ = ms.List()
	if len(maps) != 2 {
		t.Fatalf("len(List) = %d, want 2", len(maps))
	}

	// Delete
	if err := ms.Delete(m.ID); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	maps, _ = ms.List()
	if len(maps) != 1 {
		t.Fatalf("after delete len(List) = %d, want 1", len(maps))
	}
	if maps[0].ID != m2.ID {
		t.Errorf("remaining map ID = %q, want %q", maps[0].ID, m2.ID)
	}
}

func TestMapStore_CreateNormalizesExt(t *testing.T) {
	dir := t.TempDir()
	ms := NewMapStore(dir)

	m, err := ms.Create("Test", "PNG", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if m.Ext != ".png" {
		t.Errorf("ext = %q, want %q", m.Ext, ".png")
	}
}

func TestMapStore_ActiveMap(t *testing.T) {
	dir := t.TempDir()
	ms := NewMapStore(dir)

	if id := ms.ActiveID(); id != "" {
		t.Errorf("initial ActiveID = %q, want empty", id)
	}

	m, _ := ms.Create("Test", ".png", strings.NewReader("data"))
	ms.SetActive(m.ID)

	if id := ms.ActiveID(); id != m.ID {
		t.Errorf("ActiveID = %q, want %q", id, m.ID)
	}

	// Delete active map should clear active
	ms.Delete(m.ID)
	if id := ms.ActiveID(); id != "" {
		t.Errorf("after delete ActiveID = %q, want empty", id)
	}
}

func TestMapStore_Tokens(t *testing.T) {
	dir := t.TempDir()
	ms := NewMapStore(dir)

	m, _ := ms.Create("Test", ".png", strings.NewReader("data"))

	// Add token
	tok, err := ms.AddToken(m.ID, Token{
		X: 100, Y: 200, Radius: 20,
		Label: "Goblin", Color: "#ff0000",
		Visible: true, Moveable: true,
	})
	if err != nil {
		t.Fatalf("AddToken error: %v", err)
	}
	if tok.ID == "" {
		t.Error("token ID is empty")
	}
	if tok.Label != "Goblin" {
		t.Errorf("label = %q, want %q", tok.Label, "Goblin")
	}

	// Get all tokens
	tokens, err := ms.GetTokens(m.ID)
	if err != nil {
		t.Fatalf("GetTokens error: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("len(tokens) = %d, want 1", len(tokens))
	}

	// Add invisible token
	tok2, _ := ms.AddToken(m.ID, Token{
		X: 300, Y: 400, Radius: 15,
		Label: "Hidden", Color: "#0000ff",
		Visible: false,
	})

	// GetVisibleTokens should only return visible ones
	visible, err := ms.GetVisibleTokens(m.ID)
	if err != nil {
		t.Fatalf("GetVisibleTokens error: %v", err)
	}
	if len(visible) != 1 {
		t.Fatalf("len(visible) = %d, want 1", len(visible))
	}
	if visible[0].Label != "Goblin" {
		t.Errorf("visible label = %q, want %q", visible[0].Label, "Goblin")
	}

	// Update token
	tok.Label = "Orc"
	if err := ms.UpdateToken(m.ID, *tok); err != nil {
		t.Fatalf("UpdateToken error: %v", err)
	}
	tokens, _ = ms.GetTokens(m.ID)
	found := false
	for _, tk := range tokens {
		if tk.ID == tok.ID && tk.Label == "Orc" {
			found = true
		}
	}
	if !found {
		t.Error("updated token not found")
	}

	// Delete token
	if err := ms.DeleteToken(m.ID, tok2.ID); err != nil {
		t.Fatalf("DeleteToken error: %v", err)
	}
	tokens, _ = ms.GetTokens(m.ID)
	if len(tokens) != 1 {
		t.Errorf("after delete len(tokens) = %d, want 1", len(tokens))
	}
}

func TestMapStore_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	ms := NewMapStore(dir)

	_, err := ms.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent map")
	}
}

func TestMapStore_ImagePath(t *testing.T) {
	dir := t.TempDir()
	ms := NewMapStore(dir)

	m, _ := ms.Create("Test", ".png", strings.NewReader("data"))
	path, err := ms.ImagePath(m.ID)
	if err != nil {
		t.Fatalf("ImagePath error: %v", err)
	}
	if !strings.HasSuffix(path, "map.png") {
		t.Errorf("ImagePath = %q, want suffix %q", path, "map.png")
	}
}
