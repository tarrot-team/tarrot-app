package matrix

import (
	"errors"
	"fmt"
	"time"
)

type InnerNodes struct {
	A1 int `json:"a1"`
	A2 int `json:"a2"`
	B1 int `json:"b1"`
	B2 int `json:"b2"`
	C1 int `json:"c1"`
	D1 int `json:"d1"`
	F1 int `json:"f1"`
	F2 int `json:"f2"`
	G1 int `json:"g1"`
	G2 int `json:"g2"`
	H1 int `json:"h1"`
	H2 int `json:"h2"`
	I1 int `json:"i1"`
	I2 int `json:"i2"`
}

type LoveMoneyLine struct {
	K      int `json:"k"`
	L      int `json:"l"`
	Center int `json:"center"`
	Love   int `json:"love"`
	Money  int `json:"money"`
}

type Purpose struct {
	Sky       int `json:"sky"`
	Earth     int `json:"earth"`
	Personal  int `json:"personal"`
	Male      int `json:"male"`
	Female    int `json:"female"`
	Social    int `json:"social"`
	Spiritual int `json:"spiritual"`
}

type YearEnergy struct {
	Age    string `json:"age"`
	Arcana int    `json:"arcana"`
}

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

	Inner      InnerNodes    `json:"inner"`
	KarmicTail [3]int        `json:"karmic_tail"`
	LoveMoney  LoveMoneyLine `json:"love_money"`
	Purpose    Purpose       `json:"purpose"`
	Years      []YearEnergy  `json:"years"`
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

	f2 := Reduce(f + e)
	g2 := Reduce(g + e)
	h2 := Reduce(h + e)
	i2 := Reduce(i + e)

	inner := InnerNodes{
		A1: Reduce(a + j),
		A2: Reduce(j + e),
		B1: Reduce(b + k),
		B2: Reduce(k + e),
		C1: Reduce(c + l),
		D1: Reduce(d + m),
		F1: Reduce(f + f2),
		F2: f2,
		G1: Reduce(g + g2),
		G2: g2,
		H1: Reduce(h + h2),
		H2: h2,
		I1: Reduce(i + i2),
		I2: i2,
	}

	lmCenter := Reduce(m + l)
	loveMoney := LoveMoneyLine{
		K:      m,
		L:      l,
		Center: lmCenter,
		Love:   Reduce(m + lmCenter),
		Money:  Reduce(lmCenter + l),
	}

	sky := Reduce(b + d)
	earth := Reduce(a + c)
	personal := Reduce(sky + earth)
	male := Reduce(f + h)
	female := Reduce(g + i)
	social := Reduce(male + female)
	spiritual := Reduce(personal + social)

	purpose := Purpose{
		Sky:       sky,
		Earth:     earth,
		Personal:  personal,
		Male:      male,
		Female:    female,
		Social:    social,
		Spiritual: spiritual,
	}

	years := calculateYears([]int{a, f, b, g, c, h, d, i})

	return Matrix{
		A: a, B: b, C: c, D: d, E: e,
		F: f, G: g, H: h, I: i,
		J: j, K: k, L: l, M: m,
		Inner:      inner,
		KarmicTail: [3]int{d, inner.D1, m},
		LoveMoney:  loveMoney,
		Purpose:    purpose,
		Years:      years,
	}, nil
}

func calculateYears(vertices []int) []YearEnergy {
	res := make([]YearEnergy, 0, 64)
	for seg := 0; seg < len(vertices); seg++ {
		startAge := seg * 10
		v0 := vertices[seg]
		v8 := vertices[(seg+1)%len(vertices)]

		v4 := Reduce(v0 + v8)
		v2 := Reduce(v0 + v4)
		v6 := Reduce(v4 + v8)
		v1 := Reduce(v0 + v2)
		v3 := Reduce(v2 + v4)
		v5 := Reduce(v4 + v6)
		v7 := Reduce(v6 + v8)

		res = append(res,
			YearEnergy{Age: fmt.Sprintf("%d", startAge), Arcana: v0},
			YearEnergy{Age: fmt.Sprintf("%d-%d", startAge+1, startAge+2), Arcana: v1},
			YearEnergy{Age: fmt.Sprintf("%d-%d", startAge+2, startAge+3), Arcana: v2},
			YearEnergy{Age: fmt.Sprintf("%d-%d", startAge+3, startAge+4), Arcana: v3},
			YearEnergy{Age: fmt.Sprintf("%d", startAge+5), Arcana: v4},
			YearEnergy{Age: fmt.Sprintf("%d-%d", startAge+6, startAge+7), Arcana: v5},
			YearEnergy{Age: fmt.Sprintf("%d-%d", startAge+7, startAge+8), Arcana: v6},
			YearEnergy{Age: fmt.Sprintf("%d-%d", startAge+8, startAge+9), Arcana: v7},
		)
	}
	return res
}

func (m Matrix) AllArcanaNumbers() []int {
	seen := make(map[int]bool)
	var nums []int
	add := func(n int) {
		if n >= 1 && n <= 22 && !seen[n] {
			seen[n] = true
			nums = append(nums, n)
		}
	}

	base := []int{
		m.A, m.B, m.C, m.D, m.E,
		m.F, m.G, m.H, m.I,
		m.J, m.K, m.L, m.M,
		m.Inner.A1, m.Inner.A2, m.Inner.B1, m.Inner.B2,
		m.Inner.C1, m.Inner.D1,
		m.Inner.F1, m.Inner.F2, m.Inner.G1, m.Inner.G2,
		m.Inner.H1, m.Inner.H2, m.Inner.I1, m.Inner.I2,
		m.LoveMoney.Center, m.LoveMoney.Love, m.LoveMoney.Money,
		m.Purpose.Sky, m.Purpose.Earth, m.Purpose.Personal,
		m.Purpose.Male, m.Purpose.Female, m.Purpose.Social, m.Purpose.Spiritual,
	}
	for _, n := range base {
		add(n)
	}
	for _, y := range m.Years {
		add(y.Arcana)
	}
	return nums
}
