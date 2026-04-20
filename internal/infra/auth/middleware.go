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
				httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeMissingAuthorization, "missing Authorization header")
				return
			}
			parts := strings.SplitN(authz, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeInvalidAuthorization, "Authorization must be Bearer token")
				return
			}

			claims, err := jwt.ParseAccessToken(parts[1])
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeInvalidToken, "invalid or expired access token")
				return
			}

			effRoles, err := EffectiveAccessRoles(claims)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeInvalidTokenClaims, "invalid token claims")
				return
			}

			userIDStr := claims.UserID
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeInvalidTokenClaims, "invalid token claims")
				return
			}

			ctx := WithAccessClaims(r.Context(), userID, effRoles)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole allows the request if the JWT includes at least one of the allowed roles.
func RequireRole(allowed ...Role) func(http.Handler) http.Handler {
	allowedSet := make(map[Role]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, ok := RolesFromContext(r.Context())
			if !ok || len(roles) == 0 {
				httpx.WriteError(w, http.StatusUnauthorized, httpx.ErrCodeMissingRole, "missing user role")
				return
			}

			granted := false
			for _, rle := range roles {
				if _, ok := allowedSet[rle]; ok {
					granted = true
					break
				}
			}
			if !granted {
				httpx.WriteError(w, http.StatusForbidden, httpx.ErrCodeForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
