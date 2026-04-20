package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// APIError is the single machine-readable error payload used across the API.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error APIError `json:"error"`
}

const (
	// Generic.
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeInvalidID      = "INVALID_ID"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeForbidden      = "FORBIDDEN"
	ErrCodeNotImplemented = "NOT_IMPLEMENTED"
	ErrCodeRateLimited    = "RATE_LIMITED"
	ErrCodeRequestTimeout = "REQUEST_TIMEOUT"
	ErrCodeMissingAuthorization = "MISSING_AUTHORIZATION"
	ErrCodeInvalidAuthorization = "INVALID_AUTHORIZATION"
	ErrCodeInvalidToken         = "INVALID_TOKEN"
	ErrCodeInvalidTokenClaims   = "INVALID_TOKEN_CLAIMS"
	ErrCodeMissingRole          = "MISSING_ROLE"
	ErrCodeMissingSignature     = "MISSING_SIGNATURE"
	ErrCodeInvalidSignature     = "INVALID_SIGNATURE"

	// Auth.
	ErrCodeEmailNotAllowed        = "EMAIL_NOT_ALLOWED"
	ErrCodeEmailExists            = "EMAIL_EXISTS"
	ErrCodeInvalidCredentials     = "INVALID_CREDENTIALS"
	ErrCodeInvalidRefreshToken    = "INVALID_REFRESH_TOKEN"
	ErrCodeRefreshTokenConsumed   = "REFRESH_TOKEN_CONSUMED"
	ErrCodeOrganizerAlreadyActive = "ORGANIZER_ALREADY_ACTIVE"
	ErrCodeOrganizerRequestDenied = "ORGANIZER_REQUEST_FORBIDDEN"

	// Ticketing.
	ErrCodeTicketCapacityExceeded  = "TICKET_CAPACITY_EXCEEDED"
	ErrCodeTicketAlreadyRegistered = "TICKET_ALREADY_REGISTERED"
	ErrCodeEventNotPublished       = "EVENT_NOT_PUBLISHED"
	ErrCodeEventNotApproved        = "EVENT_NOT_APPROVED"
	ErrCodeEventCancelled          = "EVENT_CANCELLED"
	ErrCodeRegistrationClosed      = "REGISTRATION_CLOSED"
	ErrCodeCancellationNotAllowed  = "CANCELLATION_NOT_ALLOWED"
	ErrCodeCheckInNotOpen          = "CHECK_IN_NOT_OPEN"
	ErrCodeTicketNotFound          = "TICKET_NOT_FOUND"
	ErrCodeTicketAlreadyCancelled  = "TICKET_ALREADY_CANCELLED"
	ErrCodeTicketAlreadyUsed       = "TICKET_ALREADY_USED"
	ErrCodeTicketCannotBeUsed      = "TICKET_CANNOT_BE_USED"
	ErrCodeTicketExpired           = "ticket_expired"

	// Admin.
	ErrCodeInvalidRole   = "INVALID_ROLE"
	ErrCodeInvalidAction = "INVALID_ACTION"
)

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// WriteError writes the unified error envelope.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	if code == "" {
		code = ErrCodeInternalError
	}
	if message == "" {
		message = "internal server error"
	}
	_ = WriteJSON(w, status, ErrorResponse{
		Error: APIError{Code: code, Message: message},
	})
}

