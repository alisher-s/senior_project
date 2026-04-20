package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	notificationsModel "github.com/nu/student-event-ticketing-platform/notifications/model"
	notificationsService "github.com/nu/student-event-ticketing-platform/notifications/service"
	"github.com/nu/student-event-ticketing-platform/ticketing/repository"
	"github.com/nu/student-event-ticketing-platform/ticketing/service"
)

type Deps struct {
	DB       *pgxpool.Pool
	Redis    *redis.Client
	JWT      authx.JWT
	Cfg      struct{}
	Logger   *slog.Logger
	NotifSvc *notificationsService.Service
}

func RegisterRoutes(r chi.Router, deps Deps) {
	repo := repository.NewPostgres(deps.DB)
	svc := service.New(repo)
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	h := &handler{
		repo:     repo,
		svc:      svc,
		v:        validator.New(),
		logger:   logger,
		db:       deps.DB,
		notifSvc: deps.NotifSvc,
	}

	r.With(
		authx.AuthMiddleware(deps.JWT),
	).Route("/tickets", func(r chi.Router) {
		r.Get("/my", h.handleMyTickets)
		r.With(authx.RequireRole(authx.RoleStudent)).Post("/register", h.handleRegister)
		r.With(authx.RequireRole(authx.RoleStudent)).Post("/{id}/cancel", h.handleCancel)
		r.With(authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Post("/use", h.handleUse)
	})
}

type handler struct {
	repo     repository.TicketRepository
	svc      *service.Service
	v        *validator.Validate
	logger   *slog.Logger
	db       *pgxpool.Pool
	notifSvc *notificationsService.Service
}

// @Summary List my tickets
// @Description Returns tickets for the authenticated user (user_id from JWT). Empty `tickets` array if none.
// @Tags tickets
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Success 200 {object} MyTicketsResponseDTO
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /tickets/my [get]
func (h *handler) handleMyTickets(w http.ResponseWriter, r *http.Request) {
	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	rows, err := h.svc.ListMyTickets(r.Context(), userID)
	if err != nil {
		h.logger.Error("list_my_tickets_failed",
			"error", err,
			"request_id", httpx.GetRequestID(r),
			"user_id", userID.String(),
		)
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "internal server error"},
		})
		return
	}

	items := make([]MyTicketItemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, MyTicketItemDTO{
			TicketID:   row.ID.String(),
			Status:     string(row.Status),
			QRHashHex:  row.QRHashHex,
			EventID:    row.EventID.String(),
			EventTitle: row.EventTitle,
			EventDate:  row.EventStartsAt.UTC().Format(time.RFC3339Nano),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, MyTicketsResponseDTO{Tickets: items})
}

// @Summary Register a ticket for an event
// @Description Requires student role. On success returns qr_png_base64 (PNG data URL–ready base64) and qr_hash_hex for check-in. 409 with code capacity_full or already_registered (and other business rules — see API error codes in README).
// @Tags tickets
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (student)"
// @Param request body RegisterTicketRequestDTO true "Register ticket request"
// @Success 201 {object} RegisterTicketResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "Invalid JWT"
// @Failure 403 {object} httpx.ErrorResponse "Not a student (code: forbidden)"
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse "capacity_full, already_registered, event_not_approved, …"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /tickets/register [post]
func (h *handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterTicketRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	eventID, err := uuid.Parse(strings.TrimSpace(req.EventID))
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "invalid event_id"},
		})
		return
	}

	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	// Block direct registration for paid events — payment must go through /payments/initiate.
	var priceAmount int64
	if h.db != nil {
		_ = h.db.QueryRow(r.Context(),
			`SELECT price_amount FROM events WHERE id = $1`, eventID,
		).Scan(&priceAmount)
	}
	if priceAmount > 0 {
		writeServiceError(w, repository.ErrEventRequiresPayment)
		return
	}

	ticket, qrPNGBase64, err := h.svc.RegisterTicket(r.Context(), userID, eventID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Enqueue confirmation notifications asynchronously so they never block the response.
	go h.sendTicketNotifications(
		userID.String(),
		"Ticket Confirmed",
		fmt.Sprintf("Your ticket for the event on %s is confirmed. Check the app for your QR code.",
			ticket.CreatedAt.UTC().Format("Jan 2, 2006")),
	)

	httpx.WriteJSON(w, http.StatusCreated, RegisterTicketResponseDTO{
		TicketID:    ticket.ID.String(),
		EventID:     ticket.EventID.String(),
		UserID:      ticket.UserID.String(),
		Status:      string(ticket.Status),
		QRPNGBase64: qrPNGBase64,
		QRHashHex:   ticket.QRHashHex,
	})
}

// @Summary Cancel a ticket
// @Description **401** — JWT; **403** — not a student; **409** — `ticket_already_cancelled`, `cancellation_not_allowed`, …
// @Tags tickets
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (student)"
// @Param id path string true "Ticket ID (UUID)"
// @Success 200 {object} CancelTicketResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /tickets/{id}/cancel [post]
func (h *handler) handleCancel(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	ticketID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "invalid ticket id"},
		})
		return
	}

	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	ticket, err := h.svc.CancelTicket(r.Context(), userID, ticketID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	go h.sendTicketNotifications(
		userID.String(),
		"Ticket Cancelled",
		"Your ticket has been cancelled and the seat has been released.",
	)

	httpx.WriteJSON(w, http.StatusOK, CancelTicketResponseDTO{
		TicketID: ticket.ID.String(),
		EventID:  ticket.EventID.String(),
		UserID:   ticket.UserID.String(),
		Status:   string(ticket.Status),
	})
}

