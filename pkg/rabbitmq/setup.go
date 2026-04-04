package rabbitmq

import (
	amqp091 "github.com/rabbitmq/amqp091-go"
)

// Имена совпадают у симулятора (publisher) и бэкенда (consumer).
const (
	Exchange   = "telemetry"
	Queue      = "telemetry.samples"
	RoutingKey = "sample"
)

// DeclareTelemetryTopology объявляет exchange, очередь и привязку (идемпотентно).
func DeclareTelemetryTopology(ch *amqp091.Channel) error {
	if err := ch.ExchangeDeclare(
		Exchange, "direct", true, false, false, false, nil,
	); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		Queue, true, false, false, false, nil,
	); err != nil {
		return err
	}
	return ch.QueueBind(Queue, RoutingKey, Exchange, false, nil)
}
