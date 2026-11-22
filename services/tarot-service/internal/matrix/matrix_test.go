package matrix

import (
	"testing"
	"time"
)

func TestReduce(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{input: 1, want: 1},
		{input: 15, want: 15},
		{input: 22, want: 22},   // 22 не сводится к 4
		{input: 23, want: 5},    // 2 + 3
		{input: 29, want: 11},   // 2 + 9
		{input: 1999, want: 10}, // 1+9+9+9 = 28 -> 2+8 = 10
		{input: 2000, want: 2},  // 2+0+0+0 = 2
	}
	for _, tc := range tests {
		got := Reduce(tc.input)
		if got != tc.want {
			t.Errorf("Reduce(%d) = %d; want %d", tc.input, got, tc.want)
		}
	}
}

func TestCalculate(t *testing.T) {
	date := time.Date(1999, 12, 29, 0, 0, 0, 0, time.UTC)
	got, err := Calculate(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := Matrix{
		A: 11, B: 12, C: 10, D: 6, E: 12,
		F: 5, G: 22, H: 16, I: 17,
		J: 5, K: 6, L: 22, M: 18,
	}
	if got != want {
		t.Errorf("Calculate(%s) =\n%+v\nwant:\n%+v", date.Format("2006-01-02"), got, want)
	}
}
func TestCalculateEmptyDate(t *testing.T) {
	_, err := Calculate(time.Time{})
	if err == nil {
		t.Fatal("expected error for zero time, got nil")
	}
}
