package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	"github.com/nu/student-event-ticketing-platform/admin/service"
	authrepo "github.com/nu/student-event-ticketing-platform/auth/repository"
	eventsrepo "github.com/nu/student-event-ticketing-platform/events/repository"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	notificationsrepo "github.com/nu/student-event-ticketing-platform/notifications/repository"
	notificationssvc "github.com/nu/student-event-ticketing-platform/notifications/service"
)

type Deps struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	JWT    authx.JWT
	Logger *slog.Logger
}

func RegisterRoutes(r chi.Router, deps Deps) {
	_ = deps.Redis

	userRepo := authrepo.NewPostgres(deps.DB)
	eventRepo := eventsrepo.NewPostgres(deps.DB)
	notifier := notificationssvc.New(notificationsrepo.NewPostgres(deps.DB))
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	svc := service.New(userRepo, eventRepo, notifier)
	h := &handler{svc: svc, v: validator.New(), logger: log}

	r.Route("/admin", func(r chi.Router) {
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleAdmin)).Post("/events/{id}/moderate", h.handleModerateEvent)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleAdmin)).Patch("/users/{id}/role", h.handleSetUserRole)
	})
}

type handler struct {
	svc    *service.Service
	v      *validator.Validate
	logger *slog.Logger
}

type ModerateEventRequestDTO struct {
	Action string `json:"action" validate:"required,oneof=approve reject"`
	Reason string `json:"reason" validate:"omitempty,max=2000"`
}

type ModerateEventResponseDTO struct {
	ModerationStatus string `json:"moderation_status"`
}

type SetUserRoleRequestDTO struct {
	Role string `json:"role" validate:"required,oneof=student organizer admin"`
}

type UserRoleResponseDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// @Summary Set user role (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (admin)"
// @Param id path string true "User ID (UUID)"
// @Param request body SetUserRoleRequestDTO true "New role"
// @Success 200 {object} UserRoleResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/users/{id}/role [patch]
func (h *handler) handleSetUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	userID, err := uuid.Parse(idStr)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "invalid user id"},
		})
		return
	}

	var req SetUserRoleRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	u, err := h.svc.SetUserRole(r.Context(), userID, req.Role)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRole) {
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_role", Message: "role must be student, organizer, or admin"},
			})
			return
		}
		if errors.Is(err, authrepo.ErrUserNotFound) {
			_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_found", Message: "user not found"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to update role"},
		})
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, UserRoleResponseDTO{
		ID:    u.ID.String(),
		Email: u.Email,
		Role:  string(u.Role),
	})
}

// @Summary Moderate event visibility (admin only)
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (admin)"
// @Param id path string true "Event ID (UUID)"
// @Param request body ModerateEventRequestDTO true "approve or reject"
// @Success 200 {object} ModerateEventResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/events/{id}/moderate [post]
func (h *handler) handleModerateEvent(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(chi.URLParam(r, "id"))
	if eventID == "" {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "missing event id"},
		})
		return
	}

	adminID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	var req ModerateEventRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	st, err := h.svc.ModerateEvent(r.Context(), adminID, eventID, req.Action, req.Reason, h.logger)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEventID):
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_id", Message: "invalid event id"},
			})
		case errors.Is(err, service.ErrInvalidAction):
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_action", Message: "action must be approve or reject"},
			})
		case errors.Is(err, service.ErrEventNotFound):
			_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_found", Message: "event not found"},
			})
		default:
			_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "internal_error", Message: "moderation failed"},
			})
		}
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, ModerateEventResponseDTO{
		ModerationStatus: string(st),
	})
}
