# 013. Tool pinning on a Windows dev machine

- Status: Accepted
- Date: 2026-10-05

## Context

The primary development machine runs Windows without a 64-bit C toolchain,
so cgo-based tools (sqlc, the race detector) can't build there. The newest
TypeScript (7.x) isn't yet supported by openapi-typescript or
typescript-eslint.

## Decision

- `task` is a tool in the root `go.mod`; other Go dev tools live in
  `tools/go.mod` and run with `go tool -modfile=tools/go.mod`.
- sqlc runs from its pinned Docker image.
- The race detector runs only on Linux, which includes CI.
- TypeScript is pinned to 5.9 until the generator and linter support 7.x.

## Consequences

Only Go, Node and Docker need installing. Race conditions are caught in CI
rather than on the Windows machine.
