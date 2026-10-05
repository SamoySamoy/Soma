# 003. OpenAPI-first contract

- Status: Accepted
- Date: 2026-10-04

## Context

The backend is the product; the web UI is one client of the API.

## Decision

`api/openapi.yaml` is written first. oapi-codegen generates the Go strict
server interface and models; openapi-typescript generates the TypeScript
types used by openapi-fetch. Generated code is committed and never edited by
hand. Every operation declares `x-soma-permission`, enforced by a test.

## Consequences

Server and client can't drift. Requests are validated against the contract
before handlers run. Contract changes always come first in a change.
