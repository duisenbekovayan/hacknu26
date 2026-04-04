package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"hacknu/backend/internal/api"
	"hacknu/backend/internal/db"
	"hacknu/backend/internal/envfile"
	"hacknu/backend/internal/llm"
	"hacknu/backend/internal/rmqconsumer"
	"hacknu/backend/internal/store"
	"hacknu/backend/internal/ws"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	envfile.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx)
	if err != nil {
		log.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(context.Background(), pool); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}

	hub := ws.NewHub(log)
	st := store.NewTelemetry(pool)
	aiSvc := llm.NewService(log)
	h := api.NewHandlers(log, st, hub, aiSvc)

	rmqURL := strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	if os.Getenv("RABBITMQ_DISABLE") == "1" {
		rmqURL = ""
	} else if rmqURL == "" {
		rmqURL = "amqp://hacknu:hacknu@127.0.0.1:5672/"
	}
	rmqCtx, rmqCancel := context.WithCancel(context.Background())
	defer rmqCancel()
	if rmqURL != "" {
		go rmqconsumer.Run(rmqCtx, log, rmqURL, h)
	} else {
		log.Info("rabbitmq consumer disabled")
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(corsAll)

	// WebSocket нельзя оборачивать Timeout: после Upgrade контекст отменяется и ломает поток.
	r.Get("/ws/telemetry", h.WSTelemetry)

	r.Group(func(r chi.Router) {
		r.Use(noStore)
		r.Use(middleware.Timeout(60 * time.Second))
		h.Routes(r)
	})

	frontendDir := os.Getenv("FRONTEND_DIR")
	if frontendDir == "" {
		frontendDir = "frontend"
	}
	if _, err := os.Stat(filepath.Join(frontendDir, "index.html")); err != nil {
		log.Error("frontend: укажите каталог с index.html (FRONTEND_DIR или ./frontend от cwd)", "dir", frontendDir, "err", err)
		os.Exit(1)
	}
	log.Info("frontend static", "dir", frontendDir)
	fs := http.FileServer(http.Dir(frontendDir))
	r.Handle("/static/*", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		fs.ServeHTTP(w, req)
	})))
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		b, err := os.ReadFile(filepath.Join(frontendDir, "index.html"))
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, must-revalidate")
		_, _ = w.Write(b)
	})

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http", "err", err)
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	rmqCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("shutdown complete")
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func corsAll(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
