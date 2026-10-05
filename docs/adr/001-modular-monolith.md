# 001. Modular monolith in one Go binary

- Status: Accepted
- Date: 2026-10-04

## Context

Soma is built by one developer, must run locally as a single process, and
also scale to many users in the cloud.

## Decision

One Go binary (`cmd/soma`) serves the API, the web app and background
workers. Code is split into modules under `internal/modules` with strict
boundaries: handler → service → repository inside a module, and no module
imports another module's internals. depguard enforces this in lint.

## Consequences

Simple to run, test and deploy. The binary is stateless, so the cloud runs
several copies behind a load balancer. A module could be split into its own
service later, but only if a real need appears.
