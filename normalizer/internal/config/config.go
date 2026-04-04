package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultRabbitMQURL     = "amqp://hacknu:hacknu@127.0.0.1:5672/"
	defaultEnableSmoothing = true
	defaultEnableDedup     = true
	defaultDedupWindowMS   = 1500
	defaultStateTTLMinutes = 15
	defaultBufferSize      = 5
	defaultEMAAlpha        = 0.4
	defaultNormalizerTag   = "normalizer-service"
)

// Config описывает runtime-настройки normalizer-service.
type Config struct {
	RabbitMQURL     string
	EnableSmoothing bool
	EnableDedup     bool
	DedupWindow     time.Duration
	StateTTL        time.Duration
	BufferSize      int
	EMAAlpha        float64
	ConsumerTag     string
}

func Load() Config {
	cfg := Config{
		RabbitMQURL:     envString("RABBITMQ_URL", defaultRabbitMQURL),
		EnableSmoothing: envBool("NORMALIZER_ENABLE_SMOOTHING", defaultEnableSmoothing),
		EnableDedup:     envBool("NORMALIZER_ENABLE_DEDUP", defaultEnableDedup),
		DedupWindow:     time.Duration(envInt("NORMALIZER_DEDUP_WINDOW_MS", defaultDedupWindowMS)) * time.Millisecond,
		StateTTL:        time.Duration(envInt("NORMALIZER_STATE_TTL_MIN", defaultStateTTLMinutes)) * time.Minute,
		BufferSize:      envInt("NORMALIZER_BUFFER_SIZE", defaultBufferSize),
		EMAAlpha:        envFloat("NORMALIZER_EMA_ALPHA", defaultEMAAlpha),
		ConsumerTag:     defaultNormalizerTag,
	}

	if cfg.DedupWindow <= 0 {
		cfg.DedupWindow = time.Duration(defaultDedupWindowMS) * time.Millisecond
	}
	if cfg.StateTTL <= 0 {
		cfg.StateTTL = time.Duration(defaultStateTTLMinutes) * time.Minute
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = defaultBufferSize
	}
	if cfg.EMAAlpha <= 0 || cfg.EMAAlpha > 1 {
		cfg.EMAAlpha = defaultEMAAlpha
	}

	return cfg
}

func envString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envFloat(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}
