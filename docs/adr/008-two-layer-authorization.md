# 008. Authorization in services plus row-level security

- Status: Accepted
- Date: 2026-10-04

## Context

Soma stores medical, financial and diary data for many users in one database.

## Decision

Services check permissions with `authz.Require` before any read or write.
PostgreSQL row-level security enforces space isolation again: the app
connects as `soma_app` (NOBYPASSRLS) and every transaction sets
`soma.user_id`. Operator endpoints use a role with no grants on content
tables. A generated test runs every operation as every role.

## Consequences

A single missed check can't leak another space's data. RLS adds some query
complexity, so repository tests always run with RLS on.
