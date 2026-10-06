# 016. Money model for the MVP

- Status: Accepted
- Date: 2026-10-07

## Context

Money (FIN-01..08) must keep exact balances, handle several currencies, and
show budgets and net worth in one currency. The spec leaves several details
open.

## Decision

1. **Amounts are integers in minor units.** Each currency has a number of
   minor-unit digits (VND 0, USD 2, KWD 3, ...). The API carries `amount_minor`
   and the currency code; formatting uses the digits. No floating point.
2. **Signs come from the kind.** Income and transfer-in are positive; expense,
   transfer-out are negative. An adjustment may be either sign. A balance is
   `opening_minor + sum(amount_minor)`. Balances are never edited directly
   (FIN rule).
3. **A space has one base currency**, default VND. Budgets and totals are in
   the base currency. Accounts keep their own currency.
4. **Exchange rates are entered by hand**, per currency and day, as
   `rate_e8`: one unit of the currency equals `rate_e8 / 10^8` base major
   units. Conversion uses the latest rate on or before today, computed with
   integers and rounded half to even once. Automatic rate fetching is opt-in
   and comes later.
5. **Transfers stay in one currency** for the MVP. Cross-currency transfers
   need two amounts and come later.
6. **Categories are a two-level tree.** Expense and income categories are
   separate kinds. A space starts with defaults (Food, Transport, Bills, ...).
7. **Money is sensitive.** It is encrypted at the field level once the
   envelope encryption work (ADR-010) lands. Until then it is protected by
   the same row-level security as every other record.

## Consequences

- Balances reconcile by construction, and totals can be checked by hand.
- Converted net worth changes when rates change, as expected. Old figures are
  recomputed, not stored.
- Budget checks are simple. A budget counts expenses in its category and its
  child categories.
- Multi-currency transfers, investments, debts with schedules, and recurring
  transactions are deliberately deferred.
