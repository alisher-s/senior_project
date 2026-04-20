package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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
	deviceTokenRepo := notificationsRepo.NewDeviceTokenPostgres(deps.DB)
	svc := notificationsService.New(repo)
	h := &handler{
		svc:             svc,
		deviceTokenRepo: deviceTokenRepo,
		v:               validator.New(),
	}

	r.Route("/notifications", func(r chi.Router) {
		r.Post("/send-email", h.handleSendEmail)

		r.Group(func(r chi.Router) {
			r.Use(authx.AuthMiddleware(deps.JWT))
			r.Post("/device-token", h.handleRegisterDeviceToken)
			r.Delete("/device-token", h.handleDeleteDeviceToken)
			r.Get("/my", h.handleMyNotifications)
		})
	})
}

type handler struct {
	svc             *notificationsService.Service
	deviceTokenRepo notificationsRepo.DeviceTokenRepository
	v               *validator.Validate
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

	err := h.svc.Send(r.Context(), notificationsModel.Notification{
		Type:  notificationsModel.NotificationTypeEmail,
		To:    req.To,
		Title: req.Title,
		Body:  req.Body,
	})
	if err != nil {
		if errors.Is(err, notificationsService.ErrNotImplemented) {
			_ = httpx.WriteJSON(w, http.StatusNotImplemented, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_implemented", Message: "send-email not implemented yet"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to enqueue email"},
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

type RegisterDeviceTokenRequestDTO struct {
	Token    string `json:"token" validate:"required,min=10"`
	Platform string `json:"platform" validate:"required,oneof=android ios"`
}

// @Summary Register a push notification device token
// @Description Stores an FCM (Android) or APNs (iOS) token for the authenticated user. Used to deliver push notifications.
// @Tags notifications
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param request body RegisterDeviceTokenRequestDTO true "Device token"
// @Success 204 "No Content — token registered"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /notifications/device-token [post]
func (h *handler) handleRegisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	var req RegisterDeviceTokenRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	if err := h.deviceTokenRepo.Upsert(r.Context(), userID.String(), req.Token, req.Platform); err != nil {
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to register device token"},
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type DeleteDeviceTokenRequestDTO struct {
	Token string `json:"token" validate:"required"`
}

// @Summary Remove a push notification device token
// @Description Deletes the given token for the authenticated user (call on logout or token rotation).
// @Tags notifications
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param request body DeleteDeviceTokenRequestDTO true "Token to remove"
// @Success 204 "No Content"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /notifications/device-token [delete]
func (h *handler) handleDeleteDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	var req DeleteDeviceTokenRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	if err := h.deviceTokenRepo.Delete(r.Context(), userID.String(), req.Token); err != nil {
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to delete device token"},
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type NotificationItemDTO struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// @Summary Get my recent notifications
// @Description Returns the 20 most recent notifications for the authenticated user (for in-app notification history).
// @Tags notifications
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Success 200 {array} NotificationItemDTO
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /notifications/my [get]
func (h *handler) handleMyNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	items, err := h.svc.GetByUserID(r.Context(), userID.String(), 20)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to load notifications"},
		})
		return
	}

	out := make([]NotificationItemDTO, 0, len(items))
	for _, n := range items {
		out = append(out, NotificationItemDTO{
			ID:        n.ID,
			Type:      string(n.Type),
			Title:     n.Title,
			Body:      n.Body,
			CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	_ = httpx.WriteJSON(w, http.StatusOK, out)
}
