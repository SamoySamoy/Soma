# 015. Body map built before its milestone

- Status: Accepted
- Date: 2026-10-06
- Amends: tech spec section 20 (M6 moved ahead of M3 to M5)

## Context

The body map is the product's signature screen and the owner asked for it
now, before the Self, Journal, Tasks and Money modules. The tech spec puts it
at M6, after those modules, because its status rules read their data.

## Decision

Build the body map now, in a reduced form:

1. `GET /api/v1/bodymap` returns every area with its phase, status, whether
   it is enabled, and counts. Clients draw from that response only (BODY rule).
2. Only Heart (People) reports a status. Its rule: attention when a birthday
   falls within the next 14 days, otherwise calm.
3. Every other area is reported as disabled (status `unknown`). The map shows
   it dimmed, and its page explains which phase delivers it (BODY-04).
4. Status providers run inside one read-only transaction, so an area's
   counts are consistent with each other. A provider failure fails the whole
   snapshot for now. Per-provider isolation with savepoints comes when a
   second provider exists.
5. The rules use the server's UTC date. A user timezone field is needed for
   BODY and the spec's timezone rule; it is planned with identity (M1), which
   owns user profiles.

## Consequences

- Later modules add a provider and a registry entry, and their areas light up
  without changing the API or the map.
- The M6 exit criterion (load under 500 ms with 100,000 records) is not yet
  measured. It stays on the M6 checklist.
