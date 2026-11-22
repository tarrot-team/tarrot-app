package arcana

import "errors"

type Arcana struct {
	Number    int    `json:"number"`
	Name      string `json:"name"`
	ShortDesc string `json:"short_desc"`
}

var ErrNotFound = errors.New("arcana not found: number must be between 1 and 22")

var list = []Arcana{
	{Number: 1, Name: "Маг (The Magician)", ShortDesc: "Воля, инициатива, мастерство и реализация идей"},
	{Number: 2, Name: "Верховная Жрица (The High Priestess)", ShortDesc: "Интуиция, тайные знания, дипломатия и гармония"},
	{Number: 3, Name: "Императрица (The Empress)", ShortDesc: "Изобилие, творчество, забота и плодородие"},
	{Number: 4, Name: "Император (The Emperor)", ShortDesc: "Структура, власть, ответственность и стабильность"},
	{Number: 5, Name: "Иерофант (The Hierophant)", ShortDesc: "Традиции, обучение, наставничество и порядок"},
	{Number: 6, Name: "Влюблённые (The Lovers)", ShortDesc: "Выбор сердцем, отношения, красота и общение"},
	{Number: 7, Name: "Колесница (The Chariot)", ShortDesc: "Движение, целеустремлённость, победа и лидерство"},
	{Number: 8, Name: "Справедливость (Justice)", ShortDesc: "Баланс, честность, закон причины и следствия"},
	{Number: 9, Name: "Отшельник (The Hermit)", ShortDesc: "Мудрость, глубина, самопознание и поиск истины"},
	{Number: 10, Name: "Колесо Фортуны (Wheel of Fortune)", ShortDesc: "Поток, цикличность, удача и перемены"},
	{Number: 11, Name: "Сила (Strength)", ShortDesc: "Внутренняя мощь, выносливость, потенциал и смелость"},
	{Number: 12, Name: "Повешенный (The Hanged Man)", ShortDesc: "Новый взгляд, служение, креативность и переосмысление"},
	{Number: 13, Name: "Смерть (Death)", ShortDesc: "Трансформация, завершение старого и обновление"},
	{Number: 14, Name: "Умеренность (Temperance)", ShortDesc: "Зрелость, исцеление, мера и душевное равновесие"},
	{Number: 15, Name: "Дьявол (The Devil)", ShortDesc: "Искушение, харизма, материальная сила и проверка тенью"},
	{Number: 16, Name: "Башня (The Tower)", ShortDesc: "Пробуждение, разрушение иллюзий и духовный рост"},
	{Number: 17, Name: "Звезда (The Star)", ShortDesc: "Вдохновение, талант, надежда и яркость"},
	{Number: 18, Name: "Луна (The Moon)", ShortDesc: "Воображение, глубина психики, сны и сила мысли"},
	{Number: 19, Name: "Солнце (The Sun)", ShortDesc: "Радость, открытость, процветание и щедрость"},
	{Number: 20, Name: "Суд (Judgement)", ShortDesc: "Призвание, связь с родом, ясность и обновление"},
	{Number: 21, Name: "Мир (The World)", ShortDesc: "Целостность, масштаб, свобода и миротворчество"},
	{Number: 22, Name: "Шут (The Fool)", ShortDesc: "Свобода, лёгкость, новый путь и доверие миру"},
}

func Get(number int) (Arcana, error) {
	if number < 1 || number > len(list) {
		return Arcana{}, ErrNotFound
	}
	return list[number-1], nil
}

func All() []Arcana {
	res := make([]Arcana, len(list))
	copy(res, list)
	return res
}
