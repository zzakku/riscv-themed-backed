package dsn

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func PostgresFromEnv() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		return ""
	}
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	dbname := os.Getenv("DB_NAME")
	// И вот мы возвращаем dsn, который необходим для подключения к БД
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, dbname)
}

func MinioEndpointFromEnv() string {
	endpoint := os.Getenv("MINIO_ENDPOINT")

	return endpoint
}

func MinioAccessKeyFromEnv() string {
	access_key := os.Getenv("MINIO_ACCESS_KEY")

	return access_key
}

func MinioSecretKeyFromEnv() string {
	secret_key := os.Getenv("MINIO_SECRET_KEY")

	return secret_key
}

func MinioBucketNameFromEnv() string {
	bucket_name := os.Getenv("MINIO_BUCKET_NAME")

	return bucket_name
}

func RedisHostFromEnv() string {
	redis_host := os.Getenv("REDIS_HOST")

	return redis_host
}

func RedisPortFromEnv() (int, error) {
	redis_port := os.Getenv("REDIS_PORT")

	parsed_port, err := strconv.Atoi(redis_port)
	if err != nil {
		return 0, err
	}

	return parsed_port, nil
}

func RedisPasswordFromEnv() string {
	redis_password := os.Getenv("REDIS_PASSWORD")

	return redis_password
}

func RedisUserFromEnv() string {
	redis_user := os.Getenv("REDIS_USER")

	return redis_user
}

func RedisDialTimeoutFromEnv() (time.Duration, error) {
	redis_dial_timeout := os.Getenv("REDIS_DIAL_TIMEOUT")

	parsed_timeout, err := time.ParseDuration(redis_dial_timeout)

	if err != nil {
		return 0, err
	}

	return parsed_timeout, err
}

func RedisReadTimeoutFromEnv() (time.Duration, error) {
	redis_read_timeout := os.Getenv("REDIS_READ_TIMEOUT")

	parsed_timeout, err := time.ParseDuration(redis_read_timeout)

	if err != nil {
		return 0, err
	}

	return parsed_timeout, err
}
