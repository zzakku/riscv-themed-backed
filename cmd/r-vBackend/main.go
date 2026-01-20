package main

import (
	"fmt"
	"os"
	"r-vBackend/internal/app/config"
	"r-vBackend/internal/app/dsn"
	"r-vBackend/internal/app/handler"
	"r-vBackend/internal/app/repository"
	"r-vBackend/internal/pkg"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	router := gin.Default()
	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	postgresString := dsn.PostgresFromEnv()
	fmt.Println(postgresString)

	rep, errRep := repository.New(&repository.RepositorySettings{
		PostgresDSN:     postgresString,
		MinioEndpoint:   os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:  os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:  os.Getenv("MINIO_SECRET_KEY"),
		MinioBucketName: os.Getenv("MINIO_BUCKET_NAME"),
	})
	if errRep != nil {
		logrus.Fatalf("error initializing repository: %v", errRep)
	}

	hand := handler.NewHandler(rep)

	application := pkg.NewApp(conf, router, hand)
	application.RunApp()
}
