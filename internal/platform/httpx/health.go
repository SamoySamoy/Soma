package httpx

import (
	"context"
	"net/http"
	"time"
)

// Readiness is what /readyz checks.
type Readiness interface {
	Ping(ctx context.Context) error
	SchemaVersion(ctx context.Context) (int64, error)
}

type healthBody struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Liveness reports that the process is up. It never touches dependencies.
func Liveness() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, r, http.StatusOK, healthBody{Status: "ok"})
	})
}

// ReadinessHandler reports whether the instance can serve traffic: the
// database answers and its schema is at the version this binary expects.
func ReadinessHandler(dep Readiness, wantSchema int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		checks := map[string]string{"database": "ok", "schema": "ok"}
		ready := true

		if err := dep.Ping(ctx); err != nil {
			checks["database"] = "unreachable"
			checks["schema"] = "unknown"
			ready = false
		} else if v, err := dep.SchemaVersion(ctx); err != nil {
			checks["schema"] = "unknown"
			ready = false
		} else if v != wantSchema {
			checks["schema"] = "migrations pending"
			ready = false
		}

		if !ready {
			WriteJSON(w, r, http.StatusServiceUnavailable, healthBody{Status: "unavailable", Checks: checks})
			return
		}
		WriteJSON(w, r, http.StatusOK, healthBody{Status: "ok", Checks: checks})
	})
}
