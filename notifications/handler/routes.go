package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	notificationsModel "github.com/nu/student-event-ticketing-platform/notifications/model"
	notificationsRepo "github.com/nu/student-event-ticketing-platform/notifications/repository"
	notificationsService "github.com/nu/student-event-ticketing-platform/notifications/service"
)

type Deps struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	JWT    authx.JWT
	Logger *slog.Logger
}

func RegisterRoutes(r chi.Router, deps Deps) {
	repo := notificationsRepo.NewPostgres(deps.DB)
	svc := notificationsService.New(repo)
	h := &handler{svc: svc, v: validator.New()}

	r.Route("/notifications", func(r chi.Router) {
		// Foundation stub. Real implementation will enqueue and process emails asynchronously.
		r.Post("/send-email", h.handleSendEmail)
	})
}

type handler struct {
	svc *notificationsService.Service
	v   *validator.Validate
}

type SendEmailRequestDTO struct {
	To    string `json:"to" validate:"required,email"`
	Title string `json:"title" validate:"required,min=3,max=200"`
	Body  string `json:"body" validate:"required,min=1,max=5000"`
}

// @Summary Enqueue outbound email (foundation)
// @Description No JWT required in current build. May return 501 if sending is not wired.
// @Tags notifications
// @Accept json
// @Produce json
// @Param request body SendEmailRequestDTO true "Email payload"
// @Success 202 "Accepted — enqueued"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 501 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /notifications/send-email [post]
func (h *handler) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	var req SendEmailRequestDTO
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid json")
		return
	}
	if err := h.v.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}

	// Stub: will be replaced with async worker usage.
	err := h.svc.Send(r.Context(), notificationsModel.Notification{
		Type:  notificationsModel.NotificationTypeEmail,
		To:    req.To,
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		if status < 500 {
			// More specific client-facing message for this stub.
			if status == http.StatusNotImplemented {
				httpx.WriteError(w, status, apiErr.Code, "send-email not implemented yet")
				return
			}
			httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to enqueue email")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
