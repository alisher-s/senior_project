package handler

import (
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
	notificationsRepo "github.com/nu/student-event-ticketing-platform/notifications/repository"
	notificationsService "github.com/nu/student-event-ticketing-platform/notifications/service"
	"github.com/nu/student-event-ticketing-platform/ticketing/repository"
	"github.com/nu/student-event-ticketing-platform/ticketing/service"
)

type Deps struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	JWT    authx.JWT
	Cfg    struct{}
	Logger *slog.Logger
}

func RegisterRoutes(r chi.Router, deps Deps) {
	repo := repository.NewPostgres(deps.DB)
	svc := service.New(repo)
	notificationsQueueRepo := notificationsRepo.NewPostgres(deps.DB)
	notifier := notificationsService.New(notificationsQueueRepo)
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	h := &handler{db: deps.DB, repo: repo, svc: svc, notifier: notifier, v: validator.New(), logger: logger}

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
	db       *pgxpool.Pool
	repo     repository.TicketRepository
	svc      *service.Service
	notifier *notificationsService.Service
	v        *validator.Validate
	logger   *slog.Logger
}

// @Summary List my tickets
// @Description Returns tickets for the authenticated user (user_id from JWT). Empty `tickets` array if none. Each item `status` may be `active`, `used`, `cancelled`, or `expired` (the latter when the event end instant — `end_at` if set, otherwise `starts_at` — is strictly in the past; `end_at` is inclusive; not persisted in the database).
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
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	rows, err := h.svc.GetUserTickets(r.Context(), userID)
	if err != nil {
		h.logger.Error("list_my_tickets_failed",
			"error", err,
			"request_id", httpx.GetRequestID(r),
			"user_id", userID.String(),
		)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "internal server error")
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

	ticket, qrPNGBase64, err := h.svc.RegisterTicket(r.Context(), userID, eventID)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	// Best-effort confirmation email. Never fail registration on email issues.
	if h.notifier != nil && h.db != nil {
		var userEmail string
		if err := h.db.QueryRow(r.Context(), `SELECT email FROM users WHERE id = $1`, userID).Scan(&userEmail); err != nil {
			h.logger.Error("ticket_confirmation_email_user_lookup_failed",
				"error", err,
				"request_id", httpx.GetRequestID(r),
				"user_id", userID.String(),
			)
		} else {
			var eventTitle string
			var startsAt time.Time
			if err := h.db.QueryRow(r.Context(), `SELECT title, starts_at FROM events WHERE id = $1`, eventID).Scan(&eventTitle, &startsAt); err != nil {
				h.logger.Error("ticket_confirmation_email_event_lookup_failed",
					"error", err,
					"request_id", httpx.GetRequestID(r),
					"event_id", eventID.String(),
				)
			} else {
				subject := fmt.Sprintf("Ticket confirmed: %s", eventTitle)
				body := service.TicketConfirmationEmailHTML(eventTitle, startsAt.UTC(), ticket.ID, qrPNGBase64)
				if err := h.notifier.Send(r.Context(), notificationsModel.Notification{
					Type:  notificationsModel.NotificationTypeEmail,
					To:    userEmail,
					Title: subject,
					Body:  body,
				}); err != nil {
					h.logger.Error("ticket_confirmation_email_enqueue_failed",
						"error", err,
						"request_id", httpx.GetRequestID(r),
						"ticket_id", ticket.ID.String(),
						"to", userEmail,
					)
				}
			}
		}
	}

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
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidID, "invalid ticket id")
		return
	}

	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	ticket, err := h.svc.CancelTicket(r.Context(), userID, ticketID)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, CancelTicketResponseDTO{
		TicketID: ticket.ID.String(),
		EventID:  ticket.EventID.String(),
		UserID:   ticket.UserID.String(),
		Status:   string(ticket.Status),
	})
}

// @Summary Mark ticket as used
// @Description Check-in by QR hash; requires organizer or admin. **401** — JWT; **403** — not organizer/admin; **400** — `ticket_expired` (strictly after end instant; `end_at` is inclusive); **409** — `ticket_already_used`, `check_in_not_open`, `ticket_cannot_be_used`, …
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
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}

	ticket, err := h.svc.UseTicketByQRHash(r.Context(), req.QRHashHex)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, UseTicketResponseDTO{
		TicketID: ticket.ID.String(),
		EventID:  ticket.EventID.String(),
		UserID:   ticket.UserID.String(),
		Status:   string(ticket.Status),
	})
}
