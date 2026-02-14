package realtime

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Event struct {
	Name    string // SSE event name (e.g., "fogUpdate", "chatMessage")
	Data    string // HTML fragment or JSON
	DMOnly  bool   // If true, only sent to DM clients
}

type client struct {
	ch   chan Event
	isDM bool
}

type Broker struct {
	mu         sync.RWMutex
	clients    map[*client]struct{}
}

func NewBroker() *Broker {
	return &Broker{
		clients: make(map[*client]struct{}),
	}
}

func (b *Broker) Subscribe(isDM bool) *client {
	c := &client{
		ch:   make(chan Event, 64),
		isDM: isDM,
	}
	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()
	return c
}

func (b *Broker) Unsubscribe(c *client) {
	b.mu.Lock()
	delete(b.clients, c)
	b.mu.Unlock()
	close(c.ch)
}

func (b *Broker) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for c := range b.clients {
		if e.DMOnly && !c.isDM {
			continue
		}
		select {
		case c.ch <- e:
		default:
			// Drop event if client buffer is full
		}
	}
}

func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("SSE: connection attempt from %s, role=%s", r.RemoteAddr, r.URL.Query().Get("role"))

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("SSE: streaming not supported for %s", r.RemoteAddr)
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	isDM := r.URL.Query().Get("role") == "dm"
	c := b.Subscribe(isDM)
	log.Printf("SSE: client connected (isDM=%v)", isDM)
	defer b.Unsubscribe(c)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Send initial comment so the client knows the stream is alive
	fmt.Fprint(w, ":ok\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-c.ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Name, evt.Data)
			flusher.Flush()
		}
	}
}
