CREATE TYPE sowhat.platform AS ENUM (
    'TELEGRAM'
);

CREATE table IF NOT EXISTS sowhat.users(
    id BIGSERIAL PRIMARY KEY,
    external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    platform sowhat.platform NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()    
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_external_id_platform ON sowhat.users (external_id, platform);
