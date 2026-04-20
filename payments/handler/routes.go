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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	stripeWebhook "github.com/stripe/stripe-go/v76/webhook"
	"log/slog"

	"context"
	"fmt"

	"github.com/nu/student-event-ticketing-platform/internal/config"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	notificationsModel "github.com/nu/student-event-ticketing-platform/notifications/model"
	notificationsService "github.com/nu/student-event-ticketing-platform/notifications/service"
	"github.com/nu/student-event-ticketing-platform/payments/model"
	"github.com/nu/student-event-ticketing-platform/payments/repository"
	"github.com/nu/student-event-ticketing-platform/payments/service"
	ticketingRepository "github.com/nu/student-event-ticketing-platform/ticketing/repository"
)

type Deps struct {
	DB       *pgxpool.Pool
	Redis    *redis.Client
	JWT      authx.JWT
	Logger   *slog.Logger
	Cfg      config.Config
	NotifSvc *notificationsService.Service
}

func RegisterRoutes(r chi.Router, deps Deps) {
	ticketRepo := ticketingRepository.NewPostgres(deps.DB)

	var repo repository.PaymentRepository
	var stripeClient *service.StripeClient

	if deps.Cfg.Payments.StripeSecretKey != "" {
		repo = repository.NewPostgres(deps.DB)
		stripeClient = service.NewStripeClient(
			deps.Cfg.Payments.StripeSecretKey,
			deps.Cfg.Payments.StripeSuccessURL,
			deps.Cfg.Payments.StripeCancelURL,
		)
	} else {
		repo = repository.NewStub()
	}

	svc := service.New(repo, ticketRepo, stripeClient)
	h := &handler{
		svc:                 svc,
		v:                   validator.New(),
		webhookSecret:       deps.Cfg.Payments.WebhookSecret,
		stripeWebhookSecret: deps.Cfg.Payments.StripeWebhookSecret,
		logger:              deps.Logger,
		db:                  deps.DB,
		notifSvc:            deps.NotifSvc,
	}

	r.With(authx.AuthMiddleware(deps.JWT)).Route("/payments", func(r chi.Router) {
		r.With(authx.RequireRole(authx.RoleStudent, authx.RoleOrganizer, authx.RoleAdmin)).
			Post("/initiate", h.handleInitiate)
	})

	// Generic HMAC webhook (non-Stripe providers).
	r.Post("/payments/webhook", h.handleWebhook)

	// Stripe-specific webhook with Stripe signature verification.
	r.Post("/payments/stripe/webhook", h.handleStripeWebhook)
}

type handler struct {
	svc                 *service.Service
	v                   *validator.Validate
	webhookSecret       string
	stripeWebhookSecret string
	logger              *slog.Logger
	db                  *pgxpool.Pool
	notifSvc            *notificationsService.Service
}

// @Summary Initiate a payment
// @Description **401** — JWT; **403** — if role is not student/organizer/admin. Returns provider_url to redirect the user to (Stripe Checkout page when Stripe is configured).
// @Tags payments
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (student, organizer, or admin)"
// @Param request body InitiatePaymentRequestDTO true "Payment request"
// @Success 201 {object} InitiatePaymentResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 501 {object} httpx.ErrorResponse "Payments not configured (code: not_implemented)"
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

	// Fetch event price from DB — clients cannot supply their own amount.
	var priceAmount int64
	var priceCurrency string
	err = h.db.QueryRow(r.Context(),
		`SELECT price_amount, price_currency FROM events WHERE id = $1`, eventID,
	).Scan(&priceAmount, &priceCurrency)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_found", Message: "event not found"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to load event"},
		})
		return
	}
	if priceAmount == 0 {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "free_event", Message: "this event is free — use POST /api/v1/tickets/register"},
		})
		return
	}

	payment, providerURL, err := h.svc.Initiate(r.Context(), userID, eventID, priceAmount, priceCurrency)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentsDisabled) {
			_ = httpx.WriteJSON(w, http.StatusNotImplemented, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_implemented", Message: "payments are not enabled yet"},
			})
			return
		}
		if h.logger != nil {
			h.logger.Error("payment_initiate_failed", "error", err)
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
		Amount:      payment.Amount,
		Currency:    payment.Currency,
	})
}

