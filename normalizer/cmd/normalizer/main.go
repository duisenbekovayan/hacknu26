package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hacknu/normalizer/internal/config"
	"hacknu/normalizer/internal/consumer"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := config.Load()
	log.Info(
		"normalizer config",
		"smoothing", cfg.EnableSmoothing,
		"dedup", cfg.EnableDedup,
		"dedup_window", cfg.DedupWindow,
		"state_ttl", cfg.StateTTL,
		"buffer_size", cfg.BufferSize,
		"ema_alpha", cfg.EMAAlpha,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		consumer.Run(ctx, log, cfg)
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
