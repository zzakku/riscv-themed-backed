package handler

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"r-vBackend/internal/app/ds"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Домен услуги

func (h *Handler) GetAllCommandsAPI(ctx *gin.Context) {
	var commands []ds.Command
	var err error

	search := ctx.Query("searchQuery")
	if search == "" {
		commands, err = h.Repository.GetAllCommands()
	} else {
		commands, err = h.Repository.SearchCommandsByName(search)
	}

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   commands,
	})
}

func (h *Handler) GetCommandByIdAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	command, err := h.Repository.GetCommandByID(id)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   command,
	})
}

func (h *Handler) AddCommandAPI(ctx *gin.Context) {
	var command ds.Command
	if err := ctx.BindJSON(&command); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	err := h.Repository.AddCommand(&command)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    command,
		"message": "запись успешно добавлена",
	})
}

// Обновить ряд полей команды (нельзя обновить картинку и ID) - API

func (h *Handler) ModifyCommandAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	var command ds.Command

	if err := ctx.BindJSON(&command); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	if err = h.Repository.ModifyCommand(uint(id), &command); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	updatedCmd, err := h.Repository.GetCommandByID(int(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    updatedCmd,
		"message": "запись успешно обновлена",
	})
}

// Удаление команды. Удаление изображения встроено сюда

func (h *Handler) DeleteCommandAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	err = h.Repository.DeleteCommandWithImage(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "команда успешно удалена",
	})
}

func (h *Handler) AddCommandToProgramAPI(ctx *gin.Context) {

	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Вызов функции добавления чата в заявку
	err = h.Repository.AddToProgram(uint(id))
	if err != nil && !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "команда успешно добавлена в программу",
	})
}

func (h *Handler) AddCommandImageAPI(ctx *gin.Context) {

	strId := ctx.Param("id")
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  err.Error(),
		})
	}

	header, err := ctx.FormFile("image")

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "не удалось получить файл",
		})
		return
	}

	file, err := header.Open()

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "не удалось получить файл",
		})
		return
	}

	defer file.Close()

	buffer := make([]byte, 512) // Тип полученного файла хранится в первых 512 байтах файла
	_, err = file.Read(buffer)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "не удалось прочитать файл",
		})
		return
	}

	contentType := http.DetectContentType(buffer)
	if !isImage(contentType) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "файл должен быть изображением",
		})
		return
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "ошибка при обработке файла",
		})
		return
	}

	err = h.Repository.AddCommandImage(uint(id), file, header)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "не удалось загрузить файл на сервер",
		})
		return
	}

	updatedCmd, err := h.Repository.GetCommandByID(int(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    updatedCmd,
		"message": "изображение успешно обновлено",
	})
}

// Проверка типа файла
func isImage(contentType string) bool {
	imageTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, t := range imageTypes {
		if contentType == t {
			return true
		}
	}
	return false
}

// Домен заявки

// Получение иконки корзины

func (h *Handler) GetCartCountAPI(ctx *gin.Context) {
	count := h.Repository.GetCartCount()
	draft_id := h.Repository.GetProgramIDByCreatorID(2)

	ctx.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"draft_id": draft_id,
		"count":    count,
	})
}

//  Получить список программ
//  (поля программы, НО вместо id создателя/модератора - их логины, статус - исключить черновик и удалённые),
//  с фильтрацией по диапазону даты формирования и статусу

func (h *Handler) GetProgramsAPI(ctx *gin.Context) {

	// Запрос без JSON по шаблону ниже не сработает

	type filter_req struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Status    string `json:"status"`
	}

	var filter filter_req

	if err := ctx.ShouldBindJSON(&filter); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	var prgs []ds.Program

	prgs, err := h.Repository.GetPrograms(filter.Status, filter.StartDate, filter.EndDate)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, fmt.Errorf("ошибка обработки запроса"))
		return
	}

	type ProgramResp struct {
		ID             uint   `json:"id"`
		Status         string `json:"status"`
		DateCreate     string `json:"date_create"`
		DateUpdate     string `json:"date_update"`
		DateFinish     string `json:"date_finish"`
		CreatorLogin   string `json:"creator_login"`
		ModeratorLogin string `json:"moderator_login"`
		InitT1         *int64 `json:"init_t1"`
		InitT2         *int64 `json:"init_t2"`
		ResT1          *int64 `json:"res_t1"`
		ResT2          *int64 `json:"res_t2"`
	}

	response := make([]ProgramResp, len(prgs))
	for i, prg := range prgs {
		response[i] = ProgramResp{
			ID:           prg.ID,
			Status:       prg.Status,
			DateCreate:   prg.DateCreate.Format("02.01.2006"),
			DateUpdate:   prg.DateUpdate.Format("02.01.2006"),
			CreatorLogin: prg.Creator.Login,
			InitT1:       prg.InitT1,
			InitT2:       prg.InitT2,
			ResT1:        prg.ResT1,
			ResT2:        prg.ResT2,
		}

		// null распарсить нельзя, если в дате окончания null, то на выходе имеем пустую строку
		if prg.DateFinish.Valid {
			response[i].DateFinish = prg.DateFinish.Time.Format("02.01.2006")
		}

		// id 0 не с чем сопоставить, поэтому логин дёргаем только если id != 0
		if prg.Moderator.ID != 0 {
			response[i].ModeratorLogin = prg.Moderator.Login
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"programs": response,
	})
}