// @Summary Payment provider webhook (generic HMAC)
// @Description Verifies X-Signature (hex HMAC-SHA256 of raw body with PAYMENTS_WEBHOOK_SECRET).
// @Tags payments
// @Accept json
// @Produce json
// @Param X-Signature header string true "Hex-encoded HMAC-SHA256 of raw body"
// @Param request body PaymentWebhookRequestDTO true "Webhook payload"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
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

	if _, err = h.svc.Webhook(r.Context(), req.ProviderRef, model.PaymentStatus(req.Status)); err != nil {
		h.writeWebhookError(w, err)
		return
	}
	_ = httpx.WriteJSON(w, http.StatusOK, struct{}{})
}

// @Summary Stripe webhook
// @Description Verifies Stripe-Signature header using STRIPE_WEBHOOK_SECRET. Handles checkout.session.completed / expired and payment_intent.payment_failed.
// @Tags payments
// @Accept json
// @Produce json
// @Param Stripe-Signature header string true "Stripe webhook signature"
// @Success 200 {object} object
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /payments/stripe/webhook [post]
func (h *handler) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = 65536
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: "failed to read body"},
		})
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	if sig == "" {
		_ = httpx.WriteJSON(w, http.StatusForbidden, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "missing_signature", Message: "missing Stripe-Signature header"},
		})
		return
	}

	// Validate signature only — do not use ConstructEvent which may fail to unmarshal
	// events from newer Stripe API versions not yet modeled in this SDK version.
	if err := stripeWebhook.ValidatePayload(body, sig, h.stripeWebhookSecret); err != nil {
		_ = httpx.WriteJSON(w, http.StatusForbidden, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_signature", Message: "stripe signature verification failed"},
		})
		return
	}

	// Parse the event type and data object manually.
	var raw struct {
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: "failed to parse event"},
		})
		return
	}

	var providerRef string
	var status model.PaymentStatus

	switch raw.Type {
	case "checkout.session.completed":
		var sess stripeCheckoutSession
		if err := json.Unmarshal(raw.Data.Object, &sess); err != nil {
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_request", Message: "failed to parse session"},
			})
			return
		}
		providerRef = sess.ID
		if sess.PaymentStatus == "paid" {
			status = model.PaymentStatusSucceeded
		} else {
			status = model.PaymentStatusPending
		}

	case "checkout.session.expired":
		var sess stripeCheckoutSession
		if err := json.Unmarshal(raw.Data.Object, &sess); err != nil {
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_request", Message: "failed to parse session"},
			})
			return
		}
		providerRef = sess.ID
		status = model.PaymentStatusCanceled

	default:
		// Acknowledge all other event types (charge.succeeded, payment_intent.*, etc.)
		_ = httpx.WriteJSON(w, http.StatusOK, struct{}{})
		return
	}

	payment, err := h.svc.Webhook(r.Context(), providerRef, status)
	if err != nil {
		h.writeWebhookError(w, err)
		return
	}

	if status == model.PaymentStatusSucceeded {
		go h.sendPaymentNotification(payment.UserID.String(), payment.Amount, payment.Currency)
	}

	_ = httpx.WriteJSON(w, http.StatusOK, struct{}{})
}

// stripeCheckoutSession is a minimal struct for unmarshalling Stripe checkout session events.
type stripeCheckoutSession struct {
	ID            string `json:"id"`
	PaymentStatus string `json:"payment_status"`
}


func (h *handler) writeWebhookError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrPaymentsDisabled):
		_ = httpx.WriteJSON(w, http.StatusNotImplemented, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "not_implemented", Message: "payments are not enabled yet"},
		})
	case errors.Is(err, service.ErrPaymentNotFound), errors.Is(err, repository.ErrPaymentNotFound):
		_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "not_found", Message: "payment not found"},
		})
	default:
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "payment webhook failed"},
		})
	}
}

func (h *handler) sendPaymentNotification(userID string, amount int64, currency string) {
	if h.notifSvc == nil || h.db == nil {
		return
	}
	ctx := context.Background()

	var email string
	_ = h.db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1::uuid", userID).Scan(&email)

	title := "Payment Confirmed — Ticket Issued"
	body := fmt.Sprintf("Your payment of %d %s was successful. Your ticket is now active — check the app for your QR code.", amount, currency)

	if email != "" {
		_ = h.notifSvc.Send(ctx, notificationsModel.Notification{
			Type:   notificationsModel.NotificationTypeEmail,
			To:     email,
			UserID: userID,
			Title:  title,
			Body:   body,
		})
	}
	_ = h.notifSvc.Send(ctx, notificationsModel.Notification{
		Type:   notificationsModel.NotificationTypePush,
		To:     userID,
		UserID: userID,
		Title:  title,
		Body:   body,
	})
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
	return hmac.Equal(providedSig, mac.Sum(nil))
}
