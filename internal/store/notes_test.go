package store

import (
	"testing"
)

func TestNotes_CRUD(t *testing.T) {
	db := newTestDB(t)

	// Create
	note, err := db.CreateNote("Test Note", "Some content", false, false)
	if err != nil {
		t.Fatalf("CreateNote error: %v", err)
	}
	if note.ID == "" {
		t.Error("note ID is empty")
	}
	if note.Title != "Test Note" {
		t.Errorf("title = %q, want %q", note.Title, "Test Note")
	}

	// Get
	got, err := db.GetNote(note.ID)
	if err != nil {
		t.Fatalf("GetNote error: %v", err)
	}
	if got.Title != "Test Note" {
		t.Errorf("GetNote title = %q, want %q", got.Title, "Test Note")
	}
	if got.Content != "Some content" {
		t.Errorf("GetNote content = %q, want %q", got.Content, "Some content")
	}

	// Update
	err = db.UpdateNote(note.ID, "Updated Title", "Updated content", true, true)
	if err != nil {
		t.Fatalf("UpdateNote error: %v", err)
	}
	got, _ = db.GetNote(note.ID)
	if got.Title != "Updated Title" {
		t.Errorf("updated title = %q, want %q", got.Title, "Updated Title")
	}
	if !got.IsEntryPoint {
		t.Error("expected IsEntryPoint = true")
	}
	if !got.IsPublic {
		t.Error("expected IsPublic = true")
	}

	// List
	notes, err := db.ListNotes(false)
	if err != nil {
		t.Fatalf("ListNotes error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("len(ListNotes) = %d, want 1", len(notes))
	}

	// Delete
	err = db.DeleteNote(note.ID)
	if err != nil {
		t.Fatalf("DeleteNote error: %v", err)
	}
	_, err = db.GetNote(note.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestNotes_PublicOnly(t *testing.T) {
	db := newTestDB(t)

	// Create a public entry point and a private note
	db.CreateNote("Public", "pub content", true, true)
	db.CreateNote("Private", "priv content", false, false)

	// ListNotes(publicOnly=true) should only return public entry points
	notes, err := db.ListNotes(true)
	if err != nil {
		t.Fatalf("ListNotes error: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("len(public notes) = %d, want 1", len(notes))
	}
	if notes[0].Title != "Public" {
		t.Errorf("public note title = %q, want %q", notes[0].Title, "Public")
	}

	// ListNotes(publicOnly=false) should return all
	notes, _ = db.ListNotes(false)
	if len(notes) != 2 {
		t.Errorf("len(all notes) = %d, want 2", len(notes))
	}
}

func TestNotes_Search(t *testing.T) {
	db := newTestDB(t)

	db.CreateNote("Dragon Lore", "The ancient dragon sleeps in the mountain", false, false)
	db.CreateNote("Tavern Info", "The tavern serves ale and mead", false, false)
	db.CreateNote("Dragon Hoard", "Gold and jewels guarded by a dragon", true, true)

	// Search for "dragon"
	results, err := db.SearchNotes("dragon", false)
	if err != nil {
		t.Fatalf("SearchNotes error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(search dragon) = %d, want 2", len(results))
	}

	// Search for "tavern"
	results, _ = db.SearchNotes("tavern", false)
	if len(results) != 1 {
		t.Fatalf("len(search tavern) = %d, want 1", len(results))
	}

	// Search with publicOnly
	results, _ = db.SearchNotes("dragon", true)
	if len(results) != 1 {
		t.Fatalf("len(public search dragon) = %d, want 1", len(results))
	}
	if results[0].Title != "Dragon Hoard" {
		t.Errorf("public result = %q, want %q", results[0].Title, "Dragon Hoard")
	}
}

func TestNotes_GetNotFound(t *testing.T) {
	db := newTestDB(t)

	_, err := db.GetNote("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent note")
	}
}
