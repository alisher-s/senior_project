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
	"github.com/redis/go-redis/v9"
	"log/slog"

	"github.com/nu/student-event-ticketing-platform/admin/repository"
	"github.com/nu/student-event-ticketing-platform/admin/service"
	authrepo "github.com/nu/student-event-ticketing-platform/auth/repository"
	eventsrepo "github.com/nu/student-event-ticketing-platform/events/repository"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
	notificationsrepo "github.com/nu/student-event-ticketing-platform/notifications/repository"
	notificationssvc "github.com/nu/student-event-ticketing-platform/notifications/service"
)

type Deps struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	JWT    authx.JWT
	Logger *slog.Logger
}

func RegisterRoutes(r chi.Router, deps Deps) {
	_ = deps.Redis

	userRepo := authrepo.NewPostgres(deps.DB)
	eventRepo := eventsrepo.NewPostgres(deps.DB)
	modRepo := repository.NewPostgres(deps.DB)
	notifier := notificationssvc.New(notificationsrepo.NewPostgres(deps.DB))
	log := deps.Logger
	if log == nil {
		log = slog.Default()
	}
	svc := service.New(userRepo, eventRepo, modRepo, notifier)
	h := &handler{svc: svc, v: validator.New(), logger: log}

	r.Route("/admin", func(r chi.Router) {
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleAdmin)).Get("/events", h.handleListEvents)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleAdmin)).Get("/users", h.handleListUsers)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleAdmin)).Post("/events/{id}/moderate", h.handleModerateEvent)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleAdmin)).Patch("/users/{id}/role", h.handleSetUserRole)
		r.With(authx.AuthMiddleware(deps.JWT), authx.RequireRole(authx.RoleAdmin)).Get("/moderation-logs", h.handleListModerationLogs)
	})
}

type handler struct {
	svc    *service.Service
	v      *validator.Validate
	logger *slog.Logger
}

type ModerateEventRequestDTO struct {
	Action string `json:"action" validate:"required,oneof=approve reject"`
	Reason string `json:"reason" validate:"omitempty,max=2000"`
}

type ModerateEventResponseDTO struct {
	ModerationStatus string `json:"moderation_status"`
}

type SetUserRoleRequestDTO struct {
	Role string `json:"role" validate:"required,oneof=student organizer admin"`
}

type UserRoleResponseDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AdminUserDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AdminListUsersResponseDTO struct {
	Items  []AdminUserDTO `json:"items"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type ModerationLogEntryDTO struct {
	ID          string  `json:"id"`
	AdminUserID string  `json:"admin_user_id"`
	EventID     *string `json:"event_id,omitempty"`
	Action      string  `json:"action"`
	Reason      *string `json:"reason,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type ModerationLogsResponseDTO struct {
	Items  []ModerationLogEntryDTO `json:"items"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
}

type AdminEventDTO struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	StartsAt         string  `json:"starts_at"`
	ModerationStatus string  `json:"moderation_status"`
	OrganizerID      *string `json:"organizer_id,omitempty"`
}

type AdminListEventsResponseDTO struct {
	Items  []AdminEventDTO `json:"items"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// @Summary Set user role (admin only)
// @Description **401** — missing/invalid JWT; **403** — `forbidden` if caller is not **admin**.
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (admin)"
// @Param id path string true "User ID (UUID)"
// @Param request body SetUserRoleRequestDTO true "New role"
// @Success 200 {object} UserRoleResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/users/{id}/role [patch]
func (h *handler) handleSetUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(chi.URLParam(r, "id"))
	userID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidID, "invalid user id")
		return
	}

	var req SetUserRoleRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}

	adminID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	u, err := h.svc.SetUserRole(r.Context(), adminID, userID, req.Role, h.logger)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		if status >= 500 {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to update role")
			return
		}
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, UserRoleResponseDTO{
		ID:    u.ID.String(),
		Email: u.Email,
		Role:  string(u.Role),
	})
}

// @Summary List/search users by email (admin only)
// @Description Returns basic user info for admin user management. Search is a substring match on email.
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer access token (admin)"
// @Param q query string false "Search by email (substring match)"
// @Param limit query int false "Page size (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} AdminListUsersResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/users [get]
func (h *handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	q := strings.TrimSpace(qp.Get("q"))

	limit := 20
	if s := qp.Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 1 || v > 100 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid limit")
			return
		}
		limit = v
	}

	offset := 0
	if s := qp.Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 || v > 100000 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid offset")
			return
		}
		offset = v
	}

	items, err := h.svc.ListUsers(r.Context(), q, limit, offset)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to list users")
		return
	}

	out := make([]AdminUserDTO, 0, len(items))
	for _, u := range items {
		out = append(out, AdminUserDTO{
			ID:    u.ID.String(),
			Email: u.Email,
			Role:  string(u.Role),
		})
	}

	_ = httpx.WriteJSON(w, http.StatusOK, AdminListUsersResponseDTO{
		Items:  out,
		Limit:  limit,
		Offset: offset,
	})
}

