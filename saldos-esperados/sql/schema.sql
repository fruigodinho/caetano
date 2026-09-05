-- Espelho só de leitura, para o sqlc conseguir validar/gerar sql/queries/*.sql.
--
-- A autoridade real do schema é core/db/migrations/ (schema único "caetano",
-- partilhado por todos os módulos). Este ficheiro NUNCA é aplicado a uma base de
-- dados (não corre nenhuma migração) - se o schema em core/db/migrations mudar,
-- atualizar aqui manualmente antes de correr `sqlc generate`.

CREATE TABLE core_users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE se_balance_types (
    code VARCHAR(20) PRIMARY KEY,
    short_description VARCHAR(100) NOT NULL,
    column_indicator VARCHAR(10) NOT NULL,
    full_description TEXT NOT NULL,
    validation_rule VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE se_expected_balances (
    id SERIAL PRIMARY KEY,
    account_number VARCHAR(20) UNIQUE NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    expected_balance_type VARCHAR(20) NOT NULL,
    balance_type_code VARCHAR(20) REFERENCES se_balance_types (code),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

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
