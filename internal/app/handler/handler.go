package handler

import (
	"r-vBackend/cmd/r-vBackend/docs"
	"r-vBackend/internal/app/repository"

	//"r-vBackend/cmd/r-vBackend/docs"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
)

type Handler struct {
	Repository *repository.Repository
}

func NewHandler(r *repository.Repository) *Handler {
	return &Handler{
		Repository: r,
	}
}

// RegisterHandler Функция, в которой мы отдельно регистрируем маршруты, чтобы не писать все в одном месте
func (h *Handler) RegisterHandler(router *gin.Engine) {
	router.GET("/commands", h.GetAllCommands)
	router.GET("/command/:id", h.GetCommandById)
	router.GET("/program/:id", h.GetProgramWithCommands)
	router.POST("/delete-command", h.DeleteCommand)
	router.POST("/add-to-program", h.AddToProgram)
	router.POST("/remove-program/:id", h.DeleteProgram)

	api := router.Group("/api")
	{
		api.GET("/commands", h.GetAllCommandsAPI)
		api.GET("/commands/:id", h.GetCommandByIdAPI)
		api.POST("/commands/add", h.AddCommandAPI)
		api.PUT("/commands/:id", h.ModifyCommandAPI)
		api.DELETE("/commands/:id", h.DeleteCommandAPI)
		api.POST("/commands/:id/add-to-program", h.AddCommandToProgramAPI)
		api.POST("/commands/:id/add-image", h.AddCommandImageAPI)

		api.GET("/programs/cart-icon", h.GetCartCountAPI)
		api.GET("/programs", h.GetProgramsAPI)
		api.GET("/programs/:id", h.GetProgramAPI)
		api.PUT("/programs/:id", h.ModifyProgramFieldsAPI)
		api.PUT("/programs/:id/submit", h.SubmitProgramAPI)
		api.PUT("/programs/:id/moderate", h.ExecuteOrRejectProgramAPI)
		api.DELETE("/program/:id", h.DeleteProgramAPI)

		api.DELETE("/commands-programs", h.DeleteCommandFromProgramAPI)
		api.PUT("/commands-programs", h.ModifyCommandOperandAPI)

		api.POST("/users/register", h.RegisterUserAPI)
		api.GET("/users/profile", h.GetUserAPI)
		api.PUT("/users/profile", h.UpdateUserAPI)
		api.POST("/users/log-in", h.AuthUserAPI)
		api.POST("/users/log-out", h.DeauthUserAPI)
	}

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/doc.json")))
	router.GET("/docs/doc.json", func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Content-Type", "application/json")
		ctx.Writer.WriteHeader(200)
		ctx.Writer.Write([]byte(docs.SwaggerInfo.ReadDoc()))
	})
}

// RegisterStatic То же самое, что и с маршрутами, регистрируем статику
func (h *Handler) RegisterStatic(router *gin.Engine) {
	router.LoadHTMLGlob("../../templates/*")
	router.Static("/static", "../../resources")
}

// errorHandler для более удобного вывода ошибок
func (h *Handler) errorHandler(ctx *gin.Context, errorStatusCode int, err error) {
	logrus.Error(err.Error())
	ctx.JSON(errorStatusCode, gin.H{
		"status":      "error",
		"description": err.Error(),
	})
}
