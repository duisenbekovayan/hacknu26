package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"

	"hacknu/normalizer/internal/config"
	"hacknu/normalizer/internal/service"
	"hacknu/pkg/rabbitmq"
	"hacknu/pkg/telemetry"
)

// Run поднимает consumer raw-сэмплов и publisher normalized-сэмплов.
func Run(ctx context.Context, log *slog.Logger, cfg config.Config) {
	if cfg.RabbitMQURL == "" {
		return
	}

	store := service.NewStore(cfg.StateTTL, cfg.BufferSize)
	processor := service.NewProcessor(service.Options{
		EnableSmoothing: cfg.EnableSmoothing,
		EnableDedup:     cfg.EnableDedup,
		DedupWindow:     cfg.DedupWindow,
		EMAAlpha:        cfg.EMAAlpha,
		State:           store,
	})
	go store.RunCleanup(ctx, log)

	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := runSession(ctx, log, cfg, processor); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Warn("normalizer consumer session", "err", err, "retry_in", backoff)
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

func runSession(ctx context.Context, log *slog.Logger, cfg config.Config, processor *service.Processor) error {
	conn, err := amqp091.DialConfig(cfg.RabbitMQURL, amqp091.Config{
		Properties: amqp091.Table{"connection_name": "normalizer-service"},
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
		cfg.ConsumerTag,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Info("normalizer subscribed", "queue", rabbitmq.RawQueue, "consumer_tag", cfg.ConsumerTag)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return errors.New("delivery channel closed")
			}
			if err := handleDelivery(log, ch, d, processor); err != nil {
				return err
			}
		}
	}
}

func handleDelivery(log *slog.Logger, ch *amqp091.Channel, d amqp091.Delivery, processor *service.Processor) error {
	var in telemetry.Sample
	if err := json.Unmarshal(d.Body, &in); err != nil {
		log.Warn("normalizer bad json", "err", err)
		return handleInvalid(log, ch, d, "bad_json")
	}

	result, err := processor.Prepare(&in)
	if err != nil {
		log.Warn("normalization failed", "err", err, "train_id", in.TrainID)
		return handleInvalid(log, ch, d, "normalization_failed")
	}

	if result.Decision == service.DecisionSkipDuplicate {
		if err := d.Ack(false); err != nil {
			return fmt.Errorf("ack duplicate: %w", err)
		}
		if result.Commit != nil {
			result.Commit()
		}
		log.Info("dedup skipped sample", "train_id", result.TrainID)
		return nil
	}

	body, err := json.Marshal(result.Sample)
	if err != nil {
		log.Warn("marshal normalized failed", "err", err, "train_id", result.TrainID)
		return handleInvalid(log, ch, d, "marshal_failed")
	}

	if err := rabbitmq.PublishNormalizedSample(ch, body); err != nil {
		log.Warn("publish normalized sample failed", "err", err, "train_id", result.TrainID)
		if nackErr := d.Nack(false, true); nackErr != nil {
			return fmt.Errorf("nack requeue: %w", nackErr)
		}
		return nil
	}

	if err := d.Ack(false); err != nil {
		return fmt.Errorf("ack raw: %w", err)
	}
	if result.Commit != nil {
		result.Commit()
	}
	log.Info("published normalized sample", "train_id", result.TrainID, "queue", rabbitmq.NormalizedQueue)
	return nil
}

func handleInvalid(log *slog.Logger, ch *amqp091.Channel, d amqp091.Delivery, reason string) error {
	if err := rabbitmq.PublishToDLQ(ch, d.Body, reason); err != nil {
		log.Error("send to dlq failed", "err", err, "reason", reason)
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
