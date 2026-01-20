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

type RepositorySettings struct {
	PostgresDSN     string
	MinioEndpoint   string
	MinioAccessKey  string
	MinioSecretKey  string
	MinioBucketName string
}

func New(settings *RepositorySettings) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(settings.PostgresDSN), &gorm.Config{}) // подключаемся к БД
	if err != nil {
		return nil, err
	}

	useSSL := false // при true подключаемся к MinIO по HTTPS

	minioClient, err := minio.NewWithOptions(settings.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(settings.MinioAccessKey, settings.MinioSecretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		return nil, err
	}

	// Возвращаем указатель на получившийся Repository
	return &Repository{
		db:                db,
		minio:             minioClient,
		minio_bucket_name: settings.MinioBucketName,
	}, nil
}
