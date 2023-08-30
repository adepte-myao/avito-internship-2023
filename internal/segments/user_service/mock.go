package user_service

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"avito-internship-2023/internal/segments"
	"avito-internship-2023/internal/segments/segment_postgres"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type userProvider interface {
	GetRandom(ctx context.Context) (segments.User, error)
}

type Mock struct {
	ctx          context.Context
	userProvider userProvider // This dependency is required only for Mock, usual UserServiceProvider shouldn't have it.
	writer       *kafka.Writer
}

func NewMock(ctx context.Context, userProvider userProvider, writer *kafka.Writer) *Mock {
	return &Mock{
		ctx:          ctx,
		userProvider: userProvider,
		writer:       writer,
	}
}

// StartProducing works as http.ListenAndServe: blocks calling routine and can return only non-nil error
func (service *Mock) StartProducing(mockMaxProducePeriod int) error {
	for {
		toSleepInSeconds := rand.Intn(mockMaxProducePeriod + 1)
		time.Sleep(time.Second * time.Duration(toSleepInSeconds))

		err := service.ProduceEvent()
		if err != nil {
			return err
		}
	}
}

func (service *Mock) GetStatus(userID string) (segments.UserStatus, error) {
	usActionProb := rand.Float64()
	if usActionProb < 0.25 {
		// User tries to remove his account or is banned or there is a reason we shouldn't treat him as usual
		return segments.Excluded, nil
	}
	if usActionProb < 0.3 {
		// User was removed from user service
		return "", segments.ErrUserNotFound
	}
	// User was created or his status was returned to normal
	return segments.Active, nil
}

func (service *Mock) ProduceEvent() error {
	usActionProb := rand.Float64()

	var (
		user segments.User
		err  error
	)
	if usActionProb < 0.8 {
		// For operations with existing user
		user, err = service.userProvider.GetRandom(service.ctx)
	} else {
		// For operations with non-existing user
		user = segments.User{Id: uuid.New().String()}
	}
	if errors.Is(err, segment_postgres.ErrNoUsersToPick) {
		user = segments.User{Id: uuid.New().String()}
	} else if err != nil {
		return err
	}

	dto := segments.UserActionDTO{UserID: user.Id}
	message, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(service.ctx, 5*time.Second)
	defer cancel()

	err = service.writer.WriteMessages(writeCtx, kafka.Message{Value: message})
	if err != nil {
		return err
	}

	return nil
}
