package handler

import (
	"bytes"
	"net/http"

	"github.com/matt/dungeon-revealer-go/internal/auth"
	"github.com/matt/dungeon-revealer-go/internal/dice"
	"github.com/matt/dungeon-revealer-go/internal/realtime"
	"github.com/matt/dungeon-revealer-go/internal/store"
	"github.com/matt/dungeon-revealer-go/templates"
)

type ChatHandler struct {
	chat   *store.ChatStore
	broker *realtime.Broker
}

func NewChatHandler(chat *store.ChatStore, broker *realtime.Broker) *ChatHandler {
	return &ChatHandler{chat: chat, broker: broker}
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

	role := auth.RoleFromContext(r.Context())
	author := "Player"
	if role == auth.RoleAdmin {
		author = "DM"
	}

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
