# Architecture decision records

Each file records one decision: the context, what was decided, and the
consequences. Records are never rewritten; a later record supersedes an
earlier one. Copy `000-template.md` to add one.

| ADR | Decision | Status |
|---|---|---|
| [001](001-modular-monolith.md) | Modular monolith in one Go binary | Accepted |
| [002](002-postgresql-entity-spine.md) | PostgreSQL 18 only, with a graph-shaped entity spine | Accepted |
| [003](003-openapi-first.md) | OpenAPI-first contract with generated server and client | Accepted |
| [004](004-server-side-sessions.md) | Server-side sessions, no JWT | Accepted |
| [005](005-postgres-job-queue.md) | Background jobs in PostgreSQL with River | Accepted |
| [006](006-money-minor-units.md) | Money as integer minor units | Accepted |
| [007](007-uuidv7.md) | UUIDv7 primary keys | Accepted |
| [008](008-two-layer-authorization.md) | Authorization in services plus row-level security | Accepted |
| [009](009-embedded-spa.md) | React SPA embedded in the Go binary | Accepted |
| [010](010-envelope-encryption.md) | Envelope encryption with Tink | Accepted |
| [011](011-soma-package-format.md) | `.soma` package as the single export format | Accepted |
| [012](012-openapi-3-0.md) | OpenAPI 3.0.3 instead of 3.1 | Accepted |
| [013](013-tooling-pins.md) | Tool pinning on a Windows dev machine | Accepted |
