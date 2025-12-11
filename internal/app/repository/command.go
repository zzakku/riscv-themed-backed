package repository

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"r-vBackend/internal/app/ds"
	"strings"
	"time"

	"github.com/minio/minio-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm/clause"
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

	var command ds.Command

	err := r.db.Where("id = ? AND is_delete = FALSE", id).First(&command).Error

	if err != nil {
		return nil, err
	}

	return &command, nil
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

// Удаляет из Minio ассоциированную с командой с ID commandID картинку и очищает поле img в записи команды в Postgres

func (r *Repository) deleteImage(commandID uint) error {
	var cmd_img string

	err := r.db.Model(&ds.Command{}).Where("id = ?", commandID).Select("img").First(&cmd_img).Error

	if err != nil {
		return fmt.Errorf("не удалось найти url картинки команды с id %d: %w", commandID, err)
	}

	if cmd_img != "" {
		img_obj_name := getFileNameFromPath(cmd_img)

		r.minio.RemoveObject(r.minio_bucket_name, img_obj_name)
	}

	err = r.db.Model(&ds.Command{}).Where("id = ?", commandID).UpdateColumn("img", "").Error
	if err != nil {
		return err
	}

	return nil
}

// Переключает статус is_delete в true и удаляет из Minio ассоциированную картинку

func (r *Repository) DeleteCommandWithImage(commandID uint) error {

	err := r.deleteImage(commandID)
	if err != nil {
		return err
	}

	err = r.DeleteCommand(commandID)
	if err != nil {
		return err
	}

	return nil
}

// Вытащить название файла из пути

func getFileNameFromPath(path string) string {
	components := strings.Split(path, "/")

	if len(components) > 0 {
		return components[len(components)-1]
	}

	return path
}

// добавить команду без изображения

func (r *Repository) AddCommand(command *ds.Command) error {
	err := r.db.Model(&ds.Command{}).Create(command).Error
	if err != nil {
		return fmt.Errorf("ошибка при добавлении команды: %w", err)
	}

	return nil
}

// изменить поля в команде

func (r *Repository) ModifyCommand(id uint, command *ds.Command) error {

	var old_cmd ds.Command

	err := r.db.Model(&ds.Command{}).Where("id = ? AND is_delete=False", id).First(&old_cmd).Error

	if err != nil {
		return fmt.Errorf("не удалось найти команду с id %d: %w", id, err)
	}

	// GORM обещает non-zero fields не включать в обновление, но я лучше поступлю вернее

	to_update := map[string]interface{}{
		"com_name":    command.ComName,
		"fmt":         command.Fmt,
		"rs_num":      command.RsNum,
		"rd_num":      command.RdNum,
		"description": command.Description,
	}

	// Можно было бы по порядку обойти все поля структуры и исключить из обновляемых совпадающие с текущим состоянием записи.
	// Но это довольно много кода...

	for key, value := range to_update {
		if value == "" || value == nil {
			delete(to_update, key)
		}
	}

	err = r.db.Model(&ds.Command{}).Where("id = ?", id).Updates(to_update).Error

	if err != nil {
		return fmt.Errorf("ошибка при обновлении команды с id %d: %w", id, err)
	}

	return nil
}

