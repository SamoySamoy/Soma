# 004. Server-side sessions, no JWT

- Status: Accepted
- Date: 2026-10-04

## Context

Users must be able to revoke a session or sign out everywhere instantly
(IAM-02, IAM-06), and JavaScript should never hold a credential.

## Decision

The web app uses an opaque random token in an HttpOnly, Secure, SameSite=Lax
cookie; the database stores only its SHA-256 hash. Scripts use scoped
personal access tokens sent as bearer tokens.

## Consequences

Revocation takes effect immediately. Each request does one indexed session
lookup, which is cheap at Soma's scale.
