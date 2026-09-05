CREATE TABLE se_balance_types (
    code VARCHAR(20) PRIMARY KEY,
    short_description VARCHAR(100) NOT NULL,
    column_indicator VARCHAR(10) NOT NULL,
    full_description TEXT NOT NULL,
    validation_rule VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT se_balance_types_validation_rule_check CHECK (
        validation_rule IN ('credit_only', 'debit_only', 'both_separate', 'both_net', 'both_before_transfer')
    )
);

CREATE INDEX idx_se_balance_types_column ON se_balance_types (column_indicator);

CREATE TABLE se_expected_balances (
    id SERIAL PRIMARY KEY,
    account_number VARCHAR(20) UNIQUE NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    expected_balance_type VARCHAR(20) NOT NULL,
    balance_type_code VARCHAR(20) REFERENCES se_balance_types (code),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_se_expected_balances_account ON se_expected_balances (account_number);

CREATE TABLE se_uploads (
    id SERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    label VARCHAR(500),
    upload_date TIMESTAMPTZ NOT NULL DEFAULT now(),
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
);

CREATE TABLE se_processed_records (
    id SERIAL PRIMARY KEY,
    upload_id INTEGER NOT NULL REFERENCES se_uploads (id) ON DELETE CASCADE,
    account_number VARCHAR(20) NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    balance DECIMAL(15, 2) NOT NULL,
    expected_rule_account VARCHAR(20),
    expected_balance_type VARCHAR(20),
    balance_type_code VARCHAR(20) REFERENCES se_balance_types (code),
    is_correct BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_se_processed_records_upload ON se_processed_records (upload_id);
CREATE INDEX idx_se_processed_records_balance_type ON se_processed_records (balance_type_code);

-- user_id aponta para core_users, não para uma tabela de utilizadores própria deste
-- módulo — a identidade é sempre resolvida via Cloudflare Access + core.
CREATE TABLE se_audit_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES core_users (id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    entity VARCHAR(50) NOT NULL,
    entity_id VARCHAR(50),
    details TEXT,
    ip_address VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_se_audit_logs_user ON se_audit_logs (user_id);
CREATE INDEX idx_se_audit_logs_created_at ON se_audit_logs (created_at);
