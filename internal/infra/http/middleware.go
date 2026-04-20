package httpx

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"log/slog"
)

var defaultCORSOrigins = []string{
	"http://localhost:3000",
	"http://localhost:5173",
}

// CORS allows browser requests from local dev frontends (React on :3000, Vite on :5173).
// Allowed methods: GET, POST, PUT, DELETE, OPTIONS; headers: Content-Type, Authorization.
func CORS() func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(defaultCORSOrigins))
	for _, o := range defaultCORSOrigins {
		allowed[o] = struct{}{}
	}
	const (
		allowMethods = "GET, POST, PUT, DELETE, OPTIONS"
		allowHeaders = "Content-Type, Authorization"
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", allowMethods)
			w.Header().Set("Access-Control-Allow-Headers", allowHeaders)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type ctxKey string

const requestIDKey ctxKey = "request_id"

func RequestID() func(http.Handler) http.Handler {
	return middleware.RequestID
}

func GetRequestID(r *http.Request) string {
	if v := r.Context().Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Logging returns chi middleware which logs method/path/status/latency with request id.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			reqID := middleware.GetReqID(r.Context())
			if reqID == "" {
				reqID = uuid.NewString()
			}

			logger.Info("http_request",
				"request_id", reqID,
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"latency_ms", float64(time.Since(start).Microseconds())/1000.0,
				"remote_ip", r.RemoteAddr,
			)
		})
	}
}

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					reqID := middleware.GetReqID(r.Context())
					if reqID == "" {
						reqID = uuid.NewString()
					}
					logger.Error("panic_recovered",
						"request_id", reqID,
						"panic", rec,
					)
					httpxWriteInternalServerError(w)
				}
			}()
			ctx := context.WithValue(r.Context(), requestIDKey, middleware.GetReqID(r.Context()))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SecurityHeaders adds a minimal set of hardening headers for browser clients.
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			next.ServeHTTP(w, r)
		})
	}
}

// RequestTimeout wraps each request context with a fixed deadline.
// The cancel is deferred so downstream DB calls observe ctx cancellation.
func RequestTimeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func httpxWriteInternalServerError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"internal server error"}}`))
}

