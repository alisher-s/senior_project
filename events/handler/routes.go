package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nu/student-event-ticketing-platform/events/model"
	"github.com/nu/student-event-ticketing-platform/events/repository"
	"github.com/nu/student-event-ticketing-platform/events/service"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	notificationsModel "github.com/nu/student-event-ticketing-platform/notifications/model"
	notificationsService "github.com/nu/student-event-ticketing-platform/notifications/service"
	ticketingRepository "github.com/nu/student-event-ticketing-platform/ticketing/repository"
)

type Deps struct {
	DB         *pgxpool.Pool
	JWT        authx.JWT
	NotifSvc   *notificationsService.Service
	TicketRepo ticketingRepository.TicketRepository
}

func RegisterRoutes(r chi.Router, deps Deps) {
	repo := repository.NewPostgres(deps.DB)
	svc := service.New(repo)

	h := &handler{
		repo:       repo,
		svc:        svc,
		v:          validator.New(),
		notifSvc:   deps.NotifSvc,
		ticketRepo: deps.TicketRepo,
	}

	r.Route("/events", func(r chi.Router) {
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Post("/", h.handleCreate)
		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGetByID)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Put("/{id}", h.handleUpdate)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Delete("/{id}", h.handleDelete)
	})
}

type handler struct {
	repo       repository.EventRepository
	svc        *service.Service
	v          *validator.Validate
	notifSvc   *notificationsService.Service
	ticketRepo ticketingRepository.TicketRepository
}

// @Summary Create an event
// @Description Creates a draft/published event for the authenticated organizer or admin. New events start with moderation_status pending until an admin approves.
// @Tags events
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (organizer or admin)"
// @Param request body CreateEventRequestDTO true "Create event request"
// @Success 201 {object} EventDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "Missing/invalid JWT (see code: missing_authorization, invalid_token, …)"
// @Failure 403 {object} httpx.ErrorResponse "Authenticated but not organizer/admin (code: forbidden)"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /events [post]
func (h *handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	organizerID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing user id"},
		})
		return
	}

	ev, err := h.svc.Create(r.Context(), req.Title, req.Description, req.CoverImageURL, req.StartsAt, req.CapacityTotal, req.PriceAmount, req.PriceCurrency, organizerID)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to create event"},
		})
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, eventToDTO(ev))
}

// @Summary List events
// @Tags events
// @Accept json
// @Produce json
// @Param q query string false "Search query"
// @Param limit query int false "Page size (default 20 if omitted or invalid; max 100)"
// @Param offset query int false "Offset (default 0)"
// @Param starts_after query string false "Filter starts_after (RFC3339)"
// @Param starts_before query string false "Filter starts_before (RFC3339)"
// @Success 200 {object} ListEventsResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /events [get]
func (h *handler) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := 0
	offset := 0

	if s := r.URL.Query().Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 1 || v > 100 {
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_request", Message: "invalid limit"},
			})
			return
		}
		limit = v
	}
	if s := r.URL.Query().Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 || v > 100000 {
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_request", Message: "invalid offset"},
			})
			return
		}
		offset = v
	}

	var startsAfter *time.Time
	if s := r.URL.Query().Get("starts_after"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startsAfter = &t
		} else {
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_request", Message: "invalid starts_after"},
			})
			return
		}
	}
	var startsBefore *time.Time
	if s := r.URL.Query().Get("starts_before"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startsBefore = &t
		} else {
			_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "invalid_request", Message: "invalid starts_before"},
			})
			return
		}
	}

	filter := repository.EventFilter{
		Query:                     q,
		StartsAfter:               startsAfter,
		StartsBefore:              startsBefore,
		RequireApprovedModeration: true,
		Limit:                     limit,
		Offset:                    offset,
	}

	items, err := h.svc.List(r.Context(), filter)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to list events"},
		})
		return
	}

	resp := make([]EventDTO, 0, len(items))
	for _, ev := range items {
		resp = append(resp, eventToDTO(ev))
	}

	_ = httpx.WriteJSON(w, http.StatusOK, ListEventsResponseDTO{
		Items:  resp,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// @Summary Get event by ID
// @Description Public read. Only events with moderation_status=approved are returned; otherwise 404 with code not_found.
// @Tags events
// @Accept json
// @Produce json
// @Param id path string true "Event ID (UUID)"
// @Success 200 {object} EventDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /events/{id} [get]
func (h *handler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "invalid event id"},
		})
		return
	}

	ev, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
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

	if ev.ModerationStatus != model.ModerationApproved {
		_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "not_found", Message: "event not found"},
		})
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, eventToDTO(ev))
}

