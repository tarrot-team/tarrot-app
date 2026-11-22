package reading

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/spread"
)

func TestMemoryStoreCRUD(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	t1 := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 22, 11, 0, 0, 0, time.UTC)

	r1 := Reading{
		ID:             "r1",
		UserID:         "u1",
		Mode:           "auto",
		Question:       "Вопрос 1",
		Spread:         spread.Spread{Type: "daily", Title: "Карта дня"},
		Interpretation: "Трактовка 1",
		Source:         "ai",
		CreatedAt:      t1,
	}
	r2 := Reading{
		ID:             "r2",
		UserID:         "u1",
		Mode:           "manual",
		Question:       "Вопрос 2",
		Spread:         spread.Spread{Type: "custom", Title: "Свободный расклад"},
		Interpretation: "Трактовка 2",
		Source:         "fallback",
		CreatedAt:      t2,
	}
	rOther := Reading{
		ID:        "r3",
		UserID:    "u2",
		Mode:      "auto",
		CreatedAt: t2,
	}

	for _, r := range []Reading{r1, r2, rOther} {
		if err := store.Save(ctx, r); err != nil {
			t.Fatalf("Save(%s) error: %v", r.ID, err)
		}
	}

	got, err := store.GetByID(ctx, "r1")
	if err != nil || got.Question != "Вопрос 1" {
		t.Fatalf("GetByID(r1) = %+v, err=%v", got, err)
	}

	list, err := store.ListByUser(ctx, "u1", 10)
	if err != nil {
		t.Fatalf("ListByUser error: %v", err)
	}
	if len(list) != 2 || list[0].ID != "r2" || list[1].ID != "r1" {
		t.Fatalf("unexpected list order: %+v", list)
	}

	limited, err := store.ListByUser(ctx, "u1", 1)
	if err != nil || len(limited) != 1 || limited[0].ID != "r2" {
		t.Fatalf("unexpected limited list: %+v, err=%v", limited, err)
	}

	if err := store.Delete(ctx, "r1", "u2"); !errors.Is(err, ErrForbidden) {
		t.Errorf("Delete by wrong user err = %v; want ErrForbidden", err)
	}
	if err := store.Delete(ctx, "r1", "u1"); err != nil {
		t.Fatalf("Delete by owner error: %v", err)
	}
	if _, err := store.GetByID(ctx, "r1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID after delete err = %v; want ErrNotFound", err)
	}
}
