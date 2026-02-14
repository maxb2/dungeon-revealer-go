package realtime

import (
	"testing"
	"time"
)

func TestBroker_PubSub(t *testing.T) {
	b := NewBroker()
	c := b.Subscribe(false)
	defer b.Unsubscribe(c)

	b.Publish(Event{Name: "test", Data: "hello"})

	select {
	case evt := <-c.ch:
		if evt.Name != "test" {
			t.Errorf("event name = %q, want %q", evt.Name, "test")
		}
		if evt.Data != "hello" {
			t.Errorf("event data = %q, want %q", evt.Data, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestBroker_DMOnly_FilteredForPlayer(t *testing.T) {
	b := NewBroker()
	player := b.Subscribe(false)
	dm := b.Subscribe(true)
	defer b.Unsubscribe(player)
	defer b.Unsubscribe(dm)

	b.Publish(Event{Name: "secret", Data: "dm-only", DMOnly: true})

	// DM should receive it
	select {
	case evt := <-dm.ch:
		if evt.Name != "secret" {
			t.Errorf("DM event name = %q, want %q", evt.Name, "secret")
		}
	case <-time.After(time.Second):
		t.Fatal("DM did not receive DM-only event")
	}

	// Player should NOT receive it
	select {
	case evt := <-player.ch:
		t.Errorf("player received DM-only event: %+v", evt)
	case <-time.After(50 * time.Millisecond):
		// Expected - no event for player
	}
}

func TestBroker_DMOnly_BroadcastReachesAll(t *testing.T) {
	b := NewBroker()
	player := b.Subscribe(false)
	dm := b.Subscribe(true)
	defer b.Unsubscribe(player)
	defer b.Unsubscribe(dm)

	b.Publish(Event{Name: "chat", Data: "hi", DMOnly: false})

	// Both should receive it
	for _, name := range []string{"player", "dm"} {
		var ch <-chan Event
		if name == "player" {
			ch = player.ch
		} else {
			ch = dm.ch
		}
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("%s did not receive broadcast event", name)
		}
	}
}

func TestBroker_Unsubscribe(t *testing.T) {
	b := NewBroker()
	c := b.Subscribe(false)

	b.Unsubscribe(c)

	// Channel should be closed
	_, ok := <-c.ch
	if ok {
		t.Error("channel not closed after unsubscribe")
	}

	// Publishing after unsubscribe should not panic
	b.Publish(Event{Name: "test", Data: "after-unsub"})
}

func TestBroker_MultipleClients(t *testing.T) {
	b := NewBroker()
	c1 := b.Subscribe(false)
	c2 := b.Subscribe(false)
	defer b.Unsubscribe(c1)
	defer b.Unsubscribe(c2)

	b.Publish(Event{Name: "broadcast", Data: "all"})

	for i, c := range []*client{c1, c2} {
		select {
		case evt := <-c.ch:
			if evt.Data != "all" {
				t.Errorf("client %d data = %q, want %q", i, evt.Data, "all")
			}
		case <-time.After(time.Second):
			t.Fatalf("client %d did not receive event", i)
		}
	}
}
