-- name: InsertSettings :exec
INSERT INTO fin_settings (space_id, updated_at)
VALUES (@space_id, @now)
ON CONFLICT (space_id) DO NOTHING;

-- name: GetSettings :one
SELECT base_currency, updated_at, version
FROM fin_settings
WHERE space_id = @space_id;

-- name: UpdateSettings :one
-- Returns no rows when the expected version is stale, which the caller maps to 412.
UPDATE fin_settings
SET base_currency = @base_currency, updated_at = @now, version = version + 1
WHERE space_id = @space_id AND version = @expected_version
RETURNING version;

-- name: InsertAccount :exec
INSERT INTO fin_accounts (id, space_id, name, kind, currency, opening_minor, opened_on)
VALUES (@id, @space_id, @name, @kind, @currency, @opening_minor, @opened_on);

-- name: GetAccount :one
SELECT a.id, a.name, a.kind, a.currency, a.opening_minor, a.opened_on, e.created_at, e.updated_at, e.version
FROM fin_accounts a
JOIN entities e ON e.id = a.id
WHERE a.id = @id AND a.space_id = @space_id AND e.deleted_at IS NULL;

-- name: ListAccounts :many
SELECT a.id, a.name, a.kind, a.currency, a.opening_minor, a.opened_on, e.created_at, e.updated_at, e.version
FROM fin_accounts a
JOIN entities e ON e.id = a.id
WHERE a.space_id = @space_id AND e.deleted_at IS NULL
ORDER BY a.opened_on, a.name;

-- name: UpdateAccount :exec
UPDATE fin_accounts
SET name = @name, kind = @kind, opening_minor = @opening_minor, opened_on = @opened_on
WHERE id = @id AND space_id = @space_id;

-- name: AccountBalances :many
-- Opening balance plus every live transaction on the account. Deleted rows are
-- excluded by the LEFT JOIN on entities, so they contribute nothing.
SELECT a.id, (a.opening_minor + COALESCE(SUM(CASE WHEN tx_e.id IS NULL THEN 0 ELSE t.amount_minor END), 0))::bigint AS balance_minor
FROM fin_accounts a
JOIN entities ae ON ae.id = a.id AND ae.deleted_at IS NULL
LEFT JOIN fin_transactions t ON t.account_id = a.id
LEFT JOIN entities tx_e ON tx_e.id = t.id AND tx_e.deleted_at IS NULL
WHERE a.space_id = @space_id
GROUP BY a.id, a.opening_minor;

-- name: InsertCategory :exec
INSERT INTO fin_categories (id, space_id, parent_id, name, kind, created_at)
VALUES (@id, @space_id, @parent_id, @name, @kind, @now);

-- name: GetCategory :one
SELECT id, parent_id, name, kind
FROM fin_categories
WHERE id = @id AND space_id = @space_id;

-- name: ListCategories :many
SELECT id, parent_id, name, kind
FROM fin_categories
WHERE space_id = @space_id
ORDER BY parent_id NULLS FIRST, name;

-- name: CountCategories :one
SELECT count(*)::bigint AS total
FROM fin_categories
WHERE space_id = @space_id;

-- name: InsertTransaction :exec
INSERT INTO fin_transactions (id, space_id, account_id, kind, amount_minor, occurred_on, category_id, payee, notes, transfer_id)
VALUES (@id, @space_id, @account_id, @kind, @amount_minor, @occurred_on, @category_id, @payee, @notes, @transfer_id);

-- name: GetTransaction :one
SELECT t.id, t.account_id, t.kind, t.amount_minor, t.occurred_on, t.category_id, t.payee, t.notes, t.transfer_id,
       e.created_at, e.updated_at, e.version
FROM fin_transactions t
JOIN entities e ON e.id = t.id
WHERE t.id = @id AND t.space_id = @space_id AND e.deleted_at IS NULL;

