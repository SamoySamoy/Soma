# 009. React SPA embedded in the Go binary

- Status: Accepted
- Date: 2026-10-04

## Decision

The React app is built with Vite into `web/dist` and embedded with
`go:embed`. The Go server serves it on the same origin as the API. In
development, Vite proxies `/api` to the Go server.

## Consequences

Local mode is one binary plus PostgreSQL. Cookies stay same-origin and no
CORS is needed. A fresh clone builds the Go binary with a placeholder page
until `go tool task web:build` runs.
