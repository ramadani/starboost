package consumer

import (
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
)

type HandlerFunc func(topic string, messages <-chan *message.Message)

func NewHandler(logger *zap.Logger) HandlerFunc {
	return func(topic string, messages <-chan *message.Message) {
		for msg := range messages {
			var messageField zap.Field
			var messageContent interface{}

			if err := json.Unmarshal(msg.Payload, &messageContent); err == nil {
				messageField = zap.Any("message", messageContent)
			} else {
				messageField = zap.Any("message", string(msg.Payload))
			}

			logger.Info("received message", zap.String("id", msg.UUID), zap.String("topic", topic), messageField)
			msg.Ack()
		}
	}
}
