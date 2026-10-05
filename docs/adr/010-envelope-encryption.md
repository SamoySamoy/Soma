# 010. Envelope encryption with Tink

- Status: Accepted
- Date: 2026-10-04

## Decision

A master key (from env or a file, later a cloud KMS) wraps one data key per
space. Data keys encrypt vault secret fields, TOTP secrets and files attached
to sensitive modules, using Google Tink's AEAD and streaming AEAD.

## Consequences

Vetted primitives and key rotation by key ID. Losing the master key makes
encrypted data unreadable, so backups and docs stress keeping it safe.
