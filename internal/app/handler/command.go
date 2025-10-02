package handler

import (
	"net/http"
	"strconv"
	"strings"

	"r-vBackend/internal/app/ds"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (h *Handler) GetAllCommands(ctx *gin.Context) {
	var commands []ds.Command
	var err error

	search := ctx.Query("searchQuery")
	if search == "" {
		commands, err = h.Repository.GetAllCommands()
	} else {
		commands, err = h.Repository.SearchCommandsByName(search)
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"commands":    commands,
		"cart_count":  h.Repository.GetCartCount(),
		"searchQuery": search,
		"programID":   h.Repository.GetProgramIDByCreatorID(uint(1)), // пока что хардкод
	})
}

func (h *Handler) GetCommandById(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	command, err := h.Repository.GetCommandByID(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	ctx.HTML(http.StatusOK, "details.html", command)
}

func (h *Handler) DeleteCommand(ctx *gin.Context) {
	// считываем значение из формы, которую мы добавим в наш шаблон
	strId := ctx.PostForm("command_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	// Вызов функции добавления чата в заявку
	err = h.Repository.DeleteCommand(uint(id))
	if err != nil && !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return
	}

	// после вызова сразу произойдет обновление страницы
	ctx.Redirect(http.StatusFound, "/commands")
}

func (h *Handler) GetProgramById(ctx *gin.Context) {
	strId := ctx.Param("id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	command, err := h.Repository.GetProgramByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	ctx.HTML(http.StatusOK, "details.html", command)
}

func (h *Handler) AddToProgram(ctx *gin.Context) {
	// считываем значение из формы, которую мы добавим в наш шаблон
	strId := ctx.PostForm("command_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	// Вызов функции добавления чата в заявку
	err = h.Repository.AddToProgram(uint(id))
	if err != nil && !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
		return
	}

	// после вызова сразу произойдет обновление страницы
	ctx.Redirect(http.StatusFound, "/commands")
}

func (h *Handler) GetProgramWithCommands(ctx *gin.Context) {
	strId := ctx.Param("id")
	prgId, err := strconv.Atoi(strId)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	program, err := h.Repository.GetProgramByID(uint(prgId))

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	// если обращение к репозиторию удалось, но БД не нашла искомую программу - делаем редирект на главную страницу
	// то же нужно для удалённых программ
	if program == nil || program.Status == "удалена" {
		ctx.Redirect(http.StatusFound, "/commands")
		return
	}

	commands, err := h.Repository.GetCommandsByProgram(uint(prgId))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	operands, err := h.Repository.GetOperandsByProgram(uint(prgId))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		logrus.Error(err)
		return
	}

	ctx.HTML(http.StatusFound, "program.html", gin.H{
		"commands": commands,
		"program":  program,
		"operands": operands,
	})
}

func (h *Handler) DeleteProgram(ctx *gin.Context) {

	strId := ctx.PostForm("program_id")
	id, err := strconv.Atoi(strId)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	err = h.Repository.DeleteProgram(uint(id))
	if err != nil {
		return
	}

	// возвращаемся на главную страницу
	ctx.Redirect(http.StatusFound, "/commands")
}
