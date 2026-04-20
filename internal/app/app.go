package app

import (
	"context"
	"fmt"
	"net/http"
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
	ticketingRepo "github.com/nu/student-event-ticketing-platform/ticketing/repository"
	paymentsHandler "github.com/nu/student-event-ticketing-platform/payments/handler"
	notificationsHandler "github.com/nu/student-event-ticketing-platform/notifications/handler"
	notificationsRepo "github.com/nu/student-event-ticketing-platform/notifications/repository"
	notificationsService "github.com/nu/student-event-ticketing-platform/notifications/service"
	adminHandler "github.com/nu/student-event-ticketing-platform/admin/handler"
	analyticsHandler "github.com/nu/student-event-ticketing-platform/analytics/handler"

	_ "github.com/nu/student-event-ticketing-platform/docs"
)

type Deps struct {
	Cfg    config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Logger *slog.Logger
}

func NewRouter(cfg config.Config, db *pgxpool.Pool, rdb *redis.Client, logger *slog.Logger, workerCtx context.Context) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(httpx.CORS())
	r.Use(middleware.RealIP)
	r.Use(httpx.SecurityHeaders())
	r.Use(httpx.Logging(logger))
	r.Use(httpx.Recovery(logger))
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(rate_limit.Middleware(rdb, cfg))

	// ── Notifications infrastructure ─────────────────────────────────────────
	notificationsQueueRepo := notificationsRepo.NewPostgres(db)
	notificationsSvc := notificationsService.New(notificationsQueueRepo)

	var emailSender notificationsService.Sender
	if cfg.SMTP.Host != "" {
		emailSender = notificationsService.NewSMTPSender(cfg)
	} else {
		emailSender = notificationsService.NoopSender{}
	}

	var fcmSender *notificationsService.FCMSender
	var deviceTokenRepo notificationsRepo.DeviceTokenRepository
	if cfg.Firebase.ServerKey != "" {
		fcmSender = notificationsService.NewFCMSender(cfg.Firebase.ServerKey)
		deviceTokenRepo = notificationsRepo.NewDeviceTokenPostgres(db)
	}

	notificationsWorker := notificationsService.NewEmailWorker(
		logger,
		notificationsQueueRepo,
		emailSender,
		fcmSender,
		deviceTokenRepo,
		20,
		2*time.Second,
	)
	go notificationsWorker.Start(workerCtx)

	// ── Shared repositories ───────────────────────────────────────────────────
	// ticketRepo is shared between ticketing and events modules (events needs it
	// to fan-out cancellation/reschedule notifications to ticket holders).
	sharedTicketRepo := ticketingRepo.NewPostgres(db)

	r.Route("/api/v1", func(r chi.Router) {
		swaggerDocURL := fmt.Sprintf("http://localhost%s/api/v1/swagger/doc.json", cfg.Server.Address)
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL(swaggerDocURL),
		))

		r.Get("/healthz", healthzHandler)

		jwt := authx.NewJWT(cfg)

		authHandler.RegisterRoutes(r, authHandler.Deps{
			Cfg:    cfg,
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
		})

		eventsHandler.RegisterRoutes(r, eventsHandler.Deps{
			DB:         db,
			JWT:        jwt,
			NotifSvc:   notificationsSvc,
			TicketRepo: sharedTicketRepo,
		})

		ticketingHandler.RegisterRoutes(r, ticketingHandler.Deps{
			DB:       db,
			Redis:    rdb,
			JWT:      jwt,
			Logger:   logger,
			NotifSvc: notificationsSvc,
		})

		paymentsHandler.RegisterRoutes(r, paymentsHandler.Deps{
			DB:       db,
			Redis:    rdb,
			JWT:      jwt,
			Logger:   logger,
			Cfg:      cfg,
			NotifSvc: notificationsSvc,
		})

		notificationsHandler.RegisterRoutes(r, notificationsHandler.Deps{
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
		})

		adminHandler.RegisterRoutes(r, adminHandler.Deps{
			DB:     db,
			Redis:  rdb,
			JWT:    jwt,
			Logger: logger,
		})

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
