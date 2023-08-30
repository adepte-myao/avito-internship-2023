package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"avito-internship-2023/docs"
	"avito-internship-2023/internal/pkg/error_middleware"
	"avito-internship-2023/internal/pkg/postgres"
	"avito-internship-2023/internal/pkg/server"
	"avito-internship-2023/internal/pkg/zap_wrapper"
	"avito-internship-2023/internal/segments/segments_consumers/user_kafka_consumers"
	"avito-internship-2023/internal/segments/segments_core/segments_services"
	"avito-internship-2023/internal/segments/segments_handlers/segment_handlers"
	"avito-internship-2023/internal/segments/segments_handlers/user_handlers"
	"avito-internship-2023/internal/segments/segments_integrations/segments_dropbox"
	"avito-internship-2023/internal/segments/segments_integrations/segments_user_service"
	"avito-internship-2023/internal/segments/segments_repositories/segments_postgres"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Avito Internship Service: User Segments
// @version 1.0

func main() {
	err := godotenv.Load("deploy_user_segmenting/.env")
	if err != nil {
		log.Fatal(err)
	}

	logger, closeLoggerFunc, err := zap_wrapper.NewDevSugaredLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = closeLoggerFunc(); err != nil {
			log.Println(err)
		}
	}()

	validate := validator.New()

	appContext, cancelAppContext := context.WithCancel(context.Background())
	defer cancelAppContext()

	dbConnectionString := fmt.Sprintf("postgres://%s:%s@db:5432/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))

	postgresDB, cancelDB, err := postgres.NewDatabase(dbConnectionString)
	if err != nil {
		logger.Fatal("cannot open postgres connection: ", err)
	}
	defer cancelDB(postgresDB)

	segmentsLogger := logger.With("segments_domain", "segments")

	historyRepo := segments_postgres.NewUserSegmentHistoryRepository(
		segmentsLogger.With("caller_type", "UserSegmentHistoryRepository"), postgresDB)
	userRepo := segments_postgres.NewUserRepository(
		segmentsLogger.With("caller_type", "UserRepository"), postgresDB, historyRepo)
	segmentRepo := segments_postgres.NewSegmentRepository(
		segmentsLogger.With("caller_type", "SegmentRepository"), postgresDB, historyRepo)
	deadlineRepo := segments_postgres.NewUserSegmentDeadlineRepository(
		segmentsLogger.With("caller_type", "UserSegmentDeadlineRepository"), postgresDB, historyRepo)

	// Buffer must be big enough to hold all possible errors: it prevents goroutine leak.
	errorChannel := make(chan error, 50)

	userServiceCtx, cancelUserServiceCtx := context.WithCancel(appContext)
	defer cancelUserServiceCtx()

	kafkaUserActionWriter := &kafka.Writer{
		Addr:                   kafka.TCP(os.Getenv("KAFKA_BROKER_ADDR")),
		Topic:                  os.Getenv("KAFKA_USER_ACTION_TOPIC_NAME"),
		RequiredAcks:           kafka.RequireOne,
		AllowAutoTopicCreation: true,
	}

	mockMaxProducePeriodVal := os.Getenv("USER_SERVICE_MOCK_MAX_PRODUCE_PERIOD_IN_SECONDS")
	mockMaxProducePeriod, err := strconv.Atoi(mockMaxProducePeriodVal)
	if err != nil || mockMaxProducePeriod < 1 {
		logger.Info("value USER_SERVICE_MOCK_MAX_PRODUCE_PERIOD_IN_SECONDS is missed or incorrect, setting default value = 5")
		mockMaxProducePeriod = 5
	}

	userService := segments_user_service.NewMock(userServiceCtx, userRepo, kafkaUserActionWriter)
	go func() {
		err := userService.StartProducing(mockMaxProducePeriod)
		errorChannel <- err
	}()

	deadlineWorkerCtx, cancelDeadlineWorkerCtx := context.WithCancel(appContext)
	defer cancelDeadlineWorkerCtx()

	deadlineCheckPeriodVal := os.Getenv("DEADLINE_CHECK_PERIOD_IN_SECONDS")
	deadlineCheckPeriod, err := strconv.Atoi(deadlineCheckPeriodVal)
	if err != nil || deadlineCheckPeriod < 1 {
		logger.Info("value DEADLINE_CHECK_PERIOD_IN_SECONDS is missed or incorrect, setting default value = 30")
		deadlineCheckPeriod = 30
	}

	deadlineWorker := segments_services.NewDeadlineWorker(segmentsLogger.With("caller_type", "DeadlineWorker"), deadlineWorkerCtx, deadlineRepo, segmentRepo)
	go func() {
		err := deadlineWorker.Start(deadlineCheckPeriod)
		errorChannel <- err
	}()

	dropboxCtx, cancelDropboxCtx := context.WithCancel(appContext)
	defer cancelDropboxCtx()

	dropboxService := segments_dropbox.NewService(
		dropboxCtx, segmentsLogger.With("caller_type", "DropboxService"), os.Getenv("DROPBOX_TOKEN"))

	serviceCtx, cancelServiceCtx := context.WithCancel(appContext)
	defer cancelServiceCtx()

	service := segments_services.NewService(
		segmentsLogger.With("caller_type", "Service"), serviceCtx, userService, userRepo,
		segmentRepo, historyRepo, deadlineRepo, dropboxService)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(error_middleware.New(logger))

	docs.SwaggerInfo.BasePath = "/"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	segmentsRouter := router.Group("/segments")
	segmentsHandler := segment_handlers.NewSegmentHandler(service, validate)
	userHandler := user_handlers.NewUserHandler(service, validate)

	segmentsRouter.POST("/change-for-user", segmentsHandler.ChangeForUser)
	segmentsRouter.POST("/create", segmentsHandler.Create)
	segmentsRouter.GET("/get-for-user", segmentsHandler.GetForUser)
	segmentsRouter.GET("/get-history-report-link", segmentsHandler.GetHistoryReportLink)
	segmentsRouter.DELETE("/remove", segmentsHandler.Remove)

	segmentsRouter.POST("/create-user", userHandler.Create)
	segmentsRouter.DELETE("/remove-user", userHandler.Remove)
	segmentsRouter.PUT("/update-user", userHandler.Update)

	userActionConsumerCtx, cancelUserActionConsumerCtx := context.WithCancel(appContext)
	defer cancelUserActionConsumerCtx()

	kafkaReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{os.Getenv("KAFKA_BROKER_ADDR")},
		GroupID:     os.Getenv("KAFKA_CONSUMER_GROUP_ID"),
		Topic:       os.Getenv("KAFKA_USER_ACTION_TOPIC_NAME"),
		Dialer:      nil,
		StartOffset: 0,
	})
	userActionConsumer := user_kafka_consumers.NewUserActionConsumer(segmentsLogger, userActionConsumerCtx, kafkaReader, service)
	go func() {
		err := userActionConsumer.StartConsuming()
		errorChannel <- err

		if err = kafkaReader.Close(); err != nil {
			logger.Error("error when closing kafka reader: ", err)
		}
	}()

	addr := ":" + os.Getenv("SERVICE_PORT")
	serv := server.NewServer(logger, router, addr)
	go func() {
		err := serv.Start()
		errorChannel <- err
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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

	cancelAppContext()
	select {
	case <-tc.Done():
		return
	case <-appContext.Done():
		return
	}
}
