package user_service

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"avito-internship-2023/internal/segments/core/domain"
	"avito-internship-2023/internal/segments/core/ports"
	"avito-internship-2023/internal/segments/repositories/postgres"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Mock struct {
	ctx          context.Context
	userProvider ports.UserProvider // This dependency is required only for Mock, usual UserServiceProvider shouldn't have it.
	writer       *kafka.Writer
}

func NewMock(ctx context.Context, userProvider ports.UserProvider, writer *kafka.Writer) *Mock {
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

func (service *Mock) GetStatus(userID string) (domain.UserStatus, error) {
	usActionProb := rand.Float64()
	if usActionProb < 0.25 {
		// User tries to remove his account or is banned or there is a reason we shouldn't treat him as usual
		return domain.Excluded, nil
	}
	if usActionProb < 0.3 {
		// User was removed from user service
		return "", domain.ErrUserNotFound
	}
	// User was created or his status was returned to normal
	return domain.Active, nil
}

func (service *Mock) ProduceEvent() error {
	execCtx, cancelExec := context.WithTimeout(service.ctx, 5*time.Second)
	defer cancelExec()

	usActionProb := rand.Float64()

	var (
		user domain.User
		err  error
	)
	if usActionProb < 0.8 {
		// For operations with existing user
		user, err = service.userProvider.GetRandom(execCtx)
	} else {
		// For operations with non-existing user
		user = domain.User{Id: uuid.New().String()}
	}
	if errors.Is(err, postgres.ErrNoUsersToPick) {
		user = domain.User{Id: uuid.New().String()}
	} else if err != nil {
		return err
	}

	dto := userActionDTO{UserID: user.Id}
	message, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	err = service.writer.WriteMessages(execCtx, kafka.Message{Value: message})
	if err != nil {
		return err
	}

	return nil
}
