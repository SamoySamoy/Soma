package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"

	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/log"
)

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so that the first one listed runs first.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// HeaderRequestID carries the request ID in requests and responses.
const HeaderRequestID = "X-Request-ID"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

type requestIDKey struct{}

// RequestIDFrom returns the request ID stored by RequestID.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// RequestID assigns each request an ID, reusing a well-formed incoming one,
// and stores a logger tagged with it in the context.
func RequestID(base *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(HeaderRequestID)
			if !validRequestID.MatchString(id) {
				id = newRequestID()
			}
			w.Header().Set(HeaderRequestID, id)
			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			ctx = log.WithLogger(ctx, base.With("request_id", id))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error
	return hex.EncodeToString(b)
}

// Recover turns a panic into a 500 problem response and logs the stack.
func Recover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() { //nolint:contextcheck // writes the response with the request's own context
				p := recover()
				if p == nil {
					return
				}
				if p == http.ErrAbortHandler { //nolint:errorlint // sentinel compared by identity, as net/http does
					panic(p)
				}
				log.FromContext(r.Context()).ErrorContext(r.Context(), "panic serving request",
					"panic", p, "stack", string(debug.Stack()))
				WriteProblem(w, r, Problem{
					Status: http.StatusInternalServerError,
					Code:   "internal",
					Detail: "Something went wrong on our side. Try again later.",
				})
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// AccessLog logs one line per request: method, path (without query string,
// which may carry personal data), status, size and duration.
func AccessLog(clk clock.Clock) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := clk.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			if rec.status == 0 {
				rec.status = http.StatusOK
			}
			log.FromContext(r.Context()).InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration_ms", clk.Now().Sub(start).Milliseconds(),
			)
		})
	}
}

// contentSecurityPolicy allows only same-origin resources. Inline styles are
// needed by Mantine's CSS variables; inline scripts are never allowed.
const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; object-src 'none'; " +
	"base-uri 'none'; frame-ancestors 'none'; form-action 'self'"

// SecurityHeaders sets defensive headers on every response. hsts adds
// Strict-Transport-Security and must be true only when served over HTTPS.
func SecurityHeaders(hsts bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", contentSecurityPolicy)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "same-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("X-Frame-Options", "DENY")
			if hsts {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CrossOrigin rejects cross-origin state-changing requests using Go's
// built-in protection (Sec-Fetch-Site and Origin checks).
func CrossOrigin() Middleware {
	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteProblem(w, r, Problem{
			Status: http.StatusForbidden,
			Code:   "request.cross_origin",
			Detail: "Cross-origin requests are not allowed.",
		})
	}))
	return cop.Handler
}
