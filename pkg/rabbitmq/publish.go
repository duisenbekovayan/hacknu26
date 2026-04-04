package rabbitmq

import amqp091 "github.com/rabbitmq/amqp091-go"

// PublishRawSample публикует JSON-тело в поток сырых сэмплов.
func PublishRawSample(ch *amqp091.Channel, body []byte) error {
	return ch.Publish(
		Exchange,
		RawRoutingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
		},
	)
}

// PublishNormalizedSample публикует JSON-тело в поток нормализованных сэмплов.
func PublishNormalizedSample(ch *amqp091.Channel, body []byte) error {
	return ch.Publish(
		Exchange,
		NormalizedRoutingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
		},
	)
}

// PublishToDLQ публикует сообщение в raw DLQ с причиной в headers.
func PublishToDLQ(ch *amqp091.Channel, body []byte, reason string) error {
	headers := amqp091.Table{}
	if reason != "" {
		headers["reason"] = reason
	}
	return ch.Publish(
		"",
		RawDLQ,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Headers:      headers,
			Body:         body,
		},
	)
}
