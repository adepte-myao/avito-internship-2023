package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"avito-internship-2023/docs"
	"avito-internship-2023/internal/pkg/error_middleware"
	"avito-internship-2023/internal/pkg/postgres"
	"avito-internship-2023/internal/pkg/server"
	"avito-internship-2023/internal/segments"
	"avito-internship-2023/internal/segments/segment_dropbox"
	"avito-internship-2023/internal/segments/segment_handlers"
	"avito-internship-2023/internal/segments/segment_postgres"
	"avito-internship-2023/internal/segments/user_handlers"
	"avito-internship-2023/internal/segments/user_kafka_consumers"
	"avito-internship-2023/internal/segments/user_service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// @title Avito Internship Service: User Segments
// @version 1.0

const (
	deadlineCheckPeriodInSeconds = 30
	mockProducePeriodInSeconds   = 1
)

func main() {
	err := godotenv.Load("deploy_user_segmenting/.env")
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

	errorChannel := make(chan error, 5)

	segmentsRouter := router.Group("/segments")
	{
		segmentsLogger := logger.With("domain", "segments")

		historyRepo := segment_postgres.NewUserSegmentHistoryRepository(
			segmentsLogger.With("caller_type", "UserSegmentHistoryRepository"), postgresDB)
		userRepo := segment_postgres.NewUserRepository(
			segmentsLogger.With("caller_type", "UserRepository"), postgresDB, historyRepo)
		segmentRepo := segment_postgres.NewSegmentRepository(
			segmentsLogger.With("caller_type", "SegmentRepository"), postgresDB, historyRepo)
		deadlineRepo := segment_postgres.NewUserSegmentDeadlineRepository(
			segmentsLogger.With("caller_type", "UserSegmentDeadlineRepository"), postgresDB, historyRepo)

		kafkaWriter := &kafka.Writer{
			Addr:                   kafka.TCP("kafka:9092"),
			Topic:                  "UserAction",
			RequiredAcks:           kafka.RequireOne,
			AllowAutoTopicCreation: true,
		}
		userService := user_service.NewMock(context.Background(), userRepo, kafkaWriter)
		go func() {
			for {
				toSleepInSeconds := mockProducePeriodInSeconds + rand.Intn(4) - 2
				time.Sleep(time.Second * time.Duration(toSleepInSeconds))

				err := userService.ProduceEvent()
				if err != nil {
					errorChannel <- err
					return
				}
			}
		}()

		deadlineWorker := segments.NewDeadlineWorker(
			segmentsLogger.With("caller_type", "DeadlineWorker"), dbContext, deadlineRepo, segmentRepo)
		go func() {
			for {
				time.Sleep(time.Second * deadlineCheckPeriodInSeconds)

				err := deadlineWorker.RemoveExceededUserSegments()
				if err != nil {
					errorChannel <- err
					return
				}
			}
		}()

		dropboxService := segment_dropbox.NewService(
			context.Background(), segmentsLogger.With("caller_type", "DropboxService"), os.Getenv("DROPBOX_TOKEN"))

		service := segments.NewService(
			segmentsLogger.With("caller_type", "Service"), dbContext, userService, userRepo,
			segmentRepo, historyRepo, deadlineRepo, dropboxService)

		changeForUserHandler := segment_handlers.NewChangeForUserHandler(service, validate)
		segmentsRouter.POST("/change-for-user", changeForUserHandler.Handle)

		createHandler := segment_handlers.NewCreateHandler(service, validate)
		segmentsRouter.POST("/create", createHandler.Handle)

		getForUserHandler := segment_handlers.NewGetForUserHandler(service, validate)
		segmentsRouter.GET("/get-for-user", getForUserHandler.Handle)

		getHistoryLinkHandler := segment_handlers.NewGetHistoryReportLinkHandler(service, validate)
		segmentsRouter.GET("/get-history-report-link", getHistoryLinkHandler.Handle)

		removeHandler := segment_handlers.NewRemoveHandler(service, validate)
		segmentsRouter.DELETE("/remove", removeHandler.Handle)

		createUserHandler := user_handlers.NewCreateHandler(service, validate)
		segmentsRouter.POST("/create-user", createUserHandler.Handle)

		removeUserHandler := user_handlers.NewRemoveHandler(service, validate)
		segmentsRouter.DELETE("/remove-user", removeUserHandler.Handle)

		updateUserHandler := user_handlers.NewUpdateHandler(service, validate)
		segmentsRouter.PUT("/update-user", updateUserHandler.Handle)

		kafkaReader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{"kafka:9092"},
			GroupID:     "segments",
			Topic:       "UserAction",
			Dialer:      nil,
			StartOffset: 0,
		})
		userActionConsumer := user_kafka_consumers.NewUserActionConsumer(segmentsLogger, context.Background(), kafkaReader, service)
		go func() {
			err := userActionConsumer.StartConsuming()
			errorChannel <- err
		}()
	}

	addr := ":" + os.Getenv("SERVICE_PORT")
	serv := server.NewServer(logger, router, addr)
	go func() {
		err := serv.Start()
		errorChannel <- err
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

	tc, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	err = errors.Join(serv.Shutdown(tc))
	if err != nil {
		logger.Error(err)
	}
}
