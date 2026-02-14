package handler

import (
	"net/http"
	"path/filepath"

	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
)

type MediaHandler struct {
	media *store.MediaStore
}

func NewMediaHandler(media *store.MediaStore) *MediaHandler {
	return &MediaHandler{media: media}
}

func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = h.media.Upload(header.Filename, contentType, file)
	if err != nil {
		http.Error(w, "Upload failed", http.StatusInternalServerError)
		return
	}

	// Return updated media list
	files, _ := h.media.List()
	templates.MediaList(files).Render(r.Context(), w)
}

func (h *MediaHandler) List(w http.ResponseWriter, r *http.Request) {
	files, _ := h.media.List()
	templates.MediaList(files).Render(r.Context(), w)
}

func (h *MediaHandler) Serve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	f, err := h.media.Get(id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	path, err := h.media.FilePath(id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", f.ContentType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+filepath.Base(f.Filename)+"\"")
	http.ServeFile(w, r, path)
}

func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.media.Delete(id)

	files, _ := h.media.List()
	templates.MediaList(files).Render(r.Context(), w)
}