-- name: ListTransactions :many
-- Newest first. The cursor is the (date, ID) of the last row seen.
SELECT t.id, t.account_id, t.kind, t.amount_minor, t.occurred_on, t.category_id, t.payee, t.notes, t.transfer_id,
       e.created_at, e.updated_at, e.version
FROM fin_transactions t
JOIN entities e ON e.id = t.id
WHERE t.space_id = @space_id AND e.deleted_at IS NULL
  AND (sqlc.narg('account_id')::uuid IS NULL OR t.account_id = sqlc.narg('account_id')::uuid)
  AND (sqlc.narg('before_date')::date IS NULL
       OR (t.occurred_on, t.id) < (sqlc.narg('before_date')::date, sqlc.narg('before_id')::uuid))
ORDER BY t.occurred_on DESC, t.id DESC
LIMIT @row_limit;

-- name: UpdateTransaction :exec
UPDATE fin_transactions
SET amount_minor = @amount_minor, occurred_on = @occurred_on, category_id = @category_id,
    payee = @payee, notes = @notes
WHERE id = @id AND space_id = @space_id;

-- name: TransferLegs :many
SELECT id FROM fin_transactions WHERE transfer_id = @transfer_id AND space_id = @space_id;

-- name: UpsertBudget :exec
INSERT INTO fin_budgets (id, space_id, category_id, month, amount_minor)
VALUES (uuidv7(), @space_id, @category_id, @month, @amount_minor)
ON CONFLICT (space_id, category_id, month) DO UPDATE SET amount_minor = EXCLUDED.amount_minor;

-- name: DeleteBudget :execrows
DELETE FROM fin_budgets
WHERE space_id = @space_id AND category_id = @category_id AND month = @month;

-- name: MonthBudgets :many
-- Every expense category with the budget set for the month (base currency).
SELECT c.id AS category_id, c.name, c.parent_id, COALESCE(b.amount_minor, 0)::bigint AS budget_minor
FROM fin_categories c
LEFT JOIN fin_budgets b ON b.category_id = c.id AND b.month = @month AND b.space_id = c.space_id
WHERE c.space_id = @space_id AND c.kind = 'expense'
ORDER BY c.parent_id NULLS FIRST, c.name;

-- name: MonthExpenseByCategory :many
-- What was spent in each category this month, per account currency. Go converts
-- the amounts to the base currency, so mixed currencies never add up raw.
SELECT t.category_id, a.currency, SUM(-t.amount_minor)::bigint AS spent_minor
FROM fin_transactions t
JOIN entities tx_e ON tx_e.id = t.id AND tx_e.deleted_at IS NULL
JOIN fin_accounts a ON a.id = t.account_id
WHERE t.space_id = @space_id AND t.kind = 'expense' AND t.category_id IS NOT NULL
  AND t.occurred_on >= @month_start AND t.occurred_on < @month_end
GROUP BY t.category_id, a.currency;

-- name: MonthFlows :many
-- Income and expense for the month, per account currency. Transfers are not income.
SELECT a.currency, t.kind, SUM(t.amount_minor)::bigint AS total_minor
FROM fin_transactions t
JOIN entities tx_e ON tx_e.id = t.id AND tx_e.deleted_at IS NULL
JOIN fin_accounts a ON a.id = t.account_id
WHERE t.space_id = @space_id AND t.kind IN ('income', 'expense')
  AND t.occurred_on >= @month_start AND t.occurred_on < @month_end
GROUP BY a.currency, t.kind;

-- name: UpsertRate :exec
INSERT INTO fin_rates (space_id, currency, rate_on, rate_e8)
VALUES (@space_id, @currency, @rate_on, @rate_e8)
ON CONFLICT (space_id, currency, rate_on) DO UPDATE SET rate_e8 = EXCLUDED.rate_e8;

-- name: LatestRates :many
-- The newest rate on or before the given day, per currency.
SELECT DISTINCT ON (currency) currency, rate_on, rate_e8
FROM fin_rates
WHERE space_id = @space_id AND rate_on <= @as_of
ORDER BY currency, rate_on DESC;
