package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
)

func decodeProblem(t *testing.T, res *http.Response) Problem {
	t.Helper()
	if ct := res.Header.Get("Content-Type"); ct != ContentTypeProblem {
		t.Fatalf("Content-Type = %q, want %q", ct, ContentTypeProblem)
	}
	var p Problem
	if err := json.NewDecoder(res.Body).Decode(&p); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	return p
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want Problem
	}{
		{
			name: "validation error keeps fields",
			err:  apperr.Invalid("money.zero_amount", "Amount can't be zero.", map[string]string{"amount_minor": "must not be zero"}),
			want: Problem{Type: "about:blank", Title: "Unprocessable Entity", Status: 422, Code: "money.zero_amount",
				Detail: "Amount can't be zero.", Errors: map[string]string{"amount_minor": "must not be zero"}},
		},
		{
			name: "not found",
			err:  apperr.NotFound("people.contact_not_found", "Contact not found."),
			want: Problem{Type: "about:blank", Title: "Not Found", Status: 404, Code: "people.contact_not_found", Detail: "Contact not found."},
		},
		{
			name: "plain error hides details",
			err:  errors.New("pq: password authentication failed for user soma"),
			want: Problem{Type: "about:blank", Title: "Internal Server Error", Status: 500, Code: "internal",
				Detail: "Something went wrong on our side. Try again later."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			WriteError(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil), tt.err)
			got := decodeProblem(t, rec.Result())
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("problem mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestStatusOfCoversEveryKind(t *testing.T) {
	t.Parallel()

	want := map[apperr.Kind]int{
		apperr.KindInternal:           500,
		apperr.KindInvalid:            422,
		apperr.KindUnauthorized:       401,
		apperr.KindForbidden:          403,
		apperr.KindNotFound:           404,
		apperr.KindConflict:           409,
		apperr.KindPreconditionFailed: 412,
		apperr.KindRateLimited:        429,
	}
	for k, status := range want {
		if got := StatusOf(k); got != status {
			t.Errorf("StatusOf(%v) = %d, want %d", k, got, status)
		}
	}
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	var seen string
	h := RequestID(discardLogger())(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = RequestIDFrom(r.Context())
	}))

	t.Run("generates when missing", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
		if len(seen) != 32 {
			t.Errorf("generated ID %q, want 32 hex characters", seen)
		}
		if rec.Header().Get(HeaderRequestID) != seen {
			t.Error("response header does not match context ID")
		}
	})

	t.Run("reuses a valid incoming ID", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set(HeaderRequestID, "abc-123-def")
		h.ServeHTTP(httptest.NewRecorder(), req)
		if seen != "abc-123-def" {
			t.Errorf("ID = %q, want abc-123-def", seen)
		}
	})

	t.Run("replaces a malformed incoming ID", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		req.Header.Set(HeaderRequestID, "<script>")
		h.ServeHTTP(httptest.NewRecorder(), req)
		if seen == "<script>" {
			t.Error("malformed request ID was accepted")
		}
	})
}

func TestRecoverReturnsProblem(t *testing.T) {
	t.Parallel()

	h := Recover()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))

	p := decodeProblem(t, rec.Result())
	if p.Status != 500 || p.Code != "internal" {
		t.Errorf("problem = %+v, want status 500 code internal", p)
	}
}

