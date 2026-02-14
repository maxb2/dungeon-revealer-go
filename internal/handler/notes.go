package handler

import (
	"bytes"
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
	"github.com/yuin/goldmark"
)

type NotesHandler struct {
	db *store.DB
}

func NewNotesHandler(db *store.DB) *NotesHandler {
	return &NotesHandler{db: db}
}

func (h *NotesHandler) List(w http.ResponseWriter, r *http.Request) {
	role := auth.RoleFromContext(r.Context())
	publicOnly := role != auth.RoleAdmin

	notes, _ := h.db.ListNotes(publicOnly)
	templates.NoteList(notes, role == auth.RoleAdmin).Render(r.Context(), w)
}

func (h *NotesHandler) View(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	role := auth.RoleFromContext(r.Context())

	note, err := h.db.GetNote(id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	if role != auth.RoleAdmin && !note.IsPublic {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Render markdown
	var buf bytes.Buffer
	goldmark.Convert([]byte(note.Content), &buf)
	html := buf.String()

	templates.NoteView(note, html, role == auth.RoleAdmin).Render(r.Context(), w)
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

	// Return the note view
	note, _ := h.db.GetNote(id)
	var buf bytes.Buffer
	goldmark.Convert([]byte(note.Content), &buf)
	templates.NoteView(note, buf.String(), true).Render(r.Context(), w)
}

func (h *NotesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.db.DeleteNote(id)

	notes, _ := h.db.ListNotes(false)
	templates.NoteList(notes, true).Render(r.Context(), w)
}

func (h *NotesHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	role := auth.RoleFromContext(r.Context())
	publicOnly := role != auth.RoleAdmin

	if query == "" {
		notes, _ := h.db.ListNotes(publicOnly)
		templates.NoteList(notes, role == auth.RoleAdmin).Render(r.Context(), w)
		return
	}

	notes, _ := h.db.SearchNotes(query, publicOnly)
	templates.NoteList(notes, role == auth.RoleAdmin).Render(r.Context(), w)
}
