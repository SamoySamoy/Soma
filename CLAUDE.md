# Soma

Soma (Greek for "body") is a private, self-hostable record of a whole person. The landing page is a clickable body map; each region opens an area of life (Mind, Self, Heart, Money, ...). Go backend, React frontend, PostgreSQL. Runs locally for one person or in the cloud for many.

## Source of truth

- Business spec (SOMA-BIZ-001): https://claude.ai/artifact/2CJQpGo29ckqzRTZKBVNJW
- Tech spec (SOMA-TECH-001): https://claude.ai/artifact/Lkm7aMWD9w88ZopCwNFXg9
- Architecture decisions: `docs/adr/NNN-title.md`

Rules:
- Code implements the specs. Reference requirement IDs (`IAM-05`, `FIN-02`, `BODY-02`) in tests, PRs and commit bodies.
- If code needs to differ from the specs, stop and ask. Record any accepted change as a new ADR before writing the code.
- **Current phase: P1 (MVP).** Build milestones in order (tech spec §20). Don't start a milestone until the previous one meets its "done when" criteria.
- **Identity (M1) is deferred by decision (ADR-014).** Until it is built, the server runs only in `local` mode, as one implicit owner (`space.LocalOwnerID`), and refuses to start in any other mode. Don't add a non-local deployment path, and don't weaken the startup check, before M1.
- **Out of scope for now:** everything in Soma Mind / the simulation layer (RSI, insights, Ask Soma, embeddings, pgvector, any LLM call). Don't add code, tables, dependencies or config for it. P2/P3 features are also out of scope unless the user asks.

## Stack (pinned)

| Area | Choice |
|---|---|
| Go | 1.27 |
| Database | PostgreSQL 18 (extensions: `pg_trgm`, `unaccent` only) |
| HTTP | `net/http` + oapi-codegen strict server, kin-openapi validation |
| DB access | pgx v5, sqlc, goose migrations (embedded) |
| Jobs | River |
| Crypto | Argon2id (`x/crypto/argon2`), Tink, `pquerna/otp`, `filippo.io/age` |
| Logging | `log/slog` JSON |
| Frontend | Node 24 LTS, React 19, TypeScript (strict), Vite, Mantine, React Router, TanStack Query, openapi-typescript + openapi-fetch, react-i18next |
| Tests | Go `testing` + `go-cmp`, testcontainers-go, Vitest + Testing Library, Playwright |
| Tooling | Task (Taskfile), golangci-lint, govulncheck, ESLint, Prettier |

Dev tools are pinned, never installed globally:
- `task` is a `tool` directive in the root `go.mod`: run `go tool task <name>`.
- golangci-lint, goose, oapi-codegen and govulncheck are `tool` directives in `tools/go.mod`, kept out of the main module's dependency graph. The Taskfile runs them as `go tool -modfile=tools/go.mod <name>`.
- sqlc runs from its pinned Docker image (it needs cgo, which the Windows dev machine lacks).
- The race detector runs only on Linux (CI), for the same reason.

Don't add a dependency that isn't in this table or the tech spec without asking first. Prefer the standard library.

## Repository layout

```
cmd/soma/            main: serve, migrate, worker, backup, restore, admin
api/openapi.yaml     API contract (source of truth for server + client)
internal/platform/   config, db, httpx, crypto, blob, mail, clock, jobs, log, apperr (no business logic)
internal/iam/        users, sessions, mfa, tokens, password reset, lockout
internal/authz/      permissions, roles, policy checks
internal/space/      spaces, members, module settings
internal/core/       entity, link, graph, tag, attachment, search, audit, outbox, trash, reminder, timeline, export, pkgio
internal/bodymap/    area registry, status aggregation
internal/modules/    self, people, journal, tasks, money, assets (one package per module)
internal/testutil/   test helpers, builders, test database
pkg/somafile/        public reader/writer for .soma packages
migrations/          goose SQL files
queries/             sqlc .sql files, one folder per package
web/                 React app (build output embedded via go:embed)
deploy/              Dockerfile, compose files, Caddyfile
docs/adr/  docs/format/
```

## Commands

Run everything through Task: `go tool task <name>`.

| Task | Does |
|---|---|
| `dev` | Postgres in Docker + Go server with reload + Vite dev server |
| `gen` | Regenerate all code: oapi-codegen, sqlc, TypeScript client |
| `lint` | golangci-lint, `sqlc vet`, ESLint, Prettier check, `tsc --noEmit` |
| `test` | Go unit tests + Vitest (no Docker needed) |
| `test:integration` | Go tests with build tag `integration` (needs Docker) |
| `test:e2e` | Playwright against a running stack |
| `check` | `gen` + verify no diff, then `lint`, `test`, `test:integration`, govulncheck. **Run before declaring any task done.** |
| `migrate:new -- name` | Create a timestamped migration |
| `build` | Web build + Go binary with embedded web |