// @Summary Mark ticket as used
// @Description Check-in by QR hash; requires organizer or admin. **401** — JWT; **403** — not organizer/admin; **409** — `ticket_already_used`, `check_in_not_open`, `ticket_cannot_be_used`, …
// @Tags tickets
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (organizer or admin)"
// @Param request body UseTicketRequestDTO true "Use ticket request"
// @Success 200 {object} UseTicketResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /tickets/use [post]
func (h *handler) handleUse(w http.ResponseWriter, r *http.Request) {
	var req UseTicketRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	ticket, err := h.svc.UseTicketByQRHash(r.Context(), req.QRHashHex)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, UseTicketResponseDTO{
		TicketID: ticket.ID.String(),
		EventID:  ticket.EventID.String(),
		UserID:   ticket.UserID.String(),
		Status:   string(ticket.Status),
	})
}

// sendTicketNotifications fans out an email + push notification for the given user.
// Runs in a goroutine — failures are logged, never surface to the caller.
func (h *handler) sendTicketNotifications(userID, title, body string) {
	if h.notifSvc == nil || h.db == nil {
		return
	}
	ctx := context.Background()

	var email string
	_ = h.db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1::uuid", userID).Scan(&email)

	if email != "" {
		if err := h.notifSvc.Send(ctx, notificationsModel.Notification{
			Type:   notificationsModel.NotificationTypeEmail,
			To:     email,
			UserID: userID,
			Title:  title,
			Body:   body,
		}); err != nil {
			h.logger.Warn("ticket_email_enqueue_failed", "user_id", userID, "error", err)
		}
	}

	if err := h.notifSvc.Send(ctx, notificationsModel.Notification{
		Type:   notificationsModel.NotificationTypePush,
		To:     userID,
		UserID: userID,
		Title:  title,
		Body:   body,
	}); err != nil {
		h.logger.Warn("ticket_push_enqueue_failed", "user_id", userID, "error", err)
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrEventNotFound), errors.Is(err, service.ErrEventNotFound):
		_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "not_found", Message: "event not found"},
		})
	case errors.Is(err, repository.ErrCapacityFull), errors.Is(err, service.ErrCapacityFull):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "capacity_full", Message: "event capacity is full"},
		})
	case errors.Is(err, repository.ErrAlreadyRegistered), errors.Is(err, service.ErrAlreadyRegistered):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "already_registered", Message: "ticket already exists"},
		})
	case errors.Is(err, repository.ErrEventNotPublished), errors.Is(err, service.ErrEventNotPublished):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "event_not_published", Message: "event is not open for registration"},
		})
	case errors.Is(err, repository.ErrEventNotApproved), errors.Is(err, service.ErrEventNotApproved):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "event_not_approved", Message: "event is not approved for registration"},
		})
	case errors.Is(err, repository.ErrEventRequiresPayment):
		_ = httpx.WriteJSON(w, http.StatusPaymentRequired, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "payment_required", Message: "this is a paid event — use POST /api/v1/payments/initiate to purchase a ticket"},
		})
	case errors.Is(err, repository.ErrEventCancelled), errors.Is(err, service.ErrEventCancelled):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "event_cancelled", Message: "event is cancelled"},
		})
	case errors.Is(err, repository.ErrEventRegistrationClosed), errors.Is(err, service.ErrEventRegistrationClosed):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "registration_closed", Message: "registration is closed for this event"},
		})
	case errors.Is(err, repository.ErrCancellationNotAllowed), errors.Is(err, service.ErrCancellationNotAllowed):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "cancellation_not_allowed", Message: "ticket cannot be cancelled after the event has started"},
		})
	case errors.Is(err, repository.ErrCheckInNotOpenYet), errors.Is(err, service.ErrCheckInNotOpenYet):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "check_in_not_open", Message: "check-in is not open yet for this event"},
		})
	case errors.Is(err, repository.ErrTicketNotFound), errors.Is(err, service.ErrTicketNotFound):
		_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "ticket_not_found", Message: "ticket not found"},
		})
	case errors.Is(err, repository.ErrTicketAlreadyCancelled), errors.Is(err, service.ErrTicketAlreadyCancelled):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "ticket_already_cancelled", Message: "ticket already cancelled"},
		})
	case errors.Is(err, repository.ErrTicketAlreadyUsed), errors.Is(err, service.ErrTicketAlreadyUsed):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "ticket_already_used", Message: "ticket already used"},
		})
	case errors.Is(err, repository.ErrTicketCannotBeUsed), errors.Is(err, service.ErrTicketCannotBeUsed):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "ticket_cannot_be_used", Message: "ticket cannot be used"},
		})
	default:
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "internal server error"},
		})
	}
}
