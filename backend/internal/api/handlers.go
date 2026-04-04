package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"hacknu/backend/internal/health"
	"hacknu/backend/internal/store"
	wshub "hacknu/backend/internal/ws"
	"hacknu/pkg/telemetry"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handlers HTTP + WebSocket.
type Handlers struct {
	log   *slog.Logger
	store *store.Telemetry
	hub   *wshub.Hub
}

// NewHandlers регистрирует обработчики.
func NewHandlers(log *slog.Logger, st *store.Telemetry, hub *wshub.Hub) *Handlers {
	return &Handlers{log: log, store: st, hub: hub}
}

func (h *Handlers) Routes(r chi.Router) {
	r.Get("/healthz", h.handleHealthz)
	r.Post("/api/v1/telemetry", h.handleIngest)
	r.Get("/api/v1/telemetry/latest", h.handleLatest)
	r.Get("/api/v1/telemetry/history", h.handleHistory)
	r.Get("/ws/telemetry", h.handleWS)
}

func (h *Handlers) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handlers) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var s telemetry.Sample
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if err := h.processIngest(r.Context(), &s); err != nil {
		switch {
		case errors.Is(err, errTrainIDRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			h.log.Error("insert", "err", err)
			http.Error(w, "storage error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(s)
}

var errTrainIDRequired = errors.New("train_id required")

// ProcessIngest — то же, что HTTP POST ingest; вызывается из RabbitMQ consumer.
func (h *Handlers) ProcessIngest(ctx context.Context, s *telemetry.Sample) error {
	return h.processIngest(ctx, s)
}

// processIngest нормализует сэмпл, считает health, пишет в БД и шлёт подписчикам /ws/telemetry.
func (h *Handlers) processIngest(ctx context.Context, s *telemetry.Sample) error {
	if s.TrainID == "" {
		return errTrainIDRequired
	}
	if s.TS == "" {
		s.TS = time.Now().UTC().Format(time.RFC3339)
	}
	health.Apply(s)
	if err := h.store.Insert(ctx, s); err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	h.hub.Broadcast(s)
	return nil
}

func (h *Handlers) handleLatest(w http.ResponseWriter, r *http.Request) {
	trainID := r.URL.Query().Get("train_id")
	if trainID == "" {
		http.Error(w, "train_id required", http.StatusBadRequest)
		return
	}
	s, err := h.store.Latest(r.Context(), trainID)
	if err != nil {
		h.log.Error("latest", "err", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	if s == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s)
}

func (h *Handlers) handleHistory(w http.ResponseWriter, r *http.Request) {
	trainID := r.URL.Query().Get("train_id")
	if trainID == "" {
		http.Error(w, "train_id required", http.StatusBadRequest)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := h.store.History(r.Context(), trainID, limit)
	if err != nil {
		h.log.Error("history", "err", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (h *Handlers) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("ws upgrade", "err", err)
		return
	}
	h.hub.Register(c)

	// Держим соединение: читаем сообщения (пинги клиента) до закрытия.
	go func() {
		defer h.hub.Unregister(c)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}()
}
