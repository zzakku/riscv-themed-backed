package repository

import (
	"fmt"
	"strings"
)

type Repository struct {
}

func NewRepository() (*Repository, error) {
	return &Repository{}, nil
}

type Command struct { // вот наша новая структура
	ID          int // поля структур, которые передаются в шаблон
	ComName     string
	Fmt         string
	RsNum       int
	RdNum       int
	Description string
	CardImage   string // ОБЯЗАТЕЛЬНО должны быть написаны с заглавной буквы (то есть публичными)
}

//TO-DO: "Словарь" заявок??

type Program struct {
	ID        int
	Commands  []Command // Список команд
	NumParams []int     // Список параметров для них
}

func (r *Repository) GetCommands() ([]Command, error) {
	// имитируем работу с БД. Типа мы выполнили sql запрос и получили эти строки из БД
	commands := []Command{ // массив элементов из наших структур
		{
			ID:          1,
			ComName:     "Shift Right Logical Immediate",
			Fmt:         "rd, rs, imm",
			RsNum:       1,
			RdNum:       2,
			CardImage:   "http://127.0.0.1:9000/rvimg/shiftright.png",
			Description: "Выполняет операцию побитового сдвига числа rs на imm позиции вправо, после чего число записывается в rd.",
		},
		{
			ID:          2,
			ComName:     "Shift Left Logical Immediate",
			Fmt:         "rd, rs, imm",
			RsNum:       1,
			RdNum:       2,
			CardImage:   "http://127.0.0.1:9000/rvimg/shiftleft.png",
			Description: "Выполняет операцию побитового сдвига числа rs на imm позиции влево, после чего число записывается в rd.",
		},
		{
			ID:          3,
			ComName:     "ADD Immediate",
			Fmt:         "rd, rs, imm",
			RsNum:       1,
			RdNum:       2,
			CardImage:   "http://127.0.0.1:9000/rvimg/addi.jpg",
			Description: "Число в rs складывается с imm, результат записывается в rd.",
		},
		{
			ID:          4,
			ComName:     "NOT",
			Fmt:         "rd, rs",
			RsNum:       1,
			RdNum:       2,
			CardImage:   "http://127.0.0.1:9000/rvimg/not.jpg",
			Description: "Биты числа в rs инвертируются, результат записывается в rd.",
		},
		{
			ID:          5,
			ComName:     "XOR Immediate",
			Fmt:         "rd, rs, imm",
			RsNum:       1,
			RdNum:       2,
			CardImage:   "http://127.0.0.1:9000/rvimg/xori.png",
			Description: "Побитово производится операция исключающее ИЛИ над числами rs и imm, результат записывается в rd.",
		},
		{
			ID:          6,
			ComName:     "Bit-wise AND Immediate",
			Fmt:         "rd, rs, imm",
			RsNum:       1,
			RdNum:       2,
			CardImage:   "http://127.0.0.1:9000/rvimg/and.jpg",
			Description: "Побитово производится операция логическое И над числами rs и imm, результат записывается в rd.",
		},
		{
			ID:          7,
			ComName:     "Bit-wise OR Immediate",
			Fmt:         "rd, rs, imm",
			RsNum:       1,
			RdNum:       2,
			CardImage:   "http://127.0.0.1:9000/rvimg/or.png",
			Description: "Побитово производится операция логическое ИЛИ над числами rs и imm, результат записывается в rd.",
		},
	}
	// обязательно проверяем ошибки, и если они появились - передаем выше, то есть хендлеру
	// тут я снова искусственно обработаю "ошибку" чисто чтобы показать вам как их передавать выше
	if len(commands) == 0 {
		return nil, fmt.Errorf("массив пустой")
	}

	return commands, nil
}

func (r *Repository) GetPrograms() ([]Program, error) {
	programs := []Program{ // массив элементов из наших структур
		{
			ID: 1,
			Commands: []Command{
				{
					ID:          3,
					ComName:     "ADD Immediate",
					Fmt:         "rd, rs, imm",
					RsNum:       1,
					RdNum:       2,
					CardImage:   "http://127.0.0.1:9000/rvimg/and.jpg",
					Description: "Число в rs складывается с imm, результат записывается в rd.",
				},
			},
			NumParams: []int{1},
		},
	}
	if len(programs) == 0 {
		return nil, fmt.Errorf("массив пустой")
	}

	return programs, nil
}

func (r *Repository) GetCommand(id int) (Command, error) {
	// тут у вас будет логика получения нужной услуги, тоже наверное через цикл в первой лабе, и через запрос к БД начиная со второй
	commands, err := r.GetCommands()
	if err != nil {
		return Command{}, err // тут у нас уже есть кастомная ошибка из нашего метода, поэтому мы можем просто вернуть ее
	}

	for _, command := range commands {
		if command.ID == id {
			return command, nil // если нашли, то просто возвращаем найденный заказ (услугу) без ошибок
		}
	}
	return Command{}, fmt.Errorf("команда не найдена") // тут нужна кастомная ошибка, чтобы понимать на каком этапе возникла ошибка и что произошло
}

func (r *Repository) GetCommandsByName(cname string) ([]Command, error) {
	commands, err := r.GetCommands()
	if err != nil {
		return []Command{}, err
	}

	var result []Command
	for _, command := range commands {
		if strings.Contains(strings.ToLower(command.ComName), strings.ToLower(cname)) {
			result = append(result, command)
		}
	}

	return result, nil
}
