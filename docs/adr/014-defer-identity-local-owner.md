# 014. Defer identity; run as one implicit local owner until M1

- Status: Accepted
- Date: 2026-10-06
- Amends: tech spec section 20 (milestone order), business spec IAM requirements for the first release

## Context

The owner wants to build the core features first (the people area with real
data and a UI), and add sign-in later. The tech spec puts identity in M1 and
requires authorization before any feature. Until identity exists, nothing
can be safely reachable from the network, so the restriction has to be
enforced by the code, not by convention.

## Decision

1. Feature work may proceed before M1. M1 (identity) stays required before
   any non-local deployment.
2. Until M1 is built, the server runs only in `local` mode. It refuses to
   start in `private` or `public` mode.
3. In local mode every request acts as one implicit owner, `space.LocalOwnerID`,
   in one personal space, `space.LocalSpaceID`. A middleware sets both on the
   request context. M1 replaces that middleware with sessions. Services and
   row-level security don't change.
4. The schema includes `users`, `spaces` and `space_members` from the start,
   so M1 and M2 add behaviour rather than tables. Row-level security policies
   already separate spaces and separate read from write by role.
5. `authz.Require` still runs in every service. The local owner holds every
   permission, so the checks are exercised, not bypassed.

## Consequences

- The authorization matrix test covers the roles that can exist without
  sessions: owner, editor, viewer and a user of another space. The anonymous
  and wrong-scope-token cases are added with M1.
- Parts of M2 (spaces, membership, row-level security for isolation) arrive
  early. The exit criterion for M2 (a second user can't see or change the
  first user's data) is covered by integration tests now.
- Running Soma in the cloud is impossible until M1 is done, by design.
