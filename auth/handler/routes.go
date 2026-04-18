package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"

	"github.com/nu/student-event-ticketing-platform/internal/config"
	authx "github.com/nu/student-event-ticketing-platform/internal/infra/auth"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"

	authrepo "github.com/nu/student-event-ticketing-platform/auth/repository"
	"github.com/nu/student-event-ticketing-platform/auth/service"
)

type Deps struct {
	Cfg    config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
	JWT    authx.JWT
	Logger *slog.Logger
}

func RegisterRoutes(r chi.Router, deps Deps) {
	userRepo := authrepo.NewPostgres(deps.DB)
	svc := service.New(deps.Cfg, userRepo, userRepo, deps.JWT)

	h := &handler{
		deps: deps,
		svc:  svc,
		v:    validator.New(),
	}

	r.Group(func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", h.handleRegister)
			r.Post("/login", h.handleLogin)
			r.Post("/refresh", h.handleRefresh)
			r.With(authx.AuthMiddleware(deps.JWT)).Patch("/me/roles", h.handlePatchMeRoles)
		})
	})
}

type handler struct {
	deps Deps
	svc  *service.Service
	v    *validator.Validate
}

// @Summary Register a new user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequestDTO true "Register request"
// @Success 201 {object} AuthResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse "email already registered (code: email_exists)"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/register [post]
func (h *handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	user, access, refresh, err := h.svc.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, AuthResponseDTO{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         userToDTO(user),
	})
}

// @Summary Login and get tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequestDTO true "Login request"
// @Success 200 {object} AuthResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "invalid_credentials"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/login [post]
func (h *handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	user, access, refresh, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, AuthResponseDTO{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         userToDTO(user),
	})
}

// @Summary Refresh access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequestDTO true "Refresh request"
// @Success 200 {object} AuthResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "invalid_refresh_token, refresh_token_consumed"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/refresh [post]
func (h *handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
		})
		return
	}

	user, access, refresh, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, AuthResponseDTO{
		AccessToken:  access,
		RefreshToken: refresh,
		User:         userToDTO(user),
	})
}

// @Summary Request organizer role (pending until admin approves)
// @Description Student-only flow: body must be exactly `{"roles":["organizer"]}`. **401** — missing/invalid JWT (same codes as middleware). **403** — `organizer_request_forbidden` if caller is not an active student. **409** — `organizer_already_active` if organizer is already granted.
// @Tags auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param request body PatchMeRolesRequestDTO true "e.g. {\"roles\":[\"organizer\"]}"
// @Success 200 {object} MeRolesResponseDTO
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse "missing_authorization, invalid_token, invalid_token_claims, …"
// @Failure 403 {object} httpx.ErrorResponse "organizer_request_forbidden (RBAC / business rule)"
// @Failure 409 {object} httpx.ErrorResponse "organizer_already_active"
// @Failure 500 {object} httpx.ErrorResponse
// @Router /auth/me/roles [patch]
func (h *handler) handlePatchMeRoles(w http.ResponseWriter, r *http.Request) {
	var req PatchMeRolesRequestDTO
	if err := httpx.DecodeAndValidate(r, &req, h.v); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: err.Error()},
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

	if err := h.svc.RequestOrganizerRole(r.Context(), userID, req.Roles); err != nil {
		writeServiceError(w, err)
		return
	}

	u, err := h.svc.UserByID(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, MeRolesResponseDTO{User: userToDTO(u)})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEmailNotAllowed):
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "email_not_allowed", Message: "email domain is not allowed"},
		})
	case errors.Is(err, service.ErrEmailAlreadyExists):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "email_exists", Message: "email already exists"},
		})
	case errors.Is(err, service.ErrInvalidCredentials):
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_credentials", Message: "invalid email or password"},
		})
	case errors.Is(err, service.ErrRefreshTokenInvalid):
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_refresh_token", Message: "invalid refresh token"},
		})
	case errors.Is(err, service.ErrRefreshTokenConsumed):
		_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "refresh_token_consumed", Message: "refresh token already used"},
		})
	case errors.Is(err, service.ErrOrganizerAlreadyActive):
		_ = httpx.WriteJSON(w, http.StatusConflict, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "organizer_already_active", Message: "organizer role is already active"},
		})
	case errors.Is(err, service.ErrOrganizerRequestNotAllowed):
		_ = httpx.WriteJSON(w, http.StatusForbidden, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "organizer_request_forbidden", Message: "only active students may request organizer role"},
		})
	case errors.Is(err, service.ErrOrganizerRequestInvalidBody):
		_ = httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "invalid_request", Message: "request exactly {\"roles\":[\"organizer\"]}"},
		})
	case errors.Is(err, authrepo.ErrUserNotFound):
		_ = httpx.WriteJSON(w, http.StatusNotFound, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "not_found", Message: "user not found"},
		})
	default:
		_ = httpx.WriteJSON(w, http.StatusInternalServerError, httpx.ErrorResponse{
			Error: httpx.ErrorBody{Code: "internal_error", Message: "internal server error"},
		})
	}
}
