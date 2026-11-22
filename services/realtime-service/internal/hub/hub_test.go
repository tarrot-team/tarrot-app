package hub

import (
	"testing"
	"time"
)

func TestRoomBroadcastAndHistory(t *testing.T) {
	h := New()

	ch1, cancel1 := h.SubscribeRoom("general")
	defer cancel1()
	ch2, cancel2 := h.SubscribeRoom("general")

	if got := h.RoomSubscribers("general"); got != 2 {
		t.Fatalf("subscribers = %d; want 2", got)
	}

	ev := h.PublishMessage("general", "user-1", "Привет всем!")

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case received := <-ch:
			if received.ID != ev.ID || received.Text != "Привет всем!" {
				t.Errorf("sub %d got %+v; want %+v", i, received, ev)
			}
		case <-time.After(time.Second):
			t.Fatalf("sub %d timed out waiting for event", i)
		}
	}

	cancel2()
	if got := h.RoomSubscribers("general"); got != 1 {
		t.Fatalf("subscribers after cancel = %d; want 1", got)
	}

	h.PublishMessage("general", "user-2", "Второе сообщение")
	recent := h.RecentMessages("general", 1)
	if len(recent) != 1 || recent[0].Text != "Второе сообщение" {
		t.Fatalf("unexpected recent messages: %+v", recent)
	}
}

func TestUserNotification(t *testing.T) {
	h := New()

	ch, cancel := h.SubscribeUser("user-42")
	defer cancel()

	ev := h.NotifyUser("user-42", "user-7", "Новый комментарий к твоему посту")

	select {
	case received := <-ch:
		if received.ID != ev.ID || received.Type != "notification" || received.ToUserID != "user-42" {
			t.Errorf("got %+v; want %+v", received, ev)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for notification")
	}
}
