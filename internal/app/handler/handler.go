package handler

import (
	"net/http"
	"r-vBackend/internal/app/repository"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

func (h *Handler) GetCommands(ctx *gin.Context) {
	var commands []repository.Command
	var err error

	searchQuery := ctx.Query("searchQuery") // получаем значение из поля поиска
	if searchQuery == "" {                  // если поле поиска пусто, то просто получаем из репозитория все записи
		commands, err = h.Repository.GetCommands()
		if err != nil {
			logrus.Error(err)
		}
	} else {
		commands, err = h.Repository.GetCommandsByName(searchQuery) // в ином случае ищем заказ по заголовку
		if err != nil {
			logrus.Error(err)
		}
	}

	ctx.HTML(http.StatusOK, "index.html", gin.H{
		"commands":    commands,
		"searchQuery": searchQuery, // передаем введенный запрос обратно на страницу
		// в ином случае оно будет очищаться при нажатии на кнопку
	})
}

func (h *Handler) GetCommand(ctx *gin.Context) {
	idStr := ctx.Param("id") // получаем id заказа из урла (то есть из /command/:id)
	// через двоеточие мы указываем параметры, которые потом сможем считать через функцию выше
	id, err := strconv.Atoi(idStr) // так как функция выше возвращает нам строку, нужно ее преобразовать в int
	if err != nil {
		logrus.Error(err)
	}

	command, err := h.Repository.GetCommand(id)
	if err != nil {
		logrus.Error(err)
	}

	ctx.HTML(http.StatusOK, "details.html", gin.H{
		"command": command,
	})
}

func (h *Handler) GetProgram(ctx *gin.Context) {
	var programs []repository.Program
	var err error

	programs, err = h.Repository.GetPrograms()
	if err != nil {
		logrus.Error(err)
	}

	ctx.HTML(http.StatusOK, "program.html", gin.H{
		"programs": programs,
	})
}
