package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"avito-internship-2023/docs"
	"avito-internship-2023/internal/pkg/error_middleware"
	"avito-internship-2023/internal/pkg/postgres"
	"avito-internship-2023/internal/pkg/server"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// @title Avito Internship Service: User Segments
// @version 1.0

func main() {
	err := godotenv.Load("deploy_trajectory/.env")
	if err != nil {
		fmt.Print(err)
		return
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig = encoderConfig

	baseLogger, err := logConfig.Build()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer func() {
		if err = baseLogger.Sync(); err != nil {
			log.Fatalf("can't flush log entities: %v", err)
		}
	}()

	logger := baseLogger.Sugar()

	validate := validator.New()

	dbContext, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	dbConnectionString := fmt.Sprintf("postgres://%s:%s@db:5432/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))

	postgresDB, cancelDB, err := postgres.NewDatabase(dbConnectionString)
	if err != nil {
		logger.Fatal("cannot open postgres connection")
	}
	defer cancelDB(postgresDB)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(error_middleware.New(logger))

	docs.SwaggerInfo.BasePath = "/"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// TODO: add repositories, service, workers, etc.

	errorChannel := make(chan error, 2)

	addr := ":" + os.Getenv("SERVICE_PORT")
	serv := server.NewServer(logger, router, addr)
	go func() {
		errorChannel <- serv.Start()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	signal.Notify(sigChan, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("Received terminate, graceful shutdown. Signal:", sig)
	case err = <-errorChannel:
		logger.Error("Critical error, trying to make graceful shutdown. Error: ", err)
	}

	tc, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()

	err = errors.Join(serv.Shutdown(tc))
	if err != nil {
		logger.Error(err)
	}
}
