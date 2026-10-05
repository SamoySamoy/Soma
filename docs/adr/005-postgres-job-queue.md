# 005. Background jobs in PostgreSQL with River

- Status: Accepted
- Date: 2026-10-04

## Context

Reminders, email and exports need durable background jobs, and a reminder
must fire at most once.

## Decision

Use River, a Go job queue backed by PostgreSQL. Jobs are inserted in the same
transaction as the data that caused them. No Redis.

## Consequences

One fewer service to run. Job insertion is atomic with business writes.
Throughput is bounded by Postgres, which is ample for personal data.