// @Summary Update event by ID
// @Tags events
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (organizer or admin)"
// @Param id path string true "Event ID (UUID)"
// @Param request body UpdateEventRequestDTO true "Update event request"
// @Success 200 {object} EventDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "Missing/invalid JWT"
// @Failure 403 {object} httpx.ErrorResponse "Wrong role or organizer does not own the event"
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /events/{id} [put]
func (h *handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "invalid event id"},
		})
		return
	}

	if _, ok := authx.RoleFromContext(r.Context()); !ok {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing role"},
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

	existing, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
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

	if authx.HasRole(r.Context(), authx.RoleOrganizer) && !authx.HasRole(r.Context(), authx.RoleAdmin) {
		if existing.OrganizerID == nil || *existing.OrganizerID != userID {
			_ = httpx.WriteJSON(w, http.StatusForbidden, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "forbidden", Message: "not allowed to modify this event"},
			})
			return
		}
	}

	var req UpdateEventRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	var statusPatch *model.EventStatus
	if req.Status != nil {
		s := model.EventStatus(*req.Status)
		statusPatch = &s
	}

	ev, err := h.svc.Update(r.Context(), id, req.Title, req.Description, req.CoverImageURL, req.StartsAt, req.CapacityTotal, req.PriceAmount, req.PriceCurrency, statusPatch)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_found", Message: "event not found"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to update event"},
		})
		return
	}

	// Notify ticket holders if the event was cancelled or its start time changed.
	go h.notifyOnEventUpdate(existing, ev)

	_ = httpx.WriteJSON(w, http.StatusOK, eventToDTO(ev))
}

// @Summary Delete event by ID
// @Tags events
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (organizer or admin)"
// @Param id path string true "Event ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "Missing/invalid JWT"
// @Failure 403 {object} httpx.ErrorResponse "Wrong role or organizer does not own the event"
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /events/{id} [delete]
func (h *handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_id", Message: "invalid event id"},
		})
		return
	}

	_, ok := authx.RoleFromContext(r.Context())
	if !ok {
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "unauthorized", Message: "missing role"},
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

	existing, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
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

	if authx.HasRole(r.Context(), authx.RoleOrganizer) && !authx.HasRole(r.Context(), authx.RoleAdmin) {
		if existing.OrganizerID == nil || *existing.OrganizerID != userID {
			_ = httpx.WriteJSON(w, http.StatusForbidden, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "forbidden", Message: "not allowed to delete this event"},
			})
			return
		}
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
				Error: httpx.ErrorBody{Code: "not_found", Message: "event not found"},
			})
			return
		}
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to delete event"},
		})
		return
	}

	// Notify all ticket holders that the event has been removed.
	go h.notifyTicketHolders(
		existing.ID,
		fmt.Sprintf("Event Removed — %s", existing.Title),
		fmt.Sprintf("Unfortunately, \"%s\" has been removed by the organizer.", existing.Title),
	)

	w.WriteHeader(http.StatusNoContent)
}

// notifyOnEventUpdate sends notifications when an event is cancelled or its start time changes.
func (h *handler) notifyOnEventUpdate(before, after model.Event) {
	cancelled := after.Status == model.EventStatusCancelled && before.Status != model.EventStatusCancelled
	timeChanged := !before.StartsAt.Equal(after.StartsAt)

	if cancelled {
		h.notifyTicketHolders(
			after.ID,
			fmt.Sprintf("Event Cancelled — %s", after.Title),
			fmt.Sprintf("Unfortunately, \"%s\" has been cancelled.", after.Title),
		)
		return
	}

	if timeChanged {
		h.notifyTicketHolders(
			after.ID,
			fmt.Sprintf("Event Rescheduled — %s", after.Title),
			fmt.Sprintf("\"%s\" has been rescheduled to %s.",
				after.Title,
				after.StartsAt.UTC().Format("Jan 2, 2006 at 3:04 PM UTC")),
		)
	}
}

// notifyTicketHolders fans out email + push notifications to all active/used ticket holders.
func (h *handler) notifyTicketHolders(eventID uuid.UUID, title, body string) {
	if h.notifSvc == nil || h.ticketRepo == nil {
		return
	}
	ctx := context.Background()

	emails, err := h.ticketRepo.GetActiveTicketHolderEmails(ctx, eventID)
	if err != nil || len(emails) == 0 {
		return
	}

	for _, email := range emails {
		if err := h.notifSvc.Send(ctx, notificationsModel.Notification{
			Type:  notificationsModel.NotificationTypeEmail,
			To:    email,
			Title: title,
			Body:  body,
		}); err != nil {
			_ = err // best-effort; individual failures don't stop the fan-out
		}
	}
}

func eventToDTO(ev model.Event) EventDTO {
	return EventDTO{
		ID:                ev.ID.String(),
		Title:             ev.Title,
		Description:       ev.Description,
		CoverImageURL:     ev.CoverImageURL,
		StartsAt:          ev.StartsAt,
		CapacityTotal:     ev.CapacityTotal,
		CapacityAvailable: ev.CapacityAvailable,
		Status:            string(ev.Status),
		ModerationStatus:  string(ev.ModerationStatus),
		PriceAmount:       ev.PriceAmount,
		PriceCurrency:     ev.PriceCurrency,
		IsFree:            ev.PriceAmount == 0,
	}
}
