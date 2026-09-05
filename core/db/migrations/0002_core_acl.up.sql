CREATE TABLE core_acl_areas (
    id SERIAL PRIMARY KEY,
    app VARCHAR(50) NOT NULL,
    area_key VARCHAR(50) NOT NULL,
    label VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (app, area_key)
);

CREATE TABLE core_acl_grants (
    id SERIAL PRIMARY KEY,
    subject_type VARCHAR(10) NOT NULL,
    subject_id VARCHAR(50) NOT NULL,
    app VARCHAR(50) NOT NULL,
    area_key VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (subject_type, subject_id, app, area_key),
    CONSTRAINT core_acl_grants_subject_type_check CHECK (subject_type IN ('user', 'role'))
);

CREATE INDEX idx_core_acl_grants_lookup ON core_acl_grants (app, area_key, subject_type, subject_id);
