-- +goose Up
CREATE TABLE ledgers(
    id uuid PRIMARY KEY,
    name varchar(255) NOT NULL,
    currency varchar(10) NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE TABLE members(
    ledger_id uuid NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    telegram_id bigint NOT NULL,
    name varchar(255) NOT NULL,
    role varchar(20) NOT NULL,
    joined_at timestamptz NOT NULL,
    PRIMARY KEY (ledger_id, telegram_id)
);

-- One person, one ledger. The bot never has to ask which budget a message is
-- about, because there is only ever one answer.
CREATE UNIQUE INDEX members_one_ledger_each ON members(telegram_id);

CREATE TABLE invites(
    code varchar(32) PRIMARY KEY,
    ledger_id uuid NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    created_by bigint NOT NULL,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    used_by bigint,
    used_at timestamptz
);

CREATE TYPE account_type AS ENUM(
    'cash',
    'bank_card',
    'investment',
    'deposit'
);

CREATE TABLE accounts(
    id uuid PRIMARY KEY,
    ledger_id uuid NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    name varchar(255) NOT NULL,
    balance_amount DECIMAL(18, 2) NOT NULL,
    balance_currency varchar(10) NOT NULL,
    account_type account_type NOT NULL,
    is_archived boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX accounts_ledger ON accounts(ledger_id);

CREATE TABLE categories(
    id uuid PRIMARY KEY,
    ledger_id uuid NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    name varchar(255) NOT NULL,
    parent_id uuid REFERENCES categories(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL
);

-- Typing "еда" and "Еда" must not end up as two categories in the report.
CREATE UNIQUE INDEX categories_unique_name ON categories(ledger_id, lower(name));

CREATE INDEX categories_parent ON categories(parent_id);

CREATE TABLE transactions(
    id uuid PRIMARY KEY,
    ledger_id uuid NOT NULL REFERENCES ledgers(id) ON DELETE CASCADE,
    created_by bigint NOT NULL,
    transaction_type varchar(20) NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    currency varchar(10) NOT NULL,
    from_account_id uuid REFERENCES accounts(id) ON DELETE RESTRICT,
    to_account_id uuid REFERENCES accounts(id) ON DELETE RESTRICT,
    category_id uuid REFERENCES categories(id) ON DELETE SET NULL,
    comment text,
    date timestamptz NOT NULL,
    -- An income has a destination, an expense a source, a transfer both.
    CONSTRAINT transactions_sides CHECK (
        (transaction_type = 'income' AND from_account_id IS NULL AND to_account_id IS NOT NULL)
        OR (transaction_type = 'expense' AND from_account_id IS NOT NULL AND to_account_id IS NULL)
        OR (transaction_type = 'transfer' AND from_account_id IS NOT NULL AND to_account_id IS NOT NULL
            AND from_account_id <> to_account_id))
);

CREATE INDEX transactions_ledger_date ON transactions(ledger_id, date DESC);

-- Postgres does not index the referencing side of a foreign key on its own,
-- and deleting an account has to check both of these.
CREATE INDEX transactions_from_account ON transactions(from_account_id);

CREATE INDEX transactions_to_account ON transactions(to_account_id);

CREATE INDEX transactions_category ON transactions(category_id);

-- +goose Down
DROP TABLE transactions;

DROP TABLE categories;

DROP TABLE accounts;

DROP TYPE account_type;

DROP TABLE invites;

DROP TABLE members;

DROP TABLE ledgers;
