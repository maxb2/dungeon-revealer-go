package handler

import (
	"bytes"
	"net/http"
	"strings"
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

	// Determine author from the page they sent from, since with
	// no passwords configured everyone has admin role.
	author := "Player"
	if strings.Contains(r.Header.Get("Referer"), "/dm") {
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
