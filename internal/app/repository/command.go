package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"r-vBackend/internal/app/ds"
	"time"

	"github.com/sirupsen/logrus"
)

func (r *Repository) GetAllCommands() ([]ds.Command, error) {
	// тут мы пользуемся ORM
	var commands []ds.Command
	err := r.db.Where("is_delete = false").Find(&commands).Error // добавили условие
	if err != nil {
		return nil, err
	}
	return commands, nil
}

func (r *Repository) GetCommandByID(id int) (*ds.Command, error) {
	// тут мы пользуемся "курсором" для примера
	query := "SELECT id, img, com_name, fmt, rd_num, rs_num, description FROM commands WHERE id = $1 AND is_delete = false"

	// Создание курсора (строковый указатель)
	row := r.db.Raw(query, id).Row()

	// Создание объекта для хранения данных
	command := &ds.Command{}

	// Сканирование строки в структуру
	err := row.Scan(
		&command.ID,
		&command.Img,
		&command.ComName,
		&command.Fmt,
		&command.RdNum,
		&command.RsNum,
		&command.Description,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Возвращаем nil, если записи нет
		}
		return nil, err
	}

	return command, nil
}

func (r *Repository) SearchCommandsByName(name string) ([]ds.Command, error) {
	var commands []ds.Command
	err := r.db.Where("com_name ILIKE ? and is_delete = ?", "%"+name+"%", false).Find(&commands).Error
	if err != nil {
		return nil, err
	}
	return commands, nil
}

func (r *Repository) DeleteCommand(commandID uint) error {
	err := r.db.Model(&ds.Command{}).Where("id = ?", commandID).UpdateColumn("is_delete", true).Error
	fmt.Println(commandID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении команды с id %d: %w", commandID, err)
	}

	return nil
}

func (r *Repository) GetProgramIDByCreatorID(userID uint) int {
	var programID int
	err := r.db.Model(&ds.Program{}).Where("creator_id = ? AND status = ?", userID, "черновик").Select("id").First(&programID).Error
	if err != nil {
		return 0
	}
	return programID
}

func (r *Repository) GetCommandsByProgram(prgID uint) ([]ds.Command, error) {
	var commands []ds.Command
	var commandIDs []uint

	err := r.db.Model(&ds.CommandProgram{}).Where("program_id = ?", prgID).Pluck("command_id", &commandIDs).Error

	if err != nil {
		return nil, err
	}

	for _, commandID := range commandIDs {
		command, err := r.GetCommandByID(int(commandID))
		if err != nil {
			return nil, err
		}
		commands = append(commands, *command)
	}

	return commands, nil
}

func (r *Repository) GetOperandByCmdPrgID(cmd_prgID uint) (int, error) {
	var operand int

	err := r.db.Model(&ds.CommandProgram{}).Where("id = ?", cmd_prgID).Select("operand").First(&operand).Error

	if err != nil {
		return 0, err
	}

	return operand, nil
}

func (r *Repository) GetOperandsByProgram(prgID uint) ([]int, error) {

	var operands []int

	var cmd_prgIDs []uint

	err := r.db.Model(&ds.CommandProgram{}).Where("program_id = ?", prgID).Pluck("id", &cmd_prgIDs).Error

	if err != nil {
		return nil, err
	}

	for _, cmd_prgID := range cmd_prgIDs {
		operand, err := r.GetOperandByCmdPrgID(uint(cmd_prgID))
		if err != nil {
			return nil, err
		}
		operands = append(operands, operand)
	}

	return operands, nil
}

func (r *Repository) GetProgramByID(prgID uint) (*ds.Program, error) {

	// Логика предоставления данных о заявке для рендера страницы:
	// Ловим данные программы по id переданному по нажатию
	// Ловим м-м и компонуем в массив

	query := fmt.Sprintf("SELECT * FROM programs WHERE id = %d;", prgID)

	row := r.db.Raw(query).Row()

	program := &ds.Program{}

	err := row.Scan(
		&program.ID,
		&program.Status,
		&program.DateCreate,
		&program.DateUpdate,
		&program.DateFinish,
		&program.ModeratorID,
		&program.CreatorID,
		&program.InitX1,
		&program.InitX2,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Возвращаем nil, если записи нет
		}
		return nil, err
	}

	return program, nil
}

// GetCartCount для получения количества услуг в заявке
func (r *Repository) GetCartCount() int64 {
	var programID uint
	var count int64
	creatorID := 1
	// пока что мы захардкодили id создателя заявки, в последующем будем получать его из JWT

	err := r.db.Model(&ds.Program{}).Where("creator_id = ? AND status = ?", creatorID, "черновик").Select("id").First(&programID).Error
	if err != nil {
		return 0
	}

	err = r.db.Model(&ds.CommandProgram{}).Where("program_id = ?", programID).Count(&count).Error
	if err != nil {
		logrus.Println("Error counting records in lists_programs:", err)
	}

	return count
}

func (r *Repository) AddToProgram(cmdID uint) error {
	creatorID := 1   // пока что хардкод
	moderatorID := 2 // пока что хардкод
	var programID uint

	cmd_count := r.GetCartCount()

	if cmd_count == 0 {
		program := ds.Program{
			Status:      "черновик",
			DateCreate:  time.Now(),
			DateUpdate:  time.Now(),
			CreatorID:   uint(creatorID),
			ModeratorID: uint(moderatorID),
		}
		err := r.db.Create(&program).Error
		if err != nil {
			return err
		}
	}

	err := r.db.Model(&ds.Program{}).Where("creator_id = ? AND status = ?", creatorID, "черновик").Select("id").First(&programID).Error
	if err != nil {
		return err
	}

	new_cmd_prg := ds.CommandProgram{
		CommandID: cmdID,
		ProgramID: programID,
	}

	err = r.db.Create(&new_cmd_prg).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteProgram(programID uint) error {

	deleteQuery := "UPDATE programs SET status = $1, date_finish = $2, date_update = $3 WHERE id = $4"
	r.db.Exec(deleteQuery, "удалена", time.Now(), time.Now(), programID)

	return nil
}
