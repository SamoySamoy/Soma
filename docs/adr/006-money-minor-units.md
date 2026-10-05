# 006. Money as integer minor units

- Status: Accepted
- Date: 2026-10-04

## Context

Floating-point arithmetic can't represent money exactly.

## Decision

Amounts are `bigint` minor units plus an ISO 4217 currency code, in SQL, Go
(`money.Amount`) and JSON (`amount_minor`). Exchange rates are `numeric`,
rounded half-to-even once at the end of a conversion.

## Consequences

Exact arithmetic and simple sums. Formatting needs each currency's
minor-unit exponent, from a seeded ISO 4217 table.
