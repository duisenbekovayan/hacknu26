package rabbitmq

import (
	amqp091 "github.com/rabbitmq/amqp091-go"
)

// Поток: simulator → sample.raw → telemetry.raw → normalizer → sample.normalized → telemetry.normalized → backend → front
const (
	Exchange = "telemetry"

	RawQueue        = "telemetry.raw"
	NormalizedQueue = "telemetry.normalized"
	RawDLQ          = "telemetry.raw.dlq"

	RoutingKeyRaw        = "sample.raw"
	RoutingKeyNormalized = "sample.normalized"
)

// DeclareTelemetryTopology объявляет exchange, очереди raw/normalized/dlq и привязки (идемпотентно).
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
	if err := ch.QueueBind(RawQueue, RoutingKeyRaw, Exchange, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		NormalizedQueue, true, false, false, false, nil,
	); err != nil {
		return err
	}
	if err := ch.QueueBind(NormalizedQueue, RoutingKeyNormalized, Exchange, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		RawDLQ, true, false, false, false, nil,
	); err != nil {
		return err
	}
	return nil
}
