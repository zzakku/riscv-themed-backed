package handler

import (
	"r-vBackend/cmd/r-vBackend/docs"
	"r-vBackend/internal/app/repository"
	"r-vBackend/internal/app/role"
	"time"

	//"r-vBackend/cmd/r-vBackend/docs"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"

	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
)

type Handler struct {
	Repository *repository.Repository

	JWT JWTConfig
}

type JWTConfig struct {
	Token          string
	ExpirationTime time.Duration
	SigningMethod  jwt.SigningMethod
}

func NewHandler(r *repository.Repository) *Handler {
	expiration, err := time.ParseDuration("24h")
	if err != nil {
		expiration = 24 * time.Hour // если всё плохо
	}

	return &Handler{
		Repository: r,
		JWT: JWTConfig{
			Token:          "test",
			ExpirationTime: expiration,
			SigningMethod:  jwt.SigningMethodHS256,
		},
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

	creator := api.Group("")
	moderator := api.Group("")

	creator.Use(h.WithAuthCheck(role.Creator))
	moderator.Use(h.WithAuthCheck(role.Moderator))
	{
		api.GET("/commands", h.GetAllCommandsAPI)
		api.GET("/commands/:id", h.GetCommandByIdAPI)
		moderator.POST("/commands/add", h.AddCommandAPI)
		moderator.PUT("/commands/:id", h.ModifyCommandAPI)
		moderator.DELETE("/commands/:id", h.DeleteCommandAPI)
		creator.POST("/commands/:id/add-to-program", h.AddCommandToProgramAPI)
		moderator.POST("/commands/:id/add-image", h.AddCommandImageAPI)

		creator.GET("/programs/cart-icon", h.GetCartCountAPI)
		creator.GET("/programs", h.GetProgramsAPI)
		creator.GET("/programs/:id", h.GetProgramAPI)
		creator.PUT("/programs/:id", h.ModifyProgramFieldsAPI)
		creator.PUT("/programs/:id/submit", h.SubmitProgramAPI)
		moderator.PUT("/programs/:id/moderate", h.ExecuteOrRejectProgramAPI)
		creator.DELETE("/program/:id", h.DeleteProgramAPI)

		creator.DELETE("/commands-programs", h.DeleteCommandFromProgramAPI)
		creator.PUT("/commands-programs", h.ModifyCommandOperandAPI)

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
