-- +goose Up
CREATE TABLE users(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email varchar(255) UNIQUE NOT NULL,
    username varchar(50) UNIQUE NOT NULL,
    password_hash varchar NOT NULL,
    default_currency varchar(10) DEFAULT 'kzt',
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE TYPE account_type AS ENUM(
    'cash',
    'bank_card',
    'investment',
    'deposit'
);

CREATE TABLE accounts(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    name varchar(255) NOT NULL,
    balance_amount DECIMAL(18, 2) NOT NULL,
    balance_currency varchar(10) NOT NULL,
    account_type account_type NOT NULL,
    is_archived boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_accounts_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);

CREATE TABLE categories(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    name varchar(255) NOT NULL,
    parent_id uuid,
    icon varchar,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE SET NULL,
    CONSTRAINT fk_categories_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE transactions(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL,
    account_id uuid NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    currency varchar(10) NOT NULL,
    transaction_type varchar(20) NOT NULL,
    category_id uuid,
    description text,
    date timestamptz NOT NULL,
    tags varchar(50)[],
    to_account_id uuid,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_transactions_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_transactions_accounts FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    CONSTRAINT fk_transactions_to_accounts FOREIGN KEY (to_account_id) REFERENCES accounts(id) ON DELETE SET NULL,
    CONSTRAINT fk_transactions_categories FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);

CREATE INDEX idx_transactions_account_id ON transactions(account_id);

CREATE INDEX idx_transactions_date ON transactions(date);

-- +goose Down
DROP TABLE transactions;

DROP TABLE categories;

DROP TABLE accounts;

DROP TABLE users;

DROP TYPE account_type;

