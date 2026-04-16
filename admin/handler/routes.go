package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	"github.com/nu/student-event-ticketing-platform/admin/service"
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

	r.Route("/admin", func(r chi.Router) {
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleAdmin)).Post("/events/{id}/moderate", h.handleModerateEvent)
	})
}

type handler struct {
	svc *service.Service
	v   *validator.Validate
}

type ModerateEventRequestDTO struct {
	Action string `json:"action" validate:"required,min=3,max=64"`
}

func (h *handler) handleModerateEvent(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(chi.URLParam(r, "id"))
	if eventID == "" {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "missing event id"},
		})
		return
	}

	var req ModerateEventRequestDTO
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: "invalid json"},
		})
		return
	}
	if err := h.v.Struct(req); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	if err := h.svc.ModerateEvent(r.Context(), eventID, req.Action); err != nil {
		if errors.Is(err, service.ErrNotImplemented) {
			_ = httpx.WriteJSON(w, http.StatusNotImplemented, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_implemented", Message: "admin moderation not implemented yet"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "moderation failed"},
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

