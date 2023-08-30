package user_kafka_consumers

import (
	"context"
	"encoding/json"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments/segments_core/segments_ports"

	"github.com/segmentio/kafka-go"
)

type UserActionConsumer struct {
	logger    common.Logger
	ctx       context.Context
	reader    *kafka.Reader
	processor segments_ports.SegmentsService
}

func NewUserActionConsumer(logger common.Logger, ctx context.Context, reader *kafka.Reader, processor segments_ports.SegmentsService) *UserActionConsumer {
	return &UserActionConsumer{logger: logger, ctx: ctx, reader: reader, processor: processor}
}

// StartConsuming works as http.ListenAndServe: blocks calling routine and can return only non-nil error
func (consumer *UserActionConsumer) StartConsuming() error {
	consumer.logger.Info("starting UserActionConsumer")

	for {
		msg, err := consumer.reader.FetchMessage(consumer.ctx)
		if err != nil {
			consumer.logger.Error(err)
			continue
		}

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

		if err = consumer.reader.CommitMessages(consumer.ctx, msg); err != nil {
			consumer.logger.Error(err)
			continue
		}
	}
}
