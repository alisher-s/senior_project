package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	"github.com/nu/student-event-ticketing-platform/analytics/repository"
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
	repo := repository.NewPostgres(deps.DB)
	svc := service.New(repo, deps.Redis)

	h := &handler{svc: svc, v: validator.New(), log: deps.Logger}

	r.With(
		authx.AuthMiddleware(deps.JWT),
		authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin),
	).Route("/analytics", func(r chi.Router) {
		r.Get("/events/stats", h.handleEventStats)
	})
}

type handler struct {
	svc *service.Service
	v   *validator.Validate
	log *slog.Logger
}

// @Summary Event statistics
// @Description Registration and capacity metrics for an event or aggregated for the caller. **Requires organizer or admin** (students get **403 forbidden**). Organizers see only their events; admins see any event or global aggregates.
// @Tags analytics
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (organizer or admin)"
// @Param event_id query string false "Filter by event UUID; omit to aggregate events in scope"
// @Success 200 {object} EventStatsResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /analytics/events/stats [get]
func (h *handler) handleEventStats(w http.ResponseWriter, r *http.Request) {
	callerID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}
	if _, ok := authx.RoleFromContext(r.Context()); !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing role")
		return
	}
	isAdmin := authx.HasRole(r.Context(), authx.RoleAdmin)

	eventIDParam := strings.TrimSpace(r.URL.Query().Get("event_id"))
	var eventID *uuid.UUID
	if eventIDParam != "" {
		id, err := uuid.Parse(eventIDParam)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid event_id")
			return
		}
		eventID = &id
	}

	stats, err := h.svc.EventStats(r.Context(), callerID, isAdmin, eventID)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		if status < 500 {
			// Keep a clearer message for the client on this endpoint.
			if apiErr.Code == httpx.ErrCodeForbidden {
				httpx.WriteError(w, http.StatusForbidden, apiErr.Code, "not allowed to view stats for this event")
				return
			}
			httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
			return
		}
		if h.log != nil {
			h.log.Error("analytics: event stats failed", "err", err)
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "analytics failed")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, stats)
}
