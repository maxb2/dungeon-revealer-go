package store

import (
	"sync"
	"time"
)

type ChatMessage struct {
	ID        int       `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type ChatStore struct {
	mu       sync.RWMutex
	messages []ChatMessage
	nextID   int
	maxSize  int
}

func NewChatStore(maxSize int) *ChatStore {
	if maxSize <= 0 {
		maxSize = 200
	}
	return &ChatStore{
		messages: make([]ChatMessage, 0, maxSize),
		nextID:   1,
		maxSize:  maxSize,
	}
}

func (s *ChatStore) Add(author, content string) ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := ChatMessage{
		ID:        s.nextID,
		Author:    author,
		Content:   content,
		Timestamp: time.Now(),
	}
	s.nextID++

	s.messages = append(s.messages, msg)
	if len(s.messages) > s.maxSize {
		s.messages = s.messages[len(s.messages)-s.maxSize:]
	}

	return msg
}

func (s *ChatStore) Recent(n int) []ChatMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || n > len(s.messages) {
		n = len(s.messages)
	}
	start := len(s.messages) - n
	result := make([]ChatMessage, n)
	copy(result, s.messages[start:])
	return result
}