// MapDomainError maps known domain/repository/service errors to an HTTP status + APIError.
// Unknown errors map to 500 INTERNAL_ERROR.
func MapDomainError(err error) (int, APIError) {
	if err == nil {
		return http.StatusInternalServerError, APIError{Code: ErrCodeInternalError, Message: "internal server error"}
	}

	// Context/timeouts.
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusRequestTimeout, APIError{Code: ErrCodeRequestTimeout, Message: "request timed out"}
	}
	if strings.Contains(strings.ToLower(err.Error()), "context deadline exceeded") {
		return http.StatusRequestTimeout, APIError{Code: ErrCodeRequestTimeout, Message: "request timed out"}
	}

	// Domain mapping without importing domain packages (avoids import cycles).
	// Convention: domain packages should use stable sentinel error messages for MapDomainError.
	switch strings.TrimSpace(err.Error()) {
	// Auth.
	case "email domain not allowed":
		return http.StatusBadRequest, APIError{Code: ErrCodeEmailNotAllowed, Message: "email domain is not allowed"}
	case "email already exists", "email already exists ":
		return http.StatusConflict, APIError{Code: ErrCodeEmailExists, Message: "email already exists"}
	case "invalid credentials":
		return http.StatusUnauthorized, APIError{Code: ErrCodeInvalidCredentials, Message: "invalid email or password"}
	case "invalid refresh token":
		return http.StatusUnauthorized, APIError{Code: ErrCodeInvalidRefreshToken, Message: "invalid refresh token"}
	case "refresh token revoked or expired":
		return http.StatusUnauthorized, APIError{Code: ErrCodeRefreshTokenConsumed, Message: "refresh token already used"}
	case "organizer role already active":
		return http.StatusConflict, APIError{Code: ErrCodeOrganizerAlreadyActive, Message: "organizer role is already active"}
	case "only active students may request organizer role":
		return http.StatusForbidden, APIError{Code: ErrCodeOrganizerRequestDenied, Message: "only active students may request organizer role"}
	case "invalid roles request":
		return http.StatusBadRequest, APIError{Code: ErrCodeInvalidRequest, Message: "request exactly {\"roles\":[\"organizer\"]}"}
	case "user not found":
		return http.StatusNotFound, APIError{Code: ErrCodeNotFound, Message: "user not found"}

	// Events.
	case "event not found", "not found":
		return http.StatusNotFound, APIError{Code: ErrCodeNotFound, Message: "event not found"}

	// Ticketing.
	case "capacity full", "event capacity full":
		return http.StatusConflict, APIError{Code: ErrCodeTicketCapacityExceeded, Message: "event capacity is full"}
	case "already registered":
		return http.StatusConflict, APIError{Code: ErrCodeTicketAlreadyRegistered, Message: "ticket already exists"}
	case "event is not published":
		return http.StatusConflict, APIError{Code: ErrCodeEventNotPublished, Message: "event is not open for registration"}
	case "event is not approved for registration", "event is not approved for listing":
		return http.StatusConflict, APIError{Code: ErrCodeEventNotApproved, Message: "event is not approved for registration"}
	case "event is cancelled":
		return http.StatusConflict, APIError{Code: ErrCodeEventCancelled, Message: "event is cancelled"}
	case "event registration is closed":
		return http.StatusConflict, APIError{Code: ErrCodeRegistrationClosed, Message: "registration is closed for this event"}
	case "ticket cannot be cancelled after event start":
		return http.StatusConflict, APIError{Code: ErrCodeCancellationNotAllowed, Message: "ticket cannot be cancelled after the event has started"}
	case "ticket expired":
		return http.StatusBadRequest, APIError{Code: ErrCodeTicketExpired, Message: "ticket has expired"}
	case "check-in is not open yet", "check-in is not open yet for this event":
		return http.StatusConflict, APIError{Code: ErrCodeCheckInNotOpen, Message: "check-in is not open yet for this event"}
	case "ticket not found":
		return http.StatusNotFound, APIError{Code: ErrCodeTicketNotFound, Message: "ticket not found"}
	case "ticket already cancelled":
		return http.StatusConflict, APIError{Code: ErrCodeTicketAlreadyCancelled, Message: "ticket already cancelled"}
	case "ticket already used":
		return http.StatusConflict, APIError{Code: ErrCodeTicketAlreadyUsed, Message: "ticket already used"}
	case "ticket cannot be used":
		return http.StatusConflict, APIError{Code: ErrCodeTicketCannotBeUsed, Message: "ticket cannot be used"}

	// Admin / analytics / notifications.
	case "invalid role":
		return http.StatusBadRequest, APIError{Code: ErrCodeInvalidRole, Message: "invalid role"}
	case "invalid event id":
		return http.StatusBadRequest, APIError{Code: ErrCodeInvalidID, Message: "invalid event id"}
	case "invalid moderation action":
		return http.StatusBadRequest, APIError{Code: ErrCodeInvalidAction, Message: "invalid action"}
	case "forbidden":
		return http.StatusForbidden, APIError{Code: ErrCodeForbidden, Message: "forbidden"}
	case "not implemented":
		return http.StatusNotImplemented, APIError{Code: ErrCodeNotImplemented, Message: "not implemented"}
	}

	return http.StatusInternalServerError, APIError{Code: ErrCodeInternalError, Message: "internal server error"}
}

