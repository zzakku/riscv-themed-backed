package repository

import (
	"github.com/minio/minio-go"
	"github.com/minio/minio-go/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Repository struct {
	db                *gorm.DB
	minio             *minio.Client
	minio_bucket_name string
}

func New(dsn string, minio_endpoint string, minio_access_key string, minio_secret_key string, minio_bucket_name string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{}) // подключаемся к БД
	if err != nil {
		return nil, err
	}

	useSSL := false // Хардкод

	minioClient, err := minio.NewWithOptions(minio_endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minio_access_key, minio_secret_key, ""),
		Secure: useSSL,
	})

	if err != nil {
		return nil, err
	}

	// Возвращаем объект Repository с подключенной базой данных
	return &Repository{
		db:                db,
		minio:             minioClient,
		minio_bucket_name: minio_bucket_name,
	}, nil
}
