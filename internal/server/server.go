// Package server assembles the HTTP handler: middleware, the generated API,
// health endpoints and the embedded web app.
package server

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"

	"github.com/SamoySamoy/Soma/internal/apigen"
	"github.com/SamoySamoy/Soma/internal/platform/clock"
	"github.com/SamoySamoy/Soma/internal/platform/config"
	"github.com/SamoySamoy/Soma/internal/platform/httpx"
)

// apiPrefix is where the versioned REST API lives.
const apiPrefix = "/api/"

// Deps are the server's collaborators.
type Deps struct {
	Config config.Config
	Logger *slog.Logger
	Clock  clock.Clock
	// Ready is checked by /readyz.
	Ready httpx.Readiness
	// SchemaVersion is the migration version this binary expects.
	SchemaVersion int64
	// Web is the built single-page app.
	Web fs.FS
}

// New returns the root HTTP handler.
func New(d Deps) (http.Handler, error) {
	spec, err := apigen.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("load embedded OpenAPI spec: %w", err)
	}
	validate, err := requestValidator(spec)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", httpx.Liveness())
	mux.Handle("GET /readyz", httpx.ReadinessHandler(d.Ready, d.SchemaVersion))

	strict := apigen.NewStrictHandlerWithOptions(&api{cfg: d.Config}, nil, apigen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  badRequest,
		ResponseErrorHandlerFunc: httpx.WriteError,
	})
	apigen.HandlerWithOptions(strict, apigen.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: badRequest,
	})
	mux.Handle(apiPrefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteProblem(w, r, httpx.Problem{Status: http.StatusNotFound, Code: "request.not_found",
			Detail: "No API endpoint matches this path."})
	}))
	mux.Handle("/", httpx.SPA(d.Web))

	return httpx.Chain(mux,
		httpx.RequestID(d.Logger),
		httpx.AccessLog(d.Clock),
		httpx.Recover(),
		httpx.SecurityHeaders(strings.HasPrefix(d.Config.BaseURL, "https://")),
		httpx.CrossOrigin(),
		validate,
	), nil
}

func badRequest(w http.ResponseWriter, r *http.Request, err error) {
	httpx.WriteProblem(w, r, httpx.Problem{Status: http.StatusBadRequest, Code: "request.invalid", Detail: err.Error()})
}

// requestValidator checks API requests against the OpenAPI contract before
// any handler runs. Non-API paths pass through untouched.
func requestValidator(spec *openapi3.T) (httpx.Middleware, error) {
	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		return nil, fmt.Errorf("build OpenAPI router: %w", err)
	}
	opts := &openapi3filter.Options{
		AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		MultiError:         true,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, apiPrefix) {
				next.ServeHTTP(w, r)
				return
			}
			route, params, err := router.FindRoute(r)
			if errors.Is(err, routers.ErrMethodNotAllowed) {
				httpx.WriteProblem(w, r, httpx.Problem{Status: http.StatusMethodNotAllowed, Code: "request.method_not_allowed",
					Detail: "This endpoint does not support the " + r.Method + " method."})
				return
			}
			if err != nil {
				// Unknown path: let the mux answer with its 404 problem.
				next.ServeHTTP(w, r)
				return
			}
			in := &openapi3filter.RequestValidationInput{Request: r, PathParams: params, Route: route, Options: opts}
			if err := openapi3filter.ValidateRequest(r.Context(), in); err != nil {
				badRequest(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}
