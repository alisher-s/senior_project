package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	"github.com/nu/student-event-ticketing-platform/events/model"
	"github.com/nu/student-event-ticketing-platform/events/repository"
	"github.com/nu/student-event-ticketing-platform/events/service"
)

type Deps struct {
	DB *pgxpool.Pool
}

func RegisterRoutes(r chi.Router, deps Deps) {
	repo := repository.NewPostgres(deps.DB)
	svc := service.New(repo)

	h := &handler{repo: repo, svc: svc, v: validator.New()}

	r.Route("/events", func(r chi.Router) {
		r.Post("/", h.handleCreate)
		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGetByID)
		r.Put("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
	})
}

type handler struct {
	repo repository.EventRepository
	svc  *service.Service
	v    *validator.Validate
}

// @Summary Create an event
// @Tags events
// @Accept json
// @Produce json
// @Param request body CreateEventRequestDTO true "Create event request"
// @Success 201 {object} EventDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Router /events [post]
func (h *handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	ev, err := h.svc.Create(r.Context(), req.Title, req.Description, req.StartsAt, req.CapacityTotal)
	if err != nil {
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "failed to create event"},
		})
		return
	}

	_ = httpx.WriteJSON(w, http.StatusCreated, EventDTO{
		ID:                ev.ID.String(),
		Title:             ev.Title,
		Description:       ev.Description,
		StartsAt:          ev.StartsAt,
		CapacityTotal:     ev.CapacityTotal,
		CapacityAvailable: ev.CapacityAvailable,
		Status:            string(ev.Status),
	})
}

// @Summary List events
// @Tags events
// @Accept json
// @Produce json
// @Param q query string false "Search query"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param starts_after query string false "Filter starts_after (RFC3339)"
// @Param starts_before query string false "Filter starts_before (RFC3339)"
// @Success 200 {object} ListEventsResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
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
		Query:         q,
		StartsAfter:   startsAfter,
		StartsBefore:  startsBefore,
		Limit:         limit,
		Offset:        offset,
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
		resp = append(resp, EventDTO{
			ID:                ev.ID.String(),
			Title:             ev.Title,
			Description:       ev.Description,
			StartsAt:          ev.StartsAt,
			CapacityTotal:     ev.CapacityTotal,
			CapacityAvailable: ev.CapacityAvailable,
			Status:            string(ev.Status),
		})
	}

	_ = httpx.WriteJSON(w, http.StatusOK, ListEventsResponseDTO{
		Items:  resp,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// @Summary Get event by ID
// @Tags events
// @Accept json
// @Produce json
// @Param id path string true "Event ID (UUID)"
// @Success 200 {object} EventDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
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

	_ = httpx.WriteJSON(w, http.StatusOK, EventDTO{
		ID:                ev.ID.String(),
		Title:             ev.Title,
		Description:       ev.Description,
		StartsAt:          ev.StartsAt,
		CapacityTotal:     ev.CapacityTotal,
		CapacityAvailable: ev.CapacityAvailable,
		Status:            string(ev.Status),
	})
}

// @Summary Update event by ID
// @Tags events
// @Accept json
// @Produce json
// @Param id path string true "Event ID (UUID)"
// @Param request body UpdateEventRequestDTO true "Update event request"
// @Success 200 {object} EventDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
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

	ev, err := h.svc.Update(r.Context(), id, req.Title, req.Description, req.StartsAt, req.CapacityTotal, statusPatch)
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

	_ = httpx.WriteJSON(w, http.StatusOK, EventDTO{
		ID:                ev.ID.String(),
		Title:             ev.Title,
		Description:       ev.Description,
		StartsAt:          ev.StartsAt,
		CapacityTotal:     ev.CapacityTotal,
		CapacityAvailable: ev.CapacityAvailable,
		Status:            string(ev.Status),
	})
}

// @Summary Delete event by ID
// @Tags events
// @Accept json
// @Produce json
// @Param id path string true "Event ID (UUID)"
// @Success 204 "No Content"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
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

	w.WriteHeader(http.StatusNoContent)
}

