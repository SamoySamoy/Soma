package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"testing/fstest"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/config"
	"github.com/SamoySamoy/Soma/internal/platform/httpx"
)

type readyFake struct{}

func (readyFake) Ping(context.Context) error                   { return nil }
func (readyFake) SchemaVersion(context.Context) (int64, error) { return 1, nil }

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	h, err := New(Deps{
		Config:        config.Config{Mode: config.ModeLocal, BaseURL: "http://localhost:8080"},
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Clock:         clock.NewFake(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)),
		Ready:         readyFake{},
		SchemaVersion: 1,
		Web:           fstest.MapFS{"index.html": {Data: []byte("<html>soma</html>")}},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// validateResponse checks a real response against the OpenAPI contract.
func validateResponse(t *testing.T, req *http.Request, res response) {
	t.Helper()
	spec, err := apigen.GetSpec()
	if err != nil {
		t.Fatalf("spec: %v", err)
	}
	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	route, params, err := router.FindRoute(req)
	if err != nil {
		t.Fatalf("find route: %v", err)
	}
	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route},
		Status:                 res.status,
		Header:                 res.header,
	}
	in.SetBodyBytes(res.body)
	if err := openapi3filter.ValidateResponse(t.Context(), in); err != nil {
		t.Errorf("response does not match the OpenAPI contract: %v", err)
	}
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func do(t *testing.T, srv *httptest.Server, method, path string) (*http.Request, response) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return req, response{status: res.StatusCode, header: res.Header, body: body}
}

func TestGetMeta(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)

	req, res := do(t, srv, http.MethodGet, "/api/v1/meta")
	if res.status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", res.status, res.body)
	}
	validateResponse(t, req, res)

	var meta apigen.Meta
	if err := json.Unmarshal(res.body, &meta); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if meta.Name != "Soma" || meta.Mode != apigen.Local {
		t.Errorf("meta = %+v, want name Soma, mode local", meta)
	}
}

func TestRouting(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)

	tests := []struct {
		name        string
		method      string
		path        string
		wantStatus  int
		wantProblem string
	}{
		{"liveness", http.MethodGet, "/healthz", 200, ""},
		{"readiness", http.MethodGet, "/readyz", 200, ""},
		{"web app", http.MethodGet, "/me/heart", 200, ""},
		{"unknown API path", http.MethodGet, "/api/v1/nope", 404, "request.not_found"},
		{"wrong method on API path", http.MethodDelete, "/api/v1/meta", 405, "request.method_not_allowed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, res := do(t, srv, tt.method, tt.path)
			if res.status != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", res.status, tt.wantStatus, res.body)
			}
			if res.header.Get(httpx.HeaderRequestID) == "" {
				t.Error("missing request ID header")
			}
			if res.header.Get("Content-Security-Policy") == "" {
				t.Error("missing security headers")
			}
			if tt.wantProblem == "" {
				return
			}
			var p httpx.Problem
			if err := json.Unmarshal(res.body, &p); err != nil {
				t.Fatalf("decode problem: %v", err)
			}
			if p.Code != tt.wantProblem {
				t.Errorf("problem code = %q, want %q", p.Code, tt.wantProblem)
			}
		})
	}
}

var (
	operationIDPattern = regexp.MustCompile(`^[a-z][A-Za-z0-9]+$`)
	permissionPattern  = regexp.MustCompile(`^(public|[a-z]+:[a-z_]+)$`)
)

// TestContractDeclaresPermissions enforces CLAUDE.md: every operation has a
// camelCase operationId and an x-soma-permission. It reads the source
// contract, because the embedded copy has operation IDs rewritten.
func TestContractDeclaresPermissions(t *testing.T) {
	t.Parallel()

	spec, err := openapi3.NewLoader().LoadFromFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatalf("load api/openapi.yaml: %v", err)
	}
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			where := method + " " + path
			if !operationIDPattern.MatchString(op.OperationID) {
				t.Errorf("%s: operationId %q is not camelCase", where, op.OperationID)
			}
			perm, ok := op.Extensions["x-soma-permission"].(string)
			if !ok || !permissionPattern.MatchString(perm) {
				t.Errorf("%s: x-soma-permission = %v, want \"public\" or \"module:action\"", where, op.Extensions["x-soma-permission"])
			}
		}
	}
}
