package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments/core/ports"

	"github.com/segmentio/kafka-go"
)

type UserActionConsumer struct {
	logger    common.Logger
	ctx       context.Context
	reader    *kafka.Reader
	processor ports.SegmentsService
}

func NewUserActionConsumer(logger common.Logger, ctx context.Context, reader *kafka.Reader, processor ports.SegmentsService) *UserActionConsumer {
	return &UserActionConsumer{logger: logger, ctx: ctx, reader: reader, processor: processor}
}

// StartConsuming works as http.ListenAndServe: blocks calling routine and can return only non-nil error
func (consumer *UserActionConsumer) StartConsuming() error {
	consumer.logger.Info("starting UserActionConsumer")

	for {
		fetchCtx, cancelFetch := context.WithTimeout(consumer.ctx, 5*time.Second)

		msg, err := consumer.reader.FetchMessage(fetchCtx)
		if err != nil {
			cancelFetch()
			if !errors.Is(err, context.DeadlineExceeded) {
				consumer.logger.Error(err)
			}
			continue
		}

		cancelFetch()

		var dto userActionDTO
		if err = json.Unmarshal(msg.Value, &dto); err != nil {
			// No sensitive data is sent, so message output can include some useful info
			consumer.logger.Errorw(err.Error(),
				"key", string(msg.Key),
				"value", string(msg.Value),
				"offset", msg.Offset)

			continue
		}

		consumer.processor.ProcessUserAction(dto.UserID)

		commitCtx, cancelCommit := context.WithTimeout(consumer.ctx, 5*time.Second)

		if err = consumer.reader.CommitMessages(commitCtx, msg); err != nil {
			cancelCommit()
			consumer.logger.Error(err)
			continue
		}

		cancelCommit()
	}
}
