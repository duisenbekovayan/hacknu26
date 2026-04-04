package rabbitmq

import (
	amqp091 "github.com/rabbitmq/amqp091-go"
)

// Имена совпадают у симулятора (publisher) и бэкенда (consumer).
const (
	Exchange             = "telemetry"
	RawQueue             = "telemetry.raw"
	NormalizedQueue      = "telemetry.normalized"
	RawDLQ               = "telemetry.raw.dlq"
	RawRoutingKey        = "raw"
	NormalizedRoutingKey = "normalized"
)

// DeclareTelemetryTopology объявляет exchange, очередь и привязку (идемпотентно).
func DeclareTelemetryTopology(ch *amqp091.Channel) error {
	if err := ch.ExchangeDeclare(
		Exchange, "direct", true, false, false, false, nil,
	); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		RawQueue, true, false, false, false, nil,
	); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		NormalizedQueue, true, false, false, false, nil,
	); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		RawDLQ, true, false, false, false, nil,
	); err != nil {
		return err
	}
	if err := ch.QueueBind(RawQueue, RawRoutingKey, Exchange, false, nil); err != nil {
		return err
	}
	return ch.QueueBind(NormalizedQueue, NormalizedRoutingKey, Exchange, false, nil)
}
