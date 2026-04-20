package rate_limit

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"github.com/nu/student-event-ticketing-platform/internal/config"
	httpx "github.com/nu/student-event-ticketing-platform/internal/infra/http"
)

func Middleware(rdb *redis.Client, cfg config.Config) func(http.Handler) http.Handler {
	limit := cfg.RateLimit.RequestsPerWindow

	// Atomic INCR+EXPIRE to avoid race on first request in a new window.
	// Redis Lua guarantees a single-threaded execution per instance.
	script := redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return current
`)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For "X-Forwarded-For", take the first ip; otherwise use r.RemoteAddr.
			ip := clientIP(r)
			if ip == "" {
				ip = "unknown"
			}

			routePattern := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if p := rc.RoutePattern(); p != "" {
					routePattern = p
				}
			}

			reqID := middleware.GetReqID(r.Context())
			key := "rate:" + ip + ":" + routePattern + ":v1"

			current, err := script.Run(r.Context(), rdb, []string{key}, int64(cfg.RateLimit.WindowSeconds)).Int64()
			if err != nil {
				// If rate limiting fails, fail-open to keep service usable.
				next.ServeHTTP(w, r)
				return
			}

			if current > int64(limit) {
				ttl := rdb.TTL(r.Context(), key).Val()
				retryAfter := int(ttl.Seconds())
				if retryAfter < 0 {
					retryAfter = 0
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				httpx.WriteError(w, http.StatusTooManyRequests, httpx.ErrCodeRateLimited, "too many requests")
				return
			}

			_ = reqID // reserved for future structured logging
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// XFF can contain a comma-separated list.
		for _, part := range strings.Split(xff, ",") {
			ip := strings.TrimSpace(part)
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

