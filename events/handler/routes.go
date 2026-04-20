package handler

import (
	"net/http"
	"strconv"
	"strings"
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
	"github.com/nu/student-event-ticketing-platform/internal/infra/storage"
)

type Deps struct {
	DB      *pgxpool.Pool
	JWT     authx.JWT
	Storage storage.Service
}

func RegisterRoutes(r chi.Router, deps Deps) {
	repo := repository.NewPostgres(deps.DB)
	svc := service.New(repo)

	v := validator.New()
	// Cross-field validation: if end_at is provided, it must be strictly after starts_at.
	v.RegisterStructValidation(func(sl validator.StructLevel) {
		req, ok := sl.Current().Interface().(CreateEventRequestDTO)
		if !ok {
			return
		}
		if req.EndAt != nil && !req.EndAt.After(req.StartsAt) {
			sl.ReportError(req.EndAt, "EndAt", "end_at", "gtfield", "starts_at")
		}
	}, CreateEventRequestDTO{})
	v.RegisterStructValidation(func(sl validator.StructLevel) {
		req, ok := sl.Current().Interface().(UpdateEventRequestDTO)
		if !ok {
			return
		}
		// For updates we can only validate when both are provided in the request.
		if req.EndAt != nil && req.StartsAt != nil && !req.EndAt.After(*req.StartsAt) {
			sl.ReportError(req.EndAt, "EndAt", "end_at", "gtfield", "starts_at")
		}
	}, UpdateEventRequestDTO{})

	h := &handler{repo: repo, svc: svc, v: v, storage: deps.Storage}

	r.Route("/events", func(r chi.Router) {
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Post("/", h.handleCreate)
		r.Get("/", h.handleList)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Get("/mine", h.handleListMine)
		r.Get("/{id}", h.handleGetByID)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Post("/{id}/cover-image", h.UploadCoverImage)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Put("/{id}", h.handleUpdate)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleOrganizer, authx.RoleAdmin)).Delete("/{id}", h.handleDelete)
	})
}

type handler struct {
	repo    repository.EventRepository
	svc     *service.Service
	v       *validator.Validate
	storage storage.Service
}

// @Summary Create an event
// @Description Creates an event for the authenticated organizer or admin. New events start with moderation_status pending until an admin approves. Optional fields: location, end_at (must be after starts_at).
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
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}

	organizerID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	ev, err := h.svc.Create(r.Context(), req.Title, req.Description, req.CoverImageURL, req.StartsAt, req.Location, req.EndAt, req.CapacityTotal, organizerID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to create event")
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
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid limit")
			return
		}
		limit = v
	}
	if s := r.URL.Query().Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 || v > 100000 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid offset")
			return
		}
		offset = v
	}

	var startsAfter *time.Time
	if s := r.URL.Query().Get("starts_after"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startsAfter = &t
		} else {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid starts_after")
			return
		}
	}
	var startsBefore *time.Time
	if s := r.URL.Query().Get("starts_before"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startsBefore = &t
		} else {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid starts_before")
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
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to list events")
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

// @Summary List my events (dashboard)
// @Description Authenticated listing for organizers/admins. Organizers see their own events (any moderation status). Admins may see all events or filter by organizer_id.
// @Tags events
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (organizer or admin)"
// @Param q query string false "Search query"
// @Param limit query int false "Page size (default 20 if omitted or invalid; max 100)"
// @Param offset query int false "Offset (default 0)"
// @Param starts_after query string false "Filter starts_after (RFC3339)"
// @Param starts_before query string false "Filter starts_before (RFC3339)"
// @Param organizer_id query string false "Admin only: filter by organizer ID (UUID)"
// @Success 200 {object} ListEventsResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "Missing/invalid JWT"
// @Failure 403 {object} httpx.ErrorResponse "Wrong role"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /events/mine [get]
func (h *handler) handleListMine(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := 0
	offset := 0

	if s := r.URL.Query().Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 1 || v > 100 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid limit")
			return
		}
		limit = v
	}
	if s := r.URL.Query().Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 || v > 100000 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid offset")
			return
		}
		offset = v
	}

	var startsAfter *time.Time
	if s := r.URL.Query().Get("starts_after"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startsAfter = &t
		} else {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid starts_after")
			return
		}
	}
	var startsBefore *time.Time
	if s := r.URL.Query().Get("starts_before"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			startsBefore = &t
		} else {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid starts_before")
			return
		}
	}

	callerID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	isAdmin := authx.HasRole(r.Context(), authx.RoleAdmin)
	isOrganizer := authx.HasRole(r.Context(), authx.RoleOrganizer)

	var organizerID *uuid.UUID
	switch {
	case isAdmin:
		if s := strings.TrimSpace(r.URL.Query().Get("organizer_id")); s != "" {
			id, err := uuid.Parse(s)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid organizer_id")
				return
			}
			organizerID = &id
		}
	case isOrganizer:
		organizerID = &callerID
	default:
		httpx.WriteError(w, http.StatusForbidden, httpx.ErrCodeForbidden, "forbidden")
		return
	}

	filter := repository.EventFilter{
		Query:                     q,
		StartsAfter:               startsAfter,
		StartsBefore:              startsBefore,
		OrganizerID:               organizerID,
		RequireApprovedModeration: false,
		Limit:                     limit,
		Offset:                    offset,
	}

	items, err := h.svc.List(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to list events")
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
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidID, "invalid event id")
		return
	}

	ev, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	if ev.ModerationStatus != model.ModerationApproved {
		httpx.WriteError(w, http.StatusNotFound, httpx.ErrCodeNotFound, "event not found")
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, eventToDTO(ev))
}

