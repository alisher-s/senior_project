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
// @Description **401** — JWT; **403** — if role is not student/organizer/admin.
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
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}

	eventID, err := uuid.Parse(strings.TrimSpace(req.EventID))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidID, "invalid event_id")
		return
	}

	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	payment, providerURL, err := h.svc.Initiate(r.Context(), userID, eventID, req.Amount, req.Currency)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentsDisabled) {
			httpx.WriteError(w, http.StatusNotImplemented, httpx.ErrCodeNotImplemented, "payments are not enabled yet")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "payment initiation failed")
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
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "failed to read request body")
		return
	}

	providedSig := r.Header.Get("X-Signature")
	if providedSig == "" {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeMissingSignature, "missing X-Signature header")
		return
	}
	if !h.verifyWebhookSignature(body, providedSig) {
		httpx.WriteError(w, http.StatusForbidden, httpx.ErrCodeInvalidSignature, "webhook signature verification failed")
		return
	}

	// Decode JSON after signature verification.
	var req PaymentWebhookRequestDTO
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}
	if err := h.v.Struct(req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}

	// Signature validation should be implemented before trusting provider payload.
	_, err = h.svc.Webhook(r.Context(), req.ProviderRef, model.PaymentStatus(req.Status))
	if err != nil {
		if errors.Is(err, repository.ErrPaymentsDisabled) {
			httpx.WriteError(w, http.StatusNotImplemented, httpx.ErrCodeNotImplemented, "payments are not enabled yet")
			return
		}
		if errors.Is(err, service.ErrPaymentNotFound) || errors.Is(err, repository.ErrPaymentNotFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.ErrCodeNotFound, "payment not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "payment webhook failed")
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
