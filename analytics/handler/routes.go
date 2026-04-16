package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	"github.com/nu/student-event-ticketing-platform/analytics/service"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
)

type Deps struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	JWT    authx.JWT
	Logger *slog.Logger
}

func RegisterRoutes(r chi.Router, deps Deps) {
	_ = deps.DB
	_ = deps.Redis

	h := &handler{svc: service.New(), v: validator.New()}

	r.With(authx.AuthMiddleware(deps.JWT)).Route("/analytics", func(r chi.Router) {
		r.Get("/events/stats", h.handleEventStats)
	})
}

type handler struct {
	svc *service.Service
	v   *validator.Validate
}

// @Summary Event statistics (stub)
// @Description Requires a valid JWT; any authenticated role may call. Optional query event_id.
// @Tags analytics
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param event_id query string false "Filter by event UUID"
// @Success 200 {object} EventStatsResponseDTO
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 501 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /analytics/events/stats [get]
func (h *handler) handleEventStats(w http.ResponseWriter, r *http.Request) {
	eventIDParam := strings.TrimSpace(r.URL.Query().Get("event_id"))
	var eventID *string
	if eventIDParam != "" {
		eventID = &eventIDParam
	}

	stats, err := h.svc.EventStats(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, service.ErrNotImplemented) {
			_ = httpx.WriteJSON(w, http.StatusNotImplemented, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_implemented", Message: "analytics not implemented yet"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "analytics failed"},
		})
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, stats)
}
