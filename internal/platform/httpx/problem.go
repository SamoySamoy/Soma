// Package httpx holds HTTP plumbing shared by every module: problem details,
// middleware, health checks and static file serving.
package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/log"
)

// ContentTypeProblem is the RFC 9457 media type.
const ContentTypeProblem = "application/problem+json"

// Problem is an RFC 9457 problem details body. It mirrors the Problem schema
// in api/openapi.yaml.
type Problem struct {
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Status    int               `json:"status"`
	Detail    string            `json:"detail,omitempty"`
	Instance  string            `json:"instance,omitempty"`
	Code      string            `json:"code"`
	RequestID string            `json:"request_id,omitempty"`
	Errors    map[string]string `json:"errors,omitempty"`
}

// StatusOf maps an error kind to its HTTP status.
func StatusOf(k apperr.Kind) int {
	switch k {
	case apperr.KindInvalid:
		return http.StatusUnprocessableEntity
	case apperr.KindUnauthorized:
		return http.StatusUnauthorized
	case apperr.KindForbidden:
		return http.StatusForbidden
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindConflict:
		return http.StatusConflict
	case apperr.KindPreconditionFailed:
		return http.StatusPreconditionFailed
	case apperr.KindRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// WriteError writes err as problem details. Internal errors are logged with
// their cause and shown to the client only as a generic message.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := apperr.As(err); ok && e.Kind != apperr.KindInternal {
		WriteProblem(w, r, Problem{
			Status: StatusOf(e.Kind),
			Code:   e.Code,
			Detail: e.Message,
			Errors: e.Fields,
		})
		return
	}
	log.FromContext(r.Context()).ErrorContext(r.Context(), "internal error", "error", err)
	WriteProblem(w, r, Problem{
		Status: http.StatusInternalServerError,
		Code:   "internal",
		Detail: "Something went wrong on our side. Try again later.",
	})
}

// WriteProblem writes p, filling in type, title and request ID when empty.
func WriteProblem(w http.ResponseWriter, r *http.Request, p Problem) {
	if p.Type == "" {
		p.Type = "about:blank"
	}
	if p.Title == "" {
		p.Title = http.StatusText(p.Status)
	}
	if p.RequestID == "" {
		p.RequestID = RequestIDFrom(r.Context())
	}
	w.Header().Set("Content-Type", ContentTypeProblem)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(p.Status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.FromContext(r.Context()).WarnContext(r.Context(), "write problem response", "error", err)
	}
}

// WriteJSON writes v as JSON with the given status.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.FromContext(r.Context()).WarnContext(r.Context(), "write json response", "error", err)
	}
}
