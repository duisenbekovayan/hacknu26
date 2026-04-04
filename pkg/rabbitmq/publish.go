package rabbitmq

import amqp091 "github.com/rabbitmq/amqp091-go"

// PublishSample публикует JSON-тело в очередь телеметрии.
func PublishSample(ch *amqp091.Channel, body []byte) error {
	return ch.Publish(
		Exchange,
		RoutingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Body:         body,
		},
	)
}
