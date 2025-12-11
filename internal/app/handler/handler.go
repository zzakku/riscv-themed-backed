package handler

import (
	"fmt"
	"io"
	"net/http"
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

	Async *AsyncConfig
}

type JWTConfig struct {
	Token          string
	ExpirationTime time.Duration
	SigningMethod  jwt.SigningMethod
}

type AsyncConfig struct {
	RiscVServiceURL string
	RiscVAPIKey     string
	BackendURL      string
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
		Async: &AsyncConfig{
			BackendURL:      "http://localhost:8081",
			RiscVAPIKey:     "go-backend-secret-key-123",
			RiscVServiceURL: "http://localhost:8000",
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

	//  ПРОКСИ ДЛЯ MINIO ИЗОБРАЖЕНИЙ ===
	router.GET("/minio/*path", func(c *gin.Context) {
		path := c.Param("path")

		// Формируем URL до MinIO
		minioURL := fmt.Sprintf("http://localhost:9000%s", path)

		// Создаем HTTP клиент
		client := &http.Client{}

		// Создаем запрос к MinIO
		req, err := http.NewRequest("GET", minioURL, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request to MinIO"})
			return
		}

		// Выполняем запрос к MinIO
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to MinIO: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		// Копируем заголовки из MinIO
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// Копируем статус код
		c.Status(resp.StatusCode)

		// Копируем тело ответа
		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			logrus.Errorf("Failed to copy MinIO response: %v", err)
		}
	})

	router.PUT("/api/internal/programs/:id/callback", h.handleRiscVCallback)

	api := router.Group("/api")
	{
		api.GET("/commands", h.GetAllCommandsAPI)
		api.GET("/commands/:id", h.GetCommandByIdAPI)

		api.GET("/programs/cart-icon", h.GetProgramCartCountAPI)

		api.POST("/users/register", h.RegisterUserAPI)

		api.POST("/users/log-in", h.AuthUserAPI)

		either := api.Group("")
		either.Use(h.WithAuthCheck(role.Moderator, role.Creator))
		{
			either.GET("/programs", h.GetProgramsAPI)
			either.GET("/programs/:id", h.GetProgramAPI)
			either.PUT("/programs/:id", h.ModifyProgramFieldsAPI)
			either.PUT("/programs/:id/submit", h.SubmitProgramAPI)

			either.DELETE("/programs", h.DeleteProgramAPI)
			either.POST("/commands/:id/add-to-program", h.AddCommandToProgramAPI)
			either.DELETE("/commands-programs", h.DeleteCommandFromProgramAPI)
			either.PUT("/commands-programs", h.ModifyCommandOperandAPI)

			either.GET("/users/profile", h.GetUserAPI)
			either.PUT("/users/profile", h.UpdateUserAPI)

			either.POST("/users/log-out", h.DeauthUserAPI)
		}

		moderator := api.Group("")
		moderator.Use(h.WithAuthCheck(role.Moderator))
		{
			moderator.POST("/commands/add", h.AddCommandAPI)
			moderator.PUT("/commands/:id", h.ModifyCommandAPI)
			moderator.DELETE("/commands/:id", h.DeleteCommandAPI)
			moderator.POST("/commands/:id/add-image", h.AddCommandImageAPI)

			moderator.PUT("/programs/:id/moderate", h.ExecuteOrRejectProgramAPI)
		}
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