// GET одна запись (поля `заявки` + ее `услуги`). При получении `заявки` возвращется список ее услуг с картинками

func (h *Handler) GetProgramAPI(ctx *gin.Context) {

	// Структура ответа:
	// - Одна запись "Программа" целиком
	// - [Запись "Команда" целиком + поле "операнд" из м-м] - массив
	// Идут по порядку ID м-м

	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	var prg ds.Program

	prg, err = h.Repository.GetProgramByID(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	commands, operands, err := h.Repository.GetCommandsWithOperands(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	type CommandWithOperand struct {
		Command ds.Command
		Operand int
	}

	res_arr := make([]CommandWithOperand, len(commands))

	for i := range len(commands) {
		res_arr[i].Command = commands[i]
		res_arr[i].Operand = operands[i]
	}

	type ProgramResp struct {
		ID             uint   `json:"id"`
		Status         string `json:"status"`
		DateCreate     string `json:"date_create"`
		DateUpdate     string `json:"date_update"`
		DateFinish     string `json:"date_finish"`
		CreatorLogin   string `json:"creator_login"`
		ModeratorLogin string `json:"moderator_login"`
		InitT1         *int64 `json:"init_t1"`
		InitT2         *int64 `json:"init_t2"`
		ResT1          *int64 `json:"res_t1"`
		ResT2          *int64 `json:"res_t2"`
	}

	response := ProgramResp{
		ID:           prg.ID,
		Status:       prg.Status,
		DateCreate:   prg.DateCreate.Format("02.01.2006"),
		DateUpdate:   prg.DateUpdate.Format("02.01.2006"),
		CreatorLogin: prg.Creator.Login,
		InitT1:       prg.InitT1,
		InitT2:       prg.InitT2,
		ResT1:        prg.ResT1,
		ResT2:        prg.ResT2,
	}

	// null распарсить нельзя, если в дате окончания null, то на выходе имеем пустую строку
	if prg.DateFinish.Valid {
		response.DateFinish = prg.DateFinish.Time.Format("02.01.2006")
	}

	// id 0 не с чем сопоставить, поэтому логин дёргаем только если id != 0
	if prg.Moderator.ID != 0 {
		response.ModeratorLogin = prg.Moderator.Login
	}

	ctx.JSON(http.StatusOK, gin.H{
		"program":                response,
		"commands_with_operands": res_arr,
	})

}

// PUT изменения полей заявки по теме

func (h *Handler) ModifyProgramFieldsAPI(ctx *gin.Context) {

	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	type ModifyProgramFieldsReq struct {
		InitT1 *int64 `json:"init_t1"`
		InitT2 *int64 `json:"init_t2"`
	}

	var req ModifyProgramFieldsReq

	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	err = h.Repository.ModifyProgramFields(uint(id), req.InitT1, req.InitT2)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Получаем обновлённую программу
	prg, err := h.Repository.GetProgramByID(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	commands, operands, err := h.Repository.GetCommandsWithOperands(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	type CommandWithOperand struct {
		Command ds.Command
		Operand int
	}

	res_arr := make([]CommandWithOperand, len(commands))

	for i := range len(commands) {
		res_arr[i].Command = commands[i]
		res_arr[i].Operand = operands[i]
	}

	type ProgramResp struct {
		ID             uint   `json:"id"`
		Status         string `json:"status"`
		DateCreate     string `json:"date_create"`
		DateUpdate     string `json:"date_update"`
		DateFinish     string `json:"date_finish"`
		CreatorLogin   string `json:"creator_login"`
		ModeratorLogin string `json:"moderator_login"`
		InitT1         *int64 `json:"init_t1"`
		InitT2         *int64 `json:"init_t2"`
		ResT1          *int64 `json:"res_t1"`
		ResT2          *int64 `json:"res_t2"`
	}

	response := ProgramResp{
		ID:           prg.ID,
		Status:       prg.Status,
		DateCreate:   prg.DateCreate.Format("02.01.2006"),
		DateUpdate:   prg.DateUpdate.Format("02.01.2006"),
		CreatorLogin: prg.Creator.Login,
		InitT1:       prg.InitT1,
		InitT2:       prg.InitT2,
		ResT1:        prg.ResT1,
		ResT2:        prg.ResT2,
	}

	// null распарсить нельзя, если в дате окончания null, то на выходе имеем пустую строку
	if prg.DateFinish.Valid {
		response.DateFinish = prg.DateFinish.Time.Format("02.01.2006")
	}

	// id 0 не с чем сопоставить, поэтому логин дёргаем только если id != 0
	if prg.Moderator.ID != 0 {
		response.ModeratorLogin = prg.Moderator.Login
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"program":  response,
		"commands": res_arr,
		"message":  "поля успешно изменены",
	})
}

// PUT сформировать создателем (дата формирования). Происходит проверка на обязательные поля

func (h *Handler) SubmitProgramAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	err = h.Repository.SubmitProgram(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	var prg ds.Program

	prg, err = h.Repository.GetProgramByID(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	commands, operands, err := h.Repository.GetCommandsWithOperands(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	type CommandWithOperand struct {
		Command ds.Command
		Operand int
	}

	res_arr := make([]CommandWithOperand, len(commands))

	for i := range len(commands) {
		res_arr[i].Command = commands[i]
		res_arr[i].Operand = operands[i]
	}

	type ProgramResp struct {
		ID             uint   `json:"id"`
		Status         string `json:"status"`
		DateCreate     string `json:"date_create"`
		DateUpdate     string `json:"date_update"`
		DateFinish     string `json:"date_finish"`
		CreatorLogin   string `json:"creator_login"`
		ModeratorLogin string `json:"moderator_login"`
		InitT1         *int64 `json:"init_t1"`
		InitT2         *int64 `json:"init_t2"`
		ResT1          *int64 `json:"res_t1"`
		ResT2          *int64 `json:"res_t2"`
	}

	response := ProgramResp{
		ID:           prg.ID,
		Status:       prg.Status,
		DateCreate:   prg.DateCreate.Format("02.01.2006"),
		DateUpdate:   prg.DateUpdate.Format("02.01.2006"),
		CreatorLogin: prg.Creator.Login,
		InitT1:       prg.InitT1,
		InitT2:       prg.InitT2,
		ResT1:        prg.ResT1,
		ResT2:        prg.ResT2,
	}

	// null распарсить нельзя, если в дате окончания null, то на выходе имеем пустую строку
	if prg.DateFinish.Valid {
		response.DateFinish = prg.DateFinish.Time.Format("02.01.2006")
	}

	// id 0 не с чем сопоставить, поэтому логин дёргаем только если id != 0
	if prg.Moderator.ID != 0 {
		response.ModeratorLogin = prg.Moderator.Login
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":                 "success",
		"program":                response,
		"commands_with_operands": res_arr,
		"message":                "формирование программы завершено",
	})

}

// PUT завершить/отклонить модератором. При завершить/отклонении заявки проставляется `модератор` и дата завершения.
// Одно из доп. полей `заявки` или `м-м` рассчитывается (реализовать формулу представленную в лаб-2) при завершении заявки
// (вычисление стоимости заказа, даты доставки в течении месяца, вычисления в м-м).

func (h *Handler) ExecuteOrRejectProgramAPI(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	var input struct {
		IsAccepted *bool `json:"is_accepted" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&input); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	moderatorID := uint(2)

	err = h.Repository.ExecuteOrRejectProgram(uint(id), moderatorID, *input.IsAccepted)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	message := "программа отклонена"
	if *input.IsAccepted {
		message = "программа выполнена"
	}

	var prg ds.Program

	prg, err = h.Repository.GetProgramByID(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	commands, operands, err := h.Repository.GetCommandsWithOperands(uint(id))

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	type CommandWithOperand struct {
		Command ds.Command
		Operand int
	}

	res_arr := make([]CommandWithOperand, len(commands))

	for i := range len(commands) {
		res_arr[i].Command = commands[i]
		res_arr[i].Operand = operands[i]
	}

	type ProgramResp struct {
		ID             uint   `json:"id"`
		Status         string `json:"status"`
		DateCreate     string `json:"date_create"`
		DateUpdate     string `json:"date_update"`
		DateFinish     string `json:"date_finish"`
		CreatorLogin   string `json:"creator_login"`
		ModeratorLogin string `json:"moderator_login"`
		InitT1         *int64 `json:"init_t1"`
		InitT2         *int64 `json:"init_t2"`
		ResT1          *int64 `json:"res_t1"`
		ResT2          *int64 `json:"res_t2"`
	}

	response := ProgramResp{
		ID:           prg.ID,
		Status:       prg.Status,
		DateCreate:   prg.DateCreate.Format("02.01.2006"),
		DateUpdate:   prg.DateUpdate.Format("02.01.2006"),
		CreatorLogin: prg.Creator.Login,
		InitT1:       prg.InitT1,
		InitT2:       prg.InitT2,
		ResT1:        prg.ResT1,
		ResT2:        prg.ResT2,
	}

	// null распарсить нельзя, если в дате окончания null, то на выходе имеем пустую строку
	if prg.DateFinish.Valid {
		response.DateFinish = prg.DateFinish.Time.Format("02.01.2006")
	}

	// id 0 не с чем сопоставить, поэтому логин дёргаем только если id != 0
	if prg.Moderator.ID != 0 {
		response.ModeratorLogin = prg.Moderator.Login
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":                 "success",
		"program":                response,
		"commands_with_operands": res_arr,
		"message":                message,
	})
}

// DELETE удаление (дата формирования)

func (h *Handler) DeleteProgramAPI(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	err = h.Repository.DeleteProgram(uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "удаление программы прошло успешно",
	})
}

// Домен м-м
// DELETE удаление из заявки (без `PK м-м`)

func (h *Handler) DeleteCommandFromProgramAPI(ctx *gin.Context) {

	strId := ctx.Query("command_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	programID := h.Repository.GetProgramIDByCreatorID(2)

	err = h.Repository.DeleteCommandFromProgram(uint(programID), uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "удаление команды прошло успешно",
	})
}

// PUT изменение количества/порядка/значения в м-м (`без PK м-м`)

func (h *Handler) ModifyCommandOperandAPI(ctx *gin.Context) {

	strId := ctx.Query("command_id")
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	programID := h.Repository.GetProgramIDByCreatorID(2)

	// Запрос без JSON по шаблону ниже не сработает

	type operand_req struct {
		Operand int64 `json:"operand"`
	}

	var newVal operand_req

	if err := ctx.ShouldBindJSON(&newVal); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	err = h.Repository.ModifyCommandOperand(uint(programID), uint(id), newVal.Operand)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "значение операнда команды успешно обновлен",
	})
}

// Домен пользователь
// POST регистрация

func (h *Handler) RegisterUserAPI(ctx *gin.Context) {

	type RegisterRequest struct {
		Login    string `json:"login" binding:"required,min=3,max=25"`
		Password string `json:"password" binding:"required,min=6"`
	}

	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &ds.Users{
		Login:    req.Login,
		Password: generateHashString(req.Password),
	}

	if err := h.Repository.RegisterUser(*user); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	user.Password = ""

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    user,
		"message": "пользователь зарегистрирован",
	})
}

// GET полей пользователя после аутентификации (для личного кабинета)

func (h *Handler) GetUserAPI(ctx *gin.Context) {
	userID := uint(1) // Фиксированный ID пользователя

	user, err := h.Repository.GetUser(userID)

	if err != nil {
		h.errorHandler(ctx, http.StatusNotFound, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"user":   user,
	})
}

// PUT пользователя (личный кабинет)

func (h *Handler) UpdateUserAPI(ctx *gin.Context) {
	userID := uint(1)

	var input struct {
		Login       *string `json:"login,omitempty"`
		Name        *string `json:"name,omitempty"`
		IsModerator *bool   `json:"is_moderator,omitempty"`
	}
	if err := ctx.ShouldBindJSON(&input); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	updates := make(map[string]interface{})
	if input.Login != nil {
		updates["login"] = *input.Login
	}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.IsModerator != nil {
		updates["is_moderator"] = *input.IsModerator
	}
	if len(updates) == 0 {
		h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("нет полей для обновления"))
		return
	}
	user, err := h.Repository.UpdateUser(userID, updates)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    user,
		"message": "данные обновлены",
	})
}

// POST аутентификация

func (h *Handler) AuthUserAPI(ctx *gin.Context) {

	var request struct {
		Login    string `json:"login" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	user, err := h.Repository.AuthUser(request.Login, request.Password)
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"data":    user,
		"message": "аутентификация выполнена",
	})
}

// POST деавторизация

func (h *Handler) DeauthUserAPI(ctx *gin.Context) {

	// Функционал появится, когда можно будет получить информацию из JWT

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "деаутентификация выполнена",
	})
}

func generateHashString(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