// @Summary Moderate event visibility (admin only)
// @Description **401** / **403** — same as other admin routes (JWT + admin role).
// @Tags admin
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token (admin)"
// @Param id path string true "Event ID (UUID)"
// @Param request body ModerateEventRequestDTO true "approve or reject"
// @Success 200 {object} ModerateEventResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/events/{id}/moderate [post]
func (h *handler) handleModerateEvent(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(chi.URLParam(r, "id"))
	if eventID == "" {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidID, "missing event id")
		return
	}

	adminID, ok := authx.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeUnauthorized, "missing user id")
		return
	}

	var req ModerateEventRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, err.Error())
		return
	}

	st, err := h.svc.ModerateEvent(r.Context(), adminID, eventID, req.Action, req.Reason, h.logger)
	if err != nil {
		status, apiErr := httpx.MapDomainError(err)
		if status >= 500 {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "moderation failed")
			return
		}
		httpx.WriteError(w, status, apiErr.Code, apiErr.Message)
		return
	}

	_ = httpx.WriteJSON(w, http.StatusOK, ModerateEventResponseDTO{
		ModerationStatus: string(st),
	})
}

// @Summary List moderation audit logs (admin only)
// @Description **401** / **403** — JWT + **admin** only.
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer access token (admin)"
// @Param event_id query string false "Filter by event UUID"
// @Param admin_id query string false "Filter by acting admin user UUID"
// @Param limit query int false "Page size (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} ModerationLogsResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/moderation-logs [get]
func (h *handler) handleListModerationLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := repository.ModerationLogFilter{}

	if s := strings.TrimSpace(q.Get("event_id")); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid event_id")
			return
		}
		filter.EventID = &id
	}
	if s := strings.TrimSpace(q.Get("admin_id")); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid admin_id")
			return
		}
		filter.AdminID = &id
	}

	limit := 20
	if s := q.Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 1 || v > 100 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid limit")
			return
		}
		limit = v
	}
	filter.Limit = limit

	offset := 0
	if s := q.Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 || v > 100000 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid offset")
			return
		}
		offset = v
	}
	filter.Offset = offset

	items, err := h.svc.ListModerationLogs(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to list moderation logs")
		return
	}

	out := make([]ModerationLogEntryDTO, 0, len(items))
	for _, m := range items {
		var ev *string
		if m.EventID != nil {
			s := m.EventID.String()
			ev = &s
		}
		out = append(out, ModerationLogEntryDTO{
			ID:          m.ID.String(),
			AdminUserID: m.AdminUserID.String(),
			EventID:     ev,
			Action:      m.Action,
			Reason:      m.Reason,
			CreatedAt:   m.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}

	_ = httpx.WriteJSON(w, http.StatusOK, ModerationLogsResponseDTO{
		Items:  out,
		Limit:  limit,
		Offset: offset,
	})
}

// @Summary List events for admin moderation and browsing (admin only)
// @Description Returns events for admin tooling. Default moderation_status is `pending` when omitted.
// @Tags admin
// @Produce json
// @Param Authorization header string true "Bearer access token (admin)"
// @Param moderation_status query string false "Filter by moderation status: pending|approved|rejected (default pending)"
// @Param q query string false "Search by title (substring match)"
// @Param limit query int false "Page size (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} AdminListEventsResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /admin/events [get]
func (h *handler) handleListEvents(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	q := strings.TrimSpace(qp.Get("q"))

	ms := strings.ToLower(strings.TrimSpace(qp.Get("moderation_status")))
	if ms == "" {
		ms = "pending"
	}
	switch ms {
	case "pending", "approved", "rejected":
	default:
		httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid moderation_status")
		return
	}

	limit := 20
	if s := qp.Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 1 || v > 100 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid limit")
			return
		}
		limit = v
	}

	offset := 0
	if s := qp.Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 || v > 100000 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.ErrCodeInvalidRequest, "invalid offset")
			return
		}
		offset = v
	}

	filter := eventsrepo.EventFilter{
		Query:                     q,
		ModerationStatus:          &ms,
		RequireApprovedModeration: false,
		Limit:                     limit,
		Offset:                    offset,
	}

	items, err := h.svc.ListEvents(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, httpx.ErrCodeInternalError, "failed to list events")
		return
	}

	out := make([]AdminEventDTO, 0, len(items))
	for _, ev := range items {
		var org *string
		if ev.OrganizerID != nil {
			s := ev.OrganizerID.String()
			org = &s
		}
		out = append(out, AdminEventDTO{
			ID:               ev.ID.String(),
			Title:            ev.Title,
			StartsAt:         ev.StartsAt.UTC().Format(time.RFC3339),
			ModerationStatus: string(ev.ModerationStatus),
			OrganizerID:      org,
		})
	}

	_ = httpx.WriteJSON(w, http.StatusOK, AdminListEventsResponseDTO{
		Items:  out,
		Limit:  limit,
		Offset: offset,
	})
}
