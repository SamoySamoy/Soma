# 012. OpenAPI 3.0.3 instead of 3.1

- Status: Accepted
- Date: 2026-10-05
- Amends: tech spec section 4, which named OpenAPI 3.1

## Context

oapi-codegen v2 and kin-openapi, which generate the server and validate
requests, fully support OpenAPI 3.0 but only part of 3.1.

## Decision

Write the contract in OpenAPI 3.0.3.

## Consequences

Nothing Soma needs is missing from 3.0.3. Revisit when the generator fully
supports 3.1.
