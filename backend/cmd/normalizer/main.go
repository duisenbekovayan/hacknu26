package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"hacknu/backend/internal/rmqnormalizer"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	rmqURL := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if rmqURL == "" {
		rmqURL = "amqp://hacknu:hacknu@127.0.0.1:5672/"
	}
	consumerTag := strings.TrimSpace(os.Getenv("NORMALIZER_CONSUMER_TAG"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		rmqnormalizer.Run(ctx, log, rmqURL, consumerTag)
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		log.Warn("normalizer shutdown timeout")
	}
	log.Info("shutdown complete")
}
