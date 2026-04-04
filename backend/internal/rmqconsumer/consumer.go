package rmqconsumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"

	"hacknu/pkg/rabbitmq"
	"hacknu/pkg/telemetry"
)

// Ingester обрабатывает сэмпл так же, как HTTP POST ingest.
type Ingester interface {
	ProcessIngest(ctx context.Context, s *telemetry.Sample) error
}

// Run подключается к RabbitMQ, объявляет топологию и потребляет JSON Sample до отмены ctx.
// При обрыве соединения переподключается с паузой.
func Run(ctx context.Context, log *slog.Logger, url string, ingest Ingester) {
	if url == "" {
		return
	}
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := runSession(ctx, log, url, ingest); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Warn("rabbitmq consumer", "err", err, "retry_in", backoff)
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

func runSession(ctx context.Context, log *slog.Logger, url string, ingest Ingester) error {
	conn, err := amqp091.DialConfig(url, amqp091.Config{Properties: amqp091.Table{"connection_name": "hacknu-backend"}})
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
		rabbitmq.Queue,
		"hacknu-ingest",
		false, // manual ack
		false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Info("rabbitmq consumer subscribed", "queue", rabbitmq.Queue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return errors.New("delivery channel closed")
			}
			var s telemetry.Sample
			if err := json.Unmarshal(d.Body, &s); err != nil {
				log.Warn("rabbitmq bad json", "err", err)
				_ = d.Nack(false, false)
				continue
			}
			ictx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			err := ingest.ProcessIngest(ictx, &s)
			cancel()
			if err != nil {
				log.Warn("rabbitmq ingest", "err", err)
				_ = d.Nack(false, true)
				continue
			}
			if err := d.Ack(false); err != nil {
				return fmt.Errorf("ack: %w", err)
			}
		}
	}
}