## Go conventions

**Formatting and lint**
- gofmt + goimports. golangci-lint must pass with zero issues. `//nolint` is allowed only with a specific linter and a reason: `//nolint:gosec // G404: non-crypto jitter`.

**Packages and layering**
- Layers inside a module: `handler` → `service` → `repository`. Handlers have no business logic. Only services call repositories. Only services open transactions.
- Modules never import another module's internals. Use `internal/core` or the other module's exported service interface. depguard enforces this.
- No packages named `util`, `common`, `helpers`, `misc`, `models`.
- Define interfaces where they are used, keep them small. Return concrete types.

**Functions and types**
- `ctx context.Context` is the first parameter of anything that does I/O. Never store a context in a struct.
- Constructors take dependencies explicitly (`NewService(repo Repo, clock clock.Clock, ...)`). No package-level mutable state, no `init()` side effects, no global singletons.
- Read configuration only in `internal/platform/config`. No `os.Getenv` anywhere else.
- Names: `ID`, `URL`, `HTTP`, `JSON` stay uppercase; short receiver names; no stutter (`money.Transaction`, not `money.MoneyTransaction`).
- Every exported identifier has a doc comment. Other comments explain *why*, not *what*.

**Errors**
- Wrap with context: `fmt.Errorf("create transaction: %w", err)`. Compare with `errors.Is` / `errors.As`.
- Domain errors use `internal/platform/apperr` (kinds: `NotFound`, `Invalid`, `Conflict`, `Unauthorized`, `Forbidden`, `PreconditionFailed`, `RateLimited`) with a stable machine `code`. The HTTP layer maps them to RFC 9457 problem details. Handlers never write raw error strings.
- Handle an error once: either return it or log it, never both.
- No `panic` outside `main` and impossible-state checks. Recovery middleware is a safety net, not control flow.

**Time, money, IDs**
- Never call `time.Now()` in domain or service code. Inject `clock.Clock`. Pass `now` into status rules and recurrence.
- Store timestamps as UTC `timestamptz`; calendar dates as `date`, interpreted in the user's IANA timezone.
- Money is `money.Amount{Minor int64, Currency string}`. **`float32`/`float64` must never hold money.** Exchange rates use `numeric` in SQL and a decimal type in Go, rounding half-to-even once at the end.
- IDs are UUIDv7 from Postgres `uuidv7()`; Go uses `github.com/google/uuid`.

**Concurrency**
- Use `errgroup` with a context. Every goroutine has an owner that waits for it and a way to stop. No fire-and-forget.

**Logging**
- `slog` via the request logger from context. Log keys are snake_case.
- **Never log** request or response bodies, record content, names, emails, tokens, passwords, secrets or keys. Log IDs only.

## Database conventions

- Migrations: goose SQL in `migrations/`, named `YYYYMMDDHHMMSS_snake_description.sql`, with `-- +goose Up` and `-- +goose Down`. **Never edit a migration that has been committed**; add a new one. Breaking changes use expand → migrate → contract across releases.
- Every record table:
  - has `space_id uuid NOT NULL` and an RLS policy, and `ENABLE ROW LEVEL SECURITY` in the same migration;
  - module record tables use `id uuid PRIMARY KEY REFERENCES entities(id)` (the entity spine);
  - indexes every foreign key and every column used in a `WHERE` or `ORDER BY`.
- Types: `text` + `CHECK` instead of enum types; `timestamptz` for instants; `date` for calendar days; `bigint` for money minor units; `jsonb` only for fields that are never filtered on.
- SQL lives in `queries/` and goes through sqlc. **No SQL built with string concatenation or `fmt.Sprintf`.** No ORM.
- Use the `db.InTx(ctx, func(tx pgx.Tx) error)` helper. It sets `soma.user_id` for RLS. The app connects as `soma_app` (no BYPASSRLS); migrations run as the owner role.
- Every new table gets an integration test proving another space's user can't read or write its rows.

## API conventions

