package deck

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/arcana"
)

type Card struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Suit      string `json:"suit,omitempty"`
	Number    int    `json:"number"`
	Name      string `json:"name"`
	ShortDesc string `json:"short_desc"`
}

type DrawnCard struct {
	Card
	Reversed bool `json:"reversed"`
}

var (
	ErrInvalidCount = errors.New("draw count must be between 1 and 78")
	ErrCardNotFound = errors.New("card not found")
)

type suitDef struct {
	id      string
	ruGen   string
	enName  string
	element string
}

type rankDef struct {
	num     int
	ruName  string
	enName  string
	meaning string
}

var suits = []suitDef{
	{id: "wands", ruGen: "Жезлов", enName: "Wands", element: "энергия, действие и амбиции"},
	{id: "cups", ruGen: "Кубков", enName: "Cups", element: "чувства, отношения и интуиция"},
	{id: "swords", ruGen: "Мечей", enName: "Swords", element: "мысли, решения и ясность"},
	{id: "pentacles", ruGen: "Пентаклей", enName: "Pentacles", element: "материя, работа и ресурсы"},
}

var ranks = []rankDef{
	{num: 1, ruName: "Туз", enName: "Ace", meaning: "начало, импульс и новый шанс"},
	{num: 2, ruName: "Двойка", enName: "Two", meaning: "выбор, баланс и партнёрство"},
	{num: 3, ruName: "Тройка", enName: "Three", meaning: "рост, сотрудничество и первые плоды"},
	{num: 4, ruName: "Четвёрка", enName: "Four", meaning: "стабильность, опора и пауза"},
	{num: 5, ruName: "Пятёрка", enName: "Five", meaning: "кризис, испытание и перемены"},
	{num: 6, ruName: "Шестёрка", enName: "Six", meaning: "гармония, поддержка и движение вперёд"},
	{num: 7, ruName: "Семёрка", enName: "Seven", meaning: "проверка, стратегия и внутренняя оценка"},
	{num: 8, ruName: "Восьмёрка", enName: "Eight", meaning: "ускорение, мастерство и преодоление"},
	{num: 9, ruName: "Девятка", enName: "Nine", meaning: "зрелость, личный итог и стойкость"},
	{num: 10, ruName: "Десятка", enName: "Ten", meaning: "завершение цикла и полнота результата"},
	{num: 11, ruName: "Паж", enName: "Page", meaning: "новости, обучение и свежий интерес"},
	{num: 12, ruName: "Рыцарь", enName: "Knight", meaning: "активный порыв, движение и напор"},
	{num: 13, ruName: "Королева", enName: "Queen", meaning: "зрелое владение, глубина и забота"},
	{num: 14, ruName: "Король", enName: "King", meaning: "контроль, лидерство и авторитет"},
}

var (
	fullDeck = buildDeck()
	byID     = indexDeck(fullDeck)
)

func buildDeck() []Card {
	majors := arcana.All()
	cards := make([]Card, 0, len(majors)+len(suits)*len(ranks))

	for _, m := range majors {
		cards = append(cards, Card{
			ID:        fmt.Sprintf("major_%d", m.Number),
			Type:      "major",
			Number:    m.Number,
			Name:      m.Name,
			ShortDesc: m.ShortDesc,
		})
	}

	for _, s := range suits {
		for _, r := range ranks {
			cards = append(cards, Card{
				ID:        fmt.Sprintf("minor_%s_%d", s.id, r.num),
				Type:      "minor",
				Suit:      s.id,
				Number:    r.num,
				Name:      fmt.Sprintf("%s %s (%s of %s)", r.ruName, s.ruGen, r.enName, s.enName),
				ShortDesc: fmt.Sprintf("%s; сфера: %s", r.meaning, s.element),
			})
		}
	}

	return cards
}

func indexDeck(cards []Card) map[string]Card {
	m := make(map[string]Card, len(cards))
	for _, c := range cards {
		m[c.ID] = c
	}
	return m
}

func All() []Card {
	res := make([]Card, len(fullDeck))
	copy(res, fullDeck)
	return res
}

func GetByID(id string) (Card, error) {
	c, ok := byID[id]
	if !ok {
		return Card{}, ErrCardNotFound
	}
	return c, nil
}

func Draw(count int) ([]DrawnCard, error) {
	if count < 1 || count > len(fullDeck) {
		return nil, ErrInvalidCount
	}

	pool := All()
	for i := len(pool) - 1; i > 0; i-- {
		j, err := cryptoRandInt(i + 1)
		if err != nil {
			return nil, err
		}
		pool[i], pool[j] = pool[j], pool[i]
	}

	drawn := make([]DrawnCard, count)
	for i := 0; i < count; i++ {
		rev, err := cryptoRandInt(2)
		if err != nil {
			return nil, err
		}
		drawn[i] = DrawnCard{
			Card:     pool[i],
			Reversed: rev == 1,
		}
	}

	return drawn, nil
}

func cryptoRandInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}
