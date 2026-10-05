# 002. PostgreSQL 18 only, with a graph-shaped entity spine

- Status: Accepted
- Date: 2026-10-04

## Context

A graph database was considered because Soma's data is connected. Most data
(money, tasks, health) needs strict transactions and constraints. The
open-source graph field in 2026 is thin: Memgraph, ArangoDB and SurrealDB are
BSL-licensed, Neo4j and ArcadeDB need a JVM, and Kuzu was archived in October
2025. SQL/PGQ was removed from PostgreSQL 19 before release.

## Decision

PostgreSQL 18 is the only database. Every record has a row in `entities`
(nodes) and connections live in `links` (typed edges). Graph queries go
through a Go `graph` package implemented with recursive SQL.

## Consequences

One source of truth, one backup, and row-level security for isolation
between users. Upgrade paths if graph needs grow: SQL/PGQ when it lands
(PostgreSQL 20 at the earliest), Apache AGE fed from the outbox, or an
external graph projection.