- **Contract first.** Change `api/openapi.yaml`, run `go tool task gen`, then implement. Generated code is committed and **never edited by hand**.
- Every operation has an `operationId` (camelCase) and `x-soma-permission` (`module:action`). The authz matrix test fails if either is missing.
- Paths: `/api/v1/<module>/<resource>`, plural nouns, kebab-case. JSON fields: snake_case.
- Errors: `application/problem+json` with `code` and per-field `errors`.
- Lists: cursor pagination (`limit` ≤ 200, `cursor`, `next_cursor`). Filters are explicit query parameters.
- Updates: `PATCH` with JSON Merge Patch and `If-Match` (ETag from `entities.version`), else `412`.
- `POST` accepts `Idempotency-Key`; it is required on money endpoints.
- Money: `{"amount_minor": -125000, "currency": "VND"}`. Dates: `YYYY-MM-DD`. Instants: RFC 3339 UTC.
- A record the caller can't access returns `404`, never `403`.

## Security rules (non-negotiable)

- Check authorization in the **service** layer with `authz.Require(...)` before any read or write, even though RLS also protects the data.
- Sessions are opaque random tokens in an HttpOnly, Secure, SameSite=Lax cookie; only the SHA-256 hash is stored. No JWT. No tokens in `localStorage`.
- Passwords: Argon2id. Secrets compared with `crypto/subtle`. Random values come from `crypto/rand` only.
- Sensitive fields and files are encrypted through `internal/platform/crypto` (Tink). Never hand-roll crypto.
- Sensitive actions (export, delete account, change email, disable 2FA, reveal secrets) require recent re-authentication.
- Auth endpoints don't reveal whether an email exists.
- Security events go to the audit log in the same transaction as the action.
- No secrets in code, tests, fixtures or logs. Config comes from env; keep `.env.example` current.

## Testing conventions

- **Every change ships with tests.** A bug fix starts with a failing regression test.
- Unit tests: stdlib `testing` + `go-cmp`. Table-driven with named cases, `t.Parallel()` where safe. No testify, no mocking frameworks; write small hand-made fakes that implement the consumer's interface.
- Test names describe behaviour: `TestCreateTransaction/rejects zero amount`.
- Integration tests (`//go:build integration`): real Postgres 18 via testcontainers, one container per package, a fresh database per test cloned from a migrated template. Repository tests run with RLS on, as `soma_app`.
- API tests go through the full middleware stack with `httptest` and validate responses against the OpenAPI spec.
- The **authorization matrix** (every operation × anonymous, operator, owner, editor, viewer, other-space user, wrong-scope token) must cover every new endpoint.
- Time-dependent tests use a fake clock. **No `time.Sleep` in tests.**
- Test data comes from builders in `internal/testutil` (`testutil.NewContact(t, db, opts...)`). No shared mutable fixtures, no dependence on test order.
- Fuzz tests for parsers (CSV import, money parsing, RRULE).
- Coverage: at least 80% for `service` packages. Don't write tests just to raise a number.
- Frontend: Vitest + Testing Library for components with logic (test behaviour, query by role or label). Playwright for the critical journeys listed in the tech spec.

## Frontend conventions

- TypeScript `strict`; no `any`, no `@ts-ignore` without a reason comment. ESLint + Prettier must pass.
- Call the API only through the generated client in `web/src/api`. Never hand-write request or response types.
- Server state in TanStack Query; no global state library. Mutations invalidate affected queries, including `bodymap`.
- Use Mantine components and its theme. Write custom CSS only when Mantine can't do it (the body map, digital rain).
- Folders by area: `web/src/areas/<area>/`; shared UI in `web/src/components/`.
- Every user-facing string goes through i18next (`en` only for now). No string literals in JSX.
- Accessibility: every control has a label, everything is keyboard-usable, the body map has a list alternative.
- Never use `dangerouslySetInnerHTML`. Render Markdown with HTML disabled.

## Git

- Conventional Commits: `feat(money): add transfers (FIN-02)`, `fix(iam): ...`, `test`, `refactor`, `docs`, `chore`, `build`, `ci`.
- One logical change per commit. Generated code is committed in the same commit as its source.
- The default branch is `master` and must always be green. While Soma has a single developer, finished sections are committed and pushed straight to `master` after `go tool task check` passes. Once there are other contributors, work moves to branches named `m<N>/<short-description>` with pull requests.
- Never commit `.env`, keys, database dumps or `.soma` files.

## Definition of done

A task is done only when all of these hold:
1. `go tool task check` passes, with the output read, not assumed.
2. New behaviour has tests at the right levels, including the authz matrix for new endpoints.
3. OpenAPI, migrations and generated code are in sync.
4. Requirement IDs are referenced; any spec deviation has an ADR.
5. No TODOs without an issue reference; no commented-out code; no debug logging.
