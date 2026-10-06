//go:build integration

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/modules/people"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/config"
	"github.com/SamoySamoy/Soma/internal/platform/db"
	"github.com/SamoySamoy/Soma/internal/platform/httpx"
	"github.com/SamoySamoy/Soma/internal/space"
	"github.com/SamoySamoy/Soma/internal/testutil/pgtest"
)

// newLocalServer starts the real handler against a migrated database.
func newLocalServer(t *testing.T) *httptest.Server {
	t.Helper()
	dbs := pgtest.New(t)
	d, err := db.Open(t.Context(), dbs.AppURL)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(d.Close)

	clk := clock.NewFake(time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC))
	if err := space.EnsureLocal(t.Context(), d, clk); err != nil {
		t.Fatalf("EnsureLocal: %v", err)
	}
	h, err := New(Deps{
		Config:        config.Config{Mode: config.ModeLocal, BaseURL: "http://localhost:8080"},
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Clock:         clk,
		Ready:         d,
		SchemaVersion: 0,
		Web:           fstest.MapFS{},
		People:        people.NewService(d, clk),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// call sends a request and checks the response against the OpenAPI contract.
func call(t *testing.T, srv *httptest.Server, method, path string, body any, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, srv.URL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	validateAgainstContract(t, req, res.StatusCode, res.Header, raw)
	return res, raw
}

func validateAgainstContract(t *testing.T, req *http.Request, status int, header http.Header, body []byte) {
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
		t.Fatalf("find route for %s %s: %v", req.Method, req.URL.Path, err)
	}
	in := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route},
		Status:                 status,
		Header:                 header,
	}
	in.SetBodyBytes(body)
	if err := openapi3filter.ValidateResponse(t.Context(), in); err != nil {
		t.Errorf("%s %s (%d) does not match the contract: %v", req.Method, req.URL.Path, status, err)
	}
}

func problemCode(t *testing.T, raw []byte) string {
	t.Helper()
	var p httpx.Problem
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("decode problem %s: %v", raw, err)
	}
	return p.Code
}

func TestContactsOverHTTP(t *testing.T) {
	t.Parallel()
	srv := newLocalServer(t)

	// Create.
	//nolint:bodyclose // call reads and closes the body
	res, raw := call(t, srv, http.MethodPost, "/api/v1/people/contacts",
		map[string]any{"display_name": "Lan Nguyen", "nickname": "Lan"}, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, body %s", res.StatusCode, raw)
	}
	if got := res.Header.Get("ETag"); got != `"1"` {
		t.Errorf("create ETag = %q, want \"1\"", got)
	}
	var created apigen.Contact
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	path := "/api/v1/people/contacts/" + created.Id.String()

	// Read.
	//nolint:bodyclose // call reads and closes the body
	res, raw = call(t, srv, http.MethodGet, path, nil, nil)
	if res.StatusCode != http.StatusOK || res.Header.Get("ETag") != `"1"` {
		t.Fatalf("get = %d ETag %q, body %s", res.StatusCode, res.Header.Get("ETag"), raw)
	}

	// Update: the version must match If-Match.
	//nolint:bodyclose // call reads and closes the body
	res, raw = call(t, srv, http.MethodPatch, path, map[string]any{"phone": "+84 90 000 0000"},
		map[string]string{"If-Match": `"1"`})
	if res.StatusCode != http.StatusOK || res.Header.Get("ETag") != `"2"` {
		t.Fatalf("patch = %d ETag %q, body %s", res.StatusCode, res.Header.Get("ETag"), raw)
	}

	tests := []struct {
		name     string
		method   string
		path     string
		body     any
		headers  map[string]string
		wantCode int
		wantProb string
	}{
		{"patch without If-Match is refused", http.MethodPatch, path, map[string]any{"notes": "x"}, nil, 412, "people.version_required"},
		{"patch with a stale version is refused", http.MethodPatch, path, map[string]any{"notes": "x"},
			map[string]string{"If-Match": `"1"`}, 412, "people.stale_version"},
		{"birthday that isn't a date is 422", http.MethodPost, "/api/v1/people/contacts",
			map[string]any{"display_name": "Minh", "birthday": "1990-02-30"}, nil, 422, "people.invalid_contact"},
		{"missing name is caught by the contract (400)", http.MethodPost, "/api/v1/people/contacts",
			map[string]any{"nickname": "M"}, nil, 400, "request.invalid"},
		{"unknown contact is 404", http.MethodGet, "/api/v1/people/contacts/00000000-0000-7000-8000-0000000000ff", nil, nil, 404, "people.contact_not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//nolint:bodyclose // call reads and closes the body
			res, raw := call(t, srv, tt.method, tt.path, tt.body, tt.headers)
			if res.StatusCode != tt.wantCode {
				t.Fatalf("status = %d, want %d; body %s", res.StatusCode, tt.wantCode, raw)
			}
			if got := problemCode(t, raw); got != tt.wantProb {
				t.Errorf("code = %q, want %q", got, tt.wantProb)
			}
		})
	}

	// List, then delete.
	//nolint:bodyclose // call reads and closes the body
	res, raw = call(t, srv, http.MethodGet, "/api/v1/people/contacts", nil, nil)
	var list apigen.ContactList
	if err := json.Unmarshal(raw, &list); err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("list = %d %v, body %s", res.StatusCode, err, raw)
	}
	if len(list.Items) != 1 || list.NextCursor != nil {
		t.Errorf("list = %d items, next %v; want 1 item and no next page", len(list.Items), list.NextCursor)
	}

	//nolint:bodyclose // call reads and closes the body
	res, _ = call(t, srv, http.MethodDelete, path, nil, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", res.StatusCode)
	}
	//nolint:bodyclose // call reads and closes the body
	res, _ = call(t, srv, http.MethodGet, path, nil, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", res.StatusCode)
	}
}
