package server

import (
	"context"
	

	"github.com/segmentio/kafka-go"
)

func SendMessageToKafka(brokers []string, topic, key, value string) error {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	defer writer.Close()

	msg := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
	}
	return writer.WriteMessages(context.Background(), msg)
}


