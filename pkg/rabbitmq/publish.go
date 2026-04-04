package rabbitmq

import amqp091 "github.com/rabbitmq/amqp091-go"

// PublishRawSample публикует JSON в очередь raw (симулятор).
func PublishRawSample(ch *amqp091.Channel, body []byte) error {
	return ch.Publish(
		Exchange,
		RoutingKeyRaw,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
		},
	)
}

// PublishNormalizedSample публикует JSON в очередь normalized (после normalizer).
func PublishNormalizedSample(ch *amqp091.Channel, body []byte) error {
	return ch.Publish(
		Exchange,
		RoutingKeyNormalized,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
		},
	)
}

// PublishToDLQ кладёт исходное тело в DLQ с причиной в заголовке.
func PublishToDLQ(ch *amqp091.Channel, body []byte, reason string) error {
	return ch.Publish(
		"",
		RawDLQ,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
			Headers:      amqp091.Table{"x-dlq-reason": reason},
		},
	)
}