func (r *Repository) AddCommandImage(commandID uint, img io.Reader, header *multipart.FileHeader) error {
	err := r.deleteImage(commandID)

	if err != nil {
		return fmt.Errorf("не удалось удалить изображение команды с id %d: %w", commandID, err)
	}

	// Название будущего файла будет основано на значении поля "Название команды"

	var filename string

	err = r.db.Model(&ds.Command{}).Where("id = ?", commandID).Pluck("com_name", &filename).Error

	if err != nil {
		return fmt.Errorf("ошибка получения названия команды с id %d: %w", commandID, err)
	}

	filename = strings.ToLower(filename)
	filename = strings.ReplaceAll(filename, " ", "_")
	filename = fmt.Sprintf("%d_%s", commandID, filename)

	// А расширение файла получим из названия переданного файла

	var extension string

	piecesOfOldFilename := strings.Split(header.Filename, ".")

	extension = piecesOfOldFilename[len(piecesOfOldFilename)-1]

	filename = strings.Join([]string{filename, extension}, ".")

	// Открываем файл
	file, err := header.Open()
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	_, err = r.minio.PutObject(
		r.minio_bucket_name,
		filename,
		img,
		header.Size,
		minio.PutObjectOptions{
			ContentType: header.Header.Get("Content-Type"),
		})

	if err != nil {
		return fmt.Errorf("не удалось добавить объект в хранилище minio: %w", err)
	}

	err = r.db.Model(&ds.Command{}).Where("id = ?", commandID).UpdateColumn("img", "http://127.0.0.1:9000/"+r.minio_bucket_name+"/"+filename).Error
	if err != nil {
		// Если не удалось сохранить в БД, удаляем из MinIO
		r.minio.RemoveObject(r.minio_bucket_name, filename)
		return fmt.Errorf("ошибка сохранения пути к изображению: %w", err)
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

func (r *Repository) GetCommandsWithOperands(prgID uint) ([]ds.Command, []int, error) {
	type CommandWithOperand struct {
		ds.Command
		Operand int
	}

	var results []CommandWithOperand

	err := r.db.Model(&ds.CommandProgram{}).
		Select("commands.*, command_programs.operand").
		Joins("JOIN commands ON command_programs.command_id = commands.id").
		Where("command_programs.program_id = ?", prgID).
		Where("commands.is_delete = false").
		Find(&results).Error

	if err != nil {
		return nil, nil, err
	}

	commands := make([]ds.Command, len(results))
	operands := make([]int, len(results))

	for i, result := range results {
		commands[i].ID = result.ID
		commands[i].IsDelete = result.IsDelete
		commands[i].ComName = result.ComName
		commands[i].Description = result.Description
		commands[i].Fmt = result.Fmt
		commands[i].RdNum = result.RdNum
		commands[i].RsNum = result.RsNum
		commands[i].Img = result.Img
		operands[i] = result.Operand
	}

	return commands, operands, nil
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

func (r *Repository) GetProgramByID(prgID uint) (ds.Program, error) {

	var program ds.Program

	err := r.db.Model(&ds.Program{}).Where("id = ?", prgID).Preload("Creator").Preload("Moderator").First(&program).Error

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ds.Program{}, nil // Возвращаем nil, если записи нет
		}
		return ds.Program{}, err
	}

	return program, nil

	// Логика предоставления данных о заявке для рендера страницы:
	// Ловим данные программы по id переданному по нажатию
	// Ловим м-м и компонуем в массив
}

// для получения количества команд в программе
func (r *Repository) GetProgramCartCount(creatorID uint) int64 {

	if creatorID == 0 {
		return 0
	}

	var programID uint
	var count int64

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

func (r *Repository) GetPrograms(status string, date_start string, date_end string, userID uint) ([]ds.Program, error) {

	user, err := r.GetUser(userID)

	if err != nil {
		return nil, fmt.Errorf("не удалось найти пользователя")
	}

	var prgs []ds.Program

	query := r.db.Where("status != ? AND status != ?", "удалена", "черновик")

	if !user.IsModerator {
		query = query.Where("creator_id = ?", userID)
	}

	// Фильтр по статусу
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Фильтр по дате начала
	if date_start != "" {
		start, err := time.Parse("02.01.2006", date_start)
		if err == nil {
			query = query.Where("date_create >= ?", start)
		} else {
			fmt.Println("не удалось обработать date_start:", err)
		}
	}

	// Фильтр по дате окончания
	if date_end != "" {
		end, err := time.Parse("02.01.2006", date_end)
		if err == nil {
			query = query.Where("date_create <= ?", end.AddDate(0, 0, 1))
		} else {
			fmt.Println("не удалось обработать date_end:", err)
		}
	}

	err = query.Preload("Creator").Preload("Moderator").Find(&prgs).Error
	if err != nil {
		return nil, fmt.Errorf("ошибка получения заявок: %w", err)
	}

	return prgs, nil
}

func (r *Repository) AddToProgram(cmdID uint, creatorID uint) error {
	var programID uint

	cmd_count := r.GetProgramCartCount(creatorID)

	if cmd_count == 0 {
		program := ds.Program{
			Status:     "черновик",
			DateCreate: time.Now(),
			DateUpdate: time.Now(),
			CreatorID:  &creatorID,
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

func (r *Repository) ModifyProgramFields(programID uint, creatorID uint, init_t1 *int64, init_t2 *int64) error {
	var old_prg ds.Program

	err := r.db.Model(&ds.Program{}).Where("id = ? AND creator_id = ? AND status = 'черновик'", programID, creatorID).First(&old_prg).Error

	if err != nil {
		return fmt.Errorf("не удалось найти программу с id %d либо запрещено изменение её полей: %w", programID, err)
	}

	to_update := map[string]interface{}{
		"init_t1":     init_t1,
		"init_t2":     init_t2,
		"date_update": time.Now(),
	}

	for key, value := range to_update {
		if value == nil {
			delete(to_update, key)
		}
	}

	err = r.db.Model(&ds.Program{}).Where("id = ?", programID).Updates(to_update).Error

	if err != nil {
		return fmt.Errorf("ошибка при обновлении программы с id %d: %w", programID, err)
	}

	return nil
}

func (r *Repository) SubmitProgram(programID uint) error {
	var prg ds.Program

	// Ошибка может быть, если:
	// Нет программы с указанным ID
	// Её статус отличен от "черновик"
	// Поля InitT1 или InitT2 не инициализированы

	err := r.db.Model(&ds.Program{}).Where("id = ? AND status != 'удалена'", programID).First(&prg).Error

	if err != nil {
		return fmt.Errorf("не удалось найти программу с id %d либо она удалена: %w", programID, err)
	}

	if prg.InitT1 == nil || prg.InitT2 == nil {
		return fmt.Errorf("не задано исходное значение одного или более регистра")
	}

	err = r.db.Model(&ds.Program{}).Where("id = ?", programID).Updates(map[string]interface{}{
		"status":      "сформирована",
		"date_update": time.Now(),
	}).Error

	if err != nil {
		return fmt.Errorf("ошибка при обновлении программы с id %d: %w", programID, err)
	}

	return nil
}

func (r *Repository) executeCommand(name string, rs int64, imm int64) (int64, error) {

	var rd int64

	switch name {
	case "ADD Immediate":
		rd = rs + imm
		return rd, nil
	case "SUBtract Immediate":
		rd = rs - imm
		return rd, nil
	case "Shift Right Logical Immediate":
		rd = rs >> imm
		return rd, nil
	case "Shift Left Logical Immediate":
		rd = rs << imm
		return rd, nil
	case "NOT":
		rd = ^rs
		return rd, nil
	case "XOR Immediate":
		rd = rs ^ imm
		return rd, nil
	case "Bit-wise AND Immediate":
		rd = rs & imm
		return rd, nil
	case "Bit-wise OR Immediate":
		rd = rs | imm
		return rd, nil
	default:
		return 0, fmt.Errorf("неизвестная операция")
	}
}

// command.go - добавьте этот метод в конец файла перед последней }

// UpdateProgramResult обновляет результаты выполнения программы
func (r *Repository) UpdateProgramResult(programID uint, resT1, resT2 int64) error {
	toUpdate := map[string]interface{}{
		"res_t1":      resT1,
		"res_t2":      resT2,
		"status":      "завершена",
		"date_update": time.Now(),
		"date_finish": time.Now(),
	}

	err := r.db.Model(&ds.Program{}).
		Where("id = ?", programID).
		Updates(toUpdate).Error

	if err != nil {
		return fmt.Errorf("ошибка обновления программы: %w", err)
	}

	return nil
}

// MarkProgramAsFailed отмечает программу как завершенную с ошибкой
func (r *Repository) MarkProgramAsFailed(programID uint, errorMsg string) error {
	toUpdate := map[string]interface{}{
		"status":      "ошибка выполнения",
		"date_update": time.Now(),
		"date_finish": time.Now(),
	}

	err := r.db.Model(&ds.Program{}).
		Where("id = ?", programID).
		Updates(toUpdate).Error

	if err != nil {
		return fmt.Errorf("ошибка обновления программы: %w", err)
	}

	logrus.Errorf("Программа %d завершена с ошибкой: %s", programID, errorMsg)
	return nil
}

func (r *Repository) ExecuteOrRejectProgram(programID uint, moderatorID uint, isAccepted bool, riscvURL, apiKey, backendURL string) error {
	var prg ds.Program

	err := r.db.Model(&ds.Program{}).Where("id = ? AND status != 'удалена' AND status = 'сформирована'", programID).First(&prg).Error

	if err != nil {
		return fmt.Errorf("не удалось найти программу с id %d либо она не сформирована: %w", programID, err)
	}

	// Обновляем moderator_id и дату в любом случае
	toUpdate := map[string]interface{}{
		"moderator_id": moderatorID,
		"date_update":  time.Now(),
	}

	if !isAccepted {
		// Если отклонена - сразу завершаем
		toUpdate["date_finish"] = time.Now()
		toUpdate["status"] = "отклонена"

		return r.db.Model(&ds.Program{}).Where("id = ?", programID).Updates(toUpdate).Error
	}

	// Если принята - отправляем в RISC-V сервис

	// 1. Получаем команды программы
	type CommandWithOperand struct {
		ds.Command
		Operand int
	}

	var results []CommandWithOperand

	err = r.db.Model(&ds.CommandProgram{}).
		Select("commands.*, command_programs.operand").
		Joins("JOIN commands ON command_programs.command_id = commands.id").
		Where("command_programs.program_id = ?", programID).
		Where("commands.is_delete = false").
		Find(&results).Error

	if err != nil {
		return err
	}

	// 2. Подготавливаем данные для RISC-V сервиса
	commands := make([]map[string]interface{}, len(results))
	for i, cmd := range results {
		commands[i] = map[string]interface{}{
			"com_name": cmd.ComName,
			"rd_num":   cmd.RdNum,
			"rs_num":   cmd.RsNum,
			"operand":  cmd.Operand,
		}
	}

	// 3. Отправляем в RISC-V сервис асинхронно
	if riscvURL != "" && backendURL != "" {
		go r.sendToRiscVService(programID, prg.InitT1, prg.InitT2, commands, riscvURL, apiKey, backendURL)

		return nil
	} else {
		logrus.Errorf("Ошибка отправки в асинхронный сервис: %v", err)
		return (err)
	}

}

// Добавляем новую функцию для отправки в RISC-V сервис
func (r *Repository) sendToRiscVService(programID uint, initT1, initT2 *int64, commands []map[string]interface{}, riscvURL, apiKey, backendURL string) {
	// Подготавливаем запрос
	data := map[string]interface{}{
		"program_id":             programID,
		"init_t1":                *initT1,
		"init_t2":                *initT2,
		"commands_with_operands": commands,
		"callback_url":           fmt.Sprintf("%s/api/internal/programs/%d/callback", backendURL, programID),
		"api_key":                apiKey,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		logrus.Errorf("Ошибка маршалинга данных для RISC-V сервиса: %v", err)
		r.markProgramAsFailed(programID, err.Error())
		return
	}

	// Отправляем запрос
	req, err := http.NewRequest("PUT", riscvURL+"/api/execute/", bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.Errorf("Ошибка создания запроса: %v", err)
		r.markProgramAsFailed(programID, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Ошибка отправки в RISC-V сервис: %v", err)
		r.markProgramAsFailed(programID, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		logrus.Errorf("RISC-V сервис вернул ошибку: %d", resp.StatusCode)
		r.markProgramAsFailed(programID, fmt.Sprintf("HTTP %d", resp.StatusCode))
	} else {
		logrus.Infof("Программа %d отправлена в RISC-V сервис", programID)
	}
}

func (r *Repository) markProgramAsFailed(programID uint, errorMsg string) {
	r.db.Model(&ds.Program{}).Where("id = ?", programID).Updates(map[string]interface{}{
		"status":      "отклонена",
		"date_update": time.Now(),
		"date_finish": time.Now(),
	})
	logrus.Errorf("Программа %d завершена с ошибкой: %s", programID, errorMsg)
}

func (r *Repository) DeleteProgram(programID uint) error {
	to_update := map[string]interface{}{
		"date_update": time.Now(),
		"date_finish": time.Now(),
		"status":      "удалена",
	}

	err := r.db.Model(&ds.Program{}).Where("id = ? AND status = 'черновик'", programID).Updates(to_update).Error

	if err != nil {
		return err
	}

	return nil
}

// Удалить м-м

func (r *Repository) DeleteCommandFromProgram(programID uint, commandID uint) error {
	var prg ds.Program

	err := r.db.Model(&ds.Program{}).Where("id = ? AND status = 'черновик'", programID).First(&prg).Error

	if err != nil {
		return fmt.Errorf("не удалось найти программу с id %d, либо она не имеет статус 'черновик': %w", programID, err)
	}

	var count int64
	err = r.db.Model(&ds.CommandProgram{}).Where("program_id = ? AND command_id = ?", programID, commandID).Count(&count).Error
	if err != nil || count == 0 {
		return fmt.Errorf("команда с id %d в программе с id %d не найдена", commandID, programID)
	}

	err = r.db.Where("program_id = ? AND command_id = ?", programID, commandID).Delete(&ds.CommandProgram{}).Error
	if err != nil {
		return fmt.Errorf("ошибка при удалении услуги из заявки: %w", err)
	}

	return nil
}

func (r *Repository) ModifyCommandOperand(programID uint, commandID uint, newVal int64) error {
	var prg ds.Program

	err := r.db.Model(&ds.Program{}).Where("id = ? AND status = 'черновик'", programID).First(&prg).Error

	if err != nil {
		return fmt.Errorf("не удалось найти программу с id %d, либо она не имеет статус 'черновик': %w", programID, err)
	}

	var count int64
	err = r.db.Model(&ds.CommandProgram{}).Where("program_id = ? AND command_id = ?", programID, commandID).Count(&count).Error
	if err != nil || count == 0 {
		return fmt.Errorf("команда с id %d в программе с id %d не найдена", commandID, programID)
	}

	err = r.db.Model(&ds.CommandProgram{}).Where("program_id = ? AND command_id = ?", programID, commandID).UpdateColumn("operand", newVal).Error
	if err != nil {
		return fmt.Errorf("ошибка при обновлении операнда: %w", err)
	}

	return nil
}

// Методы работы с пользователями

func (r *Repository) RegisterUser(user ds.Users) error {

	if user.Login == "" {
		return fmt.Errorf("логин не может быть пустым")
	}

	if user.Password == "" {
		return fmt.Errorf("пароль не может быть пустым")
	}

	result := r.db.Model(&ds.Users{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "login"}},
		DoNothing: true}).Create(&user)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("пользователь с таким логином уже существует")
	}

	return nil
}

func (r *Repository) GetUser(userID uint) (*ds.Users, error) {
	var user ds.Users
	err := r.db.Where("id = ?", userID).Find(&user).Error
	if err != nil {
		return nil, err
	}

	user.Password = ""

	return &user, nil
}

func (r *Repository) UpdateUser(userID uint, updates map[string]interface{}) (*ds.Users, error) {
	if login, exists := updates["login"]; exists && login != "" {
		var existingUser ds.Users
		err := r.db.Where("login = ? AND id != ?", login, userID).First(&existingUser).Error
		if err == nil {
			return nil, fmt.Errorf("логин '%s' уже занят", login)
		}
	}
	if len(updates) > 0 {
		err := r.db.Model(&ds.Users{}).Where("id = ?", userID).Updates(updates).Error
		if err != nil {
			return nil, err
		}
	}
	var user ds.Users
	r.db.Where("id = ?", userID).First(&user)
	user.Password = ""

	return &user, nil
}

func (r *Repository) AuthUser(login, password string) (*ds.Users, error) {
	var user ds.Users
	err := r.db.Where("login = ?", login).First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("неверный логин или пароль")
	}
	if user.Password != password {
		return nil, fmt.Errorf("неверный логин или пароль")
	}
	user.Password = ""
	return &user, nil
}
