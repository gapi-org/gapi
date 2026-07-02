package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Recover converts panics into RFC 9457 Problem Details responses.
func Recover() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					writeProblem(w, problem{
						Type:   "about:blank",
						Title:  http.StatusText(http.StatusInternalServerError),
						Status: http.StatusInternalServerError,
						Detail: http.StatusText(http.StatusInternalServerError),
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID ensures each request has an X-Request-ID header and context value.
func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(requestIDHeader)
			if id == "" {
				id = newRequestID()
				r.Header.Set(requestIDHeader, id)
			}
			w.Header().Set(requestIDHeader, id)
			ctx := context.WithValue(r.Context(), requestIDContextKey{}, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestIDFromContext returns the request ID set by RequestID.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey{}).(string)
	return id
}

// Timeout limits the time a handler may spend serving a request.
func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, duration, http.StatusText(http.StatusServiceUnavailable))
	}
}

// BodyLimit caps the size of incoming request bodies.
func BodyLimit(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders adds conservative browser security headers.
func SecureHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			next.ServeHTTP(w, r)
		})
	}
}

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// CORS adds a small dependency-free CORS middleware.
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	origins := strings.Join(config.AllowedOrigins, ", ")
	methods := strings.Join(config.AllowedMethods, ", ")
	headers := strings.Join(config.AllowedHeaders, ", ")
	if origins == "" {
		origins = "*"
	}
	if methods == "" {
		methods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	}
	if headers == "" {
		headers = "Content-Type, Authorization"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origins)
			w.Header().Set("Access-Control-Allow-Methods", methods)
			w.Header().Set("Access-Control-Allow-Headers", headers)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeProblem(w http.ResponseWriter, problem problem) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(problem)
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(bytes[:])
}
