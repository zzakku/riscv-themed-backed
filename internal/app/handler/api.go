package handler

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"r-vBackend/internal/app/ds"
	"r-vBackend/internal/app/role"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
)

// Структуры данных

type registerRequest struct {
	Login    string `json:"login" binding:"required,min=3,max=25"`
	Password string `json:"password" binding:"required,min=6"`
}

type authRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResp struct {
	ExpiresIn   time.Duration `json:"expires_in" swaggertype:"integer"`
	AccessToken string        `json:"access_token"`
	TokenType   string        `json:"token_type"`
}

type successResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data" swaggertype:"object"`
}

type successMessageResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type successCartResp struct {
	Status string          `json:"status"`
	Data   programCartResp `json:"data"`
}

type errorResponse struct {
	Status      string `json:"status"`
	Description string `json:"description"`
}

type programResp struct {
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

type commandWithOperand struct {
	Command ds.Command
	Operand int
}

type programCartResp struct {
	ProgramId int   `json:"prg_id"`
	Count     int64 `json:"count"`
}

type programCmdsResp struct {
	Program     programResp          `json:"program"`
	CmdsWithOps []commandWithOperand `json:"commands_with_operands"`
}

type modifyProgramFieldsReq struct {
	InitT1 *int64 `json:"init_t1"`
	InitT2 *int64 `json:"init_t2"`
}

type moderatorDecisionReq struct {
	IsAccepted *bool `json:"is_accepted" binding:"required"`
}

type moderatedProgramResp struct {
	Status      string               `json:"status"`
	Message     string               `json:"message"`
	Program     programResp          `json:"program"`
	CmdsWithOps []commandWithOperand `json:"commands_with_operands"`
}

type operandReq struct {
	Operand int64 `json:"operand" binding:"required"`
}

type userPutReq struct {
	Login    *string `json:"login,omitempty"`
	Password *string `json:"password,omitempty"`
}

// Домен услуги

// GetAllCommandsAPI godoc
//
//		@Summary		Получить все команды
//		@Description	Получить все неудалённые команды. Доступно любому пользователю.
//		@Tags			commands
//	 @Param query path string false "поисковый запрос"
//		@Produce		json
//		@Success		200	{object}	successResponse
//		@Failure		500	{object}	errorResponse
//		@Router			/api/commands [get]
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

	ctx.JSON(http.StatusOK, successResponse{
		Status:  "success",
		Message: "команды получены",
		Data:    commands,
	})
}

// GetCommandByIdAPI godoc
//
//	@Summary		Получить команду по ID
//	@Description	Получить данные одной команды по её ID. Доступно любому пользователю.
//	@Tags			commands
//	@Produce		json
//
// @Param id path int true "id команды"
//
//	@Success		200	{object}	successResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/api/commands/{id} [get]
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

// AddCommandAPI godoc
//
//	@Summary		Добавить команду
//	@Description	Добавить команду без изображения. Доступно ревьюеру.
//	@Tags			commands
//	@Produce		json
//
// @Param request body ds.Command true "Добавляемая команда"
//
//	@Success		200	{object}	successResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/api/commands/ [post]
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

// ModifyCommandAPI godoc
//
//	@Summary		Обновить команду
//	@Description	Обновить данные неудалённой команды. Доступно ревьюеру.
//	@Tags			commands
//
// @Param id path int true "id команды"
// @Param updated_command body ds.Command true "Обновлённая команда"
//
//	@Produce		json
//	@Success		200	{object}	successResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/api/command/{id} [put]
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

// DeleteCommandAPI godoc
//
//	@Summary		Удалить команду
//	@Description	Удаляет команду и ассоциированное изображение в Minio. Доступно ревьюеру.
//	@Tags			commands
//	@Produce		json
//
// @Param id path int true "id команды"
//
//	@Success		200	{object}	successResponse
//	@Failure		500	{object}	errorResponse
//	@Router			/api/commands/{id} [delete]
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

