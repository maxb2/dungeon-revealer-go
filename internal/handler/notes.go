package handler

import (
	"bytes"
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
	"github.com/yuin/goldmark"
)

type NotesHandler struct {
	db     *store.DB
	broker *realtime.Broker
}

func NewNotesHandler(db *store.DB, broker *realtime.Broker) *NotesHandler {
	return &NotesHandler{db: db, broker: broker}
}

func renderMarkdown(content string) string {
	var buf bytes.Buffer
	goldmark.Convert([]byte(content), &buf)
	return buf.String()
}

// --- Player handlers (always enforce public-only, never expose private notes) ---

func (h *NotesHandler) PlayerList(w http.ResponseWriter, r *http.Request) {
	notes, _ := h.db.ListNotes(true)
	templates.NoteList(notes, false).Render(r.Context(), w)
}

func (h *NotesHandler) PlayerSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		notes, _ := h.db.ListNotes(true)
		templates.NoteList(notes, false).Render(r.Context(), w)
		return
	}
	notes, _ := h.db.SearchNotes(query, true)
	templates.NoteList(notes, false).Render(r.Context(), w)
}

func (h *NotesHandler) PlayerView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	note, err := h.db.GetNote(id)
	if err != nil || !note.IsPublic {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	templates.NoteView(note, renderMarkdown(note.Content), false).Render(r.Context(), w)
}

func (h *NotesHandler) PlayerEditForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	note, err := h.db.GetNote(id)
	if err != nil || !note.IsPublic {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if note.IsLocked {
		http.Error(w, "Note is locked", http.StatusForbidden)
		return
	}
	templates.NoteEditPlayer(note).Render(r.Context(), w)
}

func (h *NotesHandler) PlayerSave(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	note, err := h.db.GetNote(id)
	if err != nil || !note.IsPublic {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if note.IsLocked {
		http.Error(w, "Note is locked", http.StatusForbidden)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	if err := h.db.UpdateNote(id, title, content, note.IsEntryPoint, note.IsPublic); err != nil {
		http.Error(w, "Failed to update note", http.StatusInternalServerError)
		return
	}

	h.broker.Publish(realtime.Event{Name: "notesUpdated", DMOnly: false})

	updated, _ := h.db.GetNote(id)
	templates.NoteView(updated, renderMarkdown(updated.Content), false).Render(r.Context(), w)
}

// --- DM handlers ---

func (h *NotesHandler) List(w http.ResponseWriter, r *http.Request) {
	notes, _ := h.db.ListNotes(false)
	templates.NoteList(notes, true).Render(r.Context(), w)
}

func (h *NotesHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		notes, _ := h.db.ListNotes(false)
		templates.NoteList(notes, true).Render(r.Context(), w)
		return
	}
	notes, _ := h.db.SearchNotes(query, false)
	templates.NoteList(notes, true).Render(r.Context(), w)
}

func (h *NotesHandler) View(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	note, err := h.db.GetNote(id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	templates.NoteView(note, renderMarkdown(note.Content), true).Render(r.Context(), w)
}

func (h *NotesHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "new" {
		templates.NoteEdit(nil).Render(r.Context(), w)
		return
	}
	note, err := h.db.GetNote(id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}
	templates.NoteEdit(note).Render(r.Context(), w)
}

func (h *NotesHandler) Save(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	title := r.FormValue("title")
	content := r.FormValue("content")
	isEntryPoint := r.FormValue("isEntryPoint") == "on"
	isPublic := r.FormValue("isPublic") == "on"

	if id == "new" {
		note, err := h.db.CreateNote(title, content, isEntryPoint, isPublic)
		if err != nil {
			http.Error(w, "Failed to create note", http.StatusInternalServerError)
			return
		}
		id = note.ID
	} else {
		if err := h.db.UpdateNote(id, title, content, isEntryPoint, isPublic); err != nil {
			http.Error(w, "Failed to update note", http.StatusInternalServerError)
			return
		}
	}

	h.broker.Publish(realtime.Event{Name: "notesUpdated", DMOnly: false})

	note, _ := h.db.GetNote(id)
	templates.NoteView(note, renderMarkdown(note.Content), true).Render(r.Context(), w)
}

func (h *NotesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.db.DeleteNote(id)

	h.broker.Publish(realtime.Event{Name: "notesUpdated", DMOnly: false})

	notes, _ := h.db.ListNotes(false)
	templates.NoteList(notes, true).Render(r.Context(), w)
}

func (h *NotesHandler) LockToggle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	note, err := h.db.GetNote(id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	if err := h.db.LockNote(id, !note.IsLocked); err != nil {
		http.Error(w, "Failed to update note", http.StatusInternalServerError)
		return
	}

	h.broker.Publish(realtime.Event{Name: "notesUpdated", DMOnly: false})

	updated, _ := h.db.GetNote(id)
	templates.NoteView(updated, renderMarkdown(updated.Content), true).Render(r.Context(), w)
}
