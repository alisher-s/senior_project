package authx

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
)

func AuthMiddleware(jwt JWT) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if authz == "" {
				_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
					Error: httpx.ErrorBody{Code: "missing_authorization", Message: "missing Authorization header"},
				})
				return
			}
			parts := strings.SplitN(authz, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
					Error: httpx.ErrorBody{Code: "invalid_authorization", Message: "Authorization must be Bearer token"},
				})
				return
			}

			claims, err := jwt.ParseAccessToken(parts[1])
			if err != nil {
				_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
					Error: httpx.ErrorBody{Code: "invalid_token", Message: "invalid or expired access token"},
				})
				return
			}

			role := claims.Role
			userIDStr := claims.UserID
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
					Error: httpx.ErrorBody{Code: "invalid_token_claims", Message: "invalid token claims"},
				})
				return
			}

			ctx := WithAccessClaims(r.Context(), userID, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole allows the request only for user roles included in allowed.
func RequireRole(allowed ...Role) func(http.Handler) http.Handler {
	allowedSet := make(map[Role]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := RoleFromContext(r.Context())
			if !ok {
				_ = httpx.WriteJSON(w, http.StatusUnauthorized, httpx.ErrorResponse{
					Error: httpx.ErrorBody{Code: "missing_role", Message: "missing user role"},
				})
				return
			}

			if _, ok := allowedSet[role]; !ok {
				_ = httpx.WriteJSON(w, http.StatusForbidden, httpx.ErrorResponse{
					Error: httpx.ErrorBody{Code: "forbidden", Message: "insufficient permissions"},
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

