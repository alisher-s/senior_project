package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	authHandler "github.com/nu/student-event-ticketing-platform/auth/handler"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	"github.com/nu/student-event-ticketing-platform/internal/config"
	"github.com/nu/student-event-ticketing-platform/internal/infra/rate_limit"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	eventsHandler "github.com/nu/student-event-ticketing-platform/events/handler"
	ticketingHandler "github.com/nu/student-event-ticketing-platform/ticketing/handler"
	paymentsHandler "github.com/nu/student-event-ticketing-platform/payments/handler"
	notificationsHandler "github.com/nu/student-event-ticketing-platform/notifications/handler"
	notificationsRepo "github.com/nu/student-event-ticketing-platform/notifications/repository"
	notificationsService "github.com/nu/student-event-ticketing-platform/notifications/service"
	notificationsSender "github.com/nu/student-event-ticketing-platform/internal/notifications/sender"
	adminHandler "github.com/nu/student-event-ticketing-platform/admin/handler"
	analyticsHandler "github.com/nu/student-event-ticketing-platform/analytics/handler"
	"github.com/nu/student-event-ticketing-platform/internal/infra/storage"

	_ "github.com/nu/student-event-ticketing-platform/docs"
)

type Deps struct {
	Cfg    config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Logger *slog.Logger
}

func NewRouter(cfg config.Config, db *pgxpool.Pool, rdb *redis.Client, logger *slog.Logger, workerCtx context.Context, storageSvc storage.Service) http.Handler {
	// Chi chosen for a lightweight router with composable middleware and clean routing patterns.
	r := chi.NewRouter()

	// Standard middleware chain for production readiness.
	r.Use(middleware.RequestID)
	r.Use(httpx.CORS())
	r.Use(middleware.RealIP)
	r.Use(httpx.SecurityHeaders())
	r.Use(httpx.Logging(logger))
	r.Use(httpx.ErrorHandler(logger))
	r.Use(httpx.Recovery(logger))
	r.Use(httpx.RequestTimeout(25 * time.Second))

	// Redis-based rate limiting.
	r.Use(rate_limit.Middleware(rdb, cfg))

	// Notifications worker bootstrap (DB-backed queue); workerCtx is cancelled during API shutdown.
	notificationsQueueRepo := notificationsRepo.NewPostgres(db)
	var emailSender notificationsService.Sender = notificationsSender.NewGmailSender()
	notificationsWorker := notificationsService.NewEmailWorker(logger, notificationsQueueRepo, emailSender, 20, 2*time.Second)
	go notificationsWorker.Start(workerCtx)

	r.Route("/api/v1", func(r chi.Router) {
		swaggerDocURL := fmt.Sprintf("http://localhost%s/api/v1/swagger/doc.json", cfg.Server.Address)
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL(swaggerDocURL),
		))

		r.Get("/healthz", healthzHandler)

		// Local dev static assets (e.g., event posters). Store URLs in events.cover_image_url.
		// Example: http://localhost:8080/api/v1/static/posters/img1.jpg
		if _, err := os.Stat("static"); err == nil {
			fs := http.FileServer(http.Dir("static"))
			r.Handle("/static/*", http.StripPrefix("/api/v1/static/", fs))
		}

		jwt := authx.NewJWT(cfg)

		// Auth module (fully implemented).
		authHandler.RegisterRoutes(r, authHandler.Deps{
			Cfg:    cfg,
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
		})

		// Events CRUD.
		eventsHandler.RegisterRoutes(r, eventsHandler.Deps{DB: db, JWT: jwt, Storage: storageSvc})

		// Ticketing registration (capacity-safe + QR generation).
		ticketingHandler.RegisterRoutes(r, ticketingHandler.Deps{
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
		})

		// Payments (stub for foundation).
		paymentsHandler.RegisterRoutes(r, paymentsHandler.Deps{
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
			Cfg:    cfg,
		})

		// Notifications (stub + async worker foundation).
		notificationsHandler.RegisterRoutes(r, notificationsHandler.Deps{
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
		})

		// Admin (stub).
		adminHandler.RegisterRoutes(r, adminHandler.Deps{
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
		})

		// Analytics (stub).
		analyticsHandler.RegisterRoutes(r, analyticsHandler.Deps{
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
		})
	})

	return r
}

type HealthzResponse struct {
	Status string `json:"status"`
}

// @Summary Health check
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} HealthzResponse
// @Router /healthz [get]
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	_ = httpx.WriteJSON(w, http.StatusOK, HealthzResponse{
		Status: "ok",
	})
}