// AddCommandToProgramAPI godoc
//
//	@Summary		Добавить команду в программу
//	@Description	Позволяет добавить команду в текущую программу-черновик. Доступно авторизованным пользователям.
//	@Tags			commands
//	@Produce		json
//
// @Param id path int true "id команды"
//
//	@Success		200	{object}	successResponse
//	@Failure		500	{object}	errorResponse
//
// @Failure 400
//
//	@Router			/api/commands/{id}/add-to-program [post]
func (h *Handler) AddCommandToProgramAPI(ctx *gin.Context) {

	userID, err := h.getUserIDFromJWT(ctx)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Вызов функции добавления чата в заявку
	err = h.Repository.AddToProgram(uint(id), userID)
	if err != nil && !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "команда успешно добавлена в программу",
	})
}

// AddCommandImageAPI godoc
//
//	@Summary		Добавить изображение
//	@Description    Добавить изображение к команде, сохранив его в Minio. Доступно ревьюеру.
//	@Tags			commands
//	@Produce		json
//
// @Param id path int true "id команды"
// @Param image formData file true "Файл изображения"
//
//	@Success		200	{object}	successResponse
//
// @Failure 400 {object} errorResponse
//
//	@Failure		500	{object}	errorResponse
//	@Router			/api/commands/{id}/add-image [get]
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

// GetProgramCartCountAPI godoc
//
//	@Summary		Получить иконку корзины
//	@Description	Получает ID текущей программы-черновика и количество команд в ней. Доступно всем, для гостя всегда возвращается 0, 0
//	@Tags			programs
//	@Produce		json
//	@Success		200		{object} successCartResp
//	@Failure		500		{object}	errorResponse
//
// @Failure 403
//
//	@Router			/api/programs/cart-icon [get]
func (h *Handler) GetProgramCartCountAPI(ctx *gin.Context) {
	userID, err := h.getUserIDFromJWT(ctx)

	if err != nil && err.Error() != "jwt не имеет нужный префикс" {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	if err != nil && err.Error() == "jwt не имеет нужный префикс" {
		ctx.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data": programCartResp{
				Count:     0,
				ProgramId: 0,
			},
		})
		return
	}

	count := h.Repository.GetProgramCartCount(userID)
	prg_id := h.Repository.GetProgramIDByCreatorID(userID)

	resp := programCartResp{
		Count:     count,
		ProgramId: prg_id,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   resp,
	})
}

//  Получить список программ
//  (поля программы, НО вместо id создателя/ревьюера - их логины, статус - исключить черновик и удалённые),
//  с фильтрацией по диапазону даты формирования и статусу

// GetProgramsAPI godoc
//
//	@Summary		Получить список програм
//	@Description	Получить список неудалённых программ. Ревьюер может получить все, оператор - только свои.
//	@Tags			programs
//	@Produce		json
//	@Param			status		query	string	false	"Статус программы"
//	@Param			start_date	query	string	false	"Дата начала фильтрации (формат: DD.MM.YYYY)"
//	@Param			end_date	query	string	false	"Дата окончания фильтрации (формат: DD.MM.YYYY)"
//	@Success		200			{object}	successResponse
//	@Failure		500			{object}	errorResponse
//	@Failure		403
//	@Router			/api/programs [get]
func (h *Handler) GetProgramsAPI(ctx *gin.Context) {
	userID, err := h.getUserIDFromJWT(ctx)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	// Получаем query-параметры
	status := ctx.Query("status")
	startDate := ctx.Query("start_date")
	endDate := ctx.Query("end_date")

	// Валидация формата дат, если они переданы
	if startDate != "" {
		if _, err := time.Parse("02.01.2006", startDate); err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный формат start_date, ожидается DD.MM.YYYY"))
			return
		}
	}

	if endDate != "" {
		if _, err := time.Parse("02.01.2006", endDate); err != nil {
			h.errorHandler(ctx, http.StatusBadRequest, fmt.Errorf("неверный формат end_date, ожидается DD.MM.YYYY"))
			return
		}
	}

	prgs, err := h.Repository.GetPrograms(status, startDate, endDate, userID)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, fmt.Errorf("ошибка обработки запроса: %v", err))
		return
	}

	response := make([]programResp, len(prgs))
	for i, prg := range prgs {
		response[i] = programResp{
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
		"status":  "success",
		"message": "команды получены успешно",
		"data":    response,
	})
}

