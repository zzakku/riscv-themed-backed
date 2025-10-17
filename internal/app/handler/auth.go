package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	//	"r-vBackend/internal/app/ds"
	"r-vBackend/internal/app/role"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt"
)

type JWTClaims struct {
	jwt.StandardClaims          // все что точно необходимо по RFC
	UserID             uint     `json:"user_id"` // наши данные - uuid этого пользователя в базе данных
	Scopes             []string `json:"scopes"`  // список доступов в нашей системе
	Role               role.Role
}

const jwtPrefix = "Bearer "

func (h *Handler) WithAuthCheck(assignedRoles ...role.Role) func(ctx *gin.Context) {
	return func(gCtx *gin.Context) {
		jwtStr := gCtx.GetHeader("Authorization")
		if !strings.HasPrefix(jwtStr, jwtPrefix) { // если нет префикса то нас дурят!
			gCtx.AbortWithStatus(http.StatusForbidden) // отдаем что нет доступа

			return // завершаем обработку
		}

		// отрезаем префикс
		jwtStr = jwtStr[len(jwtPrefix):]

		// проверяем jwt в блеклист редиса
		_, err := h.Repository.RedisClient.CheckJWTInBlacklist(gCtx.Request.Context(), jwtStr)
		if err == nil { // значит что токен в блеклисте
			gCtx.AbortWithStatus(http.StatusForbidden)

			return
		}
		if !errors.Is(err, redis.Nil) { // значит что это не ошибка отсуствия - внутренняя ошибка
			gCtx.AbortWithError(http.StatusInternalServerError, err)

			return
		}

		token, err := jwt.ParseWithClaims(jwtStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.JWT.Token), nil
		})
		if err != nil {
			gCtx.AbortWithStatus(http.StatusForbidden)
			log.Println(err)

			return
		}

		myClaims := token.Claims.(*JWTClaims)

		roleAllowed := false
		for _, oneOfAssignedRole := range assignedRoles {
			if myClaims.Role == oneOfAssignedRole {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			gCtx.AbortWithStatus(http.StatusForbidden)
			log.Printf("role %s is not assigned in %s", myClaims.Role, assignedRoles)

			return
		}
	}

}

func (h *Handler) getUserIDFromJWT(ctx *gin.Context) (uint, error) {
	jwtStr := ctx.GetHeader("Authorization")
	if !strings.HasPrefix(jwtStr, jwtPrefix) {
		ctx.AbortWithStatus(http.StatusForbidden)

		return 0, fmt.Errorf("jwt не имеет нужный префикс")
	}

	// отрезаем префикс
	jwtStr = jwtStr[len(jwtPrefix):]

	token, err := jwt.ParseWithClaims(jwtStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.JWT.Token), nil
	})
	if err != nil {
		ctx.AbortWithStatus(http.StatusForbidden)
		log.Println(err)

		return 0, err // не удалось распарсить JWT
	}

	myClaims := token.Claims.(*JWTClaims)

	return myClaims.UserID, nil
}
