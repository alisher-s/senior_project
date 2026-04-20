package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/nu/student-event-ticketing-platform/internal/app"
	"github.com/nu/student-event-ticketing-platform/internal/config"
	"github.com/nu/student-event-ticketing-platform/internal/infra/db"
	"github.com/nu/student-event-ticketing-platform/internal/infra/observability"
	infraRedis "github.com/nu/student-event-ticketing-platform/internal/infra/redis"
	"github.com/nu/student-event-ticketing-platform/internal/infra/storage"
)

// @title Student Event Ticketing Platform API
// @version 0.1
// @description Modular monolith backend (Go + Chi). HTTP handlers live in domain packages (`auth/handler`, `events/handler`, `ticketing/handler`, …) under `internal/app` at base path **`/api/v1`** (отдельного пакета `internal/api/v1` в репозитории нет).
// @description **Ошибки:** тело `{ "error": { "code": "string", "message": "string" } }`. Типичные HTTP-коды: **401 Unauthorized** — нет/невалидный JWT (`missing_authorization`, `invalid_token`, `invalid_credentials`, …); **403 Forbidden** — RBAC (`forbidden`, `organizer_request_forbidden`, …); **409 Conflict** — билеты и бизнес-правила (`already_registered`, `capacity_full`, `event_not_approved`, …). Полный список `error.code` — в README.
// @description Даты событий: **RFC3339**, например `2026-01-01T10:00:00Z`. Ответ `POST /tickets/register`: поля **`qr_hash_hex`** и **`qr_png_base64`**.
// @BasePath /api/v1
// @schemes http
// @host localhost:8080

func main() {
	ctx := context.Background()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		panic(err)
	}

	logger := observability.NewLogger(cfg.AppEnv)

	dbPool, err := db.Connect(ctx, cfg)
	if err != nil {
		logger.Error("postgres_connect_failed", "error", err)
		panic(err)
	}

	rdb, err := infraRedis.Connect(ctx, cfg)
	if err != nil {
		logger.Error("redis_connect_failed", "error", err)
		dbPool.Close()
		panic(err)
	}

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	storageSvc, err := connectMinIOWithRetry(ctx, logger)
	if err != nil {
		logger.Error("minio_connect_failed", "error", err)
		dbPool.Close()
		_ = rdb.Close()
		panic(err)
	}

	srv := &http.Server{
		Addr:         cfg.Server.Address,
		Handler:      app.NewRouter(cfg, dbPool, rdb, logger, workerCtx, storageSvc),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http_server_start", "addr", cfg.Server.Address)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	// Wait for shutdown (SIGINT / SIGTERM) or fatal listen error.
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		logger.Info("shutdown_signal", "signal", sig.String())
	case err := <-errCh:
		logger.Error("server_listen_error", "error", err)
	}

	// Stop accepting new connections; drain in-flight requests.
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server_shutdown_failed", "error", err)
	}

	workerCancel()

	if err := storageSvc.Close(); err != nil {
		logger.Error("minio_close_failed", "error", err)
	}

	if err := rdb.Close(); err != nil {
		logger.Error("redis_close_failed", "error", err)
	}
	dbPool.Close()
}

// Compile-time checks for expected imports.
var _ *pgxpool.Pool
var _ *redis.Client

func connectMinIOWithRetry(ctx context.Context, logger interface {
	Error(msg string, args ...any)
}) (storage.Service, error) {
	// Docker Compose starts containers quickly but services may not be ready yet.
	// MinIO can take a moment to start accepting connections (formatting / init).
	deadlineCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	backoff := 500 * time.Millisecond
	for {
		svc, err := storage.NewMinIO(deadlineCtx)
		if err == nil {
			return svc, nil
		}

		// One-line, rate-limited-ish logs to avoid spam.
		logger.Error("minio_connect_retry", "error", err)

		if deadlineCtx.Err() != nil {
			return nil, err
		}

		time.Sleep(backoff)
		if backoff < 5*time.Second {
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
	}
}