// @Summary Update event by ID
// @Description Updates event fields. Optional fields: location, end_at (must be after starts_at).
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
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidID, "invalid event id")
		return
	}

	if _, ok := authx.RoleFromContext(r.Context()); !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing role")
		return
	}
	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	existing, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	if authx.HasRole(r.Context(), authx.RoleOrganizer) && !authx.HasRole(r.Context(), authx.RoleAdmin) {
		if existing.OrganizerID == nil || *existing.OrganizerID != userID {
			httpx.WriteError(w, http.StatusForbidden, httpx.ErrCodeForbidden, "not allowed to modify this event")
			return
		}
	}

	var req UpdateEventRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}

	// Ensure the post-update (starts_at, end_at) pair is valid, even if only one field is patched.
	newStartsAt := existing.StartsAt
	if req.StartsAt != nil {
		newStartsAt = *req.StartsAt
	}
	newEndAt := existing.EndAt
	if req.EndAt != nil {
		newEndAt = req.EndAt
	}
	if newEndAt != nil && !newEndAt.After(newStartsAt) {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "end_at must be after starts_at")
		return
	}

	var statusPatch *model.EventStatus
	if req.Status != nil {
		s := model.EventStatus(*req.Status)
		statusPatch = &s
	}

	ev, err := h.svc.Update(r.Context(), id, req.Title, req.Description, req.CoverImageURL, req.StartsAt, req.Location, req.EndAt, req.CapacityTotal, statusPatch)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		if status >= 500 {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to update event")
			return
		}
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

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
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidID, "invalid event id")
		return
	}

	_, ok := authx.RoleFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing role")
		return
	}
	userID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	existing, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	if authx.HasRole(r.Context(), authx.RoleOrganizer) && !authx.HasRole(r.Context(), authx.RoleAdmin) {
		if existing.OrganizerID == nil || *existing.OrganizerID != userID {
			httpx.WriteError(w, http.StatusForbidden, httpx.ErrCodeForbidden, "not allowed to delete this event")
			return
		}
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		status, apiErr := httpx.MapDomainError(err)
		if status >= 500 {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to delete event")
			return
		}
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func eventToDTO(ev model.Event) EventDTO {
	return EventDTO{
		ID:                ev.ID.String(),
		Title:             ev.Title,
		Description:       ev.Description,
		CoverImageURL:     ev.CoverImageURL,
		StartsAt:          ev.StartsAt,
		Location:          ev.Location,
		EndAt:             ev.EndAt,
		CapacityTotal:     ev.CapacityTotal,
		CapacityAvailable: ev.CapacityAvailable,
		Status:            string(ev.Status),
		ModerationStatus:  string(ev.ModerationStatus),
	}
}