// GET одна запись (поля `заявки` + ее `услуги`). При получении `заявки` возвращется список ее услуг с картинками

// GetProgramAPI godoc
//
//	@Summary		Получить одну программу
//	@Description	Получить одну программу. Ревьюер может получить любую, создатель - только свои.
//	@Tags			programs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"id программы"
//	@Success		200		{object} programCmdsResp
//	@Failure		500		{object}	errorResponse
//	@Router			/api/programs/{id} [get]
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

	res_arr := make([]commandWithOperand, len(commands))

	for i := range len(commands) {
		res_arr[i].Command = commands[i]
		res_arr[i].Operand = operands[i]
	}

	response := programResp{
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

	ctx.JSON(http.StatusOK, programCmdsResp{
		Program:     response,
		CmdsWithOps: res_arr,
	})

}

// PUT изменения полей заявки по теме

// ModifyProgramFieldsAPI godoc
//
//	@Summary		Изменить поля программы
//	@Description	Изменяет дополнительные поля программы. Доступно авторизованному пользователю.
//	@Tags			programs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"id программы"
//
// @Param request body modifyProgramFieldsReq true "Модифицируемые поля по теме. Можно опустить то, которое мы не будем менять."
//
//	@Success		200		{object} programCmdsResp
//	@Failure		500		{object}	errorResponse
//	@Router			/api/programs/{id} [put]
func (h *Handler) ModifyProgramFieldsAPI(ctx *gin.Context) {

	userID, err := h.getUserIDFromJWT(ctx)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	var req modifyProgramFieldsReq

	if err := ctx.ShouldBindJSON(&req); err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	err = h.Repository.ModifyProgramFields(uint(id), userID, req.InitT1, req.InitT2)

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

	res_arr := make([]commandWithOperand, len(commands))

	for i := range len(commands) {
		res_arr[i].Command = commands[i]
		res_arr[i].Operand = operands[i]
	}

	response := programResp{
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
		"program":  response,
		"commands": res_arr,
	})
}

// PUT сформировать создателем (дата формирования). Происходит проверка на обязательные поля

// SubmitProgramAPI godoc
//
//	@Summary		Сформировать программу
//	@Description	Завершить черновик заявки-программы и отправить на модерацию. Доступно авторизованному пользователю.
//	@Tags			programs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"id программы"
//	@Success		200		{object} successMessageResp
//	@Failure		500		{object}	errorResponse
//	@Router			/api/programs/{id}/submit [put]
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

	ctx.JSON(http.StatusOK, successMessageResp{
		Status:  "success",
		Message: "формирование программы завершено",
	})

}

// PUT завершить/отклонить ревьюером. При завершить/отклонении заявки проставляется `ревьюер` и дата завершения.
// Одно из доп. полей `заявки` или `м-м` рассчитывается (реализовать формулу представленную в лаб-2) при завершении заявки
// (вычисление стоимости заказа, даты доставки в течении месяца, вычисления в м-м).

// ExecuteOrRejectProgramAPI godoc
//
//	@Summary		Завершить программу
//	@Description	Исполняет или отклоняет программу, проставляет в описание программы id принявшего решение ревьюера, вычисляет конечные поля программы. Доступно ревьюеру.
//	@Tags			programs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"id программы"
//	@Param			isAccepted	body		moderatorDecisionReq	true	"Решение ревьюера"
//	@Success		200		{object} moderatedProgramResp
//	@Failure		500		{object}	errorResponse
//	@Router			/api/programs/{id}/moderate [put]
func (h *Handler) ExecuteOrRejectProgramAPI(ctx *gin.Context) {

	idStr := ctx.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	var input moderatorDecisionReq

	if err := ctx.ShouldBindJSON(&input); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}

	moderatorID, err := h.getUserIDFromJWT(ctx)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

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

	res_arr := make([]commandWithOperand, len(commands))

	for i := range len(commands) {
		res_arr[i].Command = commands[i]
		res_arr[i].Operand = operands[i]
	}

	response := programResp{
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

	ctx.JSON(http.StatusOK, moderatedProgramResp{
		Status:      "success",
		Program:     response,
		CmdsWithOps: res_arr,
		Message:     message,
	})
}

