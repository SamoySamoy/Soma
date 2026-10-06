-- Money (FIN-01..05, FIN-08). See ADR-006 (minor units) and ADR-016.

-- +goose Up
CREATE TABLE fin_settings (
    space_id      uuid PRIMARY KEY REFERENCES spaces (id),
    base_currency char(3) NOT NULL DEFAULT 'VND' CHECK (base_currency ~ '^[A-Z]{3}$'),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    version       integer NOT NULL DEFAULT 1 CHECK (version >= 1)
);

CREATE TABLE fin_accounts (
    id            uuid PRIMARY KEY REFERENCES entities (id) ON DELETE CASCADE,
    space_id      uuid NOT NULL REFERENCES spaces (id),
    name          text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    kind          text NOT NULL CHECK (kind IN ('cash', 'bank', 'credit_card', 'e_wallet', 'loan', 'investment')),
    currency      char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    opening_minor bigint NOT NULL DEFAULT 0,
    opened_on     date NOT NULL
);
CREATE INDEX fin_accounts_space_idx ON fin_accounts (space_id);

CREATE TABLE fin_categories (
    id         uuid PRIMARY KEY DEFAULT uuidv7(),
    space_id   uuid NOT NULL REFERENCES spaces (id),
    parent_id  uuid REFERENCES fin_categories (id),
    name       text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
    kind       text NOT NULL CHECK (kind IN ('expense', 'income')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX fin_categories_space_idx ON fin_categories (space_id);
CREATE INDEX fin_categories_parent_idx ON fin_categories (parent_id);

CREATE TABLE fin_transactions (
    id           uuid PRIMARY KEY REFERENCES entities (id) ON DELETE CASCADE,
    space_id     uuid NOT NULL REFERENCES spaces (id),
    account_id   uuid NOT NULL REFERENCES fin_accounts (id),
    kind         text NOT NULL CHECK (kind IN ('income', 'expense', 'transfer_in', 'transfer_out', 'adjustment')),
    amount_minor bigint NOT NULL CHECK (amount_minor <> 0),
    occurred_on  date NOT NULL,
    category_id  uuid REFERENCES fin_categories (id),
    payee        text CHECK (char_length(payee) <= 120),
    notes        text CHECK (char_length(notes) <= 2000),
    transfer_id  uuid
);
CREATE INDEX fin_transactions_space_date_idx ON fin_transactions (space_id, occurred_on DESC);
CREATE INDEX fin_transactions_account_idx ON fin_transactions (account_id, occurred_on);
CREATE INDEX fin_transactions_category_idx ON fin_transactions (category_id);
CREATE INDEX fin_transactions_transfer_idx ON fin_transactions (transfer_id);

CREATE TABLE fin_budgets (
    id           uuid PRIMARY KEY DEFAULT uuidv7(),
    space_id     uuid NOT NULL REFERENCES spaces (id),
    category_id  uuid NOT NULL REFERENCES fin_categories (id),
    month        date NOT NULL CHECK (extract(day FROM month) = 1),
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    UNIQUE (space_id, category_id, month)
);
CREATE INDEX fin_budgets_category_idx ON fin_budgets (category_id);

CREATE TABLE fin_rates (
    space_id uuid NOT NULL REFERENCES spaces (id),
    currency char(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    rate_on  date NOT NULL,
    rate_e8  bigint NOT NULL CHECK (rate_e8 > 0),
    PRIMARY KEY (space_id, currency, rate_on)
);

-- Row-level security: members read; owners and editors write.
ALTER TABLE fin_settings ENABLE ROW LEVEL SECURITY;
CREATE POLICY fin_settings_read ON fin_settings FOR SELECT USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY fin_settings_insert ON fin_settings FOR INSERT WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_settings_update ON fin_settings FOR UPDATE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor'))) WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_settings_delete ON fin_settings FOR DELETE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

ALTER TABLE fin_accounts ENABLE ROW LEVEL SECURITY;
CREATE POLICY fin_accounts_read ON fin_accounts FOR SELECT USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY fin_accounts_insert ON fin_accounts FOR INSERT WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_accounts_update ON fin_accounts FOR UPDATE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor'))) WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_accounts_delete ON fin_accounts FOR DELETE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

ALTER TABLE fin_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY fin_categories_read ON fin_categories FOR SELECT USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY fin_categories_insert ON fin_categories FOR INSERT WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_categories_update ON fin_categories FOR UPDATE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor'))) WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_categories_delete ON fin_categories FOR DELETE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

ALTER TABLE fin_transactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY fin_transactions_read ON fin_transactions FOR SELECT USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY fin_transactions_insert ON fin_transactions FOR INSERT WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_transactions_update ON fin_transactions FOR UPDATE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor'))) WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_transactions_delete ON fin_transactions FOR DELETE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

ALTER TABLE fin_budgets ENABLE ROW LEVEL SECURITY;
CREATE POLICY fin_budgets_read ON fin_budgets FOR SELECT USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY fin_budgets_insert ON fin_budgets FOR INSERT WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_budgets_update ON fin_budgets FOR UPDATE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor'))) WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_budgets_delete ON fin_budgets FOR DELETE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

ALTER TABLE fin_rates ENABLE ROW LEVEL SECURITY;
CREATE POLICY fin_rates_read ON fin_rates FOR SELECT USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id()));
CREATE POLICY fin_rates_insert ON fin_rates FOR INSERT WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_rates_update ON fin_rates FOR UPDATE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor'))) WITH CHECK (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));
CREATE POLICY fin_rates_delete ON fin_rates FOR DELETE USING (space_id IN (SELECT space_id FROM space_members WHERE user_id = soma_current_user_id() AND role IN ('owner', 'editor')));

-- +goose Down
DROP TABLE fin_rates;
DROP TABLE fin_budgets;
DROP TABLE fin_transactions;
DROP TABLE fin_categories;
DROP TABLE fin_accounts;
DROP TABLE fin_settings;
