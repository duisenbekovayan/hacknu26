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
	r.Get("/ws/ingest", h.handleWSIngest)
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

// handleWSIngest — поток телеметрии от симуляторов/края: текстовые JSON-кадры по одному Sample.
func (h *Handlers) handleWSIngest(w http.ResponseWriter, r *http.Request) {
	c, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("ws ingest upgrade", "err", err)
		return
	}
	defer func() { _ = c.Close() }()

	for {
		mt, payload, err := c.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				h.log.Warn("ws ingest read", "err", err)
			}
			return
		}
		if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
			continue
		}
		var s telemetry.Sample
		if err := json.Unmarshal(payload, &s); err != nil {
			h.log.Warn("ws ingest json", "err", err)
			_ = c.WriteJSON(map[string]string{"error": "bad json"})
			continue
		}
		// Не используем r.Context(): на нём висит middleware.Timeout на весь HTTP-запрос;
		// WebSocket живёт долго — после дедлайна Insert падал с context deadline exceeded.
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		ingestErr := h.processIngest(ctx, &s)
		cancel()
		if ingestErr != nil {
			h.log.Warn("ws ingest", "err", ingestErr)
			_ = c.WriteJSON(map[string]string{"error": ingestErr.Error()})
			continue
		}
		_ = c.WriteJSON(map[string]any{"ok": true, "health_index": s.HealthIndex, "health_grade": s.HealthGrade})
	}
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
