package main

import (
	"fmt"
	"r-vBackend/internal/app/config"
	"r-vBackend/internal/app/dsn"
	"r-vBackend/internal/app/handler"
	"r-vBackend/internal/app/repository"
	"r-vBackend/internal/pkg"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @title RVBACK
// @version 1.0
// @description Risc-V-themed Web-Service

// @contact.name API Support
// @contact.url https://github.com/zzakku
// @contact.email nuhuh@lol.com

// @license.name AS IS (NO WARRANTY)

// @host 127.0.0.1
// @schemes https http
// @BasePath /

func main() {
	router := gin.Default()
	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	postgresString := dsn.PostgresFromEnv()
	fmt.Println(postgresString)

	rep, errRep := repository.New(postgresString, dsn.MinioEndpointFromEnv(), dsn.MinioAccessKeyFromEnv(),
		dsn.MinioSecretKeyFromEnv(), dsn.MinioBucketNameFromEnv())
	if errRep != nil {
		logrus.Fatalf("error initializing repository: %v", errRep)
	}

	hand := handler.NewHandler(rep)

	application := pkg.NewApp(conf, router, hand)
	application.RunApp()
}
