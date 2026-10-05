# 011. `.soma` package as the single export format

- Status: Accepted
- Date: 2026-10-04

## Decision

Export, import and partial exports (shards) all use one format: an
age-encrypted ZIP holding `manifest.json`, one JSON Lines file per record
kind, `links.jsonl`, files named by SHA-256, and the JSON Schemas for its
version. A public Go package, `pkg/somafile`, reads and writes it.

## Consequences

Users can always take all of their data elsewhere, and other tools can read
it without Soma's internals. The format is versioned, so changes need care.
