package matrix

import (
	"errors"
	"time"
)

type Matrix struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"c"`
	D int `json:"d"`
	E int `json:"e"`

	F int `json:"f"`
	G int `json:"g"`
	H int `json:"h"`
	I int `json:"i"`

	J int `json:"j"`
	K int `json:"k"`
	L int `json:"l"`
	M int `json:"m"`
}

func Reduce(n int) int {
	for n > 22 {
		sum := 0
		for n > 0 {
			sum += n % 10
			n /= 10
		}
		n = sum
	}
	return n
}

func Calculate(birthDate time.Time) (Matrix, error) {
	if birthDate.IsZero() {
		return Matrix{}, errors.New("birth date cannot be empty")
	}

	day := birthDate.Day()
	month := int(birthDate.Month())
	year := birthDate.Year()

	a := Reduce(day)
	b := Reduce(month)
	c := Reduce(year)
	d := Reduce(a + b + c)
	e := Reduce(a + b + c + d)

	f := Reduce(a + b)
	g := Reduce(b + c)
	h := Reduce(c + d)
	i := Reduce(d + a)

	j := Reduce(a + e)
	k := Reduce(b + e)
	l := Reduce(c + e)
	m := Reduce(d + e)

	return Matrix{
		A: a, B: b, C: c, D: d, E: e,
		F: f, G: g, H: h, I: i,
		J: j, K: k, L: l, M: m,
	}, nil
}
