package dsn

import (
	"fmt"
	"os"
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
