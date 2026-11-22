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
		{input: 22, want: 22},
		{input: 23, want: 5},
		{input: 29, want: 11},
		{input: 1999, want: 10},
		{input: 2000, want: 2},
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

	if got.A != 11 || got.B != 12 || got.C != 10 || got.D != 6 || got.E != 12 ||
		got.F != 5 || got.G != 22 || got.H != 16 || got.I != 17 ||
		got.J != 5 || got.K != 6 || got.L != 22 || got.M != 18 {
		t.Errorf("unexpected base points: %+v", got)
	}
}

func TestCalculateFullChart(t *testing.T) {
	date := time.Date(1997, 1, 21, 0, 0, 0, 0, time.UTC)
	got, err := Calculate(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.A != 21 || got.B != 1 || got.C != 8 || got.D != 3 || got.E != 6 ||
		got.F != 22 || got.G != 9 || got.H != 11 || got.I != 6 ||
		got.J != 9 || got.K != 7 || got.L != 14 || got.M != 9 {
		t.Fatalf("unexpected base points: %+v", got)
	}

	wantInner := InnerNodes{
		A1: 3, A2: 15,
		B1: 8, B2: 13,
		C1: 22,
		D1: 12,
		F1: 5, F2: 10,
		G1: 6, G2: 15,
		H1: 10, H2: 17,
		I1: 18, I2: 12,
	}
	if got.Inner != wantInner {
		t.Errorf("inner = %+v; want %+v", got.Inner, wantInner)
	}

	if got.KarmicTail != [3]int{3, 12, 9} {
		t.Errorf("karmic_tail = %v; want [3 12 9]", got.KarmicTail)
	}

	wantLM := LoveMoneyLine{K: 9, L: 14, Center: 5, Love: 14, Money: 19}
	if got.LoveMoney != wantLM {
		t.Errorf("love_money = %+v; want %+v", got.LoveMoney, wantLM)
	}

	if len(got.Years) != 64 {
		t.Fatalf("len(years) = %d; want 64", len(got.Years))
	}

	wantFirstSegment := []int{21, 4, 10, 17, 7, 18, 11, 6}
	for idx, wantVal := range wantFirstSegment {
		if got.Years[idx].Arcana != wantVal {
			t.Errorf("years[%d] (%s) = %d; want %d", idx, got.Years[idx].Age, got.Years[idx].Arcana, wantVal)
		}
	}
}

func TestCalculateEmptyDate(t *testing.T) {
	_, err := Calculate(time.Time{})
	if err == nil {
		t.Fatal("expected error for zero time, got nil")
	}
}
