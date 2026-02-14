package handler

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/internal/dice"
	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
)

type ChatHandler struct {
	chat   *store.ChatStore
	broker *realtime.Broker
	auth   *auth.Auth
}

func NewChatHandler(chat *store.ChatStore, broker *realtime.Broker, a *auth.Auth) *ChatHandler {
	return &ChatHandler{chat: chat, broker: broker, auth: a}
}

func (h *ChatHandler) Messages(w http.ResponseWriter, r *http.Request) {
	msgs := h.chat.Recent(50)
	templates.ChatMessages(msgs).Render(r.Context(), w)
}

func (h *ChatHandler) Send(w http.ResponseWriter, r *http.Request) {
	content := r.FormValue("message")
	if content == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	isDM := strings.Contains(r.Header.Get("Referer"), "/dm")
	author := h.auth.GetChatName(w, r, isDM)

	// Process dice expressions
	content = dice.Process(content)

	msg := h.chat.Add(author, content)

	// Render the message as HTML and broadcast via SSE
	var buf bytes.Buffer
	templates.ChatMessageItem(msg).Render(r.Context(), &buf)
	h.broker.Publish(realtime.Event{
		Name: "chatMessage",
		Data: buf.String(),
	})

	// Return empty response (HTMX will clear the form)
	w.WriteHeader(http.StatusNoContent)
}

func (h *ChatHandler) SetName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("chatName"))
	if name == "" {
		http.Error(w, "Name cannot be empty", http.StatusBadRequest)
		return
	}
	if len(name) > 20 {
		name = name[:20]
	}
	h.auth.SetChatName(w, r, name)
	templates.ChatNameDisplay(name).Render(r.Context(), w)
}

func (h *ChatHandler) EditNameForm(w http.ResponseWriter, r *http.Request) {
	isDM := strings.Contains(r.Header.Get("Referer"), "/dm")
	name := h.auth.GetChatName(w, r, isDM)
	templates.ChatNameEdit(name).Render(r.Context(), w)
}
