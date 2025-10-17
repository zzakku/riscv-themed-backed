package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

const jwtPrefix = "jwt."

func getJWTKey(token string) string {
	return servicePrefix + jwtPrefix + token
}

// WriteJWTToBlacklist добавляет JWT токен в черный список
func (c *Client) WriteJWTToBlacklist(ctx context.Context, jwtStr string, jwtTTL time.Duration) error {
	return c.client.Set(ctx, getJWTKey(jwtStr), "blacklisted", jwtTTL).Err()
}

// RemoveJWTFromBlacklist убирает JWT-токен из черного списка
func (c *Client) RemoveJWTFromBlacklist(ctx context.Context, jwtStr string) error {
	return c.client.Del(ctx, getJWTKey(jwtStr)).Err()
}

// Возвращает nil если токен в черном списке, ошибку если нет
func (c *Client) CheckJWTInBlacklist(ctx context.Context, jwtStr string) (bool, error) {
	_, err := c.client.Get(ctx, getJWTKey(jwtStr)).Result()
	if err != nil {
		if err == redis.Nil {
			// Токена нет в черном списке
			return false, nil
		}
		// Другая ошибка Redis
		return false, err
	}
	// Токен найден в черном списке
	return true, nil
}