func TestAccessLogRecordsStatusAndDuration(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	clk := clock.NewFake(time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC))

	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		clk.Advance(42 * time.Millisecond)
		w.WriteHeader(http.StatusTeapot)
	}), RequestID(logger), AccessLog(clk))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x?email=a@b.c", nil))

	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("decode log line: %v", err)
	}
	if line["status"] != float64(418) || line["duration_ms"] != float64(42) || line["path"] != "/x" {
		t.Errorf("log line = %v, want status 418, duration 42, path /x", line)
	}
	if strings.Contains(buf.String(), "a@b.c") {
		t.Error("query string leaked into the access log")
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hsts     bool
		wantHSTS bool
	}{
		{"without https", false, false},
		{"with https", true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			SecurityHeaders(tt.hsts)(http.NotFoundHandler()).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
			h := rec.Header()
			if !strings.Contains(h.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
				t.Errorf("CSP = %q, missing frame-ancestors", h.Get("Content-Security-Policy"))
			}
			if h.Get("X-Content-Type-Options") != "nosniff" {
				t.Error("missing X-Content-Type-Options")
			}
			if got := h.Get("Strict-Transport-Security") != ""; got != tt.wantHSTS {
				t.Errorf("HSTS present = %v, want %v", got, tt.wantHSTS)
			}
		})
	}
}

func TestCrossOriginRejectsCrossSiteWrites(t *testing.T) {
	t.Parallel()

	h := CrossOrigin()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	tests := []struct {
		name     string
		method   string
		fetch    string
		wantCode int
	}{
		{"same-origin POST passes", http.MethodPost, "same-origin", http.StatusNoContent},
		{"cross-site POST is rejected", http.MethodPost, "cross-site", http.StatusForbidden},
		{"cross-site GET passes", http.MethodGet, "cross-site", http.StatusNoContent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(t.Context(), tt.method, "/api/v1/x", nil)
			req.Header.Set("Sec-Fetch-Site", tt.fetch)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}
}

type fakeReadiness struct {
	pingErr error
	version int64
	verErr  error
}

func (f fakeReadiness) Ping(context.Context) error                   { return f.pingErr }
func (f fakeReadiness) SchemaVersion(context.Context) (int64, error) { return f.version, f.verErr }

func TestReadiness(t *testing.T) {
	t.Parallel()

	const want = int64(20261005000000)
	tests := []struct {
		name       string
		dep        fakeReadiness
		wantStatus int
		wantSchema string
	}{
		{"ready", fakeReadiness{version: want}, 200, "ok"},
		{"database down", fakeReadiness{pingErr: errors.New("refused")}, 503, "unknown"},
		{"migrations pending", fakeReadiness{version: want - 1}, 503, "migrations pending"},
		{"version query fails", fakeReadiness{verErr: errors.New("no table")}, 503, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			ReadinessHandler(tt.dep, want).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var body healthBody
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Checks["schema"] != tt.wantSchema {
				t.Errorf("schema check = %q, want %q", body.Checks["schema"], tt.wantSchema)
			}
		})
	}
}

func TestSPA(t *testing.T) {
	t.Parallel()

	built := fstest.MapFS{
		"index.html":         {Data: []byte("<html>soma</html>")},
		"assets/app-1a2b.js": {Data: []byte("console.log(1)")},
		"favicon.svg":        {Data: []byte("<svg/>")},
	}

	tests := []struct {
		name      string
		fsys      fstest.MapFS
		method    string
		path      string
		wantCode  int
		wantBody  string
		wantCache string
	}{
		{"index at root", built, http.MethodGet, "/", 200, "<html>soma</html>", "no-cache"},
		{"client route falls back to index", built, http.MethodGet, "/me/heart", 200, "<html>soma</html>", "no-cache"},
		{"hashed asset is immutable", built, http.MethodGet, "/assets/app-1a2b.js", 200, "console.log(1)", "public, max-age=31536000, immutable"},
		{"other file is revalidated", built, http.MethodGet, "/favicon.svg", 200, "<svg/>", "no-cache"},
		{"writes are not allowed", built, http.MethodPost, "/", 405, "", ""},
		{"unbuilt app explains itself", fstest.MapFS{}, http.MethodGet, "/", 503, "web:build", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			SPA(tt.fsys).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, nil))
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want it to contain %q", rec.Body.String(), tt.wantBody)
			}
			if tt.wantCache != "" && rec.Header().Get("Cache-Control") != tt.wantCache {
				t.Errorf("Cache-Control = %q, want %q", rec.Header().Get("Cache-Control"), tt.wantCache)
			}
		})
	}
}