// DELETE удаление (дата формирования)

// DeleteProgramAPI godoc
// @Summary Удаляет текущую программу-черновик
// @Description Удаляет программу-черновик текущего пользователя. Заявка определяется автоматически по ID пользователя.
// @Tags programs
// @Security BearerAuth
// @Produce json
// @Success		200		{object} successMessageResp
// @Failure		500		{object}	errorResponse
// @Router /api/programs [delete]
func (h *Handler) DeleteProgramAPI(ctx *gin.Context) {

	userID, err := h.getUserIDFromJWT(ctx)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	programID := h.Repository.GetProgramIDByCreatorID(userID)

	err = h.Repository.DeleteProgram(uint(programID))
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

// DeleteCommandFromProgramAPI godoc
//
//	@Summary		Удалить команду из программы
//	@Description	Удаляет команду из программы-черновика. Доступно авторизованному пользователю.
//	@Tags			commands-programs
//
// @Security BearerAuth
//
//	@Produce		json
//
// @Param command_id query int true "Удаляемая команда"
//
//	@Success		200		{object} successMessageResp
//	@Failure		500		{object}	errorResponse
//	@Router			/api/commands-programs [delete]
func (h *Handler) DeleteCommandFromProgramAPI(ctx *gin.Context) {

	userID, err := h.getUserIDFromJWT(ctx)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	strId := ctx.Query("command_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	programID := h.Repository.GetProgramIDByCreatorID(userID)

	err = h.Repository.DeleteCommandFromProgram(uint(programID), uint(id))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, successMessageResp{
		Status:  "success",
		Message: "удаление команды прошло успешно",
	})
}

// PUT изменение количества/порядка/значения в м-м (`без PK м-м`)

// ModifyCommandOperandAPI godoc
//
//	@Summary		Изменить операнд команды в программе
//	@Description	Изменяет операнд команды в программе-черновике текущего пользователя. Доступно авторизованному пользователю.
//	@Tags			commands-programs
//
// @Security BearerAuth
//
//	@Produce		json
//
// @Param command_id query int true "Команда, чей операнд мы меняем"
// @Param request body operandReq true "Новое значение операнда"
//
//	@Success		200		{object} successMessageResp
//	@Failure		500		{object}	errorResponse
//	@Router			/api/commands-programs [put]
func (h *Handler) ModifyCommandOperandAPI(ctx *gin.Context) {

	userID, err := h.getUserIDFromJWT(ctx)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	strId := ctx.Query("command_id")
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	programID := h.Repository.GetProgramIDByCreatorID(userID)

	var newVal operandReq

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

// RegisterUserAPI godoc
//
//	@Summary		Регистрация пользователя
//	@Description	Создаёт в базе данных нового пользователя с указанными данными, если логин уникальный и данные пользователя соответствуют требованиям
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		registerRequest	true	"Введённые пользователем данные"
//	@Success		200		{object}	successResponse
//	@Failure		500		{object}	errorResponse
//	@Router			/api/users/register [post]
func (h *Handler) RegisterUserAPI(ctx *gin.Context) {

	var req registerRequest
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

// GetUserAPI godoc
// @Summary Получение данных пользователя
// @Description Предоставляет текущему пользователю свои данные. Доступно авторизованным пользователям.
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object{status=string,user=object}
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/users/profile [get]
func (h *Handler) GetUserAPI(ctx *gin.Context) {

	userID, err := h.getUserIDFromJWT(ctx)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

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

// UpdateUserAPI godoc
// @Summary Обновление данных в личном кабинете
// @Description Обновляет данные пользователя. Доступно авторизованным пользователям.
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body userPutReq true "Обновлённые данные"
// @Success 200 {object} successResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/users/profile [put]
func (h *Handler) UpdateUserAPI(ctx *gin.Context) {
	userID, err := h.getUserIDFromJWT(ctx)

	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, err)
		return
	}

	var input userPutReq

	if err := ctx.ShouldBindJSON(&input); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	updates := make(map[string]interface{})
	if input.Login != nil {
		updates["login"] = *input.Login
	}
	if input.Password != nil {
		updates["password"] = generateHashString(*input.Password)
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

// AuthUserAPI godoc
//
//	@Summary		Аутентификация юзера
//	@Description	Выдача зарегистрированному пользователю JWT-токена
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		authRequest	true	"Введённые пользователем данные"
//	@Success		200		{object}	loginResp
//	@Failure		500		{object}	errorResponse
//
// @Failure 401 {object} errorResponse "Неавторизован"
//
//	@Router			/api/users/log-in [post]
func (h *Handler) AuthUserAPI(ctx *gin.Context) {

	var request authRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		h.errorHandler(ctx, http.StatusBadRequest, err)
		return
	}
	user, err := h.Repository.AuthUser(request.Login, generateHashString(request.Password))
	if err != nil {
		h.errorHandler(ctx, http.StatusUnauthorized, err)
		return
	}

	claims := JWTClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(h.JWT.ExpirationTime).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "r-vapp",
		},
		UserID: user.ID,
		Scopes: []string{}, // test data
		Role:   role.Creator,
	}

	if user.IsModerator {
		claims.Role = role.Moderator
	}

	token := jwt.NewWithClaims(h.JWT.SigningMethod, &claims)

	if token == nil {
		h.errorHandler(ctx, http.StatusInternalServerError, fmt.Errorf("пустой токен"))
		return
	}

	strToken, err := token.SignedString([]byte(h.JWT.Token))
	if err != nil {
		h.errorHandler(ctx, http.StatusInternalServerError, fmt.Errorf("не удалось создать строку токена"))
		return
	}

	ctx.JSON(http.StatusOK, loginResp{
		ExpiresIn:   h.JWT.ExpirationTime,
		AccessToken: strToken,
		TokenType:   "Bearer",
	})
}

// POST деавторизация

// DeauthUserAPI godoc
// @Summary Выход пользователя
// @Description Добавляет JWT-токен в черный список.
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} successMessageResp
// @Failure 500 {object} errorResponse
// @Router /api/users/log-out [post]
func (h *Handler) DeauthUserAPI(ctx *gin.Context) {

	tokenString := ctx.GetHeader("Authorization")
	if tokenString == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "деаутентификация выполнена",
		})
		return
	}

	if !strings.HasPrefix(tokenString, jwtPrefix) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "деаутентификация выполнена",
		})
		return
	}

	// Отрезаем префикс
	tokenString = tokenString[len(jwtPrefix):]

	// Парсим токен чтобы получить expiration time
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.JWT.Token), nil
	})

	if err != nil || !token.Valid {
		// Если токен невалиден, все равно считаем выход успешным
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "деаутентификация выполнена",
		})
		return
	}

	// Добавляем токен в черный список
	if h.Repository.RedisClient != nil {
		// Время жизни в черном списке = оставшееся время жизни токена
		remainingTTL := time.Unix(claims.ExpiresAt, 0).Sub(time.Now())
		if remainingTTL > 0 {
			err = h.Repository.RedisClient.WriteJWTToBlacklist(ctx.Request.Context(), tokenString, remainingTTL)
			if err != nil {
				// Логируем ошибку, но все равно возвращаем успех
				logrus.Errorf("ошибка добавления токена в черный список: %v", err)
			}
		}
	}

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
