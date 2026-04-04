package rmqnormalizer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"

	"hacknu/backend/internal/normalizer"
	"hacknu/pkg/rabbitmq"
	"hacknu/pkg/telemetry"
)

const defaultConsumerTag = "hacknu-normalizer"

// Run подключается к RabbitMQ, потребляет raw-телеметрию и публикует normalized.
// При обрыве соединения переподключается с экспоненциальной паузой.
func Run(ctx context.Context, log *slog.Logger, url, consumerTag string) {
	if url == "" {
		return
	}
	if consumerTag == "" {
		consumerTag = defaultConsumerTag
	}

	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := runSession(ctx, log, url, consumerTag); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Warn("normalizer rabbitmq consumer", "err", err, "retry_in", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
			}
			continue
		}
		backoff = time.Second
	}
}

func runSession(ctx context.Context, log *slog.Logger, url, consumerTag string) error {
	conn, err := amqp091.DialConfig(url, amqp091.Config{
		Properties: amqp091.Table{"connection_name": "hacknu-normalizer"},
	})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	if err := rabbitmq.DeclareTelemetryTopology(ch); err != nil {
		return fmt.Errorf("topology: %w", err)
	}
	if err := ch.Qos(32, 0, false); err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	msgs, err := ch.Consume(
		rabbitmq.RawQueue,
		consumerTag,
		false, // manual ack
		false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Info("normalizer subscribed", "queue", rabbitmq.RawQueue, "consumer_tag", consumerTag)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return errors.New("delivery channel closed")
			}
			if err := processDelivery(log, ch, d); err != nil {
				return err
			}
		}
	}
}

func processDelivery(log *slog.Logger, ch *amqp091.Channel, d amqp091.Delivery) error {
	var in telemetry.Sample
	if err := json.Unmarshal(d.Body, &in); err != nil {
		log.Warn("normalizer bad json", "err", err)
		return rejectToDLQ(log, ch, d, "bad_json")
	}

	out, err := normalizer.NormalizeSample(&in)
	if err != nil {
		log.Warn("normalization failed", "err", err, "train_id", in.TrainID)
		return rejectToDLQ(log, ch, d, "normalization_failed")
	}

	body, err := json.Marshal(out)
	if err != nil {
		log.Warn("normalizer marshal failed", "err", err, "train_id", out.TrainID)
		return rejectToDLQ(log, ch, d, "marshal_failed")
	}

	if err := rabbitmq.PublishNormalizedSample(ch, body); err != nil {
		log.Warn("publish normalized sample failed", "err", err)
		if nackErr := d.Nack(false, true); nackErr != nil {
			return fmt.Errorf("nack requeue: %w", nackErr)
		}
		return nil
	}

	if err := d.Ack(false); err != nil {
		return fmt.Errorf("ack raw: %w", err)
	}
	log.Info("published normalized sample", "train_id", out.TrainID, "ts", out.TS)
	return nil
}

func rejectToDLQ(log *slog.Logger, ch *amqp091.Channel, d amqp091.Delivery, reason string) error {
	if err := rabbitmq.PublishToDLQ(ch, d.Body, reason); err != nil {
		log.Error("send to dlq failed, dropping message", "err", err, "reason", reason)
		if nackErr := d.Nack(false, false); nackErr != nil {
			return fmt.Errorf("nack drop: %w", nackErr)
		}
		return nil
	}

	if err := d.Ack(false); err != nil {
		return fmt.Errorf("ack after dlq: %w", err)
	}
	log.Warn("sent to dlq", "queue", rabbitmq.RawDLQ, "reason", reason)
	return nil
}
