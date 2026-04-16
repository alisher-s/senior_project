package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	"github.com/nu/student-event-ticketing-platform/internal/config"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	"github.com/nu/student-event-ticketing-platform/payments/model"
	"github.com/nu/student-event-ticketing-platform/payments/repository"
	"github.com/nu/student-event-ticketing-platform/payments/service"
	ticketingRepository "github.com/nu/student-event-ticketing-platform/ticketing/repository"
)

type Deps struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	JWT    authx.JWT
	Logger *slog.Logger
	Cfg    config.Config
}

func RegisterRoutes(r chi.Router, deps Deps) {
	_ = deps.DB
	repo := repository.NewStub()
	ticketRepo := ticketingRepository.NewPostgres(deps.DB)
	svc := service.New(repo, ticketRepo)
	h := &handler{svc: svc, v: validator.New(), webhookSecret: deps.Cfg.Payments.WebhookSecret}

	// initiate: requires authentication (student/organizer/admin in future).
	r.With(authx.AuthMiddleware(deps.JWT)).Route("/payments", func(r chi.Router) {
		r.With(authx.RequireRole(authx.RoleStudent, authx.RoleOrganizer, authx.RoleAdmin)).
			Post("/initiate", h.handleInitiate)
	})

	// webhook: usually provider-to-provider; signature validation will be added in next step.
	r.Post("/payments/webhook", h.handleWebhook)
}

type handler struct {
	svc *service.Service
	v   *validator.Validate

	webhookSecret string
}

// @Summary Initiate a payment
// @Tags payments
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (student, organizer, or admin)"
// @Param request body InitiatePaymentRequestDTO true "Payment request"
// @Success 201 {object} InitiatePaymentResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 501 {object} httpx.ErrorResponse "Payments stub disabled (code: not_implemented)"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /payments/initiate [post]
func (h *handler) handleInitiate(w http.ResponseWriter, r *http.Request) {
	var req InitiatePaymentRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	eventID, err := uuid.Parse(strings.TrimSpace(req.EventID))
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "invalid event_id"},
		})
		return
	}

	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	payment, providerURL, err := h.svc.Initiate(r.Context(), userID, eventID, req.Amount, req.Currency)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentsDisabled) {
			_ = httpx.WriteJSON(w, http.StatusNotImplemented, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_implemented", Message: "payments are not enabled yet"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "payment initiation failed"},
		})
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, InitiatePaymentResponseDTO{
		PaymentID:   payment.ID.String(),
		ProviderRef: payment.ProviderRef,
		ProviderURL: providerURL,
	})
}

// @Summary Payment provider webhook
// @Description Verifies X-Signature (hex HMAC-SHA256 of raw body with PAYMENTS_WEBHOOK_SECRET). Not for browser clients.
// @Tags payments
// @Accept json
// @Produce json
// @Param X-Signature header string true "Hex-encoded HMAC-SHA256 of raw body"
// @Param request body PaymentWebhookRequestDTO true "Webhook payload"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "missing_signature"
// @Failure 403 {object} httpx.ErrorResponse "invalid_signature"
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 501 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /payments/webhook [post]
func (h *handler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: "failed to read request body"},
		})
		return
	}

	providedSig := r.Header.Get("X-Signature")
	if providedSig == "" {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "missing_signature", Message: "missing X-Signature header"},
		})
		return
	}
	if !h.verifyWebhookSignature(body, providedSig) {
		_ = httpx.WriteJSON(w, http.StatusForbidden, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_signature", Message: "webhook signature verification failed"},
		})
		return
	}

	// Decode JSON after signature verification.
	var req PaymentWebhookRequestDTO
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}
	if err := h.v.Struct(req); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	// Signature validation should be implemented before trusting provider payload.
	_, err = h.svc.Webhook(r.Context(), req.ProviderRef, model.PaymentStatus(req.Status))
	if err != nil {
		if errors.Is(err, repository.ErrPaymentsDisabled) {
			_ = httpx.WriteJSON(w, http.StatusNotImplemented, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_implemented", Message: "payments are not enabled yet"},
			})
			return
		}
		if errors.Is(err, service.ErrPaymentNotFound) || errors.Is(err, repository.ErrPaymentNotFound) {
			_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_found", Message: "payment not found"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "payment webhook failed"},
		})
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, struct{}{})
}

func (h *handler) verifyWebhookSignature(body []byte, providedSigHex string) bool {
	if h.webhookSecret == "" {
		return false
	}

	providedSig, err := hex.DecodeString(strings.TrimSpace(providedSigHex))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	_, _ = mac.Write(body)
	expectedSig := mac.Sum(nil)

	return hmac.Equal(providedSig, expectedSig)
}
