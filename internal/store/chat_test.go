package store

import (
	"testing"
)

func TestChatStore_Add(t *testing.T) {
	cs := NewChatStore(100)

	msg := cs.Add("DM", "Hello players!")
	if msg.ID != 1 {
		t.Errorf("first message ID = %d, want 1", msg.ID)
	}
	if msg.Author != "DM" {
		t.Errorf("author = %q, want %q", msg.Author, "DM")
	}
	if msg.Content != "Hello players!" {
		t.Errorf("content = %q, want %q", msg.Content, "Hello players!")
	}

	msg2 := cs.Add("Player", "Hi!")
	if msg2.ID != 2 {
		t.Errorf("second message ID = %d, want 2", msg2.ID)
	}
}

func TestChatStore_Recent(t *testing.T) {
	cs := NewChatStore(100)
	cs.Add("DM", "msg1")
	cs.Add("DM", "msg2")
	cs.Add("DM", "msg3")

	// Get last 2
	msgs := cs.Recent(2)
	if len(msgs) != 2 {
		t.Fatalf("len(Recent(2)) = %d, want 2", len(msgs))
	}
	if msgs[0].Content != "msg2" {
		t.Errorf("msgs[0] = %q, want %q", msgs[0].Content, "msg2")
	}
	if msgs[1].Content != "msg3" {
		t.Errorf("msgs[1] = %q, want %q", msgs[1].Content, "msg3")
	}

	// N > len returns all
	msgs = cs.Recent(100)
	if len(msgs) != 3 {
		t.Errorf("len(Recent(100)) = %d, want 3", len(msgs))
	}

	// N <= 0 returns all
	msgs = cs.Recent(0)
	if len(msgs) != 3 {
		t.Errorf("len(Recent(0)) = %d, want 3", len(msgs))
	}
}

func TestChatStore_MaxSize(t *testing.T) {
	cs := NewChatStore(3)

	cs.Add("DM", "msg1")
	cs.Add("DM", "msg2")
	cs.Add("DM", "msg3")
	cs.Add("DM", "msg4") // should drop msg1

	msgs := cs.Recent(0)
	if len(msgs) != 3 {
		t.Fatalf("len = %d, want 3", len(msgs))
	}
	if msgs[0].Content != "msg2" {
		t.Errorf("oldest = %q, want %q", msgs[0].Content, "msg2")
	}
	if msgs[2].Content != "msg4" {
		t.Errorf("newest = %q, want %q", msgs[2].Content, "msg4")
	}
}

func TestChatStore_DefaultMaxSize(t *testing.T) {
	cs := NewChatStore(0)
	// Should default to 200, not panic
	cs.Add("DM", "test")
	msgs := cs.Recent(0)
	if len(msgs) != 1 {
		t.Errorf("len = %d, want 1", len(msgs))
	}
}
