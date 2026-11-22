package arcana

import (
	"errors"
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		number   int
		wantName string
	}{
		{number: 1, wantName: "Маг (The Magician)"},
		{number: 8, wantName: "Справедливость (Justice)"},
		{number: 11, wantName: "Сила (Strength)"},
		{number: 22, wantName: "Шут (The Fool)"},
	}

	for _, tc := range tests {
		got, err := Get(tc.number)
		if err != nil {
			t.Fatalf("Get(%d) returned unexpected error: %v", tc.number, err)
		}
		if got.Number != tc.number || got.Name != tc.wantName {
			t.Errorf("Get(%d) = {%d, %q}; want {%d, %q}",
				tc.number, got.Number, got.Name, tc.number, tc.wantName)
		}
	}
}

func TestGetInvalid(t *testing.T) {
	for _, num := range []int{-1, 0, 23, 100} {
		_, err := Get(num)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%d) error = %v; want ErrNotFound", num, err)
		}
	}
}

func TestAll(t *testing.T) {
	all := All()
	if len(all) != 22 {
		t.Fatalf("len(All()) = %d; want 22", len(all))
	}
	for i, a := range all {
		if a.Number != i+1 {
			t.Errorf("All()[%d].Number = %d; want %d", i, a.Number, i+1)
		}
	}
}
