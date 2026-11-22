package store

import (
	"context"
	"testing"
	"time"

	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
)

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryStore()

	base := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	events := []hub.Event{
		{ID: "1", Type: "message", Room: "general", FromUserID: "u1", Text: "Первое", CreatedAt: base},
		{ID: "2", Type: "message", Room: "general", FromUserID: "u2", Text: "Второе", CreatedAt: base.Add(time.Minute)},
		{ID: "3", Type: "message", Room: "general", FromUserID: "u1", Text: "Третье", CreatedAt: base.Add(2 * time.Minute)},
		{ID: "4", Type: "message", Room: "other", FromUserID: "u3", Text: "Другая комната", CreatedAt: base},
	}
	for _, ev := range events {
		if err := st.SaveMessage(ctx, ev); err != nil {
			t.Fatalf("SaveMessage(%s) error: %v", ev.ID, err)
		}
	}

	got, err := st.ListMessages(ctx, "general", 2)
	if err != nil {
		t.Fatalf("ListMessages error: %v", err)
	}
	if len(got) != 2 || got[0].ID != "2" || got[1].ID != "3" {
		t.Fatalf("unexpected messages: %+v", got)
	}

	empty, err := st.ListMessages(ctx, "missing", 10)
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected empty slice, got %+v, err=%v", empty, err)
	}
}
